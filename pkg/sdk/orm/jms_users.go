package orm

func (o *JMSOrm) GetUserNameByIds(ids []string) (usernames []string, err error) {
	err = o.db.Table("users_user").Where("id in (?)", ids).Pluck("name", &usernames).Error
	return
}