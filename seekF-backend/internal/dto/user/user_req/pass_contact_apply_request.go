package userreq

// PassContactApplyRequest 通过联系人申请请求
type PassContactApplyRequest struct {
	ContactId string `json:"contact_id" binding:"required"`
	GroupId   string `json:"group_id"`
}
