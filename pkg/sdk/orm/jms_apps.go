package orm

import (
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/utils"
	"log/slog"
	"os"

	"github.com/samber/lo"
)

func (orm *JMSOrm) GetApplications(appChan chan model.Application) {
	defer close(appChan)
	q := orm.db.Model(&model.Application{}).Where("category in ('db', 'cloud')").Order("id")
	page, limit := 1, 30
	for {
		offset := (page - 1) * limit
		var apps []model.Application
		err := q.Offset(offset).Limit(page).Find(&apps).Error
		if err != nil {
			slog.Error(fmt.Sprintf("查询资产失败：%s", err.Error()))
			os.Exit(0)
		}
		for _, app := range apps {
			appChan <- app
		}
		if len(apps) < limit {
			break
		}
		page += 1
	}
}

func (orm *JMSOrm) GetAppSameSystemUsers(appId string) (systemUsernames []string, err error) {
	sql := `select IF(c.username='', a.username, c.username) as uname
from assets_systemuser as a
join applications_account as c on a.id=c.systemuser_id
WHERE c.app_id = ?
GROUP BY uname
HAVING uname <> '' and COUNT(uname) > 1`

	err = orm.db.Raw(sql, appId).Scan(&systemUsernames).Error
	return
}

func (orm *JMSOrm) GetValidAppSystemUser(appId string, systemUsernames []string) (systemusers []model.AssetSystemUser, err error) {
	// 获取同名账号中连接次数最多的系统账号
	// 获取同名账号中最近连接成功的系统账号

	sql := `
	select c.id,
	IF(c.name='', a.name, c.name) as name, 
	IF(c.username='', a.username, c.username) as username, 
	IF(c.password is null, a.password, c.password) as password, 
	IF(c.private_key='', a.private_key, c.private_key) as private_key, 
	IF(c.public_key='', a.public_key, c.public_key) as public_key, 
	b.connections,
	b.duration,
	b.last_connect_time,
	a.comment
from assets_systemuser as a
join (
	SELECT REPLACE(system_user_id, '-', '') as systemuser_id, 
		count(1) as connections,
		sum(TIMESTAMPDIFF(SECOND, date_start, date_end)) as duration,
		max(date_start) as last_connect_time
	from terminal_session 
	where asset_id = ? and is_finished = true and TIMESTAMPDIFF(SECOND, date_start, date_end) > 0
	group by systemuser_id
) as b on a.id = b.systemuser_id
join applications_account as c on a.id=c.systemuser_id
WHERE c.app_id = ? `

	var loginSystemusers []model.AssetSystemUser
	err = orm.db.Raw(sql, utils.ShowUUID(appId), appId).Scan(&loginSystemusers).Error
	if err != nil {
		return
	}
	systemusers = make([]model.AssetSystemUser, 0)
	if len(loginSystemusers) == 0 {
		return
	}
	// 按用户名分组
	groupLoginSystemusers := lo.GroupBy(loginSystemusers, func(item model.AssetSystemUser) string {
		return item.Username
	})
	for _, lsus := range groupLoginSystemusers {
		suer := lo.MaxBy(lsus, func(a, b model.AssetSystemUser) bool {
			if a.Connections > b.Connections && a.Duration > b.Duration {
				// 连接次数最多且累计连接时长最多
				return true
			} else if a.Duration > b.Duration && a.LastConnectTime > b.LastConnectTime {
				// 累计连接时长最多且是最近一次连接
				return true
			} else if a.Connections > b.Connections || a.Duration > b.Duration {
				// 连接次数最多或累计连接时长最多
				return true
			} else {
				return false
			}
		})
		systemusers = append(systemusers, suer)
	}
	return
}

func (orm *JMSOrm) UpdateAppAccountBySystemUsers(appId string, systemUsers []model.AssetSystemUser) {
	sql := `update applications_account set password = ?,private_key = ?,public_key = ? where app_id = ? and systemuser_id = ?`

	for _, su := range systemUsers {
		if err := orm.db.Exec(sql, su.Password, su.PrivateKey, su.PublicKey, appId, su.Id).Error; err != nil {
			slog.Error(fmt.Sprintf("更新应用账号[%s(%s)]信息失败：%s", su.Name, su.Id, err.Error()))
		}
	}
}

// FetchWebAssets 分页获取所有Web资产并通过channel返回
// 参数:
//
//	webChan: 用于接收资产的channel
//
// 返回值:
//
//	error: 获取过程中发生的错误
//
// 注意:
//
//	函数内部会自动关闭webChan，调用方无需处理
func (orm *JMSOrm) FetchWebAssets(webChan chan model.AssetsAsset) (err error) {
	page, limit := 1, 50
	for {
		offset := (page - 1) * limit
		assets, err := orm.fetchWebAssets(limit, offset)
		if err != nil {
			return err
		}

		for _, asset := range assets {
			webChan <- asset
		}
		if len(assets) < limit {
			break
		}
		page++
	}
	return
}

