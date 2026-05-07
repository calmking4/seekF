package models

import (
	"time"

	"gorm.io/gorm"
)

type DiscoverComment struct {
	Id             int64          `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid           string         `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:评论唯一id"`
	PostId         int64          `gorm:"column:post_id;index;not null;comment:帖子id"`
	UserId         string         `gorm:"column:user_id;index;type:char(20);not null;comment:评论者uuid"`
	ParentId       string         `gorm:"column:parent_id;index;type:char(20);comment:父评论uuid，空为顶级评论"`
	ReplyToUserId  string         `gorm:"column:reply_to_user_id;type:char(20);comment:回复目标用户uuid"`
	Content        string         `gorm:"column:content;type:varchar(500);not null;comment:评论内容"`
	LikeCount      int            `gorm:"column:like_count;not null;default:0;comment:点赞数"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index;type:datetime(3);comment:删除时间"`
}

func (DiscoverComment) TableName() string {
	return "discover_comment"
}
