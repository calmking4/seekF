package models

import "time"

// SupportEntitlement 用户对指定创作者的助力权益汇总。
type SupportEntitlement struct {
	Id          int64      `gorm:"column:id;primaryKey;comment:自增ID"`
	Environment string     `gorm:"column:environment;type:varchar(16);not null;uniqueIndex:uk_support_entitlement_pair,priority:1;comment:支付环境，sandbox沙箱，production正式"`
	UserId      string     `gorm:"column:user_id;type:char(20);not null;uniqueIndex:uk_support_entitlement_pair,priority:2;comment:购买者用户UUID"`
	CreatorId   string     `gorm:"column:creator_id;type:char(20);not null;uniqueIndex:uk_support_entitlement_pair,priority:3;comment:受助力创作者用户UUID"`
	ExpiresAt   *time.Time `gorm:"column:expires_at;type:datetime(3);comment:助力权益截止时间，全部撤销或尚未付款时为空"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:datetime(3);not null;comment:记录创建时间"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:datetime(3);not null;comment:权益最后更新时间"`
}

// TableName 返回助力权益汇总表名。
func (SupportEntitlement) TableName() string { return "support_entitlement" }
