package userresp

type FollowUserItem struct {
	Uuid      string `json:"uuid"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
	IsFriend  bool   `json:"is_friend"`
}

type ListFollowRespond struct {
	List  []FollowUserItem `json:"list"`
	Total int64            `json:"total"`
}
