package models

import "time"

// DiscoverClick 保存主动打开详情的点击，同一用户同一帖子每天最多一条。
type DiscoverClick struct {
	Id        int64     `gorm:"primaryKey;comment:自增主键"`
	UserId    string    `gorm:"type:char(20);not null;uniqueIndex:uk_discover_click,priority:1;comment:点击用户UUID"`
	PostUuid  string    `gorm:"type:char(20);not null;uniqueIndex:uk_discover_click,priority:2;comment:帖子UUID"`
	ClickDay  string    `gorm:"type:date;not null;uniqueIndex:uk_discover_click,priority:3;comment:点击去重日期，按北京时间计算"`
	Source    string    `gorm:"type:varchar(20);not null;comment:首次点击来源频道或列表入口"`
	CreatedAt time.Time `gorm:"type:datetime;not null;index:idx_discover_click_time;comment:当天首次点击时间"`
}

// TableName 返回帖子点击记录表名。
func (DiscoverClick) TableName() string { return "discover_click" }
