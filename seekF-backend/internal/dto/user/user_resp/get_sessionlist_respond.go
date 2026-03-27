package userresp

type GetSessionListRespond struct {
	SessionId     string `json:"session_id"`
	Avatar        string `json:"avatar"`
	Id            string `json:"id"`
	Name          string `json:"name"`
	LastMessage   string `json:"last_message"`
	LastMessageAt string `json:"last_message_at"`
}
