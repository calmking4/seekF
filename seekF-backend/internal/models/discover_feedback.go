package models

import "time"

// DiscoverFeedback 保存用户对帖子或作者的不感兴趣状态，存在记录即生效。
type DiscoverFeedback struct {
	Id         int64     `gorm:"primaryKey;comment:自增主键"`
	UserId     string    `gorm:"type:char(20);not null;uniqueIndex:uk_discover_feedback,priority:1;comment:反馈用户UUID"`
	TargetType string    `gorm:"type:varchar(10);not null;uniqueIndex:uk_discover_feedback,priority:2;comment:目标类型，post帖子、author作者"`
	TargetUuid string    `gorm:"type:char(20);not null;uniqueIndex:uk_discover_feedback,priority:3;comment:帖子或作者UUID"`
	CreatedAt  time.Time `gorm:"type:datetime;not null;comment:首次反馈时间"`
}

// TableName 返回不感兴趣记录表名。
func (DiscoverFeedback) TableName() string { return "discover_feedback" }
