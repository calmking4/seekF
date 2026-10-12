package models

import "time"

// DiscoverMedia 帖子媒体及尺寸信息，旧媒体尺寸为零。
type DiscoverMedia struct {
	Id        int64     `gorm:"column:id;primaryKey;comment:自增id"`
	PostId    int64     `gorm:"column:post_id;index;not null;comment:帖子id"`
	Type      int8      `gorm:"column:type;not null;default:0;comment:媒体类型，0.图片，1.视频"`
	Url       string    `gorm:"column:url;type:varchar(500);not null;comment:文件地址"`
	Width     int       `gorm:"column:width;not null;default:0;comment:媒体宽度，单位像素"`
	Height    int       `gorm:"column:height;not null;default:0;comment:媒体高度，单位像素"`
	SortOrder int       `gorm:"column:sort_order;not null;default:0;comment:排序"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

// TableName 返回帖子媒体表名。
func (DiscoverMedia) TableName() string {
	return "discover_media"
}
