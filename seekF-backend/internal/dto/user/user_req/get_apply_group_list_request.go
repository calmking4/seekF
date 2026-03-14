package userreq

// GetApplyGroupListRequest 获取群聊申请列表请求
type GetApplyGroupListRequest struct {
	GroupId string `json:"group_id" binding:"required"`
}
