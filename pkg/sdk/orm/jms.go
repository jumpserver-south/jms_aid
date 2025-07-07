package orm

import "gorm.io/gorm"

type JMSOrm struct {
	db *gorm.DB
}

func NewJMSOrm(db *gorm.DB) *JMSOrm {
	return &JMSOrm{
		db: db,
	}
}
