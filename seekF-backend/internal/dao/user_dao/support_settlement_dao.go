package userdao

import (
	"context"
	"errors"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SupportSettlementDAO 只负责指定连接上的收益账户与流水读写。
type SupportSettlementDAO interface {
	// 确保账户存在并加行锁，调用方必须先持有来源订单锁。
	LockAccount(ctx context.Context, environment, creatorId string) (*models.SupportAccount, error)
	// 查询本人账户，不存在时返回空，不因只读请求创建账户。
	FindAccount(ctx context.Context, environment, creatorId string) (*models.SupportAccount, error)
	// 写入包括零值在内的全部余额字段，不修改账户归属。
	SaveAccount(ctx context.Context, row *models.SupportAccount) error
	// 保存收款密文及版本，不影响现有提现单快照。
	SavePayee(ctx context.Context, row *models.SupportAccount) error
	// 按环境及业务单号查询提现，处理结果时加行锁。
	FindWithdraw(ctx context.Context, environment, no string, lock bool) (*models.SupportWithdraw, error)
	// 查询本人请求键，只在账户行锁内用于申请幂等。
	FindWithdrawRequest(ctx context.Context, environment, creator, key string) (*models.SupportWithdraw, error)
	// 保存固定请求及密文快照。
	CreateWithdraw(ctx context.Context, row *models.SupportWithdraw) error
	// 在当前业务连接保存提现状态。
	UpdateWithdraw(ctx context.Context, id int64, values map[string]any) error
	// 分页查询本人或管理端提现记录，参数校验由Service负责。
	ListWithdraw(ctx context.Context, environment, creator, no string, page int) ([]models.SupportWithdraw, int64, error)
	// 汇总指定账户的收益流水，金额单位元。
	GetLedgerTotals(ctx context.Context, environment, creatorId string) (*SupportLedgerTotals, error)
	// 汇总指定账户的已付款订单收益、退款与预留，金额单位元。
	GetOrderTotals(ctx context.Context, environment, creatorId string) (*SupportOrderTotals, error)
	// 汇总指定账户的在途和成功提现，金额单位元。
	GetWithdrawTotals(ctx context.Context, environment, creatorId string) (*SupportWithdrawTotals, error)
	// 查询待处理或超时的处理中提现，不扫描已完成单。
	ListPendingWithdraw(ctx context.Context, environment string) ([]models.SupportWithdraw, error)
	// 用状态和更新时间原子领取单笔任务，不另设租约或计数表。
	ClaimWithdraw(ctx context.Context, id int64, now time.Time) (bool, error)
	// 提交前再次确认本次处理权，防止过期任务提交。
	RenewWithdraw(ctx context.Context, id int64, previous, now time.Time) (bool, error)
	// 网络或未知结果只保存说明，不释放冻结或改变终态。
	FinishWithdraw(ctx context.Context, row models.SupportWithdraw, message string) error
	// 锁定现有账户供手动核对，不创建不存在的账户。
	LockExistingAccount(ctx context.Context, environment string, id int64) (*models.SupportAccount, error)
	// 分页读取账户供管理员按需核对，不保存独立对账状态。
	ListAuditAccounts(ctx context.Context, environment string, page int) ([]models.SupportAccount, int64, error)
	// 插入唯一事件流水，冲突返回错误并由Service回滚整个事务。
	CreateLedger(ctx context.Context, row *models.SupportLedger) error
	// 分页查询当前环境的本人流水，不返回其他创作者信息。
	ListLedger(ctx context.Context, environment, creatorId string, page int) ([]models.SupportLedger, int64, error)
}

// SupportSettlementDAOImpl 实现对应业务的数据访问。
type SupportSettlementDAOImpl struct {
	db *gorm.DB
}

// NewSupportSettlementDAO 创建绑定业务事务或查询连接的收益DAO。
func NewSupportSettlementDAO(database *gorm.DB) SupportSettlementDAO {
	return &SupportSettlementDAOImpl{db: database}
}

// LockAccount 确保账户存在并加行锁，调用方必须先持有来源订单锁。
func (d *SupportSettlementDAOImpl) LockAccount(ctx context.Context, environment, creatorId string) (*models.SupportAccount, error) {
	row := models.SupportAccount{Environment: environment, CreatorId: creatorId}
	if err := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment = ? AND creator_id = ?", environment, creatorId).First(&row).Error
	return &row, err
}

// FindAccount 查询本人账户，不存在时返回空，不因只读请求创建账户。
func (d *SupportSettlementDAOImpl) FindAccount(ctx context.Context, environment, creatorId string) (*models.SupportAccount, error) {
	row := &models.SupportAccount{Environment: environment, CreatorId: creatorId}
	err := d.db.WithContext(ctx).Where("environment = ? AND creator_id = ?", environment, creatorId).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return row, err
}

// SaveAccount 写入包括零值在内的全部余额字段，不修改账户归属。
func (d *SupportSettlementDAOImpl) SaveAccount(ctx context.Context, row *models.SupportAccount) error {
	return d.db.WithContext(ctx).Model(&models.SupportAccount{}).Where("id = ?", row.Id).Updates(map[string]any{
		"available_amount": row.AvailableAmount, "refund_frozen_amount": row.RefundFrozenAmount,
		"refund_reserve_amount": row.RefundReserveAmount, "withdraw_frozen_amount": row.WithdrawFrozenAmount,
		"debt_amount": row.DebtAmount, "income_amount": row.IncomeAmount,
		"refund_amount": row.RefundAmount, "withdrawn_amount": row.WithdrawnAmount,
	}).Error
}

// SavePayee 保存收款密文及版本，不影响现有提现单快照。
func (d *SupportSettlementDAOImpl) SavePayee(ctx context.Context, row *models.SupportAccount) error {
	return d.db.WithContext(ctx).Model(&models.SupportAccount{}).Where("id = ?", row.Id).Updates(map[string]any{"payee_cipher": row.PayeeCipher, "account_version": row.AccountVersion, "accepted_at": row.AcceptedAt}).Error
}

// FindWithdraw 按环境及业务单号查询提现，处理结果时加行锁。
func (d *SupportSettlementDAOImpl) FindWithdraw(ctx context.Context, environment, no string, lock bool) (*models.SupportWithdraw, error) {
	var row models.SupportWithdraw
	q := d.db.WithContext(ctx).Where("environment = ? AND withdraw_no = ?", environment, no)
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
func (d *SupportSettlementDAOImpl) FindWithdrawRequest(ctx context.Context, environment, creator, key string) (*models.SupportWithdraw, error) {
	var row models.SupportWithdraw
	err := d.db.WithContext(ctx).Where("environment = ? AND creator_id = ? AND request_key = ?", environment, creator, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// CreateWithdraw 保存固定请求及密文快照。
func (d *SupportSettlementDAOImpl) CreateWithdraw(ctx context.Context, row *models.SupportWithdraw) error {
	return d.db.WithContext(ctx).Create(row).Error
}

// UpdateWithdraw 在当前业务连接保存提现状态。
func (d *SupportSettlementDAOImpl) UpdateWithdraw(ctx context.Context, id int64, values map[string]any) error {
	return d.db.WithContext(ctx).Model(&models.SupportWithdraw{}).Where("id = ?", id).Updates(values).Error
}

// ListWithdraw 分页查询本人或管理端提现记录，参数校验由Service负责。
func (d *SupportSettlementDAOImpl) ListWithdraw(ctx context.Context, environment, creator, no string, page int) ([]models.SupportWithdraw, int64, error) {
	rows := make([]models.SupportWithdraw, 0)
	q := d.db.WithContext(ctx).Model(&models.SupportWithdraw{}).Where("environment = ?", environment)
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

// SupportLedgerTotals 保存账户流水汇总查询结果，所有金额单位为元，不对应数据库表。
type SupportLedgerTotals struct {
	Income    decimal.Decimal
	Refund    decimal.Decimal
	Withdrawn decimal.Decimal
	Available decimal.Decimal
}

// SupportOrderTotals 保存已付款订单汇总查询结果，所有金额单位为元，不对应数据库表。
type SupportOrderTotals struct {
	Income  decimal.Decimal
	Refund  decimal.Decimal
	Reserve decimal.Decimal
}

// SupportWithdrawTotals 保存提现汇总查询结果，所有金额单位为元，不对应数据库表。
type SupportWithdrawTotals struct {
	Frozen decimal.Decimal
	Paid   decimal.Decimal
}

// GetLedgerTotals 使用当前业务连接汇总账户流水，不判断账户金额是否一致。
func (d *SupportSettlementDAOImpl) GetLedgerTotals(ctx context.Context, environment, creatorId string) (*SupportLedgerTotals, error) {
	var totals SupportLedgerTotals
	err := d.db.WithContext(ctx).Model(&models.SupportLedger{}).Where("environment = ? AND creator_id = ?", environment, creatorId).Select("COALESCE(SUM(CASE WHEN kind = 'income' THEN amount ELSE 0 END),0) AS income, COALESCE(SUM(CASE WHEN kind = 'refund' THEN amount ELSE 0 END),0) AS refund, COALESCE(SUM(CASE WHEN kind = 'withdraw_success' THEN amount ELSE 0 END),0) AS withdrawn, COALESCE(SUM(available_delta),0) AS available").Scan(&totals).Error
	return &totals, err
}

// GetOrderTotals 使用当前业务连接汇总已付款订单的分成、退款和预留金额。
func (d *SupportSettlementDAOImpl) GetOrderTotals(ctx context.Context, environment, creatorId string) (*SupportOrderTotals, error) {
	var totals SupportOrderTotals
	err := d.db.WithContext(ctx).Model(&models.SupportOrder{}).Where("environment = ? AND creator_id = ? AND settlement_version = 1 AND status = 1", environment, creatorId).Select("COALESCE(SUM(creator_amount),0) AS income,COALESCE(SUM(CASE WHEN refund_status = 4 THEN creator_amount ELSE 0 END),0) AS refund,COALESCE(SUM(CASE WHEN refund_reserved = 1 THEN creator_amount ELSE 0 END),0) AS reserve").Scan(&totals).Error
	return &totals, err
}

// GetWithdrawTotals 使用当前业务连接汇总在途冻结及已成功提现金额。
func (d *SupportSettlementDAOImpl) GetWithdrawTotals(ctx context.Context, environment, creatorId string) (*SupportWithdrawTotals, error) {
	var totals SupportWithdrawTotals
	err := d.db.WithContext(ctx).Model(&models.SupportWithdraw{}).Where("environment = ? AND creator_id = ?", environment, creatorId).Select("COALESCE(SUM(CASE WHEN status IN (0,1) THEN amount ELSE 0 END),0) AS frozen, COALESCE(SUM(CASE WHEN status = 2 THEN amount ELSE 0 END),0) AS paid").Scan(&totals).Error
	return &totals, err
}

// ListPendingWithdraw 查询待处理或超时的处理中提现，不扫描已完成单。
func (d *SupportSettlementDAOImpl) ListPendingWithdraw(ctx context.Context, environment string) ([]models.SupportWithdraw, error) {
	var rows []models.SupportWithdraw
	err := d.db.WithContext(ctx).Where("environment = ? AND (status = 0 OR (status = 1 AND updated_at < ?))", environment, time.Now().Add(-2*time.Minute)).Order("id").Limit(constants.SupportOrderScanBatch).Find(&rows).Error
	return rows, err
}

// ClaimWithdraw 用状态和更新时间原子领取单笔任务，不另设租约或计数表。
func (d *SupportSettlementDAOImpl) ClaimWithdraw(ctx context.Context, id int64, now time.Time) (bool, error) {
	result := d.db.WithContext(ctx).Model(&models.SupportWithdraw{}).Where("id = ? AND (status = 0 OR (status = 1 AND updated_at < ?))", id, now.Add(-2*time.Minute)).Updates(map[string]any{"status": constants.SupportWithdrawProcessing, "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

// RenewWithdraw 提交前再次确认本次处理权，防止过期任务提交。
func (d *SupportSettlementDAOImpl) RenewWithdraw(ctx context.Context, id int64, previous, now time.Time) (bool, error) {
	result := d.db.WithContext(ctx).Model(&models.SupportWithdraw{}).Where("id = ? AND status = 1 AND updated_at = ?", id, previous).Updates(map[string]any{"updated_at": now})
	return result.RowsAffected == 1, result.Error
}

// FinishWithdraw 网络或未知结果只保存说明，不释放冻结或改变终态。
func (d *SupportSettlementDAOImpl) FinishWithdraw(ctx context.Context, row models.SupportWithdraw, message string) error {
	return d.db.WithContext(ctx).Model(&models.SupportWithdraw{}).Where("id = ? AND status = 1 AND updated_at = ?", row.Id, row.UpdatedAt).Updates(map[string]any{"last_error": message, "updated_at": time.Now()}).Error
}

// LockExistingAccount 锁定现有账户供手动核对，不创建不存在的账户。
func (d *SupportSettlementDAOImpl) LockExistingAccount(ctx context.Context, environment string, id int64) (*models.SupportAccount, error) {
	var row models.SupportAccount
	err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment = ? AND id = ?", environment, id).First(&row).Error
	return &row, err
}

// ListAuditAccounts 分页读取账户供管理员按需核对，不保存独立对账状态。
func (d *SupportSettlementDAOImpl) ListAuditAccounts(ctx context.Context, environment string, page int) ([]models.SupportAccount, int64, error) {
	rows := make([]models.SupportAccount, 0)
	q := d.db.WithContext(ctx).Model(&models.SupportAccount{}).Where("environment = ?", environment)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Omit("payee_cipher").Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, total, err
}

// CreateLedger 插入唯一事件流水，冲突返回错误并由Service回滚整个事务。
func (d *SupportSettlementDAOImpl) CreateLedger(ctx context.Context, row *models.SupportLedger) error {
	return d.db.WithContext(ctx).Create(row).Error
}

// ListLedger 分页查询当前环境的本人流水，不返回其他创作者信息。
func (d *SupportSettlementDAOImpl) ListLedger(ctx context.Context, environment, creatorId string, page int) ([]models.SupportLedger, int64, error) {
	rows := make([]models.SupportLedger, 0)
	q := d.db.WithContext(ctx).Model(&models.SupportLedger{}).Where("environment = ? AND creator_id = ?", environment, creatorId)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, total, err
}
