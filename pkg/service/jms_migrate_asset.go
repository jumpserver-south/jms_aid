package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/orm"
	"jms_tools/pkg/sdk/utils/crypto"
	"log/slog"
	"os"
	"time"

	"gorm.io/gorm"
)

const (
	FILENAME_WEB_ASSETS = "web_assets.json"
	FILENAME_APP_ASSETS = "app_assets.json"
)

// JmsMigrateAssetService 资产迁移，Web 资产
type JmsMigrateAssetService struct {
	orm *orm.JMSOrm
}

func NewJmsMigrateAppAssets(db *gorm.DB) *JmsMigrateAssetService {
	return &JmsMigrateAssetService{
		orm: orm.NewJMSOrm(db),
	}
}

// Prepare 预处理迁移数据，包括：
// 1. 整理 web 相关数据
// 2. 处理数据库和云服务相关数据（因网域网关迁移导致ID变更需重新关联）
// 3. 准备应用权限数据
// 该函数应在正式迁移操作前调用
func (s *JmsMigrateAssetService) Prepare() {
	// 预处理，基于 v2, 将数据整理出来
	s.migrateWebPrepare()
	// 由于升级后，网域网关迁移到资产表中，ID 发生变更，需要重新关联应用
	// 应用域关系保存本地，升级后进行关联
	s.migrateDbCloudPrepare()
	// 由于升级后应用合并到资产表，应用 ID 变更，权限需要重新关联
	s.MigrateAppPermPrepare()
}

func (s *JmsMigrateAssetService) migrateDbCloudPrepare() {
	// Web 资产预处理
	appChan := make(chan model.AssetsAsset, 50)
	go func() {
		defer close(appChan)
		err := s.orm.FetchAppDomainAssets(appChan)
		if err != nil {
			slog.Error(fmt.Sprintf("获取 Web 资产信息失败：%s", err.Error()))
		}
	}()
	// 持久化本地
	s.dbCloudPersist(appChan)
}

func (s *JmsMigrateAssetService) dbCloudPersist(appChan chan model.AssetsAsset) {
	filepath := getFilepath(FILENAME_APP_ASSETS)
	// 打开文件（如果不存在则创建，存在则清空内容）
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("打开文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}
	defer f.Close()

	for {
		asset, ok := <-appChan
		if !ok {
			break
		}

		bt, err := json.Marshal(asset)
		if err != nil {
			slog.Error(fmt.Sprintf("序列化 Db/Cloud 资产 [%s(%s)] 失败：%s", asset.Name, asset.Address, err.Error()))
			os.Exit(1)
		}
		line := append(bt, '\n')
		_, err = f.Write(line)
		if err != nil {
			slog.Error(fmt.Sprintf("保存 Db/Cloud 资产 [%s(%s)] 失败：%s", asset.Name, asset.Address, err.Error()))
			os.Exit(1)
		}
	}
}

func (s *JmsMigrateAssetService) migrateWebPrepare() {
	// Web 资产预处理
	webChan := make(chan model.AssetsAsset, 50)
	go func() {
		defer close(webChan)
		err := s.orm.FetchWebAssets(webChan)
		if err != nil {
			slog.Error(fmt.Sprintf("获取 Web 资产信息失败：%s", err.Error()))
		}
	}()
	// 持久化本地
	s.webAssetsPersist(webChan)
}

func (s *JmsMigrateAssetService) webAssetsPersist(webChan chan model.AssetsAsset) {
	filepath := getFilepath(FILENAME_WEB_ASSETS)
	// 打开文件（如果不存在则创建，存在则清空内容）
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("打开文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}
	defer f.Close()

	for {
		asset, ok := <-webChan
		if !ok {
			break
		}

		bt, err := json.Marshal(asset)
		if err != nil {
			slog.Error(fmt.Sprintf("序列化 Web 资产 [%s(%s)] 失败：%s", asset.Name, asset.Address, err.Error()))
			os.Exit(1)
		}
		line := append(bt, '\n')
		_, err = f.Write(line)
		if err != nil {
			slog.Error(fmt.Sprintf("保存 Web 资产 [%s(%s)] 失败：%s", asset.Name, asset.Address, err.Error()))
			os.Exit(1)
		}
	}
}

// Migrate 执行资产迁移操作，包括Web资产和数据库云资产的迁移、应用
// parser: 加解密处理器，用于处理迁移过程中的敏感数据
func (s *JmsMigrateAssetService) Migrate(parser crypto.ICrypto) {
	s.migrateWebAssets(parser)
	s.migrateDbCloudAssets()

	s.MigrateAppPerms()
	// 清理授权规则中的空账号
	s.CleanPermNullAccounts()
}

func (s *JmsMigrateAssetService) migrateWebAssets(parser crypto.ICrypto) {
	fmt.Printf("web资产迁移开始...\n\n\t#### v2 Web 资产迁移到 v3 资产 ####\n\n")
	start := time.Now()
	
	// 基于 v2 整理出来的数据，迁移合并到 v3 中
	filepath, exists := existFile(FILENAME_WEB_ASSETS)
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

		var asset model.AssetsAsset
		if err := json.Unmarshal(lineBytes, &asset); err != nil {
			slog.Error(fmt.Sprintf("反序列化 Web 资产失败：%s", err.Error()))
			os.Exit(1)
		}
		asset.Secret, _ = parser.Encrypt(asset.Password)
		s.orm.AddWebAssetV3(&asset)
	}

	if err := scanner.Err(); err != nil {
		slog.Error(fmt.Sprintf("读取文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("web资产迁移完成，耗时 %f 秒\n", duration)
}

func (s *JmsMigrateAssetService) migrateDbCloudAssets() {
	fmt.Printf("dbcloud资产迁移开始...\n\n\t#### v2 数据库/云资产迁移到 v3 资产 ####\n\n")
	start := time.Now()

	// 基于 v2 整理出来的数据，将数据库/云资产的网域再 v3 中更正
	filepath, exists := existFile(FILENAME_APP_ASSETS)
	if !exists {
		slog.Error(fmt.Sprintf("dbcloud 资产文件[%s]不存在, 请先执行预处理", filepath))
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

		var asset model.AssetsAsset
		if err := json.Unmarshal(lineBytes, &asset); err != nil {
			slog.Error(fmt.Sprintf("反序列化 Web 资产失败：%s", err.Error()))
			os.Exit(1)
		}
		s.orm.SetDBCloudDomainV3(&asset)
	}

	if err := scanner.Err(); err != nil {
		slog.Error(fmt.Sprintf("读取文件[%s]失败：%s", filepath, err.Error()))
		os.Exit(1)
	}

	duration := time.Since(start).Seconds()
	fmt.Printf("dbcloud资产迁移完成，耗时 %f 秒\n", duration)
}
