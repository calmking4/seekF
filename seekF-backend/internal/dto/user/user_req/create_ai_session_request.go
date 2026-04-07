package userreq

type CreateAISessionRequest struct {
	ModelType string `json:"model_type" binding:"required"`
}
