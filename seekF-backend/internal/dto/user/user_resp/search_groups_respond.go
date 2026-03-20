package userresp

// SearchGroupsRespond 搜索群组响应结构体
type SearchGroupsRespond struct {
	GroupId   string `json:"group_id"`
	GroupName string `json:"group_name"`
	Avatar    string `json:"avatar"`
	IsInGroup bool   `json:"is_in_group"`
}
