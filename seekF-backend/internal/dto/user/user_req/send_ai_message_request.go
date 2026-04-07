package userreq

type SendAIMessageRequest struct {
	SessionId string `form:"session_id" binding:"required"`
	Content   string `form:"content" binding:"required"`
	ModelType string `form:"model_type" binding:"required"`
}
