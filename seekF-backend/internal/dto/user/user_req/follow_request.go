package userreq

type ToggleFollowRequest struct {
	FollowUserId string `json:"follow_user_id" form:"follow_user_id"`
}

type ListFollowRequest struct {
	UserId   string `json:"user_id" form:"user_id"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}
