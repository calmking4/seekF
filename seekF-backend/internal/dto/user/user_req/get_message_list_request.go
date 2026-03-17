package userreq

// GetMessageListRequest 获取聊天记录请求
type GetMessageListRequest struct {
	UserOneId string `json:"user_one_id" binding:"required"`
	UserTwoId string `json:"user_two_id" binding:"required"`
}
