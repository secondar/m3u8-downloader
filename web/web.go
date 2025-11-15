package web

import (
	"M3u8Download/downloader"
	"M3u8Download/sqlite"
	"M3u8Download/utils"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var Sqlite *sqlite.Sqlite
var TaskThreadPool *downloader.TaskThreadPool
var Token string

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Run() {
	var err error
	_ = utils.ClearDirectory(utils.GetCacheDir())
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
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	handler := TokenMiddleware(mux)
	err = http.ListenAndServe(":65533", handler)
	if err != nil {
		panic(err)
	}
}
func TokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	http.ServeFile(w, r, filepath.Join("./view", "index.html"))
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
	if err != nil || threads <= 0 {
		threads = 64
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
	fmt.Println(Uuid)
	if _, ok := TaskThreadPool.DownloaderList[Uuid]; !ok {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "任务不存在"}))
		return
	}
	_, _ = Sqlite.Cxt.Exec("DELETE FROM task WHERE uuid = ?", Uuid)
	TaskThreadPool.DownloaderList[Uuid].Status = 5
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务删除成功"}))
}
func delAll(w http.ResponseWriter, r *http.Request) {
	for i, _ := range TaskThreadPool.DownloaderList {
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
	fmt.Println(Uuid)
	if _, ok := TaskThreadPool.DownloaderList[Uuid]; !ok {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: "任务不存在"}))
		return
	}
	TaskThreadPool.DownloaderList[Uuid].Status = 4
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务终止成功"}))
}
func stopAll(w http.ResponseWriter, r *http.Request) {
	for i, _ := range TaskThreadPool.DownloaderList {
		TaskThreadPool.DownloaderList[i].Status = 4
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务终止成功"}))
}
func list(w http.ResponseWriter, r *http.Request) {
	var status = r.URL.Query().Get("status")
	var result Result
	var data []map[string]interface{}
	item := make(map[string]interface{})
	result.Code = 200
	result.Msg = "success"
	if status == "1" {
		for key, _ := range TaskThreadPool.DownloaderList {
			TaskThreadPool.DownloaderList[key].GetDownloaderTaskProgress()
			item["Speed"] = fmt.Sprintf("%.2f KB/s", TaskThreadPool.DownloaderList[key].Speed)
			item["ReadSize"] = fmt.Sprintf("%.2f MB", float64(TaskThreadPool.DownloaderList[key].ReadSize/1024)/1024)
			item["Active"] = TaskThreadPool.DownloaderList[key].Active
			item["Complete"] = TaskThreadPool.DownloaderList[key].Complete
			item["Total"] = len(TaskThreadPool.DownloaderList[key].TsList)
			item["Uuid"] = TaskThreadPool.DownloaderList[key].Uuid
			item["SavePath"] = TaskThreadPool.DownloaderList[key].SavePath
			item["Filename"] = TaskThreadPool.DownloaderList[key].Filename
			item["Url"] = TaskThreadPool.DownloaderList[key].Url
			item["UserAgent"] = TaskThreadPool.DownloaderList[key].UserAgent
			item["Cookie"] = TaskThreadPool.DownloaderList[key].Cookie
			item["Origin"] = TaskThreadPool.DownloaderList[key].Origin
			item["Referer"] = TaskThreadPool.DownloaderList[key].Referer
			item["Proxy"] = TaskThreadPool.DownloaderList[key].Proxy
			item["Threads"] = TaskThreadPool.DownloaderList[key].Threads
			item["Status"] = TaskThreadPool.DownloaderList[key].Status
			item["Msg"] = TaskThreadPool.DownloaderList[key].Msg
			item["Duration"] = TaskThreadPool.DownloaderList[key].Duration
			data = append(data, item)
		}
	} else {
		TaskThreadPool.SqliteMutex.Lock()
		rows, err := TaskThreadPool.Sqlite.Cxt.Query(fmt.Sprintf("SELECT * FROM task where Status = %s", status))
		if err == nil {
			TaskThreadPool.SqliteMutex.Unlock()
			for rows.Next() {
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
			result.Code = 400
			result.Msg = err.Error()
		}
		TaskThreadPool.SqliteMutex.Unlock()
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
	}
	_, err := Sqlite.Cxt.Exec("DELETE FROM task WHERE Status = ?", status)
	if err != nil {
		_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 400, Msg: err.Error()}))
		return
	}
	_, _ = fmt.Fprintf(w, struct2Json(Result{Code: 200, Msg: "任务清理成功"}))
}
func struct2Json(result Result) string {
	JSON, _ := json.Marshal(result)
	return string(JSON)
}
