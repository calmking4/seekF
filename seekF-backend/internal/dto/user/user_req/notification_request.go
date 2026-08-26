package userreq

// ListNotificationRequest 通知列表请求
type ListNotificationRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// MarkNotificationReadRequest 标记通知已读请求
type MarkNotificationReadRequest struct {
	Ids []int64 `json:"ids" form:"ids"`
}
