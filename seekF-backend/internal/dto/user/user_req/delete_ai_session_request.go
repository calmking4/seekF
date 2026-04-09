package userreq

type DeleteAISessionRequest struct {
	SessionId string `json:"session_id" binding:"required"`
}
