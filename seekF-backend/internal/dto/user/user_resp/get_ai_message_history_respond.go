package userresp

type GetAIMessageHistoryRespond struct {
	SessionId string `json:"session_id"`
	SendId    string `json:"send_id"`
	SendName  string `json:"send_name"`
	Content   string `json:"content"`
	Type      int8   `json:"type"`
	Url       string `json:"url"`
	Sources   string `json:"sources,omitempty"`
	Posts     string `json:"posts,omitempty"`
	CreatedAt string `json:"created_at"`
}
