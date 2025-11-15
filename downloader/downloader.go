package downloader

import (
	"M3u8Download/utils"
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/msterzhang/gpool"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Downloader struct {
	Client       *http.Client
	Uuid         string
	SavePath     string
	Filename     string
	Url          string
	UserAgent    string
	Cookie       string
	Origin       string
	Referer      string
	Proxy        string
	Threads      int
	BaseUrl      string
	TsList       []TsInfo
	cacheDirPath string
	Status       int // 0 等待 1 下载中 2 下载完成 3 下载失败 4 用户停止 5 用户删除
	Msg          string
	Speed        float64
	Complete     int
	ReadSize     int64 // 已下载大小
	Active       int
	Duration     string
}

type TsInfo struct {
	Name      string
	Url       string
	Status    int     // 0 等待 1 下载中 2 下载完成 3 下载失败
	Speed     float64 // 当前下载速度
	ReadSize  int64   // 已下载大小
	TotalSize int64   // 总大小
}

// ProgressReader 用于跟踪下载进度和速度
type ProgressReader struct {
	io.Reader
	total      int64
	current    int64
	lastTime   time.Time
	lastSize   int64
	onProgress func(current, total int64, speed float64)
	downloader *Downloader
}

func NewDownloader(client *http.Client, SavePath, Filename string, Url string, UserAgent string, Cookie string, Origin string, Referer string, Proxy string, Threads int, Uuid string, cacheDirPath string) *Downloader {
	downloader := &Downloader{Client: client, SavePath: SavePath, Filename: Filename, Url: Url, UserAgent: UserAgent, Cookie: Cookie, Origin: Origin, Referer: Referer, Proxy: Proxy, Threads: Threads, cacheDirPath: cacheDirPath, Uuid: Uuid}
	return downloader
}
func (cxt *Downloader) NewRequest(url string) (*http.Request, error) {
	url = strings.Replace(url, "\r\n", "", -1)
	url = strings.Replace(url, "\r", "", -1)
	url = strings.Replace(url, "\n", "", -1)
	var err error
	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if len(cxt.UserAgent) > 0 {
		req.Header.Set("User-Agent", cxt.UserAgent)
	}
	if len(cxt.Cookie) > 0 {
		req.Header.Set("Cookie", cxt.Cookie)
	}
	if len(cxt.Origin) > 0 {
		req.Header.Set("Origin", cxt.Origin)
	}
	if len(cxt.Referer) > 0 {
		req.Header.Set("Referer", cxt.Referer)
	}
	return req, err
}
func (cxt *Downloader) HttpRequest(url string) (string, error) {
	var body []byte
	var response *http.Response
	var err error
	var req *http.Request
	req, err = cxt.NewRequest(url)
	if err != nil {
		return "", err
	}
	response, err = cxt.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	body, err = io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
func (cxt *Downloader) GetM3u8Key(host, body string) (string, error) {
	lines := strings.Split(body, "\n")
	var key string
	var err error
	for _, line := range lines {
		if strings.Contains(line, "#EXT-X-KEY") {
			uriPos := strings.Index(line, "URI")
			quotationMarkPos := strings.LastIndex(line, "\"")
			keyUrl := strings.Split(line[uriPos:quotationMarkPos], "\"")[1]
			if !strings.Contains(line, "http") {
				keyUrl = fmt.Sprintf("%s/%s", host, keyUrl)
			}
			key, err = cxt.HttpRequest(keyUrl)
			if err != nil {
				return "", err
			}
		}
	}
	return key, err
}
func (cxt *Downloader) GetTsList(body string) {
	lines := strings.Split(body, "\n")
	index := 0
	var tsList []TsInfo
	var urls string
	var name string
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") && line != "" {
			index++
			ts := TsInfo{}
			if strings.HasPrefix(line, "http") {
				name = fmt.Sprintf("%06d.ts", index)
				urls = line
			} else {
				name = fmt.Sprintf("%06d.ts", index)
				urls = fmt.Sprintf("%s/%s", cxt.BaseUrl, line)
			}
			ts = TsInfo{
				Name:      name,
				Url:       urls,
				Status:    0,
				Speed:     0,
				ReadSize:  0,
				TotalSize: 0,
			}
			tsList = append(tsList, ts)
		}
	}
	cxt.TsList = tsList
}
func (cxt *Downloader) InitCachePath(id string) error {
	var err error
	err = os.MkdirAll(fmt.Sprintf("%s/%s", cxt.cacheDirPath, id), os.ModePerm)
	if err != nil {
		return err
	}
	// 判断文件夹是否存在
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(cxt.SavePath)
	if err != nil || !fileInfo.IsDir() {
		err = os.MkdirAll(cxt.SavePath, os.ModePerm)
		return err
	}
	_ = os.Remove(fmt.Sprintf("%s/%s", cxt.SavePath, cxt.Filename))
	return nil
}
func (cxt *Downloader) DownloadM3u8(startTime time.Time) error {
	cxt.Status = 1
	var result string
	var err error
	var key string
	result, err = cxt.HttpRequest(cxt.Url)
	if err != nil {
		utils.Error(errors.New(fmt.Sprintf("HttpRequest:%s下载失败，原因：%s", cxt.Filename, err.Error())))
		cxt.Msg = err.Error()
		cxt.Status = 3
		return err
	}
	if !strings.Contains(result, "#") {
		utils.Error(errors.New(fmt.Sprintf("%s下载失败，原因：下载链接存在未知错误[%s]", cxt.Filename, result)))
		cxt.Msg = fmt.Sprintf("下载链接存在未知错误:%s", result)
		cxt.Status = 3
		return err
	}
	if strings.Contains(result, ".m3u8") {
		utils.Error(errors.New(fmt.Sprintf("%s下载失败，原因：该文件嵌套有m3u8链接，请手动处理", cxt.Filename)))
		cxt.Msg = "该文件嵌套有m3u8链接，请手动处理"
		cxt.Status = 3
		return errors.New("该文件嵌套有m3u8链接，请手动处理")
	}
	u, _ := url.Parse(cxt.Url)
	host := u.Scheme + "://" + u.Host + strings.Replace(filepath.Dir(u.EscapedPath()), "\\", "/", -1)
	cxt.BaseUrl = host
	key, err = cxt.GetM3u8Key(host, result)
	if err != nil {
		utils.Error(errors.New(fmt.Sprintf("GetM3u8Key:%s下载失败，原因：%s", cxt.Filename, err.Error())))
		cxt.Msg = err.Error()
		cxt.Status = 3
		return err
	}
	cxt.GetTsList(result)
	id := uuid.New().String()
	err = cxt.InitCachePath(id)
	if err != nil {
		utils.Error(errors.New(fmt.Sprintf("InitCachePath:%s下载失败，原因：%s", cxt.Filename, err.Error())))
		cxt.Msg = err.Error()
		cxt.Status = 3
		return err
	}
	//防止创建的线程数比任务总量多
	thSize := cxt.Threads
	if len(cxt.TsList) < cxt.Threads {
		thSize = len(cxt.TsList)
	}
	pool := gpool.New(thSize)
	tsPath := fmt.Sprintf("%s/%s", cxt.cacheDirPath, id)
	for i := range cxt.TsList {
		if cxt.Status == 4 || cxt.Status == 5 {
			// 用户停止
			break
		}
		pool.Add(1)
		go cxt.DownloadTs(i, tsPath, key, pool)
	}
	pool.Wait()
	if cxt.Status == 4 || cxt.Status == 5 {
		cxt.Msg = fmt.Sprintf("%s下载失败，原因：用户操作", cxt.Filename)
		utils.Error(errors.New(cxt.Msg))
		_ = os.RemoveAll(tsPath)
		return err
	}
	cxt.Status = 2
	for _, v := range cxt.TsList {
		if v.Status == 3 {
			cxt.Status = 3
			if cxt.Msg == "" {
				cxt.Msg = fmt.Sprintf("%s下载失败", v.Name)
			} else {
				cxt.Msg = fmt.Sprintf("%s,%s下载失败", cxt.Msg, v.Name)
			}
		}
	}
	if cxt.Status != 2 {
		utils.Error(errors.New(fmt.Sprintf("%s下载失败，原因：%s", cxt.Filename, cxt.Msg)))
		cxt.Status = 3
		_ = os.RemoveAll(tsPath)
		return err
	}
	cxt.MergeFile(tsPath)
	cxt.Msg = fmt.Sprintf("下载完成已保存至： %s 用时： %s", cxt.Filename, time.Now().Sub(startTime))
	cxt.Status = 2
	cxt.Duration = time.Now().Sub(startTime).String()
	utils.Info(fmt.Sprintf("%s下载完成已保存至： %s 用时： %s", cxt.SavePath, cxt.Filename, time.Now().Sub(startTime)))
	return nil
}
func (cxt *Downloader) DownloadTs(index int, tsPath string, key string, pool *gpool.Pool) {
	var err error
	var req *http.Request
	var response *http.Response
	req, err = cxt.NewRequest(cxt.TsList[index].Url)
	cxt.TsList[index].Status = 1
	if err != nil {
		pool.Done()
		utils.Error(err)
		cxt.TsList[index].Status = 3
		return
	}
	// 发送请求
	client := &http.Client{}
	response, err = client.Do(req)
	if err != nil {
		pool.Done()
		utils.Error(err)
		cxt.TsList[index].Status = 3
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	// 检查响应状态
	if response.StatusCode != http.StatusOK {
		utils.Error(errors.New(fmt.Sprintf("请求失败，状态码：%d", response.StatusCode)))
		pool.Done()
		utils.Error(err)
		cxt.TsList[index].Status = 3
		return
	}
	totalSize := response.ContentLength
	progressReader := &ProgressReader{
		Reader:     response.Body,
		total:      totalSize,
		lastTime:   time.Now(),
		downloader: cxt,
		onProgress: func(current, total int64, speed float64) {
			if total > 0 {
				progress := float64(current) / float64(total) * 100
				cxt.TsList[index].ReadSize = current
				cxt.TsList[index].Speed = speed
				cxt.TsList[index].TotalSize = total
				utils.Info(fmt.Sprintf("任务：%s TS：%s 进度: %.1f%% (%d/%d bytes) 速度: %.2f KB/s", cxt.Filename, cxt.TsList[index].Name, progress, current, total, speed/1024))
			} else {
				cxt.TsList[index].ReadSize = current
				cxt.TsList[index].Speed = speed
				cxt.TsList[index].TotalSize = total
				utils.Info(fmt.Sprintf("任务：%s TS：%s 已下载: %d bytes 速度: %.2f KB/s", cxt.Filename, cxt.TsList[index].Name, current, speed/1024))
			}
		},
	}
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, progressReader)
	if err != nil {
		pool.Done()
		utils.Error(err)
		cxt.TsList[index].Status = 3
	}
	responseBytes := buffer.Bytes()
	if key != "" {
		responseBytes, err = utils.AesDecrypt(responseBytes, []byte(key))
		if err != nil {
			pool.Done()
			utils.Error(err)
			cxt.TsList[index].Status = 3
			return
		}
	}
	bLen := len(responseBytes)
	syncByte := uint8(71) //0x47
	for j := 0; j < bLen; j++ {
		if responseBytes[j] == syncByte {
			responseBytes = responseBytes[j:]
			break
		}
	}
	var file *os.File
	file, err = os.Create(fmt.Sprintf("%s/%s", tsPath, cxt.TsList[index].Name))
	if err != nil {
		pool.Done()
		utils.Error(err)
		cxt.TsList[index].Status = 3
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_, err = file.Write(responseBytes)
	if err != nil {
		pool.Done()
		utils.Error(err)
		if cxt.Status != 4 && cxt.Status != 5 {
			cxt.TsList[index].Status = 3
		}
		return
	}
	cxt.TsList[index].Status = 2
	cxt.TsList[index].Speed = 0
	cxt.Complete += 1
	pool.Done()
}
func (cxt *Downloader) MergeFile(tsPath string) {
	var saveFilename = fmt.Sprintf("%s/%s", cxt.SavePath, cxt.Filename)
	saveFilename = strings.Replace(saveFilename, "//", "/", -1)
	outFile, err := os.OpenFile(saveFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.Error(err)
		return
	}
	defer func(outFile *os.File) {
		_ = outFile.Close()
	}(outFile)
	tsFileList, _ := os.ReadDir(tsPath)
	for _, f := range tsFileList {
		tsFilePath := tsPath + "/" + f.Name()
		tsFileContent, err := os.ReadFile(tsFilePath)
		if err != nil {
			utils.Error(err)
		}
		if _, err := outFile.Write(tsFileContent); err != nil {
			utils.Error(err)
		}
		if err = os.Remove(tsFilePath); err != nil {
			utils.Error(err)
		}
	}
	_ = os.RemoveAll(tsPath)
}
func (cxt *Downloader) GetDownloaderTaskProgress() {
	cxt.Speed = 0
	cxt.ReadSize = 0
	cxt.Active = 0
	for _, v := range cxt.TsList {
		cxt.Speed += v.Speed
		cxt.ReadSize += v.ReadSize
		if v.Status == 1 {
			cxt.Active += 1
		}
	}
}
func (pr *ProgressReader) Read(p []byte) (int, error) {
	// 检查是否需要终止下载
	if pr.downloader != nil && (pr.downloader.Status == 4 || pr.downloader.Status == 5) {
		return 0, errors.New("download stopped by user")
	}
	n, err := pr.Reader.Read(p)
	pr.current += int64(n)
	// 计算下载速度
	now := time.Now()
	elapsed := now.Sub(pr.lastTime).Seconds()
	if elapsed >= 1.0 {
		speed := float64(pr.current-pr.lastSize) / elapsed
		if pr.onProgress != nil {
			pr.onProgress(pr.current, pr.total, speed)
		}
		pr.lastSize = pr.current
		pr.lastTime = now
	}
	return n, err
}
