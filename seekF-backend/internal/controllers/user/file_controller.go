package user

import (
	"net/http"

	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// FileController 文件控制器
type FileController struct {
	fileService userservice.FileService
}

// NewFileController 创建文件控制器实例
func NewFileController(fileService userservice.FileService) *FileController {
	return &FileController{
		fileService: fileService,
	}
}

// UploadFile 上传文件
func (c *FileController) UploadFile(ctx *gin.Context) {
	// 获取上传的文件
	file, err := ctx.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "请选择要上传的文件", http.StatusBadRequest)
		return
	}

	// 获取文件类型
	fileType := oss.MessageImage
	if fileTypeStr := ctx.PostForm("fileType"); fileTypeStr != "" {
		switch fileTypeStr {
		case "user_avatar":
			fileType = oss.UserAvatar
		case "group_avatar":
			fileType = oss.GroupAvatar
		case "message_image":
			fileType = oss.MessageImage
		case "message_video":
			fileType = oss.MessageVideo
		case "message_audio":
			fileType = oss.MessageAudio
		default:
			fileType = oss.MessageImage
		}
	}

	// 调用服务层上传文件
	result, err := c.fileService.UploadFile(ctx, file, fileType)
	if err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "文件上传失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "文件上传成功", result)
}
