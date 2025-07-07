package orm

import (
	"encoding/json"
	"fmt"
	"jms_tools/pkg/sdk/model"
	"jms_tools/pkg/sdk/utils"
	"log/slog"
	"strings"

	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

func (o *JMSOrm) FetchCommandFilterAcls(aclChan chan model.CommandFilterAcl) (err error) {
	page, limit := 1, 50
	for {
		offset := (page - 1) * limit
		cfs, err := o.fetchCommandFilterAcls(limit, offset)
		if err != nil {
			slog.Error(fmt.Sprintf("获取命令过滤器ACL失败：%s", err.Error()))
			return err
		}
		for _, cf := range cfs {
			// 补充命令过滤规则详情
			cf.Rules = o.fetchCommandFilterRulesById(&cf)
			cf.Users = o.fetchCommandFilterUsersById(&cf)
			cf.UserGroups = o.fetchCommandFilterUserGroupsById(&cf)
			cf.Assets = o.fetchCommandFilterAssetsById(&cf)
			cf.Nodes = o.fetchCommandFilterNodesById(&cf)
			cf.Applications = o.fetchCommandFilterAppsById(&cf)
			cf.Accounts = o.fetchCommandFilterAccountsById(&cf)
			aclChan <- cf
		}
		if len(cfs) < limit {
			break
		}
		page += 1
	}
	return nil
}

func (o *JMSOrm) fetchCommandFilterAcls(limit, offset int) (cfs []model.CommandFilterAcl, err error) {
	sql := `select id, name, is_active, comment, org_id, created_by from assets_commandfilter order by org_id, id limit ? offset ?`
	err = o.db.Raw(sql, limit, offset).Scan(&cfs).Error
	return
}

func (o *JMSOrm) fetchCommandFilterRulesById(cf *model.CommandFilterAcl) (rules []model.CommandFilterRule) {
	sql := `select t1.id, t1.type,t1.priority,t1.content,t1.ignore_case,t1.action,t1.comment,ifnull(t2.reviewers, '') as reviewers
		from assets_commandfilterrule as t1
		left join (
			select commandfilterrule_id, group_concat(user_id) as reviewers
			from assets_commandfilterrule_reviewers
			group by commandfilterrule_id
		) as t2 on t1.id=t2.commandfilterrule_id
		where filter_id = ?`
	err := o.db.Raw(sql, cf.Id).Scan(&rules).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]用户失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) fetchCommandFilterUsersById(cf *model.CommandFilterAcl) (userIds []string) {
	sql := `select user_id from assets_commandfilter_users where commandfilter_id = ?`
	err := o.db.Raw(sql, cf.Id).Scan(&userIds).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]用户失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) fetchCommandFilterUserGroupsById(cf *model.CommandFilterAcl) (usergtoupIds []string) {
	sql := `select usergroup_id from assets_commandfilter_user_groups where commandfilter_id = ?`
	err := o.db.Raw(sql, cf.Id).Scan(&usergtoupIds).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]用户失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) fetchCommandFilterAssetsById(cf *model.CommandFilterAcl) (assetIds []string) {
	sql := `select asset_id from assets_commandfilter_assets where commandfilter_id = ?`
	err := o.db.Raw(sql, cf.Id).Scan(&assetIds).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]资产失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) fetchCommandFilterNodesById(cf *model.CommandFilterAcl) (nodeIds []string) {
	sql := `select node_id from assets_commandfilter_nodes where commandfilter_id = ?`
	err := o.db.Raw(sql, cf.Id).Scan(&nodeIds).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]资产节点失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) fetchCommandFilterAppsById(cf *model.CommandFilterAcl) (apps []model.ApplicationTrans) {
	sql := `select t2.id,t2.name,
			IF(category = 'db', JSON_UNQUOTE(JSON_EXTRACT(t2.attrs, '$.host')), JSON_UNQUOTE(JSON_EXTRACT(t2.attrs, '$.cluster'))) as address,
			t2.category,
			t2.type
		from assets_commandfilter_applications as t1
		left join applications_application as t2 on t1.application_id=t2.id
		where t1.commandfilter_id = ?`
	err := o.db.Raw(sql, cf.Id).Scan(&apps).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]资产节点失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) fetchCommandFilterAccountsById(cf *model.CommandFilterAcl) (usernames []string) {
	sql := `select t2.username 
	from assets_commandfilter_system_users as t1 
	join assets_systemuser as t2 on t1.systemuser_id=t2.id 
	where t1.commandfilter_id = ? 
	group by t2.username`
	err := o.db.Raw(sql, cf.Id).Scan(&usernames).Error
	if err != nil {
		slog.Warn(fmt.Sprintf("获取命令过滤器[%s]资产节点失败：%s", cf.Name, err.Error()))
	}
	return
}

