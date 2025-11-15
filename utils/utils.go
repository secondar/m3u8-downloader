package utils

import (
	"bufio"
	"fmt"
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
