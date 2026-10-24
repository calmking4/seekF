package models

import "time"

// Notification 用户互动通知及已读状态。
type Notification struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId     string    `gorm:"column:user_id;type:char(20);not null;index;comment:接收者uuid"`
	ActorId    string    `gorm:"column:actor_id;type:char(20);not null;comment:触发者uuid"`
	Type       int8      `gorm:"column:type;not null;comment:类型：1点赞帖子，2评论帖子，3回复评论，4关注，5收藏帖子，6点赞评论，7付款确认，8全额退款确认，9收到助力"`
	TargetUuid string    `gorm:"column:target_uuid;type:char(20);comment:目标标识，帖子或评论UUID，收到助力为O前缀订单主键"`
	Content    string    `gorm:"column:content;type:varchar(500);comment:通知内容摘要"`
	IsRead     int8      `gorm:"column:is_read;not null;default:0;comment:是否已读，0.未读，1.已读"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

// TableName 返回通知表名。
func (Notification) TableName() string {
	return "notification"
}
