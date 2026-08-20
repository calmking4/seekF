package models

import (
	"time"
)

// KnowledgeChunk 知识库文档分块，存储原始文本
type KnowledgeChunk struct {
	Id        int64     `gorm:"column:id;primaryKey;comment:自增id"`
	DocUUID   string    `gorm:"column:doc_uuid;index;type:varchar(32);not null;comment:文档UUID"`
	ChunkIdx  int       `gorm:"column:chunk_idx;not null;comment:分块序号"`
	Content   string    `gorm:"column:content;type:text;not null;comment:分块文本内容"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;comment:创建时间"`
}

func (KnowledgeChunk) TableName() string {
	return "knowledge_chunk"
}
