package userreq

// SendVerifyCodeRequest 发送验证码请求
// Telephone 和 Email 二选一，优先使用 Email
type SendVerifyCodeRequest struct {
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
}
