package common

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func ConfigFileToMap(filepath string) (map[string]string, error) {
	configMap := make(map[string]string)
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("打开文件[%s]失败，错误: %v", filepath, err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(value, "#") {
			continue
		}
		items := strings.SplitN(value, "=", 2)
		if len(items) != 2 {
			continue
		}
		configMap[items[0]] = items[1]
	}
	return configMap, nil
}

func GetTerminalWidth() (int, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return 0, fmt.Errorf("无法获取终端宽度")
	}
	width, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return width, nil
}

func EnsureDir(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
