package userreq

// OpenSessionRequest 打开会话请求
type OpenSessionRequest struct {
	ReceiveId string `json:"receive_id" binding:"required"`
}
