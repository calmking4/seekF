package userreq

// InitMultipartUploadRequest 初始化分片上传请求
type InitMultipartUploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	FileType    string `json:"file_type" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required"`
	ContentType string `json:"content_type"`
}

// ListUploadPartsRequest 查询已上传分片请求
type ListUploadPartsRequest struct {
	UploadId  string `json:"upload_id" binding:"required"`
	ObjectKey string `json:"object_key" binding:"required"`
}

// CompleteMultipartUploadRequest 完成分片上传请求
type CompleteMultipartUploadRequest struct {
	UploadId  string              `json:"upload_id" binding:"required"`
	ObjectKey string              `json:"object_key" binding:"required"`
	Parts     []UploadPartRequest `json:"parts" binding:"required"`
}

// UploadPartRequest 分片信息
type UploadPartRequest struct {
	PartNumber int32  `json:"part_number" binding:"required"`
	ETag       string `json:"etag" binding:"required"`
}

// AbortMultipartUploadRequest 取消分片上传请求
type AbortMultipartUploadRequest struct {
	UploadId  string `json:"upload_id" binding:"required"`
	ObjectKey string `json:"object_key" binding:"required"`
}
