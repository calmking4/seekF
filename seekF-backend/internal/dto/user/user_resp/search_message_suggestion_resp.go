package userresp

// SearchMessageSuggestionResp 消息搜索联想响应
type SearchMessageSuggestionResp struct {
	SessionId string `json:"session_id"` // 会话ID
	Content   string `json:"content"`    // 消息内容（含高亮）
	SendName  string `json:"send_name"`  // 发送者昵称
	CreatedAt string `json:"created_at"` // 发送时间
}
