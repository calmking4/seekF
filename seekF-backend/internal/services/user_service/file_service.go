package userservice

import (
	"context"
	"fmt"
	"mime/multipart"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/upload/oss"
)

// FileService 文件服务接口
type FileService interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, fileType oss.FileCategory) (*oss.UploadResult, error)
	InitMultipartUpload(ctx context.Context, req userreq.InitMultipartUploadRequest) (*oss.MultipartInitResult, error)
	UploadPart(ctx context.Context, objectKey, uploadId string, partNumber int32, file *multipart.FileHeader) (string, error)
	ListUploadParts(ctx context.Context, objectKey, uploadId string) ([]oss.PartInfo, error)
	CompleteMultipartUpload(ctx context.Context, req userreq.CompleteMultipartUploadRequest) (*oss.UploadResult, error)
	AbortMultipartUpload(ctx context.Context, objectKey, uploadId string) error
	DeleteFile(ctx context.Context, objectKey, fileURL string) error
}

// FileServiceImpl 文件服务实现
type FileServiceImpl struct{}

// NewFileService 创建文件服务实例
func NewFileService() FileService {
	return &FileServiceImpl{}
}

// UploadFile 上传文件到OSS
func (s *FileServiceImpl) UploadFile(ctx context.Context, file *multipart.FileHeader, fileType oss.FileCategory) (*oss.UploadResult, error) {
	return oss.UploadMultipartFile(ctx, file, fileType)
}

// InitMultipartUpload 初始化分片上传
func (s *FileServiceImpl) InitMultipartUpload(ctx context.Context, req userreq.InitMultipartUploadRequest) (*oss.MultipartInitResult, error) {
	category := oss.ParseFileCategory(req.FileType)
	if !oss.IsVideoCategory(category) {
		return nil, fmt.Errorf("仅视频文件支持分片上传")
	}
	return oss.MultipartInit(ctx, req.FileName, category, req.ContentType)
}

// UploadPart 上传单个分片
func (s *FileServiceImpl) UploadPart(ctx context.Context, objectKey, uploadId string, partNumber int32, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开分片文件失败: %w", err)
	}
	defer src.Close()

	return oss.MultipartUploadPart(ctx, objectKey, uploadId, partNumber, src, file.Size)
}

// ListUploadParts 查询已上传分片
func (s *FileServiceImpl) ListUploadParts(ctx context.Context, objectKey, uploadId string) ([]oss.PartInfo, error) {
	return oss.MultipartListParts(ctx, objectKey, uploadId)
}

// CompleteMultipartUpload 完成分片上传
func (s *FileServiceImpl) CompleteMultipartUpload(ctx context.Context, req userreq.CompleteMultipartUploadRequest) (*oss.UploadResult, error) {
	parts := make([]oss.PartInfo, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = oss.PartInfo{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		}
	}
	return oss.MultipartComplete(ctx, req.ObjectKey, req.UploadId, parts)
}

// AbortMultipartUpload 取消分片上传
func (s *FileServiceImpl) AbortMultipartUpload(ctx context.Context, objectKey, uploadId string) error {
	return oss.MultipartAbort(ctx, objectKey, uploadId)
}

// DeleteFile 删除 OSS 文件
func (s *FileServiceImpl) DeleteFile(ctx context.Context, objectKey, fileURL string) error {
	if objectKey != "" {
		return oss.DeleteFile(ctx, objectKey)
	}
	if fileURL != "" {
		return oss.DeleteFileByURL(ctx, fileURL)
	}
	return fmt.Errorf("请提供 object_key 或 url")
}
