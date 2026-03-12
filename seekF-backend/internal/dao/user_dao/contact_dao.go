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
