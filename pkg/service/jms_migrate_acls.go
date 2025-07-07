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

const FILENAME_COMMANDFILTER_ACL = "acl_commandfilter.json"

type JmsMigrateAclService struct {
	orm *orm.JMSOrm
}

func NewJmsMigrateAclAssets(db *gorm.DB) *JmsMigrateAclService {
	return &JmsMigrateAclService{
		orm: orm.NewJMSOrm(db),
	}
}

func (s *JmsMigrateAclService) Prepare() {
	// 命令过滤迁移预处理
	s.MigrateCommandFilterAclPrepare()
}

func (s *JmsMigrateAclService) Post() { 
	// 命令过滤迁移
	s.MigrateCommandFilterAclPost()
}

func (s *JmsMigrateAclService) MigrateCommandFilterAclPrepare() {
	fmt.Printf("命令过滤预处理开始...\n")
	start := time.Now()
	
	// 命令过滤预处理
	aclChan := make(chan model.CommandFilterAcl, 50)
	go func() {
		defer close(aclChan)
		err := s.orm.FetchCommandFilterAcls(aclChan)
		if err != nil {
			slog.Error(fmt.Sprintf("获取命令过滤规则信息失败：%s", err.Error()))
		}
	}()
	// 持久化本地
	s.commendFilterAclPersist(aclChan)

	duration := time.Since(start).Seconds()
	fmt.Printf("命令过滤预处理完成，耗时 %f 秒\n", duration)
}

func (s *JmsMigrateAclService) commendFilterAclPersist(aclChan chan model.CommandFilterAcl) {
	filepath := getFilepath(FILENAME_COMMANDFILTER_ACL)
	// 打开文件（如果不存在则创建，存在则清空内容）
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("打开文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}
	defer f.Close()

	for {
		acl, ok := <-aclChan
		if !ok {
			break
		}

		bt, err := json.Marshal(acl)
		if err != nil {
			slog.Error(fmt.Sprintf("序列化命令过滤规则 [%s] 失败：%s", acl.Name, err.Error()))
			os.Exit(1)
		}
		line := append(bt, '\n')
		_, err = f.Write(line)
		if err != nil {
			slog.Error(fmt.Sprintf("保存命令过滤规则 [%s] 失败：%s", acl.Name, err.Error()))
			os.Exit(1)
		}
	}
}

func (s *JmsMigrateAclService) MigrateCommandFilterAclPost() {
	fmt.Printf("命令过滤迁移开始...\n")
	start := time.Now()

	// 基于 v2 整理出来的数据，迁移到 v3 中
	filepath, exists := existFile(FILENAME_COMMANDFILTER_ACL)
	if !exists {
		slog.Error(fmt.Sprintf("命令过滤文件[%s]不存在, 请先执行预处理", filepath))
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

		var acl model.CommandFilterAcl
		if err := json.Unmarshal(lineBytes, &acl); err != nil {
			slog.Error(fmt.Sprintf("反序列化 Web 资产失败：%s", err.Error()))
			os.Exit(1)
		}
		s.orm.AddCommandFilterAclV3(&acl)
	}

	if err := scanner.Err(); err != nil {
		slog.Error(fmt.Sprintf("读取文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("命令过滤迁移完成，耗时 %f 秒\n", duration)
}