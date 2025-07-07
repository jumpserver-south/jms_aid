package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"jms_tools/pkg/sdk/model"
	"log/slog"
	"os"
	"time"
)

const (
	FILENAME_APP_PERMS = "app_perms.json"
)

// MigrateAppPerms 迁移应用权限数据, 主要是数据库、k8s、chrome 资产的授权
func (s *JmsMigrateAssetService) MigrateAppPerms() {
	fmt.Printf("应用权限迁移开始...\n\n\t#### v2 应用权限迁移到 v3 资产 ####\n\n")
	start := time.Now()

	filepath, exists := existFile(FILENAME_APP_PERMS)
	if !exists {
		slog.Error(fmt.Sprintf("Web 资产文件[%s]不存在, 请先执行预处理", filepath))
		return
	}
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("打开文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()

		var perm model.ApplicationPerm
		if err := json.Unmarshal(lineBytes, &perm); err != nil {
			slog.Error(fmt.Sprintf("反序列化 Web 资产失败：%s", err.Error()))
			os.Exit(1)
		}

		s.orm.CompleteAppPermV3(&perm)
	}

	if err := scanner.Err(); err != nil {
		slog.Error(fmt.Sprintf("读取文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("应用权限迁移完成，耗时 %f 秒\n", duration)
}

func (s *JmsMigrateAssetService) MigrateAppPermPrepare() {
	permChan := make(chan model.ApplicationPerm, 50)
	go func() {
		defer close(permChan)
		err := s.orm.FetchAppPerms(permChan)
		if err != nil {
			slog.Error(fmt.Sprintf("获取应用权限信息失败：%s", err.Error()))
		}
	}()

	// 持久化本地
	s.appPermsPersist(permChan)
}

func (s *JmsMigrateAssetService) appPermsPersist(permChan chan model.ApplicationPerm) {
	filepath := getFilepath(FILENAME_APP_PERMS)
	// 打开文件（如果不存在则创建，存在则清空内容）
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("打开文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}
	defer f.Close()

	for {
		perm, ok := <-permChan
		if !ok {
			break
		}

		bt, err := json.Marshal(perm)
		if err != nil {
			slog.Error(fmt.Sprintf("序列化应用授权 [%s(%s)] 失败：%s", perm.Name, perm.AppName, err.Error()))
			os.Exit(1)
		}
		line := append(bt, '\n')
		_, err = f.Write(line)
		if err != nil {
			slog.Error(fmt.Sprintf("保存应用授权 [%s(%s)] 失败：%s", perm.Name, perm.AppName, err.Error()))
			os.Exit(1)
		}
	}
}

// CleanPermNullAccounts 清理授权规则中的空账号
// 该方法会循环调用 orm.CleanPermNullAccountsV3() 直到没有空账号可清理
// 返回:
//   error - 如果清理过程中发生错误则返回错误信息，否则返回 nil
func (s *JmsMigrateAssetService) CleanPermNullAccounts() {
	fmt.Printf("清理授权规则中空账号开始...\n\n\t#### v3 资产授权规则中空账号清理 ####\n\n")
	start := time.Now()

	// 每次清理一个空账号，直到全部清理完成
	for {
		affected, err := s.orm.CleanPermNullAccountsV3()
		if err != nil {
			slog.Error(fmt.Sprintf("清理授权规则中空账号失败：%s", err.Error()))
			return
		}
		if affected == 0 {
			break
		}
	}
	
	duration := time.Since(start).Seconds()
	fmt.Printf("清理授权规则中空账号完成，耗时 %f 秒\n", duration)
}