func (orm *JMSOrm) fetchWebAssets(limit, offset int) (assets []model.AssetsAsset, err error) {
	// 查询所有 web 资产
	sql := `select name,
	JSON_UNQUOTE(JSON_EXTRACT(attrs, '$.chrome_target')) as address,
	JSON_UNQUOTE(JSON_EXTRACT(attrs, '$.chrome_username')) as username,
	JSON_UNQUOTE(JSON_EXTRACT(attrs, '$.chrome_password')) as password, 
	org_id,
	created_by,
	date_created,
	comment
from applications_application 
where category = 'remote_app' and type = 'chrome'
order by name
limit ? offset ?`

	err = orm.db.Raw(sql, limit, offset).Scan(&assets).Error
	return
}

// AddWebAssetV3 添加 Web 类型资产到数据库
// 该方法会执行以下操作：
// 1. 获取 WebSite 平台的 ID
// 2. 在事务中依次插入资产记录、账号记录和 Web 资产特殊字段
// 3. 如果任何一步失败，则回滚整个事务
// 参数 asset 包含要添加的资产信息
// 注意：该方法会为资产和账号自动生成 UUID
func (orm *JMSOrm) AddWebAssetV3(asset *model.AssetsAsset) {
	pid, err := orm.getPlatformIdByName("WebSite")
	if err != nil {
		slog.Error(fmt.Sprintf("获取平台ID失败：%s", err.Error()))
		return
	}
	assetId := utils.NewUUID()
	// 添加资产
	insertAssetSql := `insert ignore into 
	assets_asset(id, name, address, is_active, created_by, date_created, comment, org_id, platform_id, connectivity, date_updated, custom_info, gathered_info) 
	values(?, ?, ?, 1, ?, now(), ?, ?, ?, '-', now(), '{}', '{}')`
	// 添加资产账号
	insertAccountSql := `insert ignore into 
	accounts_account(id, asset_id, org_id, name, username, secret_type, _secret, privileged, source, version, connectivity, comment, created_by, updated_by, date_created, date_updated, is_active) 
	values(?, ?, ?, ?, ?, 'password', ?, 0, 'local', 1, '-', ?, ?, ?, now(), now(), 1)`
	// 设置 web 代填信息，默认设置成不代填
	insertWebSql := `insert ignore into assets_web(asset_ptr_id, autofill, username_selector, password_selector, submit_selector, script) values(?, 'no', 'name=username', 'name=password', 'id=login_button', '[]')`

	// 开启事务添加 web 资产
	tx := orm.db.Begin()
	if err := tx.Exec(insertAssetSql, assetId, asset.Name, asset.Address, asset.CreatedBy, asset.Comment, asset.OrgId, pid).Error; err != nil {
		slog.Error(fmt.Sprintf("添加 Web 资产失败1：%s", err.Error()))
		tx.Rollback()
		return
	}
	accountId := utils.NewUUID()
	if err := tx.Exec(insertAccountSql, accountId, assetId, asset.OrgId, asset.Name, asset.Username, asset.Secret, asset.Comment, asset.CreatedBy, asset.CreatedBy).Error; err != nil {
		slog.Error(fmt.Sprintf("添加 Web 资产账号失败2：%s", err.Error()))
		tx.Rollback()
		return
	}
	if err := tx.Exec(insertWebSql, assetId).Error; err != nil {
		slog.Error(fmt.Sprintf("添加 Web 资产失败3：%s", err.Error()))
		tx.Rollback()
		return
	}
	tx.Commit()
}

func (orm *JMSOrm) getPlatformIdByName(name string) (platformId int64, err error) {
	sql := `select id from assets_platform where name = ?`
	err = orm.db.Raw(sql, name).Scan(&platformId).Error
	return
}

// FetchAppDomainAssets 获取应用域下的资产列表
func (orm *JMSOrm) FetchAppDomainAssets(appChan chan model.AssetsAsset) (err error) {
	page, limit := 1, 50
	for {
		offset := (page - 1) * limit
		assets, err := orm.fetchAppAssets(limit, offset)
		if err != nil {
			return err
		}

		for _, asset := range assets {
			appChan <- asset
		}
		if len(assets) < limit {
			break
		}
		page++
	}
	return
}

func (orm *JMSOrm) fetchAppAssets(limit, offset int) (assets []model.AssetsAsset, err error) {
	// 查询所有 web 资产
	sql := `select a.name,
	IF(category = 'db', JSON_UNQUOTE(JSON_EXTRACT(a.attrs, '$.host')), JSON_UNQUOTE(JSON_EXTRACT(a.attrs, '$.cluster'))) as address,
	a.org_id,
	a.category,
	a.created_by,
	a.date_created,
	a.domain_id,
	d.name as domain_name,
	a.comment
from applications_application as a
left join assets_domain as d on a.domain_id = d.id
where a.category in ('db', 'cloud')
order by a.name
limit ? offset ?`

	err = orm.db.Raw(sql, limit, offset).Scan(&assets).Error
	return
}

