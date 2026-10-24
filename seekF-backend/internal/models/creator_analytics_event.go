package models

import "time"

// CreatorAnalyticsEvent 保存粉丝变化及成功分享，启用标记用于标注历史统计边界。
type CreatorAnalyticsEvent struct {
	Id         int64     `gorm:"primaryKey;comment:事件自增ID"`
	CreatorId  string    `gorm:"type:char(20);not null;index:idx_creator_analytics,priority:1;comment:指标所属创作者UUID，启用标记为空"`
	ActorId    string    `gorm:"type:char(20);not null;uniqueIndex:uk_creator_analytics_request,priority:1;comment:操作用户UUID，启用标记为空"`
	TargetUuid string    `gorm:"type:char(20);not null;comment:分享帖子UUID，粉丝变化及启用标记为空"`
	Kind       string    `gorm:"type:varchar(16);not null;index:idx_creator_analytics,priority:2;comment:事件类型，follower粉丝变化、share分享、started启用标记"`
	Delta      int64     `gorm:"not null;comment:数量增量，关注为1、取消关注为-1、分享为1、启用标记为0，单位次"`
	RequestKey string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_creator_analytics_request,priority:2;comment:操作幂等标识，同一用户内唯一"`
	CreatedAt  time.Time `gorm:"type:datetime(3);not null;index:idx_creator_analytics,priority:3;comment:事件发生时间，按北京时间统计"`
}

// TableName 返回创作者统计事件表名。
func (CreatorAnalyticsEvent) TableName() string { return "creator_analytics_event" }
