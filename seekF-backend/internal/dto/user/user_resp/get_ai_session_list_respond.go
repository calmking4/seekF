package userresp

type AISessionItem struct {
	SessionId    string `json:"session_id"`
	FirstMessage string `json:"first_message"`
	CreatedAt    string `json:"created_at"`
}

type GetAISessionListRespond struct {
	List  []AISessionItem `json:"list"`
	Total int64           `json:"total"`
}
