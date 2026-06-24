package userreq

// LoginByCodeRequest 验证码登录请求
// Telephone 和 Email 二选一，优先使用 Email
type LoginByCodeRequest struct {
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	Code      string `json:"code" binding:"required"`
}
