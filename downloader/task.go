package downloader

import (
	"M3u8Download/sqlite"
	"M3u8Download/utils"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TaskThreadPool 动态任务线程池
type TaskThreadPool struct {
	DownloaderList map[string]*Downloader
	Sqlite         *sqlite.Sqlite
	SqliteMutex    sync.Mutex
	maxWorkers     int32         // 最大工作协程数
	activeWorkers  int32         // 当前活跃协程数
	tasks          chan func()   // 任务队列
	quit           chan struct{} // 退出信号
	mutex          sync.Mutex
	cond           *sync.Cond     // 条件变量用于阻塞和唤醒
	wg             sync.WaitGroup // 等待所有任务完成
}

// NewTaskThreadPool 创建新的动态任务线程池
func NewTaskThreadPool(maxWorkers int, sqlite *sqlite.Sqlite) *TaskThreadPool {
	cxt := &TaskThreadPool{
		DownloaderList: make(map[string]*Downloader),
		maxWorkers:     int32(maxWorkers),
		tasks:          make(chan func(), 1000),
		quit:           make(chan struct{}),
		Sqlite:         sqlite,
	}
	cxt.cond = sync.NewCond(&cxt.mutex)
	return cxt
}

// SetMaxWorkers 动态设置最大协程数
func (cxt *TaskThreadPool) SetMaxWorkers(max int) {
	cxt.mutex.Lock()
	oldMax := cxt.maxWorkers
	cxt.maxWorkers = int32(max)
	cxt.mutex.Unlock()
	// 如果最大值增加了，唤醒可能等待的协程
	if int32(max) > oldMax {
		cxt.cond.Broadcast()
	}
	utils.Info(fmt.Sprintf("最大协程数从 %d 调整为 %d", oldMax, max))
}

// GetMaxWorkers 获取最大协程数
func (cxt *TaskThreadPool) GetMaxWorkers() int {
	cxt.mutex.Lock()
	defer cxt.mutex.Unlock()
	return int(cxt.maxWorkers)
}

// GetActiveWorkers 获取当前活跃协程数
func (cxt *TaskThreadPool) GetActiveWorkers() int {
	return int(atomic.LoadInt32(&cxt.activeWorkers))
}

// AddTask 添加单个任务
func (cxt *TaskThreadPool) AddTask(task func()) {
	cxt.wg.Add(1)
	select {
	case cxt.tasks <- task:
		// 尝试启动新协程（如果需要）
		go cxt.startWorkerIfNeeded()
	case <-cxt.quit:
		cxt.wg.Done()
	}
}

// worker 工作协程
func (cxt *TaskThreadPool) worker(workerID int) {
	defer func() {
		atomic.AddInt32(&cxt.activeWorkers, -1)
		utils.Info(fmt.Sprintf("协程 %d 退出，当前活跃协程数: %d", workerID, cxt.GetActiveWorkers()))
	}()
	utils.Info(fmt.Sprintf("协程 %d 启动", workerID))
	for {
		select {
		case task, ok := <-cxt.tasks:
			if !ok {
				return
			}
			// 执行任务
			task()
			cxt.wg.Done() // 任务完成
		case <-cxt.quit:
			return
		}
	}
}

// startWorkerIfNeeded 根据条件启动协程
func (cxt *TaskThreadPool) startWorkerIfNeeded() {
	cxt.mutex.Lock()
	defer cxt.mutex.Unlock()
	// 检查是否可以启动新协程
	for cxt.activeWorkers >= cxt.maxWorkers && cxt.maxWorkers > 0 {
		// 等待直到可以启动新协程
		cxt.cond.Wait()
	}
	// 检查是否已退出
	select {
	case <-cxt.quit:
		return
	default:
	}
	// 启动新协程
	atomic.AddInt32(&cxt.activeWorkers, 1)
	currentActive := cxt.activeWorkers
	go cxt.worker(int(currentActive))
}

// Wait 等待所有任务完成
func (cxt *TaskThreadPool) Wait() {
	cxt.wg.Wait()
}

// Close 关闭线程池
func (cxt *TaskThreadPool) Close() {
	close(cxt.quit)
	cxt.cond.Broadcast() // 唤醒所有等待的协程
}

// RunTask 运行任务
func (cxt *TaskThreadPool) RunTask() error {
	cxt.SqliteMutex.Lock()
	rows, err := cxt.Sqlite.Cxt.Query(fmt.Sprintf("SELECT * FROM task where Status = 0"))
	if err != nil {
		cxt.SqliteMutex.Unlock()
		return err
	}
	var taskList []Downloader
	for rows.Next() {
		row := Downloader{}
		err = rows.Scan(&row.Uuid, &row.SavePath, &row.Filename, &row.Url, &row.UserAgent, &row.Cookie, &row.Origin, &row.Referer, &row.Proxy, &row.Threads, &row.Status, &row.Msg, &row.Duration)
		if err == nil {
			updateSQL := `UPDATE task SET Status = ? WHERE uuid = ?`
			_, _ = cxt.Sqlite.Cxt.Exec(updateSQL, 1, row.Uuid)
			row.SavePath = strings.Trim(row.SavePath, "\n")
			row.SavePath = strings.Trim(row.SavePath, "\r")
			row.SavePath = strings.Trim(row.SavePath, " ")
			row.Filename = strings.Trim(row.Filename, "\n")
			row.Filename = strings.Trim(row.Filename, "\r")
			row.Filename = strings.Trim(row.Filename, " ")
			row.Url = strings.Trim(row.Url, "\n")
			row.Url = strings.Trim(row.Url, "\r")
			row.Url = strings.Trim(row.Url, " ")
			row.UserAgent = strings.Trim(row.UserAgent, "\n")
			row.UserAgent = strings.Trim(row.UserAgent, "\r")
			row.UserAgent = strings.Trim(row.UserAgent, " ")
			row.Cookie = strings.Trim(row.Cookie, "\n")
			row.Cookie = strings.Trim(row.Cookie, "\r")
			row.Cookie = strings.Trim(row.Cookie, " ")
			row.Origin = strings.Trim(row.Origin, "\n")
			row.Origin = strings.Trim(row.Origin, "\r")
			row.Origin = strings.Trim(row.Origin, " ")
			row.Referer = strings.Trim(row.Referer, "\n")
			row.Referer = strings.Trim(row.Referer, "\r")
			row.Referer = strings.Trim(row.Referer, " ")
			row.Proxy = strings.Trim(row.Proxy, "\n")
			row.Proxy = strings.Trim(row.Proxy, "\r")
			row.Proxy = strings.Trim(row.Proxy, " ")
			taskList = append(taskList, row)
		}
	}
	_ = rows.Close()
	cxt.SqliteMutex.Unlock()
	utils.Info(fmt.Sprintf("取得%d个新任务", len(taskList)))
	for _, taskRows := range taskList {
		taskRow := taskRows
		cxt.DownloaderList[taskRow.Uuid] = &taskRow
		client := &http.Client{}
		if taskRow.Proxy != "" {
			Transport := &http.Transport{}
			var proxyURL *url.URL
			proxyURL, err = url.Parse(taskRow.Proxy)
			if err == nil {
				Transport.Proxy = http.ProxyURL(proxyURL)
				client.Transport = Transport
			}
		}
		var updateSQL = ""
		cxt.DownloaderList[taskRow.Uuid].Client = client
		if err != nil {
			updateSQL = `UPDATE task SET Status = ?,Msg= ? WHERE uuid = ?`
			_, _ = cxt.Sqlite.Cxt.Exec(updateSQL, 3, err.Error(), taskRow.Uuid)
			utils.Error(errors.New(fmt.Sprintf("新任务代理[%s]设置失败：%s", taskRow.Filename, err.Error())))
		} else {
			cxt.DownloaderList[taskRow.Uuid] = NewDownloader(cxt.DownloaderList[taskRow.Uuid].Client,
				cxt.DownloaderList[taskRow.Uuid].SavePath,
				cxt.DownloaderList[taskRow.Uuid].Filename,
				cxt.DownloaderList[taskRow.Uuid].Url,
				cxt.DownloaderList[taskRow.Uuid].UserAgent,
				cxt.DownloaderList[taskRow.Uuid].Cookie,
				cxt.DownloaderList[taskRow.Uuid].Origin,
				cxt.DownloaderList[taskRow.Uuid].Referer,
				cxt.DownloaderList[taskRow.Uuid].Proxy,
				64,
				taskRow.Uuid,
				utils.GetCacheDir())
			updateSQL = `UPDATE task SET Status = ?,Msg= ? WHERE uuid = ?`
			_, _ = cxt.Sqlite.Cxt.Exec(updateSQL, 1, cxt.DownloaderList[taskRow.Uuid].Msg, taskRow.Uuid)
		}
		cxt.AddTask(func() {
			err = cxt.DownloaderList[taskRow.Uuid].DownloadM3u8(time.Now())
			if err != nil {
				updateSQL = `UPDATE task SET Status = ?,Msg= ? WHERE uuid = ?`
				_, _ = cxt.Sqlite.Cxt.Exec(updateSQL, cxt.DownloaderList[taskRow.Uuid].Status, err.Error(), taskRow.Uuid)
			} else {
				updateSQL = `UPDATE task SET Status = ?,Msg= ?,Duration=? WHERE uuid = ?`
				_, _ = cxt.Sqlite.Cxt.Exec(updateSQL, cxt.DownloaderList[taskRow.Uuid].Status, cxt.DownloaderList[taskRow.Uuid].Msg, cxt.DownloaderList[taskRow.Uuid].Duration, taskRow.Uuid)
			}
			delete(cxt.DownloaderList, taskRow.Uuid)
		})
	}
	return nil
}
