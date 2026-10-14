package models

import "time"

// UserFollow 用户与博主的关注关系。
type UserFollow struct {
	Id           int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId       string    `gorm:"column:user_id;index:idx_follow_feed,priority:1;type:char(20);not null;index;comment:关注者uuid"`
	FollowUserId string    `gorm:"column:follow_user_id;index:idx_follow_feed,priority:2;type:char(20);not null;index;comment:被关注者uuid"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

// TableName 返回关注关系表名。
func (UserFollow) TableName() string {
	return "user_follow"
}
