package userresp

import "time"

// SupportReceivedOrderRespond 创作者本人可见的已付款助力记录，金额单位为分。
type SupportReceivedOrderRespond struct {
	OrderNo       string     `json:"order_no"`
	BuyerId       string     `json:"buyer_id"`
	BuyerNickname string     `json:"buyer_nickname"`
	BuyerAvatar   string     `json:"buyer_avatar"`
	AmountFen     int64      `json:"amount_fen"`
	RefundStatus  int8       `json:"refund_status"`
	PaidAt        *time.Time `json:"paid_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
}

// SupportPaymentRespond 同一订单的网页收银台与内嵌扫码页面地址。
type SupportPaymentRespond struct {
	PayURL string `json:"pay_url"`
	QRURL  string `json:"qr_url"`
}

// SupportOrderRespond 本人可见的订单快照，不暴露渠道凭据和后台领取信息。
type SupportOrderRespond struct {
	OrderNo       string     `json:"order_no"`
	RequestKey    string     `json:"request_key"`
	CreatorId     string     `json:"creator_id"`
	AmountFen     int64      `json:"amount_fen"`
	DurationHours int        `json:"duration_hours"`
	Status        int8       `json:"status"`
	RefundStatus  int8       `json:"refund_status"`
	ExpiresAt     time.Time  `json:"expires_at"`
	PaidAt        *time.Time `json:"paid_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	HasError      bool       `json:"has_error"`
}
