package userdao

import (
	"errors"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"github.com/shopspring/decimal"
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

// SavePayee 保存收款密文及版本，不影响现有提现单快照。
func (d *SupportSettlementDAO) SavePayee(row *models.SupportAccount) error {
	return d.db.Model(&models.SupportAccount{}).Where("id = ?", row.Id).Updates(map[string]any{"payee_cipher": row.PayeeCipher, "account_version": row.AccountVersion, "accepted_at": row.AcceptedAt}).Error
}

// FindWithdraw 按环境及业务单号查询提现，处理结果时加行锁。
func (d *SupportSettlementDAO) FindWithdraw(environment, no string, lock bool) (*models.SupportWithdraw, error) {
	var row models.SupportWithdraw
	q := d.db.Where("environment = ? AND withdraw_no = ?", environment, no)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// FindWithdrawRequest 查询本人请求键，只在账户行锁内用于申请幂等。
func (d *SupportSettlementDAO) FindWithdrawRequest(environment, creator, key string) (*models.SupportWithdraw, error) {
	var row models.SupportWithdraw
	err := d.db.Where("environment = ? AND creator_id = ? AND request_key = ?", environment, creator, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// CreateWithdraw 保存固定请求及密文快照。
func (d *SupportSettlementDAO) CreateWithdraw(row *models.SupportWithdraw) error {
	return d.db.Create(row).Error
}

// UpdateWithdraw 在当前业务连接保存提现状态。
func (d *SupportSettlementDAO) UpdateWithdraw(id int64, values map[string]any) error {
	return d.db.Model(&models.SupportWithdraw{}).Where("id = ?", id).Updates(values).Error
}

// ListWithdraw 分页查询本人或管理端提现记录，参数校验由Service负责。
func (d *SupportSettlementDAO) ListWithdraw(environment, creator, no string, page int) ([]models.SupportWithdraw, int64, error) {
	rows := make([]models.SupportWithdraw, 0)
	q := d.db.Model(&models.SupportWithdraw{}).Where("environment = ?", environment)
	if creator != "" {
		q = q.Where("creator_id = ?", creator)
	}
	if no != "" {
		q = q.Where("withdraw_no = ?", no)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, total, err
}

// AuditAccount 在账户锁下核对订单、流水和提现资金，全部查询使用当前业务连接。
func (d *SupportSettlementDAO) AuditAccount(account *models.SupportAccount) (string, error) {
	var flow struct {
		Income    decimal.Decimal
		Refund    decimal.Decimal
		Withdrawn decimal.Decimal
		Available decimal.Decimal
	}
	err := d.db.Model(&models.SupportLedger{}).Where("environment = ? AND creator_id = ?", account.Environment, account.CreatorId).Select("COALESCE(SUM(CASE WHEN kind = 'income' THEN amount ELSE 0 END),0) AS income, COALESCE(SUM(CASE WHEN kind = 'refund' THEN amount ELSE 0 END),0) AS refund, COALESCE(SUM(CASE WHEN kind = 'withdraw_success' THEN amount ELSE 0 END),0) AS withdrawn, COALESCE(SUM(available_delta),0) AS available").Scan(&flow).Error
	if err != nil {
		return "", err
	}
	if !flow.Income.Equal(account.IncomeAmount) || !flow.Refund.Equal(account.RefundAmount) || !flow.Withdrawn.Equal(account.WithdrawnAmount) || !flow.Available.Equal(account.AvailableAmount) {
		return "账户金额与累计流水不一致", nil
	}
	var orders struct {
		Income  decimal.Decimal
		Refund  decimal.Decimal
		Reserve decimal.Decimal
	}
	if err := d.db.Model(&models.SupportOrder{}).Where("environment = ? AND creator_id = ? AND settlement_version = 1 AND status = 1", account.Environment, account.CreatorId).Select("COALESCE(SUM(creator_amount),0) AS income,COALESCE(SUM(CASE WHEN refund_status = 4 THEN creator_amount ELSE 0 END),0) AS refund,COALESCE(SUM(CASE WHEN refund_reserved = 1 THEN creator_amount ELSE 0 END),0) AS reserve").Scan(&orders).Error; err != nil {
		return "", err
	}
	if !orders.Income.Equal(account.IncomeAmount) || !orders.Refund.Equal(account.RefundAmount) || !orders.Reserve.Equal(account.RefundReserveAmount) {
		return "订单收益、退款或预留与账户不一致", nil
	}
	var withdrawals struct {
		Frozen decimal.Decimal
		Paid   decimal.Decimal
	}
	if err := d.db.Model(&models.SupportWithdraw{}).Where("environment = ? AND creator_id = ?", account.Environment, account.CreatorId).Select("COALESCE(SUM(CASE WHEN status IN (0,1) THEN amount ELSE 0 END),0) AS frozen, COALESCE(SUM(CASE WHEN status = 2 THEN amount ELSE 0 END),0) AS paid").Scan(&withdrawals).Error; err != nil {
		return "", err
	}
	if !withdrawals.Frozen.Equal(account.WithdrawFrozenAmount) || !withdrawals.Paid.Equal(account.WithdrawnAmount) {
		return "提现单状态与账户冻结或已提现金额不一致", nil
	}
	return "", nil
}

// ListPendingWithdraw 查询待处理或超时的处理中提现，不扫描已完成单。
func (d *SupportSettlementDAO) ListPendingWithdraw(environment string) ([]models.SupportWithdraw, error) {
	var rows []models.SupportWithdraw
	err := d.db.Where("environment = ? AND (status = 0 OR (status = 1 AND updated_at < ?))", environment, time.Now().Add(-2*time.Minute)).Order("id").Limit(constants.SupportOrderScanBatch).Find(&rows).Error
	return rows, err
}

// ClaimWithdraw 用状态和更新时间原子领取单笔任务，不另设租约或计数表。
func (d *SupportSettlementDAO) ClaimWithdraw(id int64, now time.Time) (bool, error) {
	result := d.db.Model(&models.SupportWithdraw{}).Where("id = ? AND (status = 0 OR (status = 1 AND updated_at < ?))", id, now.Add(-2*time.Minute)).Updates(map[string]any{"status": constants.SupportWithdrawProcessing, "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

// RenewWithdraw 提交前再次确认本次处理权，防止过期任务提交。
func (d *SupportSettlementDAO) RenewWithdraw(id int64, previous, now time.Time) (bool, error) {
	result := d.db.Model(&models.SupportWithdraw{}).Where("id = ? AND status = 1 AND updated_at = ?", id, previous).Updates(map[string]any{"updated_at": now})
	return result.RowsAffected == 1, result.Error
}

// FinishWithdraw 网络或未知结果只保存说明，不释放冻结或改变终态。
func (d *SupportSettlementDAO) FinishWithdraw(row models.SupportWithdraw, message string) error {
	return d.db.Model(&models.SupportWithdraw{}).Where("id = ? AND status = 1 AND updated_at = ?", row.Id, row.UpdatedAt).Updates(map[string]any{"last_error": message, "updated_at": time.Now()}).Error
}

// LockExistingAccount 锁定现有账户供手动核对，不创建不存在的账户。
func (d *SupportSettlementDAO) LockExistingAccount(environment string, id int64) (*models.SupportAccount, error) {
	var row models.SupportAccount
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment = ? AND id = ?", environment, id).First(&row).Error
	return &row, err
}

// ListAuditAccounts 分页读取账户供管理员按需核对，不保存独立对账状态。
func (d *SupportSettlementDAO) ListAuditAccounts(environment string, page int) ([]models.SupportAccount, int64, error) {
	rows := make([]models.SupportAccount, 0)
	q := d.db.Model(&models.SupportAccount{}).Where("environment = ?", environment)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Omit("payee_cipher").Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, total, err
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
