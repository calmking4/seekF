package userreq

type GetAISessionListRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
