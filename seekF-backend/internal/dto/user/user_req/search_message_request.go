package userreq

// SearchMessageRequest 搜索聊天消息请求
type SearchMessageRequest struct {
	SessionId string `json:"session_id" binding:"required"`
	Keyword   string `json:"keyword" binding:"required"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

// SearchMessageSuggestionsRequest 搜索消息联想请求（跨会话）
type SearchMessageSuggestionsRequest struct {
	Keyword  string `json:"keyword" binding:"required"`
	PageSize int    `json:"page_size"` // 默认5
}
