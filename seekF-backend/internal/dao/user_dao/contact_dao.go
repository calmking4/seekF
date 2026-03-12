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

// RemoveContactsByContactId 批量删除指定联系ID的联系人
func RemoveContactsByContactId(contactId string) error {
	result := db.GormDB.Where("contact_id = ?", contactId).Delete(&models.UserContact{})
	return result.Error
}

// RemoveContactApply 根据用户ID和联系人ID删除联系人申请记录
func RemoveContactApply(userId string, contactId string) error {
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).Delete(&models.ContactApply{})
	return result.Error
}

// RemoveContactAppliesByContactId 批量删除指定联系ID的申请记录
func RemoveContactAppliesByContactId(contactId string) error {
	result := db.GormDB.Where("contact_id = ?", contactId).Delete(&models.ContactApply{})
	return result.Error
}

// UpdateUserContactStatusAndDelete 根据用户ID和联系人ID更新联系人状态并删除
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
