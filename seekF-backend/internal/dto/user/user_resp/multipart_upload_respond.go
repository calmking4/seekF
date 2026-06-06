package userresp

// MultipartInitRespond 分片上传初始化响应
type MultipartInitRespond struct {
	UploadId  string `json:"uploadId"`
	ObjectKey string `json:"objectKey"`
	URL       string `json:"url"`
	PartSize  int64  `json:"partSize"`
}

// UploadPartRespond 分片上传响应
type UploadPartRespond struct {
	PartNumber int32  `json:"partNumber"`
	ETag       string `json:"etag"`
	Size       int64  `json:"size,omitempty"`
}

// ListUploadPartsRespond 已上传分片列表响应
type ListUploadPartsRespond struct {
	Parts []UploadPartRespond `json:"parts"`
}
