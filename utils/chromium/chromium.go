package chromium

import (
	"M3u8Download/utils"
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/ulikunitz/xz"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Manager 管理UnGoogled Chromium的安装和运行
type Manager struct {
	ChromiumPath string
	DataDir      string
	IsInstalled  bool
	Platform     string
}

// M3U8ExtractResult 存储提取结果
type M3U8ExtractResult struct {
	URL      string   `json:"url"`
	M3U8URLs []string `json:"m3u8_urls"`
}

type progressReader struct {
	io.Reader
	total      int64
	current    int64
	lastTime   time.Time
	lastSize   int64
	onProgress func(current, total int64, speed float64)
}

// NewChromiumManager 创建Chromium管理器
func NewChromiumManager(dataDir string) *Manager {
	return &Manager{
		DataDir:     dataDir,
		Platform:    runtime.GOOS,
		IsInstalled: false,
	}
}

// InstallUnGoogledChromium 下载并安装UnGoogled Chromium
func (cm *Manager) InstallUnGoogledChromium() error {
	if err := os.MkdirAll(cm.DataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}
	switch cm.Platform {
	case "linux":
		return cm.installLinuxChromium()
	case "windows":
		return cm.installWindowsChromium()
	case "darwin":
		return cm.installMacOSChromium()
	case "freebsd":
		return cm.installLinuxChromium()
	default:
		return fmt.Errorf("不支持的操作系统: %s", cm.Platform)
	}
}

// installLinuxChromium 在Linux系统安装UnGoogled Chromium
func (cm *Manager) installLinuxChromium() error {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("不支持的CPU架构: %s，Linux下由于UnGoogled Chromium只支持64位的原因，所以不可用", runtime.GOARCH)
	}
	chromiumDir := filepath.Join(cm.DataDir, "ungoogled-chromium")
	cm.ChromiumPath = filepath.Join(chromiumDir, "chrome")
	if cm.checkChromiumInstalled(chromiumDir) {
		cm.IsInstalled = true
		utils.Info(fmt.Sprintf("UnGoogled Chromium已安装:%s", cm.ChromiumPath))
		return nil
	}
	// 创建Chromium目录
	if err := os.MkdirAll(chromiumDir, 0755); err != nil {
		return fmt.Errorf("创建Chromium目录失败: %v", err)
	}
	downloadURL := "https://github.com/ungoogled-software/ungoogled-chromium-portablelinux/releases/download/142.0.7444.162-1/ungoogled-chromium-142.0.7444.162-1-x86_64_linux.tar.xz"
	if runtime.GOARCH == "arm64" {
		downloadURL = "https://github.com/ungoogled-software/ungoogled-chromium-portablelinux/releases/download/142.0.7444.162-1/ungoogled-chromium-142.0.7444.162-1-arm64_linux.tar.xz"
	}
	err := cm.downloadAndExtract(downloadURL, chromiumDir)
	if err != nil {
		return fmt.Errorf("下载并解压UnGoogled Chromium失败: %v", err)
	}
	// 验证安装
	if !cm.checkChromiumInstalled(chromiumDir) {
		return fmt.Errorf("chromium安装验证失败")
	}
	cm.IsInstalled = true
	utils.Info(fmt.Sprintf("UnGoogled Chromium安装完成:%s", cm.ChromiumPath))
	return nil
}

// installWindowsChromium 在Windows系统安装UnGoogled Chromium
func (cm *Manager) installWindowsChromium() error {
	chromiumDir := filepath.Join(cm.DataDir, "ungoogled-chromium")
	cm.ChromiumPath = filepath.Join(chromiumDir, "chrome.exe")
	if cm.checkChromiumInstalled(chromiumDir) {
		cm.IsInstalled = true
		return nil
	}
	downloadURL := "https://github.com/ungoogled-software/ungoogled-chromium-windows/releases/download/142.0.7444.162-1.1/ungoogled-chromium_142.0.7444.162-1.1_windows_x86.zip"
	if runtime.GOARCH == "amd64" {
		downloadURL = "https://github.com/ungoogled-software/ungoogled-chromium-windows/releases/download/142.0.7444.162-1.1/ungoogled-chromium_142.0.7444.162-1.1_windows_x64.zip"
	}
	if runtime.GOARCH == "arm64" {
		downloadURL = "https://github.com/ungoogled-software/ungoogled-chromium-windows/releases/download/142.0.7444.162-1.1/ungoogled-chromium_142.0.7444.162-1.1_windows_arm64.zip"
	}
	if runtime.GOARCH == "arm" {
		return fmt.Errorf("不支持的CPU架构: %s，Windows ARM下由于UnGoogled Chromium只支持64位的原因，所以不可用", runtime.GOARCH)
	}
	if err := cm.downloadAndExtract(downloadURL, chromiumDir); err != nil {
		return fmt.Errorf("下载并解压UnGoogled Chromium失败: %v", err)
	}

	cm.IsInstalled = true
	return nil
}

