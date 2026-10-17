package userdao

import (
	"errors"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SupportDAO 助力开通配置及观看权益的数据访问接口。
type SupportDAO interface {
	// 查询创作者的助力开通记录。
	FindActivate(userId string) (*models.SupportActivate, error)
	// 保存开通记录或更新现有价格，保留首次开通时间。
	SaveActivate(activate *models.SupportActivate) error
	// 批量查询当前用户对指定创作者的权益，包含已过期记录。
	ListEntitlements(environment, userId string, creatorIds []string) ([]models.SupportEntitlement, error)
}

// SupportDAOImpl 绑定数据库或业务事务的助力数据访问实现。
type SupportDAOImpl struct{ db *gorm.DB }

// NewSupportDAO 创建绑定指定连接的助力DAO。
func NewSupportDAO(db *gorm.DB) SupportDAO { return &SupportDAOImpl{db: db} }

// FindActivate 查询创作者的助力开通记录。
func (d *SupportDAOImpl) FindActivate(userId string) (*models.SupportActivate, error) {
	var activate models.SupportActivate
	err := d.db.Where("user_id = ?", userId).First(&activate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &activate, err
}

// SaveActivate 使用用户唯一键原子保存价格，避免并发首次开通产生重复记录。
func (d *SupportDAOImpl) SaveActivate(activate *models.SupportActivate) error {
	return d.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"price_fen", "updated_at"})}).Create(activate).Error
}

// ListEntitlements 批量读取权益，观看资格由Service按当前时间判断。
func (d *SupportDAOImpl) ListEntitlements(environment, userId string, creatorIds []string) ([]models.SupportEntitlement, error) {
	if userId == "" || len(creatorIds) == 0 || (environment != "sandbox" && environment != "production") {
		return nil, nil
	}
	var rows []models.SupportEntitlement
	err := d.db.Where("environment = ? AND user_id = ? AND creator_id IN ?", environment, userId, creatorIds).Find(&rows).Error
	return rows, err
}