func (o *JMSOrm) AddCommandFilterAclV3(acl *model.CommandFilterAcl) {
	// v2 命令过滤对象转换成 v3 对象
	aclV3 := o.commandFilterAclToV3(acl)

	// 根据命令组动作分组, v2 动作在命令组中，v3 在规则中，所以需要根据动作分组
	grouped := lo.GroupBy(acl.Rules, func(cg model.CommandFilterRule) string {
		return cg.ActionEN()
	})
	baseName := aclV3.Name
	for action, rules := range grouped {
		// 生成新的 uuid 、名称、动作
		aclV3.Name = baseName + "-" + action
		aclV3.Action = action

		// 生成相对固定的 id，防止
		aclV3.Id = utils.NewUUIDBy(aclV3.String())

		// v3 中添加命令组
		commandGroups, err := o.addCommandFilterAclV3CommandGroups(acl, rules)
		if err != nil {
			slog.Error(fmt.Sprintf("添加命令过滤规则[%s]命令组[%s]失败：%s", acl.Name, action, err.Error()))
			continue
		}

		if err = o.db.Clauses(clause.OnConflict{DoNothing: true}).Create(aclV3).Error; err != nil {
			slog.Error(fmt.Sprintf("添加命令过滤规则[%s]失败：%s", acl.Name, err.Error()))
			continue
		}
		// 命令过滤规则关联命令组
		relations := lo.Map(commandGroups, func(item model.CommandFilterAclV3CommandGroup, idx int) model.CommandFilterAclV3CommandGroupRelate {
			return model.CommandFilterAclV3CommandGroupRelate{
				CommandfilteraclId: aclV3.Id,
				CommandgroupId:     item.Id,
			}
		})
		if len(relations) > 0 {
			err = o.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&relations).Error
			if err != nil {
				slog.Error(fmt.Sprintf("关联命令过滤器[%s]的规则失败：%s", acl.Name, err.Error()))
				continue
			}
		}

		// 复核/通知用户。v2 是在命令组中设置的复核人，v3 是在规则中设置，这里同动作的命令组复核人进行合并
		reviewers := lo.FlatMap(rules, func(item model.CommandFilterRule, idx int) []model.CommandFilterAclV3Reviewers {
			return lo.Map(item.ReviewerList(), func(id string, idx int) model.CommandFilterAclV3Reviewers {
				return model.CommandFilterAclV3Reviewers{
					CommandfilteraclId: aclV3.Id,
					UserId:             id,
				}
			})
		})
		if len(reviewers) > 0 {
			err = o.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&reviewers).Error
			if err != nil {
				slog.Error(fmt.Sprintf("添加命令过滤器[%s]的规则失败：%s", acl.Name, err.Error()))
				continue
			}
		}
	}
}

func (o *JMSOrm) commandFilterAclToV3(a *model.CommandFilterAcl) *model.CommandFilterAclV3 {
	cf := new(model.CommandFilterAclV3)
	cf.Id = a.Id
	cf.Name = a.Name
	cf.Priority = 50
	cf.Action = "accept" // 默认允许
	cf.IsActive = a.IsActive
	cf.OrgId = a.OrgId
	cf.Comment = a.Comment
	cf.CreatedBy = a.CreatedBy
	cf.UpdatedBy = a.CreatedBy

	accounts := make([]string, 0)
	accounts = append(accounts, "@SPEC")
	accounts = append(accounts, a.Accounts...)
	baccs, _ := json.Marshal(accounts)
	cf.Accounts = string(baccs)

	jassets := &model.CommandFilterAclV3Filter{
		Type: "ids",
		Ids:  a.Assets,
	}
	bassets, _ := json.Marshal(jassets)
	cf.Assets = string(bassets)

	jusers := &model.CommandFilterAclV3Filter{
		Type: "ids",
		Ids:  a.Users,
	}
	busers, _ := json.Marshal(jusers)
	cf.Users = string(busers)

	return cf
}

// addCommandFilterAclV3CommandGroups 为命令过滤器ACL添加V3版本的命令组规则
// 参数:
//   - cf: 命令过滤器ACL对象，包含规则和基本信息
//
// 返回值:
//   - []model.CommandFilterAclV3CommandGroup: 成功创建的命令组规则列表
//   - nil: 当数据库创建失败时返回nil
//
// 功能说明:
//  1. 遍历cf.Rules生成V3格式的命令组规则
//  2. 将生成的规则批量创建到数据库中
//  3. 如果创建失败会记录警告日志并返回nil
func (o *JMSOrm) addCommandFilterAclV3CommandGroups(acl *model.CommandFilterAcl, cfs []model.CommandFilterRule) (commandGroups []model.CommandFilterAclV3CommandGroup, err error) {
	commandGroups = make([]model.CommandFilterAclV3CommandGroup, len(cfs))
	for i, rule := range cfs {
		// 获取审核人姓名添加到备注，以便升级后管理员查看
		reviewers, err := o.GetUserNameByIds(rule.ReviewerList())
		if err != nil {
			slog.Warn(fmt.Sprintf("获取命令过滤器[%s]规则[%s]复核人信息失败：%s", acl.Name, rule.Id, err.Error()))
		}
		comment := fmt.Sprintf("%s\nreviewers: %s", rule.Comment, strings.Join(reviewers, ", "))
		
		commandGroup := model.CommandFilterAclV3CommandGroup{
			Name:       rule.GenerateName(acl.Name),
			Type:       rule.Type,
			Content:    rule.Content,
			IgnoreCase: rule.IgnoreCase,
			Comment:    comment,
			OrgId:      acl.OrgId,
			CreatedBy:  acl.CreatedBy,
			UpdatedBy:  acl.CreatedBy,
			ActionEN:   rule.ActionEN(),
		}
		commandGroup.Id = utils.NewUUIDBy(commandGroup.String())
		commandGroups[i] = commandGroup
	}
	err = o.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&commandGroups).Error
	return
}
