package userreq

type GetAIMessageHistoryRequest struct {
	SessionId string `json:"session_id" binding:"required"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}
