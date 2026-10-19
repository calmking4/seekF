package adminresp

import userresp "seekF-backend/internal/dto/user/user_resp"

// SupportOrder 管理端订单查询结果，不包含渠道支付凭据。
type SupportOrder struct {
	*userresp.SupportOrderRespond
	UserId     string `json:"user_id"`
	ReviewerId string `json:"reviewer_id"`
	LastError  string `json:"last_error"`
}
