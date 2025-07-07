package model

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AccountTemplate struct {
	Id             string         `gorm:"id" json:"id"`
	Name           string         `gorm:"name" json:"name"`
	Username       string         `gorm:"username" json:"username"`
	SecretType     string         `gorm:"secret_type" json:"secret_type"`
	Secret         string         `gorm:"column:_secret" json:"_secret"`
	Privileged     bool           `gorm:"privileged" json:"privileged"`
	IsActive       bool           `gorm:"is_active" json:"is_active"`
	AutoPush       bool           `gorm:"auto_push" json:"auto_push"`
	PushParams     string         `gorm:"push_params" json:"push_params"`
	SecretStrategy string         `gorm:"secret_strategy" json:"secret_strategy"`
	PasswordRules  string         `gorm:"password_rules" json:"password_rules"`
	OrgId          string         `gorm:"org_id" json:"org_id"`
	SuFromId       sql.NullString `gorm:"su_from_id" json:"su_from_id"`
	Comment        string         `gorm:"comment" json:"comment"`
	CreatedBy      string         `gorm:"created_by" json:"created_by"`
	UpdatedBy      string         `gorm:"updated_by" json:"updated_by"`
	DateCreated    string         `gorm:"date_created" json:"date_created"`
	DateUpdated    string         `gorm:"date_updated" json:"date_updated"`
}

func (AccountTemplate) TableName() string {
	return "accounts_accounttemplate"
}

func (a *AccountTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	if len(a.PushParams) == 0 {
		a.PushParams = "{}"
	}
	if len(a.SecretStrategy) == 0 {
		a.SecretStrategy = "specific"
	}
	if len(a.PasswordRules) == 0 {
		a.PasswordRules = `{"length": 16, "lowercase": true, "uppercase": true, "digit": true, "symbol": true, "exclude_symbols": ""}`
	}
	if len(a.CreatedBy) == 0 {
		a.CreatedBy = "Administrator"
	}
	if len(a.UpdatedBy) == 0 {
		a.UpdatedBy = a.CreatedBy
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if len(a.DateCreated) == 0 {
		a.DateCreated = now
	}
	if len(a.DateUpdated) == 0 {
		a.DateUpdated = now
	}
	return nil
}

func (a *AccountTemplate) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", a.Name, a.Username, a.Secret, a.SecretType)
}
