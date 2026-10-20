package userreq

// SupportPayeeRequest 本人收款声明及登录密码复核，敏感字段仅用于当前请求。
type SupportPayeeRequest struct {
	Account      string `json:"account"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	TermsVersion string `json:"terms_version"`
	Accepted     bool   `json:"accepted"`
}

// SupportWithdrawRequest 本人金额、收款版本与幂等键，不接受其他用户标识。
type SupportWithdrawRequest struct {
	Amount         string `json:"amount"`
	RequestKey     string `json:"request_key"`
	AccountVersion int64  `json:"account_version"`
	WithdrawNo     string `json:"withdraw_no"`
	Page           int    `json:"page"`
}
