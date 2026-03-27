package userreq

// GetUserMessageListRequest 获取用户聊天记录请求
type GetUserMessageListRequest struct {
	UserOneId string `json:"user_one_id" binding:"required"`
	UserTwoId string `json:"user_two_id" binding:"required"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}
