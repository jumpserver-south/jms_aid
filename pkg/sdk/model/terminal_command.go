package model

const TerminalCommandTable = "terminal_command"

type TerminalCommand struct {
	Id         string `json:"id" gorm:"id"`
	User       string `json:"user" gorm:"user"`
	Asset      string `json:"asset" gorm:"asset"`
	SystemUser string `json:"system_user" gorm:"system_user"`
	Input      string `json:"input" gorm:"input"`
	Output     string `json:"output" gorm:"output"`
	Session    string `json:"session" gorm:"session"`
	Timestamp  int64  `json:"timestamp" gorm:"timestamp"`
	OrgId      string `json:"org_id" gorm:"org_id"`
	RiskLevel  int    `json:"risk_level" gorm:"risk_level"`
}

func (TerminalCommand) TableName() string {
	return TerminalCommandTable
}
