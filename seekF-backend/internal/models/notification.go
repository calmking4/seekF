package models

import "time"

type Notification struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId     string    `gorm:"column:user_id;type:char(20);not null;index;comment:接收者uuid"`
	ActorId    string    `gorm:"column:actor_id;type:char(20);not null;comment:触发者uuid"`
	Type       int8      `gorm:"column:type;not null;comment:类型：1.点赞帖子，2.评论帖子，3.回复评论，4.关注，5.收藏帖子"`
	TargetUuid string    `gorm:"column:target_uuid;type:char(20);comment:目标uuid（帖子/评论）"`
	Content    string    `gorm:"column:content;type:varchar(500);comment:通知内容摘要"`
	IsRead     int8      `gorm:"column:is_read;not null;default:0;comment:是否已读，0.未读，1.已读"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

func (Notification) TableName() string {
	return "notification"
}
