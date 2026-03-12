package userresp

type LoadMyJoinedGroupRespond struct {
	GroupId   string `json:"group_id"`
	GroupName string `json:"group_name"`
	Avatar    string `json:"avatar"`
}
