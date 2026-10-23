package models

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Session struct {
	Id                int64          `gorm:"column:id;primaryKey;comment:自增id"`
	Uuid              string         `gorm:"column:uuid;uniqueIndex;type:char(20);comment:会话uuid"`
	SendId            string         `gorm:"column:send_id;Index;type:char(20);not null;comment:创建会话人id"`
	ReceiveId         string         `gorm:"column:receive_id;Index;type:char(20);not null;comment:接受会话人id"`
	LastMessage       string         `gorm:"column:last_message;type:TEXT;comment:最新的消息"`
	FirstMessage      string         `gorm:"column:first_message;type:TEXT;comment:第一条消息"`
	LastMessageAt     sql.NullTime   `gorm:"column:last_message_at;type:datetime;comment:最近接收时间"`
	CreatedAt         time.Time      `gorm:"column:created_at;Index;type:datetime;comment:创建时间"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;Index;type:datetime;comment:删除时间"`
	LastReadMessageId int64          `gorm:"column:last_read_message_id;not null;default:0;comment:本人已读入站私信的消息主键上界，0表示尚未阅读"`
	PinnedAt          *time.Time     `gorm:"column:pinned_at;type:datetime(3);comment:本人会话置顶时间，NULL表示未置顶"`
}

func (Session) TableName() string {
	return "session"
}
