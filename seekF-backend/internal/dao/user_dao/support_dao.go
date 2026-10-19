package userdao

import (
	"errors"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SupportDAO 助力开通配置、订单及观看权益的数据访问接口。
type SupportDAO interface {
	// 查询创作者的助力开通记录。
	FindActivate(userId string) (*models.SupportActivate, error)
	// 保存开通记录或更新现有价格，保留首次开通时间。
	SaveActivate(activate *models.SupportActivate) error
	// 批量查询当前用户对指定创作者的权益，包含已过期记录。
	ListEntitlements(environment, userId string, creatorIds []string) ([]models.SupportEntitlement, error)
	// 查询订单，确认业务时可加行锁。
	FindOrder(environment, orderNo string, lock bool) (*models.SupportOrder, error)
	// 查询当前购买者已使用的请求键。
	FindRequest(environment, userId, key string) (*models.SupportOrder, error)
	// 查询同一购买者与创作者的有效待支付订单。
	FindPending(environment, userId, creatorId string, now time.Time) (*models.SupportOrder, error)
	// 保存订单价格和期限快照。
	CreateOrder(row *models.SupportOrder) error
	// 在当前连接更新订单。
	UpdateOrder(id int64, values map[string]any) error
	// 确保权益行存在并锁定，必须在锁定订单后调用。
	LockEntitlement(environment, userId, creatorId string) (*models.SupportEntitlement, error)
	// 更新持锁权益的截止时间。
	SaveExpiry(id int64, expires time.Time) error
	// 分页查询本人已确认付款的订单。
	ListOrders(environment, userId string, page int) ([]models.SupportOrder, int64, error)
	// 分页查询本人作为创作者收到的已付款助力记录。
	ListReceivedOrders(environment, creatorId string, page int) ([]SupportReceivedOrder, int64, error)
	// 分页查询本人权益。
	ListMyEntitlements(environment, userId string, page int) ([]models.SupportEntitlement, error)
	// 汇总创作者实收、退款及净额。
	Income(environment, creatorId string) (*SupportIncome, error)
	// 在业务事务中分批领取到期订单。
	ClaimOrders(environment string, now time.Time) ([]models.SupportOrder, error)
	// 仅释放当前租约的待支付任务。
	FinishAttempt(row models.SupportOrder, message string) error
}

// SupportDAOImpl 绑定数据库或业务事务的助力数据访问实现。
type SupportDAOImpl struct{ db *gorm.DB }

// SupportReceivedOrder 创作者收到的助力查询结果，金额单位为分，不包含渠道凭据。
type SupportReceivedOrder struct {
	OrderNo       string
	BuyerId       string
	BuyerNickname string
	BuyerAvatar   string
	AmountFen     int64
	RefundStatus  int8
	PaidAt        *time.Time
	ConfirmedAt   *time.Time
}

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

// FindOrder 查询订单，事务确认时可加行锁。
func (d *SupportDAOImpl) FindOrder(environment, orderNo string, lock bool) (*models.SupportOrder, error) {
	var row models.SupportOrder
	q := d.db.Where("environment = ? AND order_no = ?", environment, orderNo)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// FindRequest 查询购买者提交过的幂等键。
func (d *SupportDAOImpl) FindRequest(environment, userId, key string) (*models.SupportOrder, error) {
	var row models.SupportOrder
	err := d.db.Where("environment = ? AND user_id = ? AND request_key = ?", environment, userId, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// FindPending 查询同一购买者与创作者尚未过期的待付款订单。
func (d *SupportDAOImpl) FindPending(environment, userId, creatorId string, now time.Time) (*models.SupportOrder, error) {
	var row models.SupportOrder
	err := d.db.Where("environment = ? AND user_id = ? AND creator_id = ? AND status = 0 AND expires_at > ?", environment, userId, creatorId, now).Order("id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// CreateOrder 保存服务端价格与期限快照。
func (d *SupportDAOImpl) CreateOrder(row *models.SupportOrder) error {
	return d.db.Create(row).Error
}

// UpdateOrder 在当前连接中更新订单状态。
func (d *SupportDAOImpl) UpdateOrder(id int64, values map[string]any) error {
	return d.db.Model(&models.SupportOrder{}).Where("id = ?", id).Updates(values).Error
}

// LockEntitlement 以唯一键确保权益行存在并锁定，统一在锁定订单后调用。
func (d *SupportDAOImpl) LockEntitlement(environment, userId, creatorId string) (*models.SupportEntitlement, error) {
	row := models.SupportEntitlement{Environment: environment, UserId: userId, CreatorId: creatorId}
	if err := d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("environment = ? AND user_id = ? AND creator_id = ?", environment, userId, creatorId).First(&row).Error
	return &row, err
}

// SaveExpiry 写入持锁权益的截止时间。
func (d *SupportDAOImpl) SaveExpiry(id int64, expires time.Time) error {
	return d.db.Model(&models.SupportEntitlement{}).Where("id = ?", id).Update("expires_at", expires).Error
}

// ListOrders 分页查询本人已付款订单，未付款与已关闭订单保留供支付及补偿使用。
func (d *SupportDAOImpl) ListOrders(environment, userId string, page int) ([]models.SupportOrder, int64, error) {
	rows := make([]models.SupportOrder, 0)
	var count int64
	q := d.db.Model(&models.SupportOrder{}).Where("environment = ? AND user_id = ? AND status = ?", environment, userId, constants.SupportOrderPaid)
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, count, err
}

// ListReceivedOrders 按当前环境和收款创作者查询已付款订单，关联公开买家资料。
func (d *SupportDAOImpl) ListReceivedOrders(environment, creatorId string, page int) ([]SupportReceivedOrder, int64, error) {
	rows := make([]SupportReceivedOrder, 0)
	var total int64
	query := d.db.Model(&models.SupportOrder{}).Where("support_order.environment = ? AND support_order.creator_id = ? AND support_order.status = ?", environment, creatorId, constants.SupportOrderPaid)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Select("support_order.order_no, support_order.user_id AS buyer_id, COALESCE(user_info.nickname, '') AS buyer_nickname, COALESCE(user_info.avatar, '') AS buyer_avatar, support_order.amount_fen, support_order.refund_status, support_order.paid_at, support_order.confirmed_at").Joins("LEFT JOIN user_info ON user_info.uuid = support_order.user_id AND user_info.deleted_at IS NULL").Order("support_order.id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Scan(&rows).Error
	return rows, total, err
}

// ListMyEntitlements 分页查询本人的权益汇总。
func (d *SupportDAOImpl) ListMyEntitlements(environment, userId string, page int) ([]models.SupportEntitlement, error) {
	rows := make([]models.SupportEntitlement, 0)
	err := d.db.Where("environment = ? AND user_id = ?", environment, userId).Order("id DESC").Offset((page - 1) * constants.SupportOrderPageSize).Limit(constants.SupportOrderPageSize).Find(&rows).Error
	return rows, err
}

// SupportIncome 创作者按订单汇总的实收、退款和净额，单位分。
type SupportIncome struct {
	GrossFen   int64 `json:"gross_fen"`
	RefundFen  int64 `json:"refund_fen"`
	NetFen     int64 `json:"net_fen"`
	PaidOrders int64 `json:"paid_orders"`
}

// Income 仅汇总指定环境及本人作为创作者的已付款订单。
func (d *SupportDAOImpl) Income(environment, creatorId string) (*SupportIncome, error) {
	var row SupportIncome
	err := d.db.Model(&models.SupportOrder{}).Select("COALESCE(SUM(amount_fen),0) AS gross_fen, COALESCE(SUM(CASE WHEN refund_status = 4 THEN amount_fen ELSE 0 END),0) AS refund_fen, COUNT(*) AS paid_orders").Where("environment = ? AND creator_id = ? AND status = 1", environment, creatorId).Scan(&row).Error
	row.NetFen = row.GrossFen - row.RefundFen
	return &row, err
}

// ClaimOrders 锁定索引范围内到期任务，跳过其他实例持锁记录并持久化领取租约。
func (d *SupportDAOImpl) ClaimOrders(environment string, now time.Time) ([]models.SupportOrder, error) {
	rows := make([]models.SupportOrder, 0)
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("environment = ? AND status = 0 AND next_attempt_at <= ? AND (processing_until IS NULL OR processing_until < ?)", environment, now, now).Order("next_attempt_at, id").Limit(constants.SupportOrderScanBatch).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		// 租约字段为DATETIME(3)，统一到毫秒以确保释放时的相等条件能够匹配。
		until := now.Add(constants.SupportOrderLease).Truncate(time.Millisecond)
		if err := d.UpdateOrder(rows[i].Id, map[string]any{"processing_until": until, "next_attempt_at": until}); err != nil {
			return nil, err
		}
		rows[i].ProcessingUntil = &until
	}
	return rows, nil
}

// FinishAttempt 只释放仍属于本次租约的待付款任务，避免覆盖并发通知或后续领取结果。
func (d *SupportDAOImpl) FinishAttempt(row models.SupportOrder, message string) error {
	return d.db.Model(&models.SupportOrder{}).Where("id = ? AND status = 0 AND processing_until = ?", row.Id, row.ProcessingUntil).Updates(map[string]any{"processing_until": nil, "next_attempt_at": time.Now().Add(constants.SupportOrderScanInterval), "last_error": message}).Error
}
