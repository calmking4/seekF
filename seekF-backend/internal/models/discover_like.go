package models

import "time"

type DiscoverLike struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId     string    `gorm:"column:user_id;type:char(20);not null;comment:点赞者uuid"`
	TargetUuid string    `gorm:"column:target_uuid;type:char(20);not null;comment:目标uuid"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

func (DiscoverLike) TableName() string {
	return "discover_like"
}
