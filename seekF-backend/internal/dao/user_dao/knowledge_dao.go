package userdao

import (
	"errors"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

type KnowledgeDAO interface {
	Create(doc *models.Knowledge) error
	FindByUuid(uuid string) (*models.Knowledge, error)
	FindByUserId(userId string) ([]models.Knowledge, error)
	Delete(uuid string) error
}

type KnowledgeDAOImpl struct{}

func NewKnowledgeDAO() KnowledgeDAO {
	return &KnowledgeDAOImpl{}
}

func (d *KnowledgeDAOImpl) Create(doc *models.Knowledge) error {
	result := db.GormDB.Create(doc)
	return result.Error
}

func (d *KnowledgeDAOImpl) FindByUuid(uuid string) (*models.Knowledge, error) {
	var doc models.Knowledge
	result := db.GormDB.Where("uuid = ?", uuid).First(&doc)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &doc, result.Error
}

func (d *KnowledgeDAOImpl) FindByUserId(userId string) ([]models.Knowledge, error) {
	var docs []models.Knowledge
	result := db.GormDB.Where("user_id = ?", userId).Order("created_at DESC").Find(&docs)
	return docs, result.Error
}

func (d *KnowledgeDAOImpl) Delete(uuid string) error {
	result := db.GormDB.Where("uuid = ?", uuid).Delete(&models.Knowledge{})
	return result.Error
}
