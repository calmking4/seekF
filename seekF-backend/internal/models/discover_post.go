package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type DiscoverPost struct {
	Id           int64          `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid         string         `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:帖子唯一id"`
	UserId       string         `gorm:"column:user_id;index;type:char(20);not null;comment:发布者uuid"`
	Title        string         `gorm:"column:title;type:varchar(50);not null;comment:标题"`
	Content      string         `gorm:"column:content;type:text;comment:正文"`
	MediaType    int8           `gorm:"column:media_type;not null;default:0;comment:媒体类型，0.图片，1.视频"`
	Tags         json.RawMessage `gorm:"column:tags;type:json;comment:标签数组"`
	LikeCount    int            `gorm:"column:like_count;not null;default:0;comment:点赞数"`
	CommentCount int            `gorm:"column:comment_count;not null;default:0;comment:评论数"`
	CollectCount int            `gorm:"column:collect_count;not null;default:0;comment:收藏数"`
	Status       int8           `gorm:"column:status;not null;default:0;comment:状态，0.正常，1.下架"`
	CreatedAt    time.Time      `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index;type:datetime(3);comment:删除时间"`
}

func (DiscoverPost) TableName() string {
	return "discover_post"
}
