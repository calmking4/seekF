package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	contactstatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_status_enum"
	"time"

	"gorm.io/gorm"
)

type ContactDAO interface {
	CreateUserContact(contact *models.UserContact) error
	RemoveContact(userId string, contactId string) error
	RemoveContactsByContactId(contactId string) error
	UpdateUserContactStatusAndDelete(userId string, contactId string, status int8) error
	UpdateUserContactStatus(userId string, contactId string, status int8) error
	GetUserJoinedGroupContactsByUserId(userId string) ([]models.UserContact, error)
	GetUserContactList(ownerId string) ([]models.UserContact, error)
	GetUserContactByUserIdAndContactId(userId string, contactId string) (*models.UserContact, error)
}

type ContactDAOImpl struct{}

func NewContactDAO() ContactDAO {
	return &ContactDAOImpl{}
}

// CreateUserContact 创建用户联系人关系
func (d *ContactDAOImpl) CreateUserContact(contact *models.UserContact) error {
	result := db.GormDB.Create(contact)
	return result.Error
}

// RemoveContact 根据用户ID和联系人ID更新联系人
func (d *ContactDAOImpl) RemoveContact(userId string, contactId string) error {
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).Delete(&models.UserContact{})
	return result.Error
}

// RemoveContactsByContactId 批量删除指定联系ID的联系人
func (d *ContactDAOImpl) RemoveContactsByContactId(contactId string) error {
	result := db.GormDB.Where("contact_id = ?", contactId).Delete(&models.UserContact{})
	return result.Error
}

// UpdateUserContactStatusAndDelete 根据用户ID和联系人ID更新联系人状态并删除
func (d *ContactDAOImpl) UpdateUserContactStatusAndDelete(userId string, contactId string, status int8) error {
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	result := db.GormDB.Model(&models.UserContact{}).Where("user_id = ? AND contact_id = ?", userId, contactId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     status,
	})
	return result.Error
}

// GetUserJoinedGroupContactsByUserId 根据用户ID获取用户加入的群聊联系列表
func (d *ContactDAOImpl) GetUserJoinedGroupContactsByUserId(userId string) ([]models.UserContact, error) {
	var contactList []models.UserContact
	result := db.GormDB.Order("created_at DESC").Where("user_id = ? AND status != ? AND status != ?", userId, contactstatusenum.QUIT_GROUP, contactstatusenum.KICK_OUT_GROUP).Find(&contactList)
	return contactList, result.Error
}

// GetUserContactList 根据用户 ID 获取用户联系人列表
func (d *ContactDAOImpl) GetUserContactList(ownerId string) ([]models.UserContact, error) {
	var contactList []models.UserContact
	result := db.GormDB.Order("created_at DESC").Where("user_id = ? AND status != ?", ownerId, contactstatusenum.DELETE).Find(&contactList)
	return contactList, result.Error
}

// UpdateUserContactStatus 根据用户ID和联系人ID更新联系人状态
func (d *ContactDAOImpl) UpdateUserContactStatus(userId string, contactId string, status int8) error {
	result := db.GormDB.Model(&models.UserContact{}).Where("user_id = ? AND contact_id = ?", userId, contactId).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})
	return result.Error
}

// GetUserContactByUserIdAndContactId 根据用户ID和联系人ID获取联系人记录
func (d *ContactDAOImpl) GetUserContactByUserIdAndContactId(userId string, contactId string) (*models.UserContact, error) {
	var contact models.UserContact
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).First(&contact)
	return &contact, result.Error
}
