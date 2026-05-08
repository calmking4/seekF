package models

import (
	"time"

	"gorm.io/gorm"
)

type DiscoverCollectionFolder struct {
	Id          int64          `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid        string         `gorm:"column:uuid;uniqueIndex;type:char(20);not null;comment:收藏夹唯一id"`
	UserId      string         `gorm:"column:user_id;index;type:char(20);not null;comment:用户uuid"`
	Name        string         `gorm:"column:name;type:varchar(50);not null;comment:收藏夹名称"`
	Description string         `gorm:"column:description;type:varchar(200);default:'';comment:描述"`
	IsPublic    int8           `gorm:"column:is_public;not null;default:0;comment:0.私密 1.公开"`
	PostCount   int            `gorm:"column:post_count;not null;default:0;comment:帖子数"`
	CreatedAt   time.Time      `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;type:datetime;not null;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index;type:datetime(3);comment:删除时间"`
}

func (DiscoverCollectionFolder) TableName() string {
	return "discover_collection_folder"
}
