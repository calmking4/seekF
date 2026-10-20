package userdao

import (
	"errors"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SupportSettlementDAO 只负责指定连接上的收益账户与流水读写。
type SupportSettlementDAO struct{ db *gorm.DB }

// NewSupportSettlementDAO 创建绑定业务事务或查询连接的收益DAO。
func NewSupportSettlementDAO(conn *gorm.DB) *SupportSettlementDAO {
	return &SupportSettlementDAO{db: conn}
}

// LockAccount 确保账户存在并加行锁，调用方必须先持有来源订单锁。
func (d *SupportSettlementDAO) LockAccount(environment, creatorId string) (*models.SupportAccount, error) {
	row := models.SupportAccount{Environment: environment, CreatorId: creatorId}
	if err := d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment = ? AND creator_id = ?", environment, creatorId).First(&row).Error
	return &row, err
}

// FindAccount 查询本人账户，不因只读请求创建余额记录。
func (d *SupportSettlementDAO) FindAccount(environment, creatorId string) (*models.SupportAccount, error) {
	row := &models.SupportAccount{Environment: environment, CreatorId: creatorId}
	err := d.db.Where("environment = ? AND creator_id = ?", environment, creatorId).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.SupportAccount{}, nil
	}
	return row, err
}

// SaveAccount 写入包括零值在内的全部余额字段，不修改账户归属。
func (d *SupportSettlementDAO) SaveAccount(row *models.SupportAccount) error {
	return d.db.Model(&models.SupportAccount{}).Where("id = ?", row.Id).Updates(map[string]any{
		"available_amount": row.AvailableAmount, "refund_frozen_amount": row.RefundFrozenAmount,
		"refund_reserve_amount": row.RefundReserveAmount, "withdraw_frozen_amount": row.WithdrawFrozenAmount,
		"debt_amount": row.DebtAmount, "income_amount": row.IncomeAmount,
		"refund_amount": row.RefundAmount, "withdrawn_amount": row.WithdrawnAmount,
	}).Error
}

// CreateLedger 插入唯一事件流水，冲突返回错误并由Service回滚整个事务。
func (d *SupportSettlementDAO) CreateLedger(row *models.SupportLedger) error {
	return d.db.Create(row).Error
}

// ListLedger 分页查询当前环境的本人流水，不返回其他创作者信息。
func (d *SupportSettlementDAO) ListLedger(environment, creatorId string, page int) ([]models.SupportLedger, int64, error) {
	rows := make([]models.SupportLedger, 0)
	q := d.db.Model(&models.SupportLedger{}).Where("environment = ? AND creator_id = ?", environment, creatorId)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, total, err
}
