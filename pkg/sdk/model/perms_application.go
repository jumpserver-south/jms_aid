package model

type ApplicationPerm struct {
	Id          string `json:"id" gorm:"id"`
	Name        string `json:"name" gorm:"name"`
	AppId       string `json:"app_id" gorm:"app_id"`
	AppName     string `json:"app_name" gorm:"app_name"`
	AppCategory string `json:"app_category" gorm:"app_category"`
	Accounts    string `json:"accounts" gorm:"accounts"`
	OrgId       string `json:"org_id" gorm:"org_id"`
}
