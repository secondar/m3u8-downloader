package utils

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-version"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func IsDockerByCGroup() bool {
	file, err := os.Open("/proc/1/cgroup")
	if err != nil {
		return false
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "docker") || strings.Contains(line, "containerd") {
			return true
		}
	}
	return false
}
func GetCacheDir() string {
	if IsDockerByCGroup() {
		return "/cache"
	} else {
		return "./cache"
	}
}
func ClearDirectory(dirPath string) error {
	// 读取目录内容
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %v", err)
	}
	// 删除每个条目
	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())
		err := os.RemoveAll(fullPath)
		if err != nil {
			return fmt.Errorf("删除 %s 失败: %v", fullPath, err)
		}
	}
	return nil
}
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
func DirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return info.IsDir()
}
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(sourceFile *os.File) {
		_ = sourceFile.Close()
	}(sourceFile)
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(destFile *os.File) {
		_ = destFile.Close()
	}(destFile)
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	err = destFile.Sync()
	return err
}
func CheckForUpdates() map[string]interface{} {
	client := github.NewClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})
	ctx := context.Background()
	// 指定要查询的仓库
	owner := "secondar"
	repo := "m3u8-downloader"
	// 获取最新release信息
	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	localVersion := "v1.0.1"
	result := make(map[string]interface{})
	result["localVersion"] = localVersion
	result["update"] = 0
	if err != nil {
		return result
	}
	local, _ := version.NewVersion(localVersion)
	remote, _ := version.NewVersion(release.GetTagName())
	if local.Compare(remote) < 0 {
		result["update"] = 1
		result["body"] = release.GetBody()
		result["remoteVersion"] = release.GetTagName()
		result["releaseTime"] = release.GetPublishedAt().Format("2006-01-02 15:04:05")
	}
	return result
}