func (orm *JMSOrm) getDomainIdByName(name string) (domainId string, err error) {
	sql := `select id from assets_domain where name = ?`
	err = orm.db.Raw(sql, name).Scan(&domainId).Error
	return
}

func (orm *JMSOrm) SetDBCloudDomainV3(asset *model.AssetsAsset) (err error) {
	if asset.DomainName == "" {
		return
	}
	domainId, err := orm.getDomainIdByName(asset.DomainName)
	if err != nil {
		slog.Error(fmt.Sprintf("获取网域[%s]ID失败：%s", asset.DomainName, err.Error()))
		return
	}
	newName := ""
	if asset.Category == "db" {
		newName = fmt.Sprintf("DB-%s", asset.Name)
	} else {
		newName = fmt.Sprintf("Cloud-%s", asset.Name)
	}
	sql := `update assets_asset set domain_id = ? where org_id = ? and name = ? and address = ?`
	err = orm.db.Exec(sql, domainId, asset.OrgId, newName, asset.Address).Error
	if err != nil {
		slog.Error(fmt.Sprintf("设置资产[%s]网域[%s]失败：%s", newName, asset.DomainName, err.Error()))
	}
	return
}

// FetchAppPerms 分页获取应用权限数据并通过通道返回
// 参数:
//
//	permChan: 用于接收权限数据的通道
//
// 返回值:
//
//	error: 获取过程中发生的错误
//
// 说明:
//
//	该方法会分页查询应用权限数据，每页50条，直到获取完所有数据
func (orm *JMSOrm) FetchAppPerms(permChan chan model.ApplicationPerm) (err error) {
	page, limit := 1, 50
	for {
		offset := (page - 1) * limit
		perms, err := orm.fetchAppPerms(limit, offset)
		if err != nil {
			return err
		}

		for _, perm := range perms {
			permChan <- perm
		}
		if len(perms) < limit {
			break
		}
		page++
	}
	return
}

func (orm *JMSOrm) fetchAppPerms(limit, offset int) (perms []model.ApplicationPerm, err error) {
	sql := `select org_id, id, name, app_id, app_name, app_category, JSON_ARRAYAGG(username) as accounts
from (
  select p.org_id, p.id, p.name, a.id as app_id, a.name as app_name, a.category as app_category, u.username
  from (
	select org_id, id, name from perms_applicationpermission order by id limit ? offset ?
  ) as p
  left join perms_applicationpermission_applications as p1 on p.id=p1.applicationpermission_id
  left join applications_application as a on p1.application_id = a.id
  left join perms_applicationpermission_system_users as pu on p.id=pu.applicationpermission_id
  left join assets_systemuser as u on pu.systemuser_id = u.id
  group by p.org_id, p.id, p.name, a.id, a.name, a.category, u.username
) as t
group by org_id, id, name, app_id, app_name, app_category`
	err = orm.db.Raw(sql, limit, offset).Scan(&perms).Error
	return
}

// CompleteAppPermV3 完成应用权限V3的关联操作
// 根据应用类别（db/cloud）构造新的应用名称，查询对应的资产ID
// 并将权限规则与资产进行关联
// 参数:
//   perm: 应用权限对象指针
// 返回:
//   error: 操作过程中出现的错误
func (orm *JMSOrm) CompleteAppPermV3(perm *model.ApplicationPerm) (err error) {
	// 查询 v3 中应用 ID
	newAppName := ""
	if perm.AppCategory == "db" {
		newAppName = fmt.Sprintf("DB-%s", perm.AppName)
	} else {
		newAppName = fmt.Sprintf("Cloud-%s", perm.AppName)
	}
	assetId, err := orm.getAssetIdByNameV3(perm.OrgId, newAppName)
	if err != nil {
		slog.Error(fmt.Sprintf("获取应用[%s]失败：%s", newAppName, err.Error()))
		return
	}
	if len(assetId) <= 0 {
		slog.Warn(fmt.Sprintf("应用[%s]未找到，请核查！", newAppName))
		return
	}
	err = orm.relatedAssetPermV3(perm.Id, assetId)
	if err != nil {
		slog.Error(fmt.Sprintf("授权规则[%s]关联应用资产[%s]失败：%s", perm.Name, newAppName, err.Error()))
		return
	}
	return
}

func (orm *JMSOrm) getAssetIdByNameV3(orgId, name string) (id string, err error) {
	sql := `select id from assets_asset where org_id = ? and name = ?`
	err = orm.db.Raw(sql, orgId, name).Scan(&id).Error
	return
}

func (orm *JMSOrm) relatedAssetPermV3(permId, assetId string) (err error) {
	sql := `INSERT IGNORE INTO perms_assetpermission_assets(assetpermission_id, asset_id) values (?, ?)`
	err = orm.db.Exec(sql, permId, assetId).Error
	return
}