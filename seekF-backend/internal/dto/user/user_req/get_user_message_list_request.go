package userreq

// GetUserMessageListRequest 获取用户聊天记录请求
type GetUserMessageListRequest struct {
	Refresh   bool     `json:"refresh"`
	AfterId   string   `json:"after_id" binding:"omitempty,max=20"`
	KnownIds  []string `json:"known_ids" binding:"max=100,dive,max=20"`
	UserOneId string   `json:"user_one_id" binding:"required"`
	UserTwoId string   `json:"user_two_id" binding:"required"`
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
	// 历史分页的最早消息编号，避免个人删除后页码偏移漏消息。
	BeforeMessageId string `json:"before_message_id" binding:"omitempty,max=20"`
}
