package userresp

// UploadFileRespond 文件上传响应
type UploadFileRespond struct {
	URL       string `json:"url"`       // 文件访问URL
	ObjectKey string `json:"objectKey"` // OSS对象键
}
