package userreq

// DeleteFileRequest 删除文件请求
type DeleteFileRequest struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
}
