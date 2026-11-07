package userreq

type SendAIMessageRequest struct {
	SessionId    string `form:"session_id" binding:"required"`
	Content      string `form:"content"`
	ModelType    string `form:"model_type" binding:"required"`
	ImageURL     string `form:"image_url"`
	UseKnowledge bool   `form:"use_knowledge"`
	UseWebSearch bool   `form:"use_web_search"`
}

// AIStreamRequest 指定需要恢复的回答及已处理的事件游标。
type AIStreamRequest struct {
	SessionId string `form:"session_id" binding:"required"`
	StreamId  string `form:"stream_id" binding:"required,uuid"`
	After     string `form:"after"`
}
