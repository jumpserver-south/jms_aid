package model

const (
	AssetSystemUserTable = "assets_systemuser"
	AssetAuthbookTable   = "assets_authbook"
)

type AssetSystemUser struct {
	Id         string `json:"id" gorm:"id"`
	Name       string `json:"name" gorm:"name"`
	Username   string `json:"username" gorm:"username"`
	Password   string `json:"password" gorm:"password"`
	PrivateKey string `json:"private_key" gorm:"private_key"`
	PublicKey  string `json:"public_key" gorm:"public_key"`
	Token      string `json:"token" gorm:"token"`
	Protocol   string `json:"protocol" gorm:"protocol"`
	LoginMode  string `json:"login_mode" gorm:"login_mode"`
	ADDomain   string `json:"ad_domain" gorm:"ad_domain"`
	Type       string `json:"type" gorm:"type"`
	OrgId      string `json:"org_id" gorm:"org_id"`
	SuEnabled  bool   `json:"su_enabled" gorm:"su_enabled"`
	SuFromId   string `json:"su_from_id" gorm:"su_from_id"`
	Comment    string `json:"comment" gorm:"comment"`

	Connections     int64  `json:"connections" gorm:"connections"`
	Duration        int64  `json:"duration" gorm:"duration"`
	LastConnectTime string `json:"last_connect_time" gorm:"last_connect_time"`
}

func (c *AssetSystemUser) TableName() string {
	return AssetSystemUserTable
}

func (c *AssetSystemUser) AuthTypes() []string {
	authTypes := make([]string, 0)
	if c.Password != "" {
		authTypes = append(authTypes, "password")
	}
	if c.PrivateKey != "" {
		authTypes = append(authTypes, "ssh_key")
	}
	if c.Token != "" {
		authTypes = append(authTypes, "token")
	}
	return authTypes
}

type AssetAccount struct {
	Id           string `json:"id" gorm:"id"`
	Name         string `json:"name" gorm:"name"`
	Username     string `json:"username" gorm:"username"`
	Password     string `json:"password" gorm:"password"`
	PrivateKey   string `json:"private_key" gorm:"private_key"`
	PublicKey    string `json:"public_key" gorm:"public_key"`
	AssetId      string `json:"asset_id" gorm:"asset_id"`
	SecretType string `json:"secret_type" gorm:"secret_type"`
	SystemUserId string `json:"systemuser_id" gorm:"systemuser_id"`
}

func (a *AssetAccount) AssetAccountsSQL() string {
	// 删除资产账号中不匹配资产协议的账号
	const ds = `DELETE a
FROM assets_authbook a
JOIN assets_systemuser s ON a.systemuser_id = s.id
JOIN assets_asset aa ON a.asset_id = aa.id
WHERE s.login_mode = 'auto'
  AND INSTR(aa.protocols, s.protocol) = 0;`
	// 查询资产账号
	const dd = `select a.id,a.asset_id,a.systemuser_id, 
	IF(a.name='', s.name, a.name) as name, 
	IF(a.username='', IF(s.username='', 'null', s.username), a.username) as username, 
	IFNULL(a.password, IFNULL(s.password, "")) as password,
	IFNULL(a.private_key, IFNULL(s.private_key, "")) as private_key,
	IFNULL(a.public_key, IFNULL(s.public_key, "")) as public_key
from assets_authbook a 
join assets_systemuser s on a.systemuser_id=s.id
join assets_asset aa on a.asset_id=aa.id
where s.login_mode='auto'
	and INSTR(aa.protocols, s.protocol) > 0`
	return `select a.id,a.asset_id,a.systemuser_id, a.name, a.username, IFNULL(a.password, IFNULL(s.password, "")) as password,
	IFNULL(a.private_key, IFNULL(s.private_key, "")) as private_key,IFNULL(a.public_key, IFNULL(s.public_key, "")) as public_key
from assets_authbook a 
join assets_systemuser s on a.systemuser_id=s.id`
}

func (a *AssetAccount) AppAccountsSQL() string {
	return `select a.id,a.app_id as asset_id,a.systemuser_id, a.name, a.username, IFNULL(a.password, IFNULL(s.password, "")) as password,
	IFNULL(a.private_key, IFNULL(s.private_key, "")) as private_key,IFNULL(a.public_key, IFNULL(s.public_key, "")) as public_key
from applications_account a 
join assets_systemuser s on a.systemuser_id=s.id`
}

type AssetAccountList []AssetAccount
