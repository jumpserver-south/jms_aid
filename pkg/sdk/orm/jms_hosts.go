package orm

import (
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/utils"
	"log/slog"
	"os"

	"github.com/samber/lo"
)

// UpdateAssetAccountBySystemUsers 根据系统用户列表更新资产账号信息
// 参数:
//
//	systemUsers: 资产系统用户列表
//
// 返回:
//
//	error: 操作错误，成功返回nil
func (orm *JMSOrm) UpdateAssetAccountBySystemUsers(assetId string, systemUsers []model.AssetSystemUser) {
	sql := `update assets_authbook as a 
			join assets_systemuser as s on a.systemuser_id = s.id 
			set a.password = ?,a.private_key = ?,a.public_key = ? 
			where a.asset_id = ? and s.username = ?`

	for _, su := range systemUsers {
		if err := orm.db.Exec(sql, su.Password, su.PrivateKey, su.PublicKey, assetId, su.Username).Error; err != nil {
			slog.Error(fmt.Sprintf("更新账号[%s(%s)]信息失败：%s", su.Name, su.Id, err.Error()))
		}
	}
}

// GetAssetSameSystemUsers 根据资产ID获取具有相同系统用户的用户名列表
// 参数:
//
//	assetId - 资产ID
//
// 返回:
//
//	[]string - 系统用户名列表
//	error - 错误信息
func (orm *JMSOrm) GetAssetSameSystemUsers(assetId string) (systemUsernames []string, err error) {
	sql := `select IF(c.username='', a.username, c.username) as uname
from assets_systemuser as a
join assets_authbook as c on a.id=c.systemuser_id
WHERE c.asset_id = ?
GROUP BY uname
HAVING uname <> '' and COUNT(uname) > 1`

	err = orm.db.Raw(sql, assetId).Scan(&systemUsernames).Error
	return
}

// GetValidSystemUser 根据资产ID和系统用户名列表，获取有效的系统用户列表
// 优先返回同名账号中连接次数最多或最近连接成功的系统账号
// 参数:
//
//	assetId: 资产ID
//	systemUsernames: 系统用户名列表
//
// 返回:
//
//	[]model.AssetSystemUser: 符合条件的系统用户列表
//	error: 查询错误信息
func (orm *JMSOrm) GetValidSystemUser(assetId string, systemUsernames []string) (systemusers []model.AssetSystemUser, err error) {
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
join assets_authbook as c on a.id=c.systemuser_id
WHERE c.asset_id = ? `

	var loginSystemusers []model.AssetSystemUser
	err = orm.db.Raw(sql, utils.ShowUUID(assetId), assetId).Scan(&loginSystemusers).Error
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

// GetAssets 从数据库分页获取所有资产记录，并通过 channel 返回
// 参数 assetChan: 用于接收资产记录的 channel
// 注意: 函数会在处理完所有记录后自动关闭 channel
// 错误处理: 如果查询失败会记录错误并退出程序
func (orm *JMSOrm) GetAssets(assetChan chan model.AssetsAsset) {
	sql := `select id,ip as address,hostname as name,protocols,is_active,
				public_ip,created_by,date_created,comment,domain_id,org_id,
				platform_id,admin_user_id 
			from assets_asset 
			order by id limit ? offset ?`
	page, limit := 1, 30
	for {
		offset := (page - 1) * limit
		var assets []model.AssetsAsset
		err := orm.db.Raw(sql, limit, offset).Scan(&assets).Error
		if err != nil {
			slog.Error(fmt.Sprintf("查询资产失败：%s", err.Error()))
			os.Exit(0)
		}
		for _, asset := range assets {
			assetChan <- asset
		}
		if len(assets) < limit {
			break
		}
		page += 1
	}
}

// DeleteIllegalAssetAccounts 清理非法的资产账号（即资产协议组中未包含系统账号协议的资产账号）
// 返回:
//   - affected: 删除的记录数
//   - err: 执行过程中发生的错误
func (orm *JMSOrm) DeleteIllegalAssetAccounts() (affected int64, err error) {
	sql := `DELETE a 
FROM assets_authbook a 
JOIN assets_systemuser s ON a.systemuser_id = s.id 
JOIN assets_asset aa ON a.asset_id = aa.id 
WHERE s.login_mode = 'auto' AND INSTR(aa.protocols, s.protocol) = 0;`
	query := orm.db.Exec(sql)
	return query.RowsAffected, query.Error
}
