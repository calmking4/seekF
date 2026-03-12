package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	"time"

	"gorm.io/gorm"
)

// CreateUserContact 创建用户联系人关系
func CreateUserContact(contact *models.UserContact) error {
	result := db.GormDB.Create(contact)
	return result.Error
}

// RemoveContact 根据用户ID和联系人ID更新联系人
func RemoveContact(userId string, contactId string) error {
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).Delete(&models.UserContact{})
	return result.Error
}

// RemoveContactApply 根据用户ID和联系人ID删除联系人申请记录
func RemoveContactApply(userId string, contactId string) error {
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).Delete(&models.ContactApply{})
	return result.Error
}

func UpdateUserContactStatusAndDelete(userId string, contactId string, status int8) error {
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	result := db.GormDB.Model(&models.UserContact{}).Where("user_id = ? AND contact_id = ?", userId, contactId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     status,
	})
	return result.Error
}
