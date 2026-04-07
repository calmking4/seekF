package userresp

type CreateAISessionRespond struct {
	SessionId string `json:"session_id"`
	ReceiveId string `json:"receive_id"`
	ModelType string `json:"model_type"`
}
