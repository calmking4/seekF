package models

import (
	"github.com/shopspring/decimal"
	"time"
)

// SupportWithdraw 记录一次自动提现，金额单位元，收款信息只存加密快照。
type SupportWithdraw struct {
	Id             int64           `gorm:"column:id;primaryKey;comment:自增ID"`
	Environment    string          `gorm:"column:environment;type:varchar(16);not null;uniqueIndex:uk_support_withdraw_no,priority:1;uniqueIndex:uk_support_withdraw_request,priority:1;uniqueIndex:uk_support_withdraw_channel,priority:1;index:idx_support_withdraw_creator,priority:1;index:idx_support_withdraw_status,priority:1;comment:支付环境，sandbox沙箱，production正式"`
	WithdrawNo     string          `gorm:"column:withdraw_no;type:varchar(64);not null;uniqueIndex:uk_support_withdraw_no,priority:2;comment:固定提现单号，同时作为渠道out_biz_no"`
	CreatorId      string          `gorm:"column:creator_id;type:char(20);not null;uniqueIndex:uk_support_withdraw_request,priority:2;index:idx_support_withdraw_creator,priority:2;comment:提现创作者UUID"`
	RequestKey     string          `gorm:"column:request_key;type:varchar(64);not null;uniqueIndex:uk_support_withdraw_request,priority:3;comment:本人请求幂等键，防止重复申请"`
	Amount         decimal.Decimal `gorm:"column:amount;type:decimal(18,2);not null;comment:冻结及转账金额，单位元"`
	PayeeCipher    string          `gorm:"column:payee_cipher;type:text;not null;comment:收款账号姓名及商户场景的不可变AES-GCM密文快照"`
	Status         int8            `gorm:"column:status;not null;default:0;index:idx_support_withdraw_status,priority:2;comment:提现状态，0待处理，1处理中含结果未知，2成功，3确认失败"`
	ChannelOrderID *string         `gorm:"column:channel_order_id;type:varchar(64);uniqueIndex:uk_support_withdraw_channel,priority:2;comment:支付宝转账单号，未取得时为空"`
	LastError      string          `gorm:"column:last_error;type:varchar(300);not null;default:;comment:不含收款信息的中文处理说明"`
	CreatedAt      time.Time       `gorm:"column:created_at;type:datetime(3);not null;index:idx_support_withdraw_creator,priority:3;comment:申请时间"`
	UpdatedAt      time.Time       `gorm:"column:updated_at;type:datetime(3);not null;comment:最后处理时间，处理中超过2分钟可重新领取查询"`
}

// TableName 返回提现记录表名。
func (SupportWithdraw) TableName() string { return "support_withdraw" }
