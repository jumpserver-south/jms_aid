package orm

func (orm *JMSOrm) CleanPermNullAccountsV3() (affected int64, err error) {
	sql := `UPDATE perms_assetpermission SET accounts=JSON_REMOVE(accounts, JSON_UNQUOTE(JSON_SEARCH(accounts, 'one', ''))) WHERE JSON_SEARCH(accounts, 'one', '') is not null`
	res := orm.db.Exec(sql)
	return res.RowsAffected, err
}