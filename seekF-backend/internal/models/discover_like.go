package models

import "time"

// DiscoverLike 用户对帖子或评论的唯一点赞记录。
type DiscoverLike struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId     string    `gorm:"column:user_id;index:idx_discover_like_activity,priority:3;uniqueIndex:uk_discover_like_user_target,priority:1;type:char(20);not null;comment:点赞者uuid"`
	TargetUuid string    `gorm:"column:target_uuid;index:idx_discover_like_activity,priority:2;index:idx_discover_like_target;uniqueIndex:uk_discover_like_user_target,priority:2;type:char(20);not null;comment:目标uuid"`
	CreatedAt  time.Time `gorm:"column:created_at;index:idx_discover_like_activity,priority:1;type:datetime;not null;comment:创建时间"`
}

// TableName 返回发现页点赞表名。
func (DiscoverLike) TableName() string {
	return "discover_like"
}
