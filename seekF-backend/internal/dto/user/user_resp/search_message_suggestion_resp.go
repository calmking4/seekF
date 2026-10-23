package userresp

// SearchMessageSuggestionResp 消息搜索联想响应
type SearchMessageSuggestionResp struct {
	Uuid      string `json:"uuid"`       // 匹配消息编号，用于定位原消息。
	SessionId string `json:"session_id"` // 会话ID
	Content   string `json:"content"`    // 原始文字内容，由前端安全高亮。
	SendName  string `json:"send_name"`  // 发送者昵称
	CreatedAt string `json:"created_at"` // 发送时间
}
