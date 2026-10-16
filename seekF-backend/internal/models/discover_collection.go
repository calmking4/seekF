package models

import "time"

type DiscoverCollection struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	UserId     string    `gorm:"column:user_id;index:idx_discover_collection_activity,priority:3;type:char(20);not null;comment:用户uuid"`
	FolderId   int64     `gorm:"column:folder_id;not null;comment:收藏夹id"`
	TargetUuid string    `gorm:"column:target_uuid;index:idx_discover_collection_activity,priority:2;type:char(20);not null;comment:帖子uuid"`
	CreatedAt  time.Time `gorm:"column:created_at;index:idx_discover_collection_activity,priority:1;type:datetime;not null;comment:创建时间"`
}

func (DiscoverCollection) TableName() string {
	return "discover_collection"
}