// installMacOSChromium 在macOS系统安装UnGoogled Chromium
func (cm *Manager) installMacOSChromium() error {
	chromiumDir := filepath.Join(cm.DataDir, "ungoogled-chromium")
	cm.ChromiumPath = filepath.Join(chromiumDir, "Chromium.app", "Contents", "MacOS", "Chromium")
	if cm.checkChromiumInstalled(chromiumDir) {
		cm.IsInstalled = true
		return nil
	}
	return fmt.Errorf("MacOS 下需要自行安装 Chromium 下载地址：https://ungoogled-software.github.io/ungoogled-chromium-binaries/")
}

// downloadAndExtract 下载并解压Chromium
func (cm *Manager) downloadAndExtract(url, targetDir string) error {
	utils.Info(fmt.Sprintf("正在从 %s 下载UnGoogled Chromium", url))
	// 创建临时目录
	tempDir := filepath.Join(utils.GetCacheDir(), "/temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	// 下载文件
	archivePath := filepath.Join(tempDir, "chromium.archive")
	if err := cm.downloadFile(archivePath, url); err != nil {
		return fmt.Errorf("下载并解压UnGoogled Chromium失败: %v", err)
	}
	// 根据文件类型解压
	if strings.HasSuffix(url, ".tar.xz") || strings.HasSuffix(url, ".tar.gz") {
		if strings.HasSuffix(url, ".tar.xz") {
			return cm.extractTarArchive(archivePath, targetDir, "xz")
		}
		if strings.HasSuffix(url, ".tar.gz") {
			return cm.extractTarArchive(archivePath, targetDir, "gz")
		}
		return cm.extractTarArchive(archivePath, targetDir, "")
	} else if strings.HasSuffix(url, ".zip") {
		return cm.extractZipArchive(archivePath, targetDir)
	}
	_ = utils.ClearDirectory(utils.GetCacheDir())
	return fmt.Errorf("不支持的压缩格式: %s", url)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	// 检查是否需要终止下载
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

// downloadFile 下载文件
func (cm *Manager) downloadFile(filepath string, url string) error {
	var err error
	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	client := &http.Client{}
	Transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = Transport
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)
	totalSize := resp.ContentLength
	progressReader := &progressReader{
		Reader:   resp.Body,
		total:    totalSize,
		lastTime: time.Now(),
		onProgress: func(current, total int64, speed float64) {
			if total > 0 {
				progress := float64(current) / float64(total) * 100
				utils.Info(fmt.Sprintf("进度: %.1f%% (%d/%d bytes) 速度: %.2f KB/s", progress, current, total, speed/1024))
			} else {
				utils.Info(fmt.Sprintf("已下载: %d bytes 速度: %.2f KB/s", current, speed/1024))
			}
		},
	}
	_, err = io.Copy(out, progressReader)
	return err
}

// extractTarArchive 解压tar归档文件
func (cm *Manager) extractTarArchive(archivePath, targetDir, types string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var gr *gzip.Reader
	var xr *xz.Reader
	if types == "gz" {
		gr, err = gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer func(gr *gzip.Reader) {
			_ = gr.Close()
		}(gr)
	} else if types == "xz" {
		xr, err = xz.NewReader(file)
		if err != nil {
			return err
		}
	} else {
		gr = nil
	}

	var tr *tar.Reader
	if gr != nil {
		tr = tar.NewReader(gr)
	} else if xr != nil {
		tr = tar.NewReader(xr)
	} else {
		tr = tar.NewReader(file)
	}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fName := header.Name
		fNames := strings.Split(fName, "/")
		fName = strings.Replace(fName, fNames[0], "", -1)
		path := filepath.Join(targetDir, fName)
		utils.Info(fmt.Sprintf("正在解压文件: %s", path))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			outFile, err := os.Create(path)
			if err != nil {
				return err
			}
			if _, err = io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return err
			}
			_ = outFile.Close()
		}
	}

	return nil
}

