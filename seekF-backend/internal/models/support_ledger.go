package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// SupportLedger 不可覆盖的收益变更流水，金额单位为元。
type SupportLedger struct {
	Id             int64           `gorm:"column:id;primaryKey;comment:自增ID"`
	Environment    string          `gorm:"column:environment;type:varchar(16);not null;uniqueIndex:uk_support_ledger_event,priority:1;index:idx_support_ledger_creator,priority:1;comment:支付环境，sandbox沙箱，production正式"`
	CreatorId      string          `gorm:"column:creator_id;type:char(20);not null;index:idx_support_ledger_creator,priority:2;comment:创作者用户UUID"`
	OrderNo        string          `gorm:"column:order_no;type:varchar(64);not null;uniqueIndex:uk_support_ledger_event,priority:2;comment:来源助力订单号"`
	Kind           string          `gorm:"column:kind;type:varchar(32);not null;uniqueIndex:uk_support_ledger_event,priority:3;comment:事件类型，income收益入账，refund_reserve申请预留，refund_audit_reserve渠道退款预留，refund_release驳回释放，refund退款冲销"`
	Amount         decimal.Decimal `gorm:"column:amount;type:decimal(18,2);not null;comment:本事件创作者份额，单位元"`
	OffsetAmount   decimal.Decimal `gorm:"column:offset_amount;type:decimal(18,2);not null;comment:本次抵扣既有欠款金额，单位元"`
	AvailableDelta decimal.Decimal `gorm:"column:available_delta;type:decimal(18,2);not null;comment:可提现余额变化，可为负，单位元"`
	AvailableAfter decimal.Decimal `gorm:"column:available_after;type:decimal(18,2);not null;comment:事件后可提现余额，单位元"`
	CreatedAt      time.Time       `gorm:"column:created_at;type:datetime(3);index:idx_support_ledger_creator,priority:3;comment:流水创建时间"`
}

// TableName 返回创作者收益流水表名。
func (SupportLedger) TableName() string { return "support_ledger" }
