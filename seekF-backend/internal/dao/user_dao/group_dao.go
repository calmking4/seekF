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
