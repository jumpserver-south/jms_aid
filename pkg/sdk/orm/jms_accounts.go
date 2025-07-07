package orm

import (
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/utils"
	"log/slog"
	"strings"

	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

func (o *JMSOrm) FetchSystemusers(suserChan chan model.AssetSystemUser) (err error) {
	page, limit := 1, 50
	for {
		offset := (page - 1) * limit
		systemusers, err := o.fetchSystemusers(limit, offset)
		if err != nil {
			slog.Error(fmt.Sprintf("分批[limit: %d, offset: %d]获取系统用户失败：%s", limit, offset, err.Error()))
			return err
		}
		for _, systemuser := range systemusers {
			suserChan <- systemuser
		}
		if len(systemusers) < limit {
			break
		}
		page += 1
	}
	return nil
}

// fetchSystemusers 根据分页参数获取自动登录模式的系统用户列表
// 参数:
//
//	limit - 每页记录数
//	offset - 偏移量
//
// 返回:
//
//	[]model.AssetSystemUser - 系统用户列表
//	error - 查询错误
func (o *JMSOrm) fetchSystemusers(limit, offset int) (systemusers []model.AssetSystemUser, err error) {
	sql := `SELECT id,name,username,
				ifnull(password, '') as password,
				ifnull(public_key, '') as public_key,
				ifnull(private_key, '') as private_key, 
				type,
				org_id, comment, created_by
			FROM assets_systemuser where login_mode='auto' order by id LIMIT ? OFFSET ?`
	err = o.db.Raw(sql, limit, offset).Scan(&systemusers).Error
	return
}

// AddAccountTemplateV3 添加系统用户账户模板到数据库（V3版本）
// 使用 OnConflict 子句避免重复插入，如果插入失败会记录错误日志
// 参数 systemuser: 要迁移的资产系统用户对象
func (o *JMSOrm) AddAccountTemplateV3(systemuser *model.AssetSystemUser) {
	templates := toAccountTemplate(systemuser)

	err := o.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&templates).Error
	if err != nil {
		slog.Error(fmt.Sprintf("迁移系统用户[%s]失败：%s", systemuser.Name, err.Error()))
	}
}

func toAccountTemplate(systemuser *model.AssetSystemUser) []model.AccountTemplate {
	templates := make([]model.AccountTemplate, 0)
	if systemuser == nil {
		return templates
	}
	// 特权账号判断
	privileged := systemuser.Type == "admin"
	if !privileged {
		// v2 未标识特权账号情况下，根据账号名判断是否特权账号
		uname := strings.ToLower(systemuser.Username)
		if uname == "root" || uname == "administrator" {
			privileged = true
		}
	}
	authTypes := systemuser.AuthTypes()
	authTypeNum := len(authTypes)
	if lo.Contains(authTypes, "password") {
		t := model.AccountTemplate{
			Id:         systemuser.Id,
			Name:       systemuser.Name,
			Username:   systemuser.Username,
			SecretType: "password",
			Secret:     systemuser.Password,
			Privileged: privileged,
			IsActive:   true,
			AutoPush:   false,
			OrgId:      systemuser.OrgId,
			Comment:    systemuser.Comment,
		}
		// 存在多种认证方式，模板名称添加认证方式后缀
		if authTypeNum > 1 {
			t.Name = fmt.Sprintf("%s-password", systemuser.Name)
		}
		t.Id = utils.NewUUIDBy(t.String())
		templates = append(templates, t)
	}

	if lo.Contains(authTypes, "ssh_key") {
		t := model.AccountTemplate{
			Name:       systemuser.Name,
			Username:   systemuser.Username,
			SecretType: "ssh_key",
			Secret:     systemuser.PrivateKey,
			Privileged: privileged,
			IsActive:   true,
			AutoPush:   false,
			OrgId:      systemuser.OrgId,
			Comment:    systemuser.Comment,
		}
		// 存在多种认证方式，模板名称添加认证方式后缀
		if authTypeNum > 1 {
			t.Name = fmt.Sprintf("%s-sshkey", systemuser.Name)
		}
		t.Id = utils.NewUUIDBy(t.String())

		templates = append(templates, t)
	}

	if lo.Contains(authTypes, "token") {
		t := model.AccountTemplate{
			Name:       systemuser.Name,
			Username:   systemuser.Username,
			SecretType: "token",
			Secret:     systemuser.Token,
			Privileged: privileged,
			IsActive:   true,
			AutoPush:   false,
			OrgId:      systemuser.OrgId,
			Comment:    systemuser.Comment,
		}
		// 存在多种认证方式，模板名称添加认证方式后缀
		if authTypeNum > 1 {
			t.Name = fmt.Sprintf("%s-token", systemuser.Name)
		}
		t.Id = utils.NewUUIDBy(t.String())

		templates = append(templates, t)
	}
	return templates
}

func (o *JMSOrm) FetchManyTypeAccount(accountChan chan model.AssetAccount) (err error) {
	page, limit := 1, 50
	for {
		offset := (page - 1) * limit
		accounts, err := o.fetchManyTypeAccount(limit, offset)
		if err != nil {
			slog.Error(fmt.Sprintf("分批[limit: %d, offset: %d]获取多类型账号失败：%s", limit, offset, err.Error()))
			return err
		}
		for _, account := range accounts {
			accountChan <- account
		}
		if len(accounts) < limit {
			break
		}
		page++
	}
	return 
}

// fetchManyTypeAccount 查询拥有多个类型账号的资产账户
// 参数:
//
//	limit: 返回结果的最大数量
//	offset: 查询结果的偏移量
//
// 返回:
//
//	[]model.AssetAccount: 符合条件的资产账户列表
//	error: 查询过程中遇到的错误
func (o *JMSOrm) fetchManyTypeAccount(limit, offset int) (accounts []model.AssetAccount, err error) {
	sql := `select t1.id, t1.name, t1.username, t1.asset_id, t1.secret_type
	from accounts_account as t1 
	inner join (
		SELECT asset_id, username 
		FROM accounts_account 
		group by asset_id, username 
		having count(distinct id) > 1
	) as t2 on t1.asset_id = t2.asset_id and t1.username = t2.username
	order by id 
	LIMIT ? OFFSET ?`
	err = o.db.Raw(sql, limit, offset).Scan(&accounts).Error
	return
}


// AccountNameStandard 标准化资产账号名称并更新到数据库
// 如果账号名为空则使用用户名，如果账号名包含"-"则替换最后一段为密钥类型
// 最后将标准化后的名称更新到数据库
func (o *JMSOrm) AccountNameStandard(acc *model.AssetAccount) {
	if acc == nil {
		return
	}
	name := acc.Name
	if acc.Name == "" {
		name = acc.Username
	}else {
		if strings.Contains(acc.Name, "-") {
			newName := acc.Name[:strings.LastIndex(acc.Name, "-")]
			name = fmt.Sprintf("%s-%s", newName, acc.SecretType)
		}
	}
	
	sql := `update accounts_account set name = ? where id = ?`
	if err := o.db.Exec(sql, name, acc.Id).Error; err != nil {
		slog.Error(fmt.Sprintf("更新账号[%s]名称[%s -> %s]失败：%s", acc.Id, acc.Name, name, err.Error()))
	}
}