// extractZipArchive 解压zip文件的具体实现
func (cm *Manager) extractZipArchive(zipPath, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		_ = r.Close()
	}(r)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		fName := f.Name
		fNames := strings.Split(fName, "/")
		fName = strings.Replace(fName, fNames[0], "", -1)
		path := filepath.Join(targetDir, fName)
		utils.Info(fmt.Sprintf("正在解压文件: %s", path))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				_ = rc.Close()
				return err
			}
		} else {
			// 创建文件目录
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				_ = rc.Close()
				return err
			}
			w, err := os.Create(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(w, rc)
			_ = w.Close()
		}
	}

	return nil
}

// checkChromiumInstalled 检查Chromium是否已安装
func (cm *Manager) checkChromiumInstalled(chromiumDir string) bool {
	var chromiumBinary string
	switch cm.Platform {
	case "linux":
		chromiumBinary = filepath.Join(chromiumDir, "chrome")
	case "windows":
		chromiumBinary = filepath.Join(chromiumDir, "chrome.exe")
	case "darwin":
		chromiumBinary = filepath.Join(chromiumDir, "Chromium.app", "Contents", "MacOS", "Chromium")
	default:
		return false
	}
	if _, err := os.Stat(chromiumBinary); err != nil {
		return false
	}

	// 验证可执行权限
	if cm.Platform != "windows" {
		if err := os.Chmod(chromiumBinary, 0755); err != nil {
			return false
		}
	}

	return true
}
func (cm *Manager) CheckChromiumInstalled() bool {
	chromiumDir := filepath.Join(cm.DataDir, "ungoogled-chromium")
	cm.ChromiumPath = filepath.Join(chromiumDir, "chrome")
	cm.IsInstalled = cm.checkChromiumInstalled(chromiumDir)
	return cm.IsInstalled
}

// GetChromiumOptions 获取Chromium启动选项
func (cm *Manager) GetChromiumOptions() []chromedp.ExecAllocatorOption {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.IgnoreCertErrors,
	}

	if cm.IsInstalled {
		opts = append(opts, chromedp.ExecPath(cm.ChromiumPath))
	}

	// UnGoogled Chromium专用配置
	opts = append(opts,
		chromedp.UserDataDir(filepath.Join(cm.DataDir, "user-data")),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("remote-debugging-port", "0"),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("disable-background-timer-throttle", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("disable-domain-reliability", true),
		chromedp.Flag("disable-features", "TranslateUI,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-back-forward-cache", true),
	)

	return opts
}

// ExtractM3U8 提取m3u8链接
func (cm *Manager) ExtractM3U8(url string) (*M3U8ExtractResult, error) {
	if !cm.IsInstalled {
		return nil, fmt.Errorf("未安装UnGoogled Chromium")
	}
	opts := cm.GetChromiumOptions()
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var pageContent string
	var networkRequests []string
	// 监听网络请求
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			// 检查URL是否包含.m3u8
			if strings.Contains(e.Request.URL, ".m3u8") {
				networkRequests = append(networkRequests, e.Request.URL)
				utils.Info(fmt.Sprintf("发现m3u8链接: %s", e.Request.URL))
			}
		}
	})
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML("html", &pageContent),
	)
	if err != nil {
		return nil, fmt.Errorf("页面导航失败: %v", err)
	}
	// 从页面内容中匹配m3u8链接
	m3u8Regex := regexp.MustCompile(`https?://[^\s"']*\.m3u8[^\s"']*`)
	contentMatches := m3u8Regex.FindAllString(pageContent, -1)
	// 合并结果
	allM3U8s := make(map[string]bool)
	for _, req := range networkRequests {
		allM3U8s[req] = true
	}
	for _, match := range contentMatches {
		allM3U8s[match] = true
	}
	var m3u8URLs []string
	for url := range allM3U8s {
		m3u8URLs = append(m3u8URLs, url)
	}
	result := &M3U8ExtractResult{
		URL:      url,
		M3U8URLs: m3u8URLs,
	}
	return result, nil
}
