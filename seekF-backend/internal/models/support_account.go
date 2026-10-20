package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// SupportAccount 创作者按环境隔离的收益账户，所有金额单位为元。
type SupportAccount struct {
	PayeeCipher          string          `gorm:"column:payee_cipher;type:text;comment:支付宝收款账号及姓名AES-GCM密文，不存明文"`
	AccountVersion       int64           `gorm:"column:account_version;not null;default:0;comment:收款账号版本，换绑递增，不修改既有提现快照"`
	AcceptedAt           *time.Time      `gorm:"column:accepted_at;type:datetime(3);comment:本人确认收款及退款抵扣规则的时间"`
	Id                   int64           `gorm:"column:id;primaryKey;comment:自增ID"`
	Environment          string          `gorm:"column:environment;type:varchar(16);not null;uniqueIndex:uk_support_account,priority:1;comment:支付环境，sandbox沙箱，production正式"`
	CreatorId            string          `gorm:"column:creator_id;type:char(20);not null;uniqueIndex:uk_support_account,priority:2;comment:创作者用户UUID"`
	AvailableAmount      decimal.Decimal `gorm:"column:available_amount;type:decimal(18,2);not null;default:0.00;comment:可提现余额，单位元"`
	RefundFrozenAmount   decimal.Decimal `gorm:"column:refund_frozen_amount;type:decimal(18,2);not null;default:0.00;comment:已覆盖退款预留的冻结余额，单位元"`
	RefundReserveAmount  decimal.Decimal `gorm:"column:refund_reserve_amount;type:decimal(18,2);not null;default:0.00;comment:尚未确认的退款预留总额，含未覆盖部分，单位元"`
	WithdrawFrozenAmount decimal.Decimal `gorm:"column:withdraw_frozen_amount;type:decimal(18,2);not null;default:0.00;comment:在途提现冻结余额，单位元"`
	DebtAmount           decimal.Decimal `gorm:"column:debt_amount;type:decimal(18,2);not null;default:0.00;comment:已退款但尚待未来收益抵扣的金额，单位元"`
	IncomeAmount         decimal.Decimal `gorm:"column:income_amount;type:decimal(18,2);not null;default:0.00;comment:累计创作者份额入账金额，单位元"`
	RefundAmount         decimal.Decimal `gorm:"column:refund_amount;type:decimal(18,2);not null;default:0.00;comment:累计已退款创作者份额，单位元"`
	WithdrawnAmount      decimal.Decimal `gorm:"column:withdrawn_amount;type:decimal(18,2);not null;default:0.00;comment:累计确认提现成功金额，单位元"`
	CreatedAt            time.Time       `gorm:"column:created_at;type:datetime(3);comment:账户创建时间"`
	UpdatedAt            time.Time       `gorm:"column:updated_at;type:datetime(3);comment:账户最后变更时间"`
}

// TableName 返回创作者收益账户表名。
func (SupportAccount) TableName() string { return "support_account" }
