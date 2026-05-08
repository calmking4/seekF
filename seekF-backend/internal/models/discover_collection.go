package models

import "time"

type DiscoverCollection struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId     string    `gorm:"column:user_id;type:char(20);not null;comment:用户uuid"`
	FolderId   int64     `gorm:"column:folder_id;not null;comment:收藏夹id"`
	TargetUuid string    `gorm:"column:target_uuid;type:char(20);not null;comment:帖子uuid"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

func (DiscoverCollection) TableName() string {
	return "discover_collection"
}
