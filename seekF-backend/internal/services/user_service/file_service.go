package userservice

import (
	"context"
	"mime/multipart"

	"seekF-backend/internal/pkg/upload/oss"
)

// FileService 文件服务接口
type FileService interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, fileType oss.FileCategory) (*oss.UploadResult, error)
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
