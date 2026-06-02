package userdao

import (
	"errors"
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

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
