package model

const TerminalSessionTable = "terminal_session"

type TerminalSession struct {
	Id           string `json:"id" gorm:"id"`
	User         string `json:"user" gorm:"user"`
	Asset        string `json:"asset" gorm:"asset"`
	SystemUser   string `json:"system_user" gorm:"system_user"`
	LoginFrom    string `json:"login_from" gorm:"login_from"`
	IsFinished   string `json:"is_finished" gorm:"is_finished"`
	HasReplay    bool   `json:"has_replay" gorm:"has_replay"`
	HasCommand   bool   `json:"has_command" gorm:"has_command"`
	DateStart    string `json:"date_start" gorm:"date_start"`
	DateEnd      string `json:"date_end" gorm:"date_end"`
	TerminalId   string `json:"terminal_id" gorm:"terminal_id"`
	RemoteAddr   string `json:"remote_addr" gorm:"remote_addr"`
	Protocol     string `json:"protocol" gorm:"protocol"`
	OrgId        string `json:"org_id" gorm:"org_id"`
	AssetId      string `json:"asset_id" gorm:"asset_id"`
	SystemUserId string `json:"system_user_id" gorm:"system_user_id"`
	UserId       string `json:"user_id" gorm:"user_id"`
	IsSuccess    bool   `json:"is_success" gorm:"is_success"`
}

func (TerminalSession) TableName() string {
	return TerminalSessionTable
}
