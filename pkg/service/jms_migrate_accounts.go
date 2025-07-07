package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/orm"
	"log/slog"
	"os"
	"time"

	"gorm.io/gorm"
)

const FILENAME_SYSTEMUSER = "systemuser.json"

type MigrateAccountService struct {
	orm *orm.JMSOrm
}

func NewMigrateAccountService(db *gorm.DB) *MigrateAccountService {
	return &MigrateAccountService{
		orm: orm.NewJMSOrm(db),
	}
}

func (s *MigrateAccountService) Prepare() {
	s.MigrateAccountPrepare()
}

func (s *MigrateAccountService) MigrateAccountPrepare() {
	fmt.Printf("系统用户迁移账号模板预处理开始...\n")
	start := time.Now()

	// 系统用户迁移预处理
	systemuserChan := make(chan model.AssetSystemUser, 50)
	go func() {
		defer close(systemuserChan)
		err := s.orm.FetchSystemusers(systemuserChan)
		if err != nil {
			slog.Error(fmt.Sprintf("获取 Web 资产信息失败：%s", err.Error()))
		}
	}()
	// 持久化本地
	s.systemuserPersist(systemuserChan)

	duration := time.Since(start).Seconds()
	fmt.Printf("系统用户迁移账号模板预处理完成，耗时 %f 秒\n", duration)
}

func (s *MigrateAccountService) systemuserPersist(systemuserChan chan model.AssetSystemUser) {
	filepath := getFilepath(FILENAME_SYSTEMUSER)
	// 打开文件（如果不存在则创建，存在则清空内容）
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("打开文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}
	defer f.Close()

	for {
		suser, ok := <-systemuserChan
		if !ok {
			break
		}

		bt, err := json.Marshal(suser)
		if err != nil {
			slog.Error(fmt.Sprintf("序列化系统用户 [%s] 失败：%s", suser.Name, err.Error()))
			os.Exit(1)
		}
		line := append(bt, '\n')
		_, err = f.Write(line)
		if err != nil {
			slog.Error(fmt.Sprintf("保存系统用户 [%s] 失败：%s", suser.Name, err.Error()))
			os.Exit(1)
		}
	}
}

func (s *MigrateAccountService) Post() {
	s.MigrateAccountPost()
	s.AccountNameStandard()
}

func (s *MigrateAccountService) MigrateAccountPost() {
	fmt.Printf("系统用户迁移账号模板开始...\n\n\t#### v2 系统用户迁移到 v3 账号模板，不会关联到资产，请注意！####\n\n")
	start := time.Now()

	// 基于 v2 整理出来的数据，迁移到 v3 中
	filepath, exists := existFile(FILENAME_SYSTEMUSER)
	if !exists {
		slog.Error(fmt.Sprintf("系统用户文件[%s]不存在, 请先执行预处理", filepath))
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

		var systemuser model.AssetSystemUser
		if err := json.Unmarshal(lineBytes, &systemuser); err != nil {
			slog.Error(fmt.Sprintf("反序列化系统用户失败：%s", err.Error()))
			os.Exit(1)
		}
		s.orm.AddAccountTemplateV3(&systemuser)
	}

	if err := scanner.Err(); err != nil {
		slog.Error(fmt.Sprintf("读取文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("系统用户迁移账号模板完成，耗时 %f 秒\n", duration)
}

// AccountStandard 标准化账号名称，确保账号名称高可读性
func (s *MigrateAccountService) AccountNameStandard() {
	fmt.Printf("资产账号名称标准化开始...\n")
	start := time.Now()

	accountChan := make(chan model.AssetAccount, 50)
	go func() {
		defer close(accountChan)
		if err := s.orm.FetchManyTypeAccount(accountChan); err != nil {
			slog.Error(fmt.Sprintf("获取资产账号信息失败：%s", err.Error()))
		}
	}()

	for {
		account, ok := <-accountChan
		if !ok {
			break
		}
		s.orm.AccountNameStandard(&account)
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("资产账号名称标准化完成，耗时 %f 秒\n", duration)
}
