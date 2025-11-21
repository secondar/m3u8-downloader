package web

import (
	"M3u8Download/downloader"
	"M3u8Download/sqlite"
	"M3u8Download/utils"
	"M3u8Download/utils/chromium"
	"M3u8Download/utils/m3u8"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var Sqlite *sqlite.Sqlite
var TaskThreadPool *downloader.TaskThreadPool
var Token string
var chromiumManager *chromium.Manager

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Run() {
	var err error
	_ = utils.ClearDirectory(utils.GetCacheDir())
	// 创建Chromium管理器
	chromiumManager = chromium.NewChromiumManager(utils.GetDataDir())
	if !chromiumManager.CheckChromiumInstalled() {
		utils.Info("正在安装UnGoogled Chromium...")
		err = chromiumManager.InstallUnGoogledChromium()
		if err != nil {
			utils.Error(errors.New(fmt.Sprintf("UnGoogled Chromium 安装失败：%s", err.Error())))
		}
	}
	Sqlite, err = sqlite.GetSqliteCxt()
	if err != nil {
		utils.Error(err)
		os.Exit(0)
	}
	var key, value string
	var rows *sql.Rows
	rows, err = Sqlite.Cxt.Query(fmt.Sprintf("SELECT * FROM conf where `key` = 'Token'"))
	if err != nil {
		utils.Error(err)
		os.Exit(0)
	}
	rows.Next()
	err = rows.Scan(&key, &value)
	_ = rows.Close()
	if err != nil {
		utils.Error(err)
		os.Exit(0)
	}
	Token = value
	rows, err = Sqlite.Cxt.Query(fmt.Sprintf("SELECT * FROM conf where `key` = 'maxWorkers'"))
	if err != nil {
		utils.Error(err)
		os.Exit(0)
	}
	rows.Next()
	err = rows.Scan(&key, &value)
	_ = rows.Close()
	if err != nil {
		utils.Error(err)
		os.Exit(0)
	}
	var maxWorkers int
	maxWorkers, err = strconv.Atoi(value)
	if err != nil || maxWorkers <= 0 {
		maxWorkers = 5
	}
	_, _ = Sqlite.Cxt.Exec(`UPDATE task SET Status = ? WHERE Status = ?`, 0, 1)
	TaskThreadPool = downloader.NewTaskThreadPool(maxWorkers, Sqlite)
	_ = TaskThreadPool.RunTask()
	mux := http.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/checkToken", checkToken)
	mux.HandleFunc("/editToken", editToken)
	mux.HandleFunc("/add", add)
	mux.HandleFunc("/del", del)
	mux.HandleFunc("/delAll", delAll)
	mux.HandleFunc("/stop", stop)
	mux.HandleFunc("/stopAll", stopAll)
	mux.HandleFunc("/list", list)
	mux.HandleFunc("/cleanUp", cleanUp)
	mux.HandleFunc("/getConf", getConf)
	mux.HandleFunc("/setConf", setConf)
	mux.HandleFunc("/checkForUpdates", checkForUpdates)
	mux.HandleFunc("/extractM3U8ByUrl", extractM3U8ByUrl)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	handler := TokenMiddleware(mux)
	utils.Info(fmt.Sprintf("M3u8Download服务已启动，请访问：http://localhost:65533"))
	err = http.ListenAndServe(":65533", handler)
	if err != nil {
		panic(err)
	}
}
func TokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 添加CORS头部
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Token")
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// 定义不需要token验证的路径
		exemptPaths := []string{"/", "/checkToken", "/static/"}
		// 检查当前路径是否在豁免列表中
		shouldExempt := false
		for _, path := range exemptPaths {
			if r.URL.Path == path || (path == "/static/" && strings.HasPrefix(r.URL.Path, "/static/")) {
				shouldExempt = true
				break
			}
		}
		// 如果不在豁免列表中，则验证token
		if !shouldExempt {
			token := r.Header.Get("X-Token")
			if token == "" {
				http.Error(w, "Missing X-Token", http.StatusUnauthorized)
				return
			}
			if token != Token {
				http.Error(w, "Invalid X-Token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), "token", token)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			// 豁免路径直接通过
			next.ServeHTTP(w, r)
		}
	})
}
func checkToken(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	token = strings.Trim(token, "\n")
	token = strings.Trim(token, "\r")
	token = strings.Trim(token, " ")
	if token != Token {
		http.Error(w, "Invalid X-Token", http.StatusUnauthorized)
		return
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "验证成功"}))
}
func editToken(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	token = strings.Trim(token, "\n")
	token = strings.Trim(token, "\r")
	token = strings.Trim(token, " ")
	if token == "" {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "token不能为空"}))
		return
	}
	_, err = Sqlite.Cxt.Exec("UPDATE conf SET value = ? WHERE key = ?", token, "Token")
	if err != nil {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: fmt.Sprintf("修改Token失败：%s", err.Error())}))
		return
	}
	Token = token
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "修改Token成功"}))
}
func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, filepath.Join("./static", "index.html"))
}
func add(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	SavePath := r.FormValue("SavePath")
	Filename := r.FormValue("Filename")
	Url := r.FormValue("Url")
	UserAgent := r.FormValue("UserAgent")
	Cookie := r.FormValue("Cookie")
	Origin := r.FormValue("Origin")
	Referer := r.FormValue("Referer")
	Proxy := r.FormValue("Proxy")
	Threads := r.FormValue("Threads")
	SavePath = strings.Trim(SavePath, "\n")
	SavePath = strings.Trim(SavePath, "\r")
	SavePath = strings.Trim(SavePath, " ")
	if SavePath == "" {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "保存地址必须填写"}))
		return
	}
	if utils.IsDockerByCGroup() {
		SavePath = fmt.Sprintf("/download/%s", SavePath)
	}
	Filename = strings.Trim(Filename, "\n")
	Filename = strings.Trim(Filename, "\r")
	Filename = strings.Trim(Filename, " ")
	if Filename == "" {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "文件名不能为空"}))
		return
	}
	if len(Filename) < 4 || !strings.HasSuffix(strings.ToLower(Filename), ".mp4") {
		Filename += ".mp4"
	}
	Url = strings.Trim(Url, "\n")
	Url = strings.Trim(Url, "\r")
	Url = strings.Trim(Url, " ")
	if Url == "" {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "Url不能为空"}))
		return
	}
	var threads int
	threads, err = strconv.Atoi(Threads)
	if err != nil || threads <= 0 || threads > 64 {
		threads = 64
	}
	// 解析m3u8
	client := &http.Client{}
	Transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if Proxy != "" {
		var proxyURL *url.URL
		proxyURL, err = url.Parse(Proxy)
		if err == nil {
			Transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	client.Transport = Transport
	var m3u8Info *m3u8.Info
	m3u8Info, err = m3u8.ParseM3U8(client, Url, UserAgent, Cookie, Origin, Referer)
	if err != nil {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "任务添加失败：" + err.Error()}))
		return
	}
	if m3u8Info.Type != m3u8.MASTER && m3u8Info.Type != m3u8.VOD {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: fmt.Sprintf("不支持的M3u8类型：%s", m3u8Info.Type)}))
		return
	}
	if m3u8Info.Type == m3u8.MASTER {
		type MASTER struct {
			Name string `json:"name"`
			Url  string `json:"url"`
		}
		var master []MASTER
		for _, stream := range m3u8Info.Streams {
			master = append(master, MASTER{Url: stream.URL, Name: fmt.Sprintf("名称：%s 带宽：%d bps 分辨率： %s 编码：%s", stream.Name, stream.Bandwidth, stream.Resolution, stream.Codecs)})
		}
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 201, Msg: "选择m3u8", Data: master}))
		return
	}
	query := "INSERT INTO task (uuid,SavePath,Filename,Url,UserAgent,Cookie,Origin,Referer,Proxy,Threads,Status) VALUES (?,?,?,?,?,?,?,?,?,?,?)"
	_, err = Sqlite.Cxt.Exec(query, uuid.New().String(), SavePath, Filename, Url, UserAgent, Cookie, Origin, Referer, Proxy, Threads, 0)
	if err != nil {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: fmt.Sprintf("任务添加失败：%s", err.Error())}))
		return
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务添加成功"}))
	_ = TaskThreadPool.RunTask()
}
func del(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	Uuid := r.FormValue("uuid")
	Uuid = strings.Trim(Uuid, "\n")
	Uuid = strings.Trim(Uuid, "\r")
	Uuid = strings.Trim(Uuid, " ")
	if _, ok := TaskThreadPool.DownloaderList[Uuid]; !ok {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "任务不存在"}))
		return
	}
	_, _ = Sqlite.Cxt.Exec("DELETE FROM task WHERE uuid = ?", Uuid)
	TaskThreadPool.DownloaderList[Uuid].Status = 5
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务删除成功"}))
}
func delAll(w http.ResponseWriter, r *http.Request) {
	_ = r
	for i := range TaskThreadPool.DownloaderList {
		TaskThreadPool.DownloaderList[i].Status = 5
		_, _ = Sqlite.Cxt.Exec("DELETE FROM task WHERE uuid = ?", TaskThreadPool.DownloaderList[i].Uuid)
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务删除成功"}))
}
func stop(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	Uuid := r.FormValue("uuid")
	Uuid = strings.Trim(Uuid, "\n")
	Uuid = strings.Trim(Uuid, "\r")
	Uuid = strings.Trim(Uuid, " ")
	if _, ok := TaskThreadPool.DownloaderList[Uuid]; !ok {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "任务不存在"}))
		return
	}
	TaskThreadPool.DownloaderList[Uuid].Status = 4
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务终止成功"}))
}
func stopAll(w http.ResponseWriter, r *http.Request) {
	_ = r
	for i := range TaskThreadPool.DownloaderList {
		TaskThreadPool.DownloaderList[i].Status = 4
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务终止成功"}))
}
func list(w http.ResponseWriter, r *http.Request) {
	var status = r.URL.Query().Get("status")
	var result Result
	var data []map[string]interface{}
	result.Code = 200
	result.Msg = "success"
	DownloaderList := TaskThreadPool.DownloaderList
	if status == "1" {
		// 创建临时切片用于排序
		var downloaders []*downloader.Downloader
		for _, v := range DownloaderList {
			downloaders = append(downloaders, v)
		}

		// 按照 Uuid 排序确保一致性
		sort.Slice(downloaders, func(i, j int) bool {
			return downloaders[i].Uuid < downloaders[j].Uuid
		})

		// 按排序后的顺序处理
		for _, v := range downloaders {
			item := make(map[string]interface{})
			v.GetDownloaderTaskProgress()
			item["Speed"] = fmt.Sprintf("%.2f KB/s", v.Speed/1024)
			item["ReadSize"] = fmt.Sprintf("%.2f MB", float64(v.ReadSize/1024)/1024)
			item["Active"] = v.Active
			item["Complete"] = v.Complete
			item["Total"] = len(v.TsList)
			item["TsList"] = v.TsList
			item["Uuid"] = v.Uuid
			item["SavePath"] = v.SavePath
			item["Filename"] = v.Filename
			item["Url"] = v.Url
			item["UserAgent"] = v.UserAgent
			item["Cookie"] = v.Cookie
			item["Origin"] = v.Origin
			item["Referer"] = v.Referer
			item["Proxy"] = v.Proxy
			item["Threads"] = v.Threads
			item["Status"] = v.Status
			item["Msg"] = v.Msg
			item["Duration"] = v.Duration
			data = append(data, item)
		}
	} else {
		TaskThreadPool.SqliteMutex.Lock()
		rows, err := TaskThreadPool.Sqlite.Cxt.Query(fmt.Sprintf("SELECT * FROM task where Status = %s", status))
		if err == nil {
			TaskThreadPool.SqliteMutex.Unlock()
			for rows.Next() {
				item := make(map[string]interface{})
				row := downloader.Downloader{}
				err = rows.Scan(&row.Uuid, &row.SavePath, &row.Filename, &row.Url, &row.UserAgent, &row.Cookie, &row.Origin, &row.Referer, &row.Proxy, &row.Threads, &row.Status, &row.Msg, &row.Duration)
				if err == nil {
					item["Uuid"] = row.Uuid
					item["SavePath"] = row.SavePath
					item["Filename"] = row.Filename
					item["Url"] = row.Url
					item["UserAgent"] = row.UserAgent
					item["Cookie"] = row.Cookie
					item["Origin"] = row.Origin
					item["Referer"] = row.Referer
					item["Proxy"] = row.Proxy
					item["Threads"] = row.Threads
					item["Status"] = row.Status
					item["Msg"] = row.Msg
					item["Duration"] = row.Duration
					data = append(data, item)
				} else {
					result.Code = 400
					result.Msg = err.Error()
				}
			}
			_ = rows.Close()
		} else {
			TaskThreadPool.SqliteMutex.Unlock()
			result.Code = 400
			result.Msg = err.Error()
		}
	}
	result.Data = data
	_, _ = fmt.Fprintf(w, struct2Json(result))
}
func cleanUp(w http.ResponseWriter, r *http.Request) {
	var status = r.URL.Query().Get("status")
	status = strings.Trim(status, "\n")
	status = strings.Trim(status, "\r")
	status = strings.Trim(status, " ")
	if status == "" || status == "1" {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "进行中的任务不可清理"}))
		return
	}
	_, err := Sqlite.Cxt.Exec("DELETE FROM task WHERE Status = ?", status)
	if err != nil {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: err.Error()}))
		return
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务清理成功"}))
}
func getConf(w http.ResponseWriter, r *http.Request) {
	_ = r
	var key, value string
	var rows *sql.Rows
	var err error
	rows, err = Sqlite.Cxt.Query(fmt.Sprintf("SELECT * FROM conf"))
	if err != nil {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "进行中的任务不可清理"}))
		return
	}
	data := make(map[string]interface{})
	for rows.Next() {
		key = ""
		value = ""
		err = rows.Scan(&key, &value)
		data[key] = value
	}
	data["isInstallChromium"] = 0
	data["ChromiumMsg"] = "未安装UnGoogled Chromium"
	if chromiumManager.IsInstalled {
		data["isInstallChromium"] = 1
		data["ChromiumMsg"] = "已安装UnGoogled Chromium"
	}
	data["CanBeInstalled"] = 0
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			data["CanBeInstalled"] = 1
		} else {
			data["ChromiumMsg"] = "由于UnGoogled Chromium发型限制，不支持32位操作系统"
		}
	case "windows":
		if runtime.GOARCH != "arm" {
			data["CanBeInstalled"] = 1
		} else {
			data["ChromiumMsg"] = "由于UnGoogled Chromium发型限制，不支持arm32位操作系统"
		}
	case "darwin":
		data["CanBeInstalled"] = 1
		data["ChromiumMsg"] = "MacOs下需要自行安装UnGoogled Chromium，请到 https://ungoogled-software.github.io/ungoogled-chromium-binaries/releases/macos/ 选择适合自己的版本"
	case "freebsd":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			data["CanBeInstalled"] = 1
		} else {
			data["ChromiumMsg"] = "由于UnGoogled Chromium发型限制，不支持32位操作系统"
		}
	}
	_ = rows.Close()
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "success", Data: data}))
}
func setConf(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	key := r.FormValue("key")
	key = strings.Trim(key, "\n")
	key = strings.Trim(key, "\r")
	key = strings.Trim(key, " ")
	if key == "" {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	value := r.FormValue("value")
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")
	value = strings.Trim(value, " ")
	var maxWorkers int
	if key == "maxWorkers" {
		maxWorkers, err = strconv.Atoi(value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if maxWorkers < 0 || maxWorkers > 100 {
			http.Error(w, "同时下载数必须大于0小于等于100", http.StatusBadRequest)
			return
		}
	}
	_, err = Sqlite.Cxt.Exec(`UPDATE conf SET value = ? WHERE key = ?`, value, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if key == "maxWorkers" {
		TaskThreadPool.SetMaxWorkers(maxWorkers)
	}
	if key == "Token" {
		Token = value
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "success"}))
}
func checkForUpdates(w http.ResponseWriter, r *http.Request) {
	_ = r
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "success", Data: utils.CheckForUpdates()}))
}
func extractM3U8ByUrl(w http.ResponseWriter, r *http.Request) {
	if !chromiumManager.IsInstalled {
		http.Error(w, "UnGoogled Chromium is not installed", http.StatusBadRequest)
		return
	}
	var Url = r.URL.Query().Get("url")
	resp, err := chromiumManager.ExtractM3U8(Url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(resp.M3U8URLs) > 0 {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "success", Data: resp.M3U8URLs}))
	} else {
		http.Error(w, "未找到m3u8文件", http.StatusBadRequest)
	}
}
func struct2Json(result Result) string {
	JSON, _ := json.Marshal(result)
	return string(JSON)
}
