package userreq

// SessionReadRequest 确认已加载的入站私信主键边界，不接受客户端时间。
type SessionReadRequest struct {
	SessionId string `json:"session_id" binding:"required"`
	MessageId int64  `json:"message_id,string" binding:"gt=0"`
}
