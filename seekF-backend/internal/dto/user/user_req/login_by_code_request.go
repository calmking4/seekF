package userreq

// LoginByCodeRequest 验证码登录请求
type LoginByCodeRequest struct {
	Telephone string `json:"telephone" binding:"required"`
	Code      string `json:"code" binding:"required"`
}
