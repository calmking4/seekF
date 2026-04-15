package models

import (
	"time"
)

type Knowledge struct {
	Id         int64     `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid       string    `gorm:"column:uuid;uniqueIndex;type:varchar(32);not null;comment:唯一标识"`
	UserId     string    `gorm:"column:user_id;index;type:varchar(32);not null;comment:用户id"`
	FileName   string    `gorm:"column:file_name;type:varchar(255);not null;comment:文件名"`
	FileUrl    string    `gorm:"column:file_url;type:varchar(500);not null;comment:文件URL"`
	FileType   string    `gorm:"column:file_type;type:varchar(10);not null;comment:文件类型"`
	ChunkCount int       `gorm:"column:chunk_count;default:0;comment:向量块数量"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

func (Knowledge) TableName() string {
	return "knowledge"
}
