package userreq

type GetGroupMessageListRequest struct {
	GroupId  string `json:"group_id"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
