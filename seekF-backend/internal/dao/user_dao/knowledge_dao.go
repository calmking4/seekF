package userdao

import (
	"context"
	"errors"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

// ==================== KnowledgeDAO ====================

// KnowledgeDAO 提供知识库文档记录的数据访问能力。
type KnowledgeDAO interface {
	// 创建知识库文档记录
	Create(ctx context.Context, doc *models.Knowledge) error
	// 根据UUID查询文档
	FindByUuid(ctx context.Context, uuid string) (*models.Knowledge, error)
	// 根据用户ID查询文档列表
	FindByUserId(ctx context.Context, userId string) ([]models.Knowledge, error)
	// 根据UUID删除文档
	Delete(ctx context.Context, uuid string) error
}

// KnowledgeDAOImpl 使用指定数据库或事务连接访问知识库文档。
type KnowledgeDAOImpl struct {
	db *gorm.DB
}

// NewKnowledgeDAO 创建知识库DAO实例
func NewKnowledgeDAO(database *gorm.DB) KnowledgeDAO {
	return &KnowledgeDAOImpl{db: database}
}

// Create 创建知识库文档记录
func (d *KnowledgeDAOImpl) Create(ctx context.Context, doc *models.Knowledge) error {
	result := d.db.WithContext(ctx).Create(doc)
	return result.Error
}

// FindByUuid 根据UUID查询文档
func (d *KnowledgeDAOImpl) FindByUuid(ctx context.Context, uuid string) (*models.Knowledge, error) {
	var doc models.Knowledge
	result := d.db.WithContext(ctx).Where("uuid = ?", uuid).First(&doc)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &doc, result.Error
}

// FindByUserId 根据用户ID查询文档列表
func (d *KnowledgeDAOImpl) FindByUserId(ctx context.Context, userId string) ([]models.Knowledge, error) {
	var docs []models.Knowledge
	result := d.db.WithContext(ctx).Where("user_id = ?", userId).Order("created_at DESC").Find(&docs)
	return docs, result.Error
}

// Delete 根据UUID删除文档
func (d *KnowledgeDAOImpl) Delete(ctx context.Context, uuid string) error {
	result := d.db.WithContext(ctx).Where("uuid = ?", uuid).Delete(&models.Knowledge{})
	return result.Error
}

// ==================== KnowledgeChunkDAO ====================

// KnowledgeChunkDAO 提供知识库文本分块的数据访问能力。
type KnowledgeChunkDAO interface {
	// 批量创建分块记录
	BatchCreate(ctx context.Context, chunks []models.KnowledgeChunk) error
	// 根据文档UUID查询所有分块
	FindByDocUUID(ctx context.Context, docUUID string) ([]models.KnowledgeChunk, error)
	// 根据文档UUID和分块序号查询分块
	FindByDocUUIDsAndChunkIdx(ctx context.Context, docUUID string, chunkIdxs []int) ([]models.KnowledgeChunk, error)
	// 批量查询多个文档的分块
	FindByDocUUIDs(ctx context.Context, docUUIDs []string) ([]models.KnowledgeChunk, error)
	// 根据文档UUID删除所有分块
	DeleteByDocUUID(ctx context.Context, docUUID string) error
}

// KnowledgeChunkDAOImpl 使用指定数据库或事务连接访问文档分块。
type KnowledgeChunkDAOImpl struct {
	db *gorm.DB
}

// NewKnowledgeChunkDAO 创建知识库分块DAO实例
func NewKnowledgeChunkDAO(database *gorm.DB) KnowledgeChunkDAO {
	return &KnowledgeChunkDAOImpl{db: database}
}

// BatchCreate 批量创建分块记录
func (d *KnowledgeChunkDAOImpl) BatchCreate(ctx context.Context, chunks []models.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	result := d.db.WithContext(ctx).CreateInBatches(chunks, 100) // 每批100条
	return result.Error
}

// FindByDocUUID 根据文档UUID查询所有分块
func (d *KnowledgeChunkDAOImpl) FindByDocUUID(ctx context.Context, docUUID string) ([]models.KnowledgeChunk, error) {
	var chunks []models.KnowledgeChunk
	result := d.db.WithContext(ctx).Where("doc_uuid = ?", docUUID).Order("chunk_idx ASC").Find(&chunks)
	return chunks, result.Error
}

// FindByDocUUIDsAndChunkIdx 根据文档UUID和分块序号查询分块
func (d *KnowledgeChunkDAOImpl) FindByDocUUIDsAndChunkIdx(ctx context.Context, docUUID string, chunkIdxs []int) ([]models.KnowledgeChunk, error) {
	if len(chunkIdxs) == 0 {
		return nil, nil
	}
	var chunks []models.KnowledgeChunk
	result := d.db.WithContext(ctx).Where("doc_uuid = ? AND chunk_idx IN ?", docUUID, chunkIdxs).Find(&chunks)
	return chunks, result.Error
}

// FindByDocUUIDs 批量查询多个文档的分块
func (d *KnowledgeChunkDAOImpl) FindByDocUUIDs(ctx context.Context, docUUIDs []string) ([]models.KnowledgeChunk, error) {
	if len(docUUIDs) == 0 {
		return nil, nil
	}
	var chunks []models.KnowledgeChunk
	result := d.db.WithContext(ctx).Where("doc_uuid IN ?", docUUIDs).Find(&chunks)
	return chunks, result.Error
}

// DeleteByDocUUID 根据文档UUID删除所有分块
func (d *KnowledgeChunkDAOImpl) DeleteByDocUUID(ctx context.Context, docUUID string) error {
	result := d.db.WithContext(ctx).Where("doc_uuid = ?", docUUID).Delete(&models.KnowledgeChunk{})
	return result.Error
}
