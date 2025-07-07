package model

import (
	"database/sql"
	"encoding/json"
)

const ApplicationTable = "applications_application"

type ApplicationAttrs struct {
	// db
	Host             string         `json:"host,omitempty"`
	Port             int            `json:"port,omitempty"`
	UseSSL           bool           `json:"use_ssl,omitempty"`
	Database         string         `json:"database,omitempty"`
	AllowInvalidCert bool           `json:"allow_invalid_cert,omitempty"`
	CaCert           sql.NullString `json:"ca_cert,omitempty"`
	CertKey          sql.NullString `json:"cert_key,omitempty"`
	ClientCert       sql.NullString `json:"client_cert,omitempty"`

	// k8s
	Cluster string `json:"cluster"`

	// app
	Path  string `json:"path"`
	Asset string `json:"asset"`
}

type Application struct {
	Id          string         `json:"id" gorm:"id"`
	Name        string         `json:"name" gorm:"name"`
	Category    string         `json:"category" gorm:"category"`
	Type        string         `json:"type" gorm:"type"`
	Attrs       []byte         `json:"attrs" gorm:"attrs"`
	DomainId    sql.NullString `json:"domain_id" gorm:"domain_id"`
	OrgId       string         `json:"org_id" gorm:"org_id"`
	CreatedBy   string         `json:"created_by" gorm:"created_by"`
	DateCreated string         `json:"date_created" gorm:"date_created"`
	DateUpdated string         `json:"date_updated" gorm:"date_updated"`
	Comment     string         `json:"comment" gorm:"comment"`
}

func (Application) TableName() string {
	return ApplicationTable
}

func (a Application) GetAttrs() ApplicationAttrs {
	if len(a.Attrs) == 0 {
		return ApplicationAttrs{}
	}
	var attrs ApplicationAttrs
	err := json.Unmarshal(a.Attrs, &attrs)
	if err != nil {
		return ApplicationAttrs{}
	}
	return attrs
}

func (a Application) Address() string {
	attrs := a.GetAttrs()
	switch a.Category {
	case "db":
		return attrs.Host
	case "k8s":
		return attrs.Cluster
	case "app":
		return attrs.Path
	default:
		return ""
	}
}