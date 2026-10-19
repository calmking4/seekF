package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// SupportActivate 创作者助力开通与定价配置。
type SupportActivate struct {
	Id        int64           `gorm:"column:id;primaryKey;comment:自增ID"`
	UserId    string          `gorm:"column:user_id;type:char(20);not null;uniqueIndex:uk_support_activate_user,priority:1;comment:开通助力的创作者用户UUID"`
	Price     decimal.Decimal `gorm:"column:price;type:decimal(10,2);not null;default:10.00;comment:每30天助力价格，单位元，5至100且必须为整数元"`
	CreatedAt time.Time       `gorm:"column:created_at;type:datetime(3);not null;comment:开通时间"`
	UpdatedAt time.Time       `gorm:"column:updated_at;type:datetime(3);not null;comment:最后改价时间"`
}

// TableName 返回创作者助力配置表名。
func (SupportActivate) TableName() string { return "support_activate" }
