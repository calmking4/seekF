package userreq

// SendVerifyCodeRequest 发送验证码请求
type SendVerifyCodeRequest struct {
	Telephone string `json:"telephone" binding:"required"`
}
