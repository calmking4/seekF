package userreq

// BlackApplyRequest 拉黑申请请求
type BlackApplyRequest struct {
	ContactId string `json:"contact_id" binding:"required"`
}
