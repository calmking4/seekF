package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

// CreateUserContact 创建用户联系人关系
func CreateUserContact(contact *models.UserContact) error {
	result := db.GormDB.Create(contact)
	return result.Error
}
