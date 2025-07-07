package model

import (
	"fmt"
	"jms_tools/pkg/sdk/utils"
	"strings"
	"time"

	"gorm.io/gorm"
)

type CommandFilterAcl struct {
	Id        string `json:"id" gorm:"id"`
	Name      string `json:"filter" gorm:"name"`
	IsActive  bool   `json:"filter_type" gorm:"is_active"`
	Comment   string `json:"comment" gorm:"comment"`
	OrgId     string `json:"org_id" gorm:"org_id"`
	CreatedBy string `json:"created_by" gorm:"created_by"`

	Rules        []CommandFilterRule `json:"rules" gorm:"-"`
	Assets       []string            `json:"assets" gorm:"-"`
	Applications []ApplicationTrans  `json:"applications" gorm:"-"`
	Nodes        []string            `json:"nodes" gorm:"-"`
	Users        []string            `json:"users" gorm:"-"`
	UserGroups   []string            `json:"user_groups" gorm:"-"`
	Accounts     []string            `json:"accounts" gorm:"-"`
}

type ApplicationTrans struct {
	Id       string `json:"id" gorm:"id"`
	Name     string `json:"name" gorm:"name"`
	Address  string `json:"address" gorm:"address"`
	Category string `json:"category" gorm:"category"`
	Type     string `json:"type" gorm:"type"`
}

type CommandFilterRule struct {
	Id         string `json:"id" gorm:"id"`
	Type       string `json:"type" gorm:"type"`
	Priority   int    `json:"priority" gorm:"priority"`
	Content    string `json:"content" gorm:"content"`
	Action     int    `json:"action" gorm:"action"`
	IgnoreCase bool   `json:"ignore_case" gorm:"ignore_case"`
	Comment    string `json:"comment" gorm:"comment"`

	Reviewers string `json:"reviewers" gorm:"reviewers"`
}

func (r *CommandFilterRule) ActionCN() string {
	switch r.Action {
	case 9:
		return "允许"
	case 0:
		return "拒绝"
	case 2:
		return "复核"
	default:
		return "未知"
	}
}

func (r *CommandFilterRule) ActionEN() string {
	switch r.Action {
	case 9:
		return "accept"
	case 0:
		return "reject"
	case 2:
		return "review"
	default:
		return "accept"
	}
}

// GenerateName 根据前缀生成规则名称，格式为"前缀-动作类型-随机后缀"
// 前缀由参数指定，动作类型来自规则自身的ActionCN()方法，随机后缀取UUID的最后4位
func (r *CommandFilterRule) GenerateName(prefix string) string {
	code := fmt.Sprintf("%s%s%s%s", prefix, r.ActionEN(), r.Content, r.Comment)
	uid := utils.NewUUIDBy(code)
	last := uid[len(uid)-4:]
	return prefix + "-" + r.ActionEN() + "-" + last
}

// ReviewerList 返回命令过滤规则的审阅者列表，以逗号分隔的字符串形式存储在 Reviewers 字段中。
// 如果 Reviewers 为空，则返回空列表。

func (r *CommandFilterRule) ReviewerList() (reviewers []string) {
	if len(r.Reviewers) == 0 {
		return
	} else {
		return strings.Split(r.Reviewers, ",")
	}
}

type CommandFilterAclV3 struct {
	Id          string `json:"id" gorm:"id"`
	Name        string `json:"name" gorm:"name"`
	Priority    int    `json:"priority" gorm:"priority"`
	Action      string `json:"action" gorm:"action"` // accept 、reject 、review、warning
	IsActive    bool   `json:"is_active" gorm:"is_active"`
	Accounts    string `json:"accounts" gorm:"accounts"`
	Assets      string `json:"assets" gorm:"assets"`
	Users       string `json:"users" gorm:"users"`
	Comment     string `json:"comment" gorm:"comment"`
	OrgId       string `json:"org_id" gorm:"org_id"`
	CreatedBy   string `json:"created_by" gorm:"created_by"`
	UpdatedBy   string `json:"update_by" gorm:"update_by"`
	DateCreated string `json:"date_created" gorm:"column:date_created"`
	DateUpdated string `json:"date_updated" gorm:"column:date_updated"`
}

