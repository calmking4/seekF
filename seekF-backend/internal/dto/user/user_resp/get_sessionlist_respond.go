package userresp

type GetSessionListRespond struct {
	SessionId string `json:"session_id"`
	Avatar    string `json:"avatar"`
	Id        string `json:"id"`
	Name      string `json:"name"`
}
