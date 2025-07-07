package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

const AssetsAssetTable = "assets_asset"

type AssetsAsset struct {
	Id          string `json:"id" gorm:"id"`
	Address     string `json:"address" gorm:"address"`
	Name        string `json:"name" gorm:"name"`
	IsActive    bool   `json:"is_active" gorm:"is_active"`
	PublicIp    string `json:"public_ip" gorm:"public_ip"`
	DomainId    string `json:"domain_id" gorm:"domain_id"`
	Protocols   string `json:"protocols" gorm:"protocols"`
	OrgId       string `json:"org_id" gorm:"org_id"`
	PlatformId  int    `json:"platform_id" gorm:"platform_id"`
	AdminUserId string `json:"admin_user_id" gorm:"admin_user_id"`
	CreatedBy   string `json:"created_by" gorm:"created_by"`
	DateCreated string `json:"date_created" gorm:"date_created"`
	Comment     string `json:"comment" gorm:"comment"`

	Category string `json:"category" gorm:"category"`
	// web
	Username string `json:"username" gorm:"username"`
	Password string `json:"password" gorm:"password"`
	Secret   string `json:"secret" gorm:"-"`

	// db
	Database      string `json:"database" gorm:"database"`
	DomainName    string `json:"domain_name" gorm:"domain_name"`
	DomainAddress string `json:"domain_address" gorm:"domain_address"`
}

func (AssetsAsset) TableName() string {
	return AssetsAssetTable
}

func (a AssetsAsset) ProtocolList() []AssetProtocolMini {
	ps := strings.Split(a.Protocols, " ")
	return lo.Map(ps, func(item string, index int) AssetProtocolMini {
		np := strings.Split(item, "/")
		port, _ := strconv.Atoi(np[1])
		return AssetProtocolMini{
			Name: np[0],
			Port: port,
		}
	})
}

// CommentIP 公网 IP 写入 Comment
func (a AssetsAsset) CommentIP() string {
	if a.PublicIp == "" {
		return a.Comment
	}
	if a.Comment == "" {
		return fmt.Sprintf("public_ip: %s", a.PublicIp)
	}
	return fmt.Sprintf("%s\n\npublic_ip: %s", a.Comment, a.PublicIp)
}
