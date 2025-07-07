package service

import (
	"fmt"
	"log/slog"
	"os"
	"path"
)

func getFilepath(filename string) string {
	exepath, err := os.Executable()
	if err != nil {
		slog.Error(fmt.Sprintf("获取可执行文件路径失败：%s", err.Error()))
		os.Exit(1)
	}
	dir := path.Join(path.Dir(exepath), "data")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error(fmt.Sprintf("创建数据目录[%s]失败：%s", dir, err.Error()))
			os.Exit(1)
		}
	}
	filepath := path.Join(dir, filename)
	return filepath
}

func existFile(filename string) (filepath string, exists bool) {
	filepath = getFilepath(filename)
	stats, err := os.Stat(filepath)
	if err != nil {
		if err == os.ErrNotExist {
			return filepath, false
		} else {
			slog.Error(fmt.Sprintf("获取文件信息失败：%s", err.Error()))
			os.Exit(1)
		}
	}
	if stats.IsDir() {
		return filepath, false
	}
	return filepath, true
}
