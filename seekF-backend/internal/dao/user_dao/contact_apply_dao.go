package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	contactapplystatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_apply_status_enum"
)

type ContactApplyDAO interface {
	GetContactApplyByUserIdAndContactId(userId string, contactId string) (models.ContactApply, error)
	GetPendingContactAppliesByContactId(contactId string) ([]models.ContactApply, error)
	GetContactAppliesByUserId(userId string) ([]models.ContactApply, error)
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
	// 使用 First 以便在未查询到记录时返回 gorm.ErrRecordNotFound。
	// 否则使用 Find 在未命中时不会返回错误，调用方就会误以为记录存在并走更新分支。
	result := db.GormDB.Where("user_id = ? AND contact_id = ?", userId, contactId).First(&contactApply)
	return contactApply, result.Error
}

// GetPendingContactAppliesByContactId 根据联系人ID获取待处理的联系人申请列表
func (d *ContactApplyDAOImpl) GetPendingContactAppliesByContactId(contactId string) ([]models.ContactApply, error) {
	var contactApplyList []models.ContactApply
	result := db.GormDB.Where("contact_id = ? AND status = ?", contactId, contactapplystatusenum.PENDING).Find(&contactApplyList)
	return contactApplyList, result.Error
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

// GetContactAppliesByUserId 根据用户ID获取联系人申请列表
func (d *ContactApplyDAOImpl) GetContactAppliesByUserId(userId string) ([]models.ContactApply, error) {
	var contactApplyList []models.ContactApply
	result := db.GormDB.Where("user_id = ?", userId).Find(&contactApplyList)
	return contactApplyList, result.Error
}
