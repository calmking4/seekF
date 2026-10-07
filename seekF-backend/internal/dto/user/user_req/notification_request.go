package userreq

// ListNotificationRequest 通知列表请求
type ListNotificationRequest struct {
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"page_size" form:"page_size"`
	Category   string `json:"category" form:"category" binding:"omitempty,oneof=comments likes follows"`
	UnreadOnly bool   `json:"unread_only" form:"unread_only"`
}

// MarkNotificationReadRequest 标记通知已读请求
type MarkNotificationReadRequest struct {
	Ids []int64 `json:"ids" form:"ids"`
}

// MarkNotificationCategoryReadRequest 标记指定分类通知已读。
type MarkNotificationCategoryReadRequest struct {
	Category string `json:"category" form:"category" binding:"required,oneof=comments likes follows"`
}
