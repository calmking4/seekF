package models

import "time"

// SupportOrder 助力订单、付款权益快照及全额退款审核记录。
type SupportOrder struct {
	Id                int64      `gorm:"column:id;primaryKey;index:idx_support_order_user,priority:3;index:idx_support_order_creator,priority:3;index:idx_support_order_due,priority:4;index:idx_support_order_refund_due,priority:4;comment:自增ID"`
	Environment       string     `gorm:"column:environment;type:varchar(16);not null;uniqueIndex:uk_support_order_request,priority:1;uniqueIndex:uk_support_order_trade,priority:1;index:idx_support_order_user,priority:1;index:idx_support_order_creator,priority:1;index:idx_support_order_due,priority:1;index:idx_support_order_pair,priority:1;index:idx_support_order_refund_due,priority:1;comment:支付环境，sandbox沙箱，production正式"`
	OrderNo           string     `gorm:"column:order_no;type:varchar(64);not null;uniqueIndex:uk_support_order_no;comment:商户订单号，同时作为支付宝out_trade_no"`
	UserId            string     `gorm:"column:user_id;type:char(20);not null;uniqueIndex:uk_support_order_request,priority:2;index:idx_support_order_user,priority:2;index:idx_support_order_pair,priority:2;comment:购买者用户UUID"`
	CreatorId         string     `gorm:"column:creator_id;type:char(20);not null;index:idx_support_order_creator,priority:2;index:idx_support_order_pair,priority:3;comment:受助力创作者用户UUID"`
	RequestKey        string     `gorm:"column:request_key;type:varchar(64);not null;uniqueIndex:uk_support_order_request,priority:3;comment:购买者提交的请求幂等标识，区分大小写"`
	Channel           string     `gorm:"column:channel;type:varchar(16);not null;default:alipay;comment:支付渠道，alipay支付宝"`
	TradeNo           *string    `gorm:"column:trade_no;type:varchar(64);uniqueIndex:uk_support_order_trade,priority:2;comment:支付宝交易号，未创建交易时为空"`
	AmountFen         int64      `gorm:"column:amount_fen;not null;comment:订单价格快照，单位分，500至10000且为100的倍数"`
	DurationHours     int        `gorm:"column:duration_hours;type:int;not null;default:720;comment:购买权益时长快照，单位小时，固定720小时"`
	Status            int8       `gorm:"column:status;not null;default:0;index:idx_support_order_due,priority:2;index:idx_support_order_pair,priority:4;comment:支付状态，0待付款，1已付款，2已关闭"`
	RefundStatus      int8       `gorm:"column:refund_status;not null;default:0;index:idx_support_order_refund_due,priority:2;comment:退款状态，0未申请，1待审核，2已驳回，3处理中，4已退款，5退款失败"`
	RefundRequestNo   *string    `gorm:"column:refund_request_no;type:varchar(64);uniqueIndex:uk_support_order_refund_request;comment:固定退款请求号，作为支付宝out_request_no，未申请时为空"`
	RefundReason      string     `gorm:"column:refund_reason;type:varchar(500);not null;default:;comment:用户退款申请原因"`
	RefundRequestedAt *time.Time `gorm:"column:refund_requested_at;type:datetime(3);comment:退款申请提交时间，每单只允许申请一次"`
	RefundReviewerId  string     `gorm:"column:refund_reviewer_id;type:char(20);not null;default:;comment:审核管理员UUID，未审核时为空"`
	RefundReviewNote  string     `gorm:"column:refund_review_note;type:varchar(500);not null;default:;comment:退款审核说明或驳回原因"`
	RefundReviewedAt  *time.Time `gorm:"column:refund_reviewed_at;type:datetime(3);comment:管理员退款审核时间"`
	RefundedAt        *time.Time `gorm:"column:refunded_at;type:datetime(3);comment:支付宝确认全额退款时间"`
	ExpiresAt         time.Time  `gorm:"column:expires_at;type:datetime(3);not null;index:idx_support_order_pair,priority:5;comment:待支付有效期截止时间，创建后15分钟"`
	PaidAt            *time.Time `gorm:"column:paid_at;type:datetime(3);comment:支付宝确认的付款时间"`
	ConfirmedAt       *time.Time `gorm:"column:confirmed_at;type:datetime(3);comment:本站确认支付并发放权益的时间，退款重算起点"`
	ClosedAt          *time.Time `gorm:"column:closed_at;type:datetime(3);comment:确认交易关闭时间"`
	NextAttemptAt     time.Time  `gorm:"column:next_attempt_at;type:datetime(3);not null;index:idx_support_order_due,priority:3;index:idx_support_order_refund_due,priority:3;comment:下次支付或退款查询、补偿时间"`
	ProcessingUntil   *time.Time `gorm:"column:processing_until;type:datetime(3);comment:后台处理领取截止时间，防止多实例重复调度"`
	LastError         string     `gorm:"column:last_error;type:varchar(1000);not null;default:;comment:最近一次支付或退款处理错误"`
	CreatedAt         time.Time  `gorm:"column:created_at;type:datetime(3);not null;comment:订单创建时间"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;type:datetime(3);not null;comment:订单更新时间"`
}

// TableName 返回助力订单表名。
func (SupportOrder) TableName() string { return "support_order" }
