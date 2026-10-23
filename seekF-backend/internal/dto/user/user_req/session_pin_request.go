package userreq

// SessionPinRequest 明确设置本人私聊置顶状态，允许false表示取消置顶。
type SessionPinRequest struct {
	SessionId string `json:"session_id" binding:"required"`
	Pinned    *bool  `json:"pinned" binding:"required"`
}
