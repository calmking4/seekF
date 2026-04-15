package userreq

type AddKnowledgeRequest struct {
	FileName string `json:"file_name" form:"file_name"`
	FileUrl  string `json:"file_url" form:"file_url"`
	FileType string `json:"file_type" form:"file_type"`
}

type ListKnowledgeRequest struct {
}

type RemoveKnowledgeRequest struct {
	Uuid string `json:"uuid" form:"uuid"`
}

type GetKnowledgeContentRequest struct {
	Uuid string `json:"uuid" form:"uuid"`
}
