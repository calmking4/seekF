package userresp

type GetAIMessageHistoryRespond struct {
	Uuid      string `json:"uuid"`
	SessionId string `json:"session_id"`
	SendId    string `json:"send_id"`
	SendName  string `json:"send_name"`
	Content   string `json:"content"`
	Type      int8   `json:"type"`
	Url       string `json:"url"`
	Sources   string `json:"sources,omitempty"`
	Posts     string `json:"posts,omitempty"`
	CreatedAt string `json:"created_at"`
}

// AIStreamInfo 描述一次可回放回答及其对应的正式消息。
type AIStreamInfo struct {
	StreamId      string `json:"stream_id"`
	SessionId     string `json:"session_id"`
	MessageId     string `json:"message_id"`
	UserMessageId string `json:"user_message_id"`
	ModelType     string `json:"model_type"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

// AIStreamEvent 保存有序事件编号、类型与JSON载荷，供SSE转发。
type AIStreamEvent struct {
	Id   string
	Type string
	Data string
}
