package userresp

// SearchUsersRespond 搜索用户响应结构体
type SearchUsersRespond struct {
	UserId   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email,omitempty"`
}
