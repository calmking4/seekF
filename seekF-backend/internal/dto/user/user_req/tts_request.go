package userreq

type TTSRequest struct {
	Content string `json:"content" binding:"required"`
	Voice   string `json:"voice"`
}
