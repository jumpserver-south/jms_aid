package model

type LoginAssetAcl struct {
	Name      string             `json:"name"`
	Priority  int                `json:"priority"`
	Users     LoginAssetAclParam `json:"users"`
	Assets    LoginAssetAclParam `json:"assets"`
	Accounts  []string           `json:"accounts"`
	Rules     LoginAssetAclRule  `json:"rules"`
	IsActive  bool               `json:"is_active"`
	Comment   string             `json:"comment"`
	Reviewers []string           `json:"reviewers"`
	OrgId     string             `json:"org_id"`
}

type LoginAssetAclParamType string

const (
	LoginAclParamTypeAll   LoginAssetAclParamType = "all"
	LoginAclParamTypeIds   LoginAssetAclParamType = "ids"
	LoginAclParamTypeAttrs LoginAssetAclParamType = "attrs"
)

type LoginAssetAclParam struct {
	Type  LoginAssetAclParamType  `json:"type"`
	Ids   []string                `json:"ids,omitempty"`
	Atrrs LoginAssetAclParamAttrs `json:"attrs,omitempty"`
}

type LoginAssetAclParamAttrs struct {
	Name  string      `json:"name"`
	Match string      `json:"match"`
	Value interface{} `json:"value"`
}

type LoginAssetAclRule struct {
	IPGroup    []string                      `json:"ip_group"`
	TimePeriod []LoginAssetAclRuleTimePeriod `json:"time_period"`
}

type LoginAssetAclRuleTimePeriod struct {
	Id    int    `json:"id"`
	Value string `json:"value"`
}
