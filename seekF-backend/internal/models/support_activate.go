package models

import "time"

// SupportActivate 创作者助力开通与定价配置。
type SupportActivate struct {
	Id        int64     `gorm:"column:id;primaryKey;comment:自增ID"`
	UserId    string    `gorm:"column:user_id;type:char(20);not null;uniqueIndex:uk_support_activate_user,priority:1;comment:开通助力的创作者用户UUID"`
	PriceFen  int64     `gorm:"column:price_fen;not null;default:1000;comment:每30天助力价格，单位分，500至10000且为100的倍数"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null;comment:开通时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime(3);not null;comment:最后改价时间"`
}

// TableName 返回创作者助力配置表名。
func (SupportActivate) TableName() string { return "support_activate" }
