package userreq

type GetUserMessageListRequest struct {
	UserOneId string `json:"user_one_id" binding:"required"`
	UserTwoId string `json:"user_two_id" binding:"required"`
}
