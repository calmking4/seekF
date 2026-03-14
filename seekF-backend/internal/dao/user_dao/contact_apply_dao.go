package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

type ContactApplyDAO interface {
	GetContactApplyByUserIdAndContactId(userId string, contactId string) (models.ContactApply, error)
	CreateContactApply(apply *models.ContactApply) error
	UpdateContactApply(apply *models.ContactApply) error
	RemoveContactApply(userId string, contactId string) error
	RemoveContactAppliesByContactId(contactId string) error
}

type ContactApplyDAOImpl struct{}

func NewContactApplyDAO() ContactApplyDAO {
	return &ContactApplyDAOImpl{}
}

// GetContactApplyByUserIdAndContactId 根据用户ID和联系人ID获取联系人申请记录
func (d *ContactApplyDAOImpl) GetContactApplyByUserIdAndContactId(userId string, contactId string) (models.ContactApply, error) {
	var contactApply models.ContactApply
	//可以使用 Take 方法代替 First 方法，因为 Take 在找不到记录时不会写入日志。
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).First(&contactApply)
	return contactApply, result.Error
}

// CreateContactApply 创建联系人申请记录
func (d *ContactApplyDAOImpl) CreateContactApply(apply *models.ContactApply) error {
	result := db.GormDB.Create(apply)
	return result.Error
}

// UpdateContactApply 更新联系人申请记录
func (d *ContactApplyDAOImpl) UpdateContactApply(apply *models.ContactApply) error {
	result := db.GormDB.Save(apply)
	return result.Error
}

// RemoveContactApply 根据用户ID和联系人ID删除联系人申请记录
func (d *ContactApplyDAOImpl) RemoveContactApply(userId string, contactId string) error {
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).Delete(&models.ContactApply{})
	return result.Error
}

// RemoveContactAppliesByContactId 批量删除指定联系ID的申请记录
func (d *ContactApplyDAOImpl) RemoveContactAppliesByContactId(contactId string) error {
	result := db.GormDB.Where("contact_id = ?", contactId).Delete(&models.ContactApply{})
	return result.Error
}
