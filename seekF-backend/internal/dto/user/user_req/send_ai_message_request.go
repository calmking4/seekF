package userreq

type SendAIMessageRequest struct {
	SessionId string `form:"session_id" binding:"required"`
	Content   string `form:"content"`
	ModelType string `form:"model_type" binding:"required"`
	ImageURL  string `form:"image_url"`
}
