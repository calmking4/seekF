package userresp

type FollowUserItem struct {
	Uuid      string `json:"uuid"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
}

type ListFollowRespond struct {
	List  []FollowUserItem `json:"list"`
	Total int64            `json:"total"`
}
