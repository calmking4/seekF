package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// DiscoverPost 发现页帖子及媒体封面信息。
type DiscoverPost struct {
	Id                 int64           `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid               string          `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:帖子唯一id"`
	UserId             string          `gorm:"column:user_id;index;uniqueIndex:uk_discover_publish,priority:1;type:char(20);not null;comment:发布者uuid"`
	Title              string          `gorm:"column:title;type:varchar(50);not null;comment:标题"`
	Content            string          `gorm:"column:content;type:text;comment:正文"`
	MediaType          int8            `gorm:"column:media_type;not null;default:0;comment:媒体类型，0.图片，1.视频"`
	CoverUrl           string          `gorm:"column:cover_url;type:varchar(500);default:;comment:视频封面图URL"`
	CoverWidth         int             `gorm:"column:cover_width;not null;default:0;comment:封面宽度，单位像素"`
	CoverHeight        int             `gorm:"column:cover_height;not null;default:0;comment:封面高度，单位像素"`
	Tags               json.RawMessage `gorm:"column:tags;type:json;comment:标签数组"`
	LikeCount          int             `gorm:"column:like_count;not null;default:0;comment:点赞数"`
	CommentCount       int             `gorm:"column:comment_count;not null;default:0;comment:评论数"`
	CollectCount       int             `gorm:"column:collect_count;not null;default:0;comment:收藏数"`
	ViewCount          int             `gorm:"column:view_count;not null;default:0;comment:浏览量"`
	Status             int8            `gorm:"column:status;not null;default:0;comment:状态，0.已发布，1.已下架，2.审核中，3.未通过"`
	CreatedAt          time.Time       `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
	DeletedAt          gorm.DeletedAt  `gorm:"column:deleted_at;index;type:datetime(3);comment:删除时间"`
	PublishRequestKey  *string         `gorm:"type:varchar(64);collate:utf8mb4_bin;uniqueIndex:uk_discover_publish,priority:2;comment:发布请求标识，历史帖子为空"`
	PublishPayloadHash string          `gorm:"type:char(64);not null;default:;comment:首次发布内容SHA256摘要，编辑时保持不变"`
	CoverAssetId       *int64          `gorm:"index;comment:封面资源ID，关联file_asset，无封面时为空"`
	Visibility         int8            `gorm:"not null;default:0;index;comment:可见性，0.公开，1.仅自己可见"`
	Version            uint64          `gorm:"not null;default:1;comment:编辑版本号"`
}

// TableName 返回发现页帖子表名。
func (DiscoverPost) TableName() string {
	return "discover_post"
}
