package models

import "time"

// UserBlock 用户主动设置的单向拉黑关系。
type UserBlock struct {
	Id            int64     `gorm:"primaryKey;comment:拉黑关系自增主键"`
	UserId        string    `gorm:"type:char(20);not null;uniqueIndex:uk_user_block,priority:1;comment:设置拉黑的用户UUID"`
	BlockedUserId string    `gorm:"type:char(20);not null;uniqueIndex:uk_user_block,priority:2;index:idx_blocked_user;comment:被拉黑用户UUID"`
	CreatedAt     time.Time `gorm:"type:datetime;not null;comment:拉黑时间"`
}

// TableName 返回用户拉黑关系表名。
func (UserBlock) TableName() string { return "user_block" }
