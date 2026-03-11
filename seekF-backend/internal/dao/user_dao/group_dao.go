package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

// CreateGroup 创建群组
func CreateGroup(group *models.GroupInfo) error {
	result := db.GormDB.Create(group)
	return result.Error
}

// LoadMyGroup 加载我创建的群组
func LoadMyGroup(ownerId string) ([]models.GroupInfo, error) {
	var groupList []models.GroupInfo
	result := db.GormDB.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList)
	return groupList, result.Error
}
