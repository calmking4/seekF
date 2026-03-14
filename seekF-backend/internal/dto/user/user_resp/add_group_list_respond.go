package userresp

// AddGroupListRespond 群聊申请列表响应
type AddGroupListRespond struct {
	ContactId     string `json:"contact_id"`
	ContactName   string `json:"contact_name"`
	ContactAvatar string `json:"contact_avatar"`
	Message       string `json:"message"`
}
