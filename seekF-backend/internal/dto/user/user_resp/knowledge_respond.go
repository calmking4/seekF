package userresp

type KnowledgeDocItem struct {
	Uuid      string `json:"uuid"`
	FileName  string `json:"file_name"`
	FileUrl   string `json:"file_url"`
	FileType  string `json:"file_type"`
	ChunkCnt  int    `json:"chunk_count"`
	CreatedAt string `json:"created_at"`
}

type ListKnowledgeRespond struct {
	List []KnowledgeDocItem `json:"list"`
}

type AddKnowledgeRespond struct {
	Uuid     string `json:"uuid"`
	ChunkCnt int    `json:"chunk_count"`
}
