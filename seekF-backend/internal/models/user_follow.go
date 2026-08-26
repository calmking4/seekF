package models

import "time"

type UserFollow struct {
	Id           int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId       string    `gorm:"column:user_id;type:char(20);not null;index;comment:关注者uuid"`
	FollowUserId string    `gorm:"column:follow_user_id;type:char(20);not null;index;comment:被关注者uuid"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

func (UserFollow) TableName() string {
	return "user_follow"
}
