package userdao

import (
	"errors"
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

// ==================== KnowledgeDAO ====================

type KnowledgeDAO interface {
	Create(doc *models.Knowledge) error
	FindByUuid(uuid string) (*models.Knowledge, error)
	FindByUserId(userId string) ([]models.Knowledge, error)
	Delete(uuid string) error
}

type KnowledgeDAOImpl struct {
	db *gorm.DB
}

// NewKnowledgeDAO 创建知识库DAO实例
func NewKnowledgeDAO(db *gorm.DB) KnowledgeDAO {
	return &KnowledgeDAOImpl{db: db}
}

// Create 创建知识库文档记录
func (d *KnowledgeDAOImpl) Create(doc *models.Knowledge) error {
	result := d.db.Create(doc)
	return result.Error
}

// FindByUuid 根据UUID查询文档
func (d *KnowledgeDAOImpl) FindByUuid(uuid string) (*models.Knowledge, error) {
	var doc models.Knowledge
	result := d.db.Where("uuid = ?", uuid).First(&doc)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &doc, result.Error
}

// FindByUserId 根据用户ID查询文档列表
func (d *KnowledgeDAOImpl) FindByUserId(userId string) ([]models.Knowledge, error) {
	var docs []models.Knowledge
	result := d.db.Where("user_id = ?", userId).Order("created_at DESC").Find(&docs)
	return docs, result.Error
}

// Delete 根据UUID删除文档
func (d *KnowledgeDAOImpl) Delete(uuid string) error {
	result := d.db.Where("uuid = ?", uuid).Delete(&models.Knowledge{})
	return result.Error
}

// ==================== KnowledgeChunkDAO ====================

type KnowledgeChunkDAO interface {
	BatchCreate(chunks []models.KnowledgeChunk) error
	FindByDocUUID(docUUID string) ([]models.KnowledgeChunk, error)
	FindByDocUUIDsAndChunkIdx(docUUID string, chunkIdxs []int) ([]models.KnowledgeChunk, error)
	FindByDocUUIDs(docUUIDs []string) ([]models.KnowledgeChunk, error)
	DeleteByDocUUID(docUUID string) error
}

type KnowledgeChunkDAOImpl struct {
	db *gorm.DB
}

// NewKnowledgeChunkDAO 创建知识库分块DAO实例
func NewKnowledgeChunkDAO(db *gorm.DB) KnowledgeChunkDAO {
	return &KnowledgeChunkDAOImpl{db: db}
}

// BatchCreate 批量创建分块记录
func (d *KnowledgeChunkDAOImpl) BatchCreate(chunks []models.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	result := d.db.CreateInBatches(chunks, 100) // 每批100条
	return result.Error
}

// FindByDocUUID 根据文档UUID查询所有分块
func (d *KnowledgeChunkDAOImpl) FindByDocUUID(docUUID string) ([]models.KnowledgeChunk, error) {
	var chunks []models.KnowledgeChunk
	result := d.db.Where("doc_uuid = ?", docUUID).Order("chunk_idx ASC").Find(&chunks)
	return chunks, result.Error
}

// FindByDocUUIDsAndChunkIdx 根据文档UUID和分块序号查询分块
func (d *KnowledgeChunkDAOImpl) FindByDocUUIDsAndChunkIdx(docUUID string, chunkIdxs []int) ([]models.KnowledgeChunk, error) {
	if len(chunkIdxs) == 0 {
		return nil, nil
	}
	var chunks []models.KnowledgeChunk
	result := d.db.Where("doc_uuid = ? AND chunk_idx IN ?", docUUID, chunkIdxs).Find(&chunks)
	return chunks, result.Error
}

// FindByDocUUIDs 批量查询多个文档的分块
func (d *KnowledgeChunkDAOImpl) FindByDocUUIDs(docUUIDs []string) ([]models.KnowledgeChunk, error) {
	if len(docUUIDs) == 0 {
		return nil, nil
	}
	var chunks []models.KnowledgeChunk
	result := d.db.Where("doc_uuid IN ?", docUUIDs).Find(&chunks)
	return chunks, result.Error
}

// DeleteByDocUUID 根据文档UUID删除所有分块
func (d *KnowledgeChunkDAOImpl) DeleteByDocUUID(docUUID string) error {
	result := d.db.Where("doc_uuid = ?", docUUID).Delete(&models.KnowledgeChunk{})
	return result.Error
}