func (CommandFilterAclV3) TableName() string {
	return "acls_commandfilteracl"
}

func (c *CommandFilterAclV3) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if len(c.DateCreated) == 0 {
		c.DateCreated = now
	}
	if len(c.DateUpdated) == 0 {
		c.DateUpdated = now
	}
	return nil
}

func (c *CommandFilterAclV3) BeforeUpdate(tx *gorm.DB) (err error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if len(c.DateUpdated) == 0 {
		c.DateUpdated = now
	}
	return nil
}

func (c *CommandFilterAclV3) String() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s", c.Action, c.Name, c.Users, c.Assets, c.Accounts, c.Comment)
}

type CommandFilterAclV3Filter struct {
	Type  string                          `json:"type" gorm:"type"` // all 、ids 、attrs
	Ids   []string                        `json:"ids,omitempty" gorm:"ids"`
	Attrs []CommandFilterAclV3FilterAttrs `json:"attrs,omitempty" gorm:"attrs"`
}

type CommandFilterAclV3FilterAttrs struct {
	Name  string   `json:"name" gorm:"name"`
	Match string   `json:"match" gorm:"match"`
	Value []string `json:"value" gorm:"value"`
}

type CommandFilterAclV3CommandGroup struct {
	Id          string `json:"id" gorm:"id"`
	Name        string `json:"name" gorm:"name"`
	Type        string `json:"type" gorm:"type"`
	Content     string `json:"content" gorm:"content"`
	IgnoreCase  bool   `json:"ignore_case" gorm:"ignore_case"`
	Comment     string `json:"comment" gorm:"comment"`
	OrgId       string `json:"org_id" gorm:"org_id"`
	CreatedBy   string `json:"created_by" gorm:"created_by"`
	UpdatedBy   string `json:"update_by" gorm:"update_by"`
	DateCreated string `json:"date_created" gorm:"column:date_created"`
	DateUpdated string `json:"date_updated" gorm:"column:date_updated"`

	ActionEN  string   `json:"-" gorm:"-"`
	Reviewers []string `json:"-" gorm:"-"`
}

func (CommandFilterAclV3CommandGroup) TableName() string {
	return "acls_commandgroup"
}

func (c *CommandFilterAclV3CommandGroup) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if len(c.DateCreated) == 0 {
		c.DateCreated = now
	}
	if len(c.DateUpdated) == 0 {
		c.DateUpdated = now
	}
	return nil
}

func (c *CommandFilterAclV3CommandGroup) BeforeUpdate(tx *gorm.DB) (err error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	if len(c.DateUpdated) == 0 {
		c.DateUpdated = now
	}
	return nil
}

func (c *CommandFilterAclV3CommandGroup) String() string {
	return fmt.Sprintf("%s:%s:%s:%t:%s:%s:%s", c.Type, c.Name, c.Content, c.IgnoreCase, c.Comment, c.ActionEN, c.OrgId)
}

type CommandFilterAclV3CommandGroupRelate struct {
	Id                 int    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	CommandfilteraclId string `json:"commandfilteracl_id" gorm:"commandfilteracl_id"`
	CommandgroupId     string `json:"commandgroup_id" gorm:"commandgroup_id"`
}

func (CommandFilterAclV3CommandGroupRelate) TableName() string {
	return "acls_commandfilteracl_command_groups"
}

type CommandFilterAclV3Reviewers struct {
	Id                 int    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	CommandfilteraclId string `json:"commandfilteracl_id" gorm:"commandfilteracl_id"`
	UserId             string `json:"user_id" gorm:"user_id"`
}

func (CommandFilterAclV3Reviewers) TableName() string {
	return "acls_commandfilteracl_reviewers"
}
