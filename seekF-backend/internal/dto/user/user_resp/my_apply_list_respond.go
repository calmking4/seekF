package userresp

// MyApplyListRespond 用户自己的申请状态响应结构体
type MyApplyListRespond struct {
	ContactId     string `json:"contact_id"`
	ContactName   string `json:"contact_name"`
	ContactAvatar string `json:"contact_avatar"`
	ContactType   string `json:"contact_type"` // user 或 group
	Status        int    `json:"status"`      // 0: 待处理, 1: 已同意, 2: 已拒绝, 3: 已拉黑
	Message       string `json:"message"`
	ApplyTime     string `json:"apply_time"`
	IsReceived    bool   `json:"is_received"` // 是否是收到的申请
}
