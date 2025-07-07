package service

import (
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/orm"
	"log/slog"
	"os"
	"sync"

	"gorm.io/gorm"
)

// JMSMergeAccountService 账号合并
type JMSMergeAccountService struct {
	orm *orm.JMSOrm
}

func NewJMSMergeAccountService(db *gorm.DB) *JMSMergeAccountService {
	return &JMSMergeAccountService{
		orm: orm.NewJMSOrm(db),
	}
}

func (s *JMSMergeAccountService) Run() {
	// 主机资产同名账号合并
	s.MergeHostAccounts()
	// 应用资产同名账号合并
	s.MergeAppAccounts()
}

// MergeHostAccounts 处理主机资产下的同名账号合并
// 1. 首先清理资产下的非法账号
// 2. 通过channel获取资产数据
// 3. 对每个资产执行同名账号合并操作
func (s *JMSMergeAccountService) MergeHostAccounts() {
	// 清理非法的资产账号(即资产协议组中未包含系统账号协议的资产账号)
	s.DeleteIllegalAssetAccounts()

	fmt.Println("开始处理主机资产同名账号.....")
	assetChan := make(chan model.AssetsAsset, 30)
	// 获取资产
	go func() {
		defer close(assetChan)
		s.orm.GetAssets(assetChan)
	}()

	// 并发处理
	var wg sync.WaitGroup
	wg.Add(appConfig.Workers)

	for i := 0; i < appConfig.Workers; i++ {
		go func() {
			defer wg.Done()
			for asset := range assetChan {
				// 主机资产同名账号合并
				s.mergeHostAccount(asset)
			}
		}()
	}
	wg.Wait()

	fmt.Println("主机资产同名账号处理完成。")
}

// DeleteIllegalAssetAccounts 清理非法资产账号
// 该方法会删除数据库中所有非法的资产账号记录
// 返回:
//   - 如果清理过程中发生错误，会记录错误日志并退出程序
//   - 成功时记录清理的账号数量
func (s *JMSMergeAccountService) DeleteIllegalAssetAccounts() {
	fmt.Println("开始清理非法的资产账号(即资产协议组中未包含系统账号协议的资产账号)....")
	affected, err := s.orm.DeleteIllegalAssetAccounts()
	if err != nil {
		slog.Error(fmt.Sprintf("清理非法资产账号失败：%s", err.Error()))
		os.Exit(1)
	}
	fmt.Printf("非法的资产账号清理完成，共清理 %d 个账号\n", affected)
}

// mergeHostAccount 合并资产下的同名账号
// 1. 查询资产同名账号
// 2. 通过会话记录表验证并获取有效的系统用户ID
// 3. 使用有效系统用户ID更新资产下同名账号的密码/密钥
// 4. 未处理的同名账号将被记录（TODO待实现）
// 参数 asset: 需要合并账号的资产对象
func (s *JMSMergeAccountService) mergeHostAccount(asset model.AssetsAsset) {
	// 查询资产同名账号
	systemUsernames, err := s.orm.GetAssetSameSystemUsers(asset.Id)
	if err != nil {
		slog.Error(fmt.Sprintf("查询资产 %s(%s):%s 可用系统用户失败：%s", asset.Name, asset.Address, asset.Id, err.Error()))
		return
	}

	// 通过会话记录表查询资产正确系统用户 ID
	systemUsers, err := s.orm.GetValidSystemUser(asset.Id, systemUsernames)
	if err != nil {
		slog.Error(fmt.Sprintf("查询资产 %s(%s):%s 可用系统用户失败：%s", asset.Name, asset.Address, asset.Id, err.Error()))
		return
	}

	// 通过可用系统用户 ID 更新资产下同名账号的密码/密钥
	s.orm.UpdateAssetAccountBySystemUsers(asset.Id, systemUsers)

	// TODO 未处理的同名账号考虑记录下来
}

// MergeAppAccounts 合并应用账号，处理数据库和k8s资产账号的合并逻辑
func (s *JMSMergeAccountService) MergeAppAccounts() {
	fmt.Println("开始处理应用(数据库、k8s)资产同名账号.....")
	appChan := make(chan model.Application, 30)
	// 获取资产
	go s.orm.GetApplications(appChan)

	// 并发处理
	var wg sync.WaitGroup
	wg.Add(appConfig.Workers)

	for i := 0; i < appConfig.Workers; i++ {
		go func() {
			defer wg.Done()
			for app := range appChan {
				// 应用资产同名账号合并
				s.mergeAppAccount(app)
			}
		}()
	}
	wg.Wait()

	fmt.Println("主机资产同名账号处理完成。")
}

// mergeAppAccount 合并应用账号
// 1. 查询资产同名系统账号
// 2. 通过会话记录验证有效系统用户ID
// 3. 使用有效系统用户更新应用账号密码/密钥
// 参数 app: 待合并账号的应用对象
func (s *JMSMergeAccountService) mergeAppAccount(app model.Application) {
	// 查询资产同名账号
	systemUsernames, err := s.orm.GetAppSameSystemUsers(app.Id)
	if err != nil {
		slog.Error(fmt.Sprintf("查询资产 %s(%s):%s 可用系统用户失败：%s", app.Name, app.GetAttrs().Host, app.Id, err.Error()))
		return
	}

	// 通过会话记录表查询资产正确系统用户 ID
	systemUsers, err := s.orm.GetValidAppSystemUser(app.Id, systemUsernames)
	if err != nil {
		slog.Error(fmt.Sprintf("查询资产 %s(%s):%s 可用系统用户失败：%s", app.Name, app.Address(), app.Id, err.Error()))
		return
	}

	// 通过可用系统用户 ID 更新资产下同名账号的密码/密钥
	s.orm.UpdateAppAccountBySystemUsers(app.Id, systemUsers)
}
