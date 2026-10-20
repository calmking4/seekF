package userresp

// GetMessageListRespond 表示历史记录与实时投递的聊天消息。
type GetMessageListRespond struct {
	Uuid       string `json:"uuid"` // 消息唯一编号，用于历史记录和实时回显去重。
	SessionId  string `json:"session_id"`
	SendId     string `json:"send_id"`
	SendName   string `json:"send_name"`
	SendAvatar string `json:"send_avatar"`
	ReceiveId  string `json:"receive_id"`
	Type       int8   `json:"type"`
	Content    string `json:"content"`
	Url        string `json:"url"`
	FileType   string `json:"file_type"`
	FileName   string `json:"file_name"`
	FileSize   string `json:"file_size"`
	CreatedAt  string `json:"created_at"` // 先用CreatedAt排序，后面考虑改成SentAt
}
