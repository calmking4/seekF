package models

import "time"

// DiscoverFeedInbox 关注流候选收件箱，不保存正文或权限。
type DiscoverFeedInbox struct {
	UserId        string    `gorm:"type:char(20);primaryKey;index:idx_inbox_page,priority:1;comment:接收用户UUID"`
	PostUuid      string    `gorm:"type:char(20);primaryKey;comment:帖子UUID，与接收用户联合唯一"`
	PostCreatedAt time.Time `gorm:"type:datetime;not null;index:idx_inbox_page,priority:2;comment:帖子原发布时间，用于候选保留窗口"`
}

// TableName 返回关注收件箱表名。
func (DiscoverFeedInbox) TableName() string { return "discover_feed_inbox" }
