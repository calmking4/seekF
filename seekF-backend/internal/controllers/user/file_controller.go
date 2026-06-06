package user

import (
	"net/http"
	"strconv"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
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

	fileType := oss.ParseFileCategory(ctx.PostForm("fileType"))

	// 调用服务层上传文件
	result, err := c.fileService.UploadFile(ctx, file, fileType)
	if err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "文件上传失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "文件上传成功", result)
}

// InitMultipartUpload 初始化视频分片上传
func (c *FileController) InitMultipartUpload(ctx *gin.Context) {
	var req userreq.InitMultipartUploadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error("初始化分片上传参数错误: " + err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	result, err := c.fileService.InitMultipartUpload(ctx.Request.Context(), req)
	if err != nil {
		zlog.Error("初始化分片上传失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "初始化成功", userresp.MultipartInitRespond{
		UploadId:  result.UploadId,
		ObjectKey: result.ObjectKey,
		URL:       result.URL,
		PartSize:  result.PartSize,
	})
}

// UploadPart 上传视频分片
func (c *FileController) UploadPart(ctx *gin.Context) {
	uploadId := ctx.PostForm("uploadId")
	objectKey := ctx.PostForm("objectKey")
	partNumberStr := ctx.PostForm("partNumber")

	if uploadId == "" || objectKey == "" || partNumberStr == "" {
		resp.Error(ctx, "缺少必要参数", http.StatusBadRequest)
		return
	}

	partNumber, err := strconv.ParseInt(partNumberStr, 10, 32)
	if err != nil || partNumber < 1 {
		resp.Error(ctx, "分片序号无效", http.StatusBadRequest)
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		zlog.Error("读取分片文件失败: " + err.Error())
		resp.Error(ctx, "请选择分片文件", http.StatusBadRequest)
		return
	}

	etag, err := c.fileService.UploadPart(ctx.Request.Context(), objectKey, uploadId, int32(partNumber), file)
	if err != nil {
		zlog.Error("上传分片失败: " + err.Error())
		resp.Error(ctx, "上传分片失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "分片上传成功", userresp.UploadPartRespond{
		PartNumber: int32(partNumber),
		ETag:       etag,
	})
}

// ListUploadParts 查询已上传分片（断点续传）
func (c *FileController) ListUploadParts(ctx *gin.Context) {
	var req userreq.ListUploadPartsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error("查询分片参数错误: " + err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	parts, err := c.fileService.ListUploadParts(ctx.Request.Context(), req.ObjectKey, req.UploadId)
	if err != nil {
		zlog.Error("查询已上传分片失败: " + err.Error())
		resp.Error(ctx, "查询分片失败", http.StatusInternalServerError)
		return
	}

	var respondParts []userresp.UploadPartRespond
	for _, p := range parts {
		respondParts = append(respondParts, userresp.UploadPartRespond{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
			Size:       p.Size,
		})
	}

	resp.Success(ctx, "查询成功", userresp.ListUploadPartsRespond{Parts: respondParts})
}

// CompleteMultipartUpload 完成视频分片上传
func (c *FileController) CompleteMultipartUpload(ctx *gin.Context) {
	var req userreq.CompleteMultipartUploadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error("完成分片上传参数错误: " + err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	result, err := c.fileService.CompleteMultipartUpload(ctx.Request.Context(), req)
	if err != nil {
		zlog.Error("完成分片上传失败: " + err.Error())
		resp.Error(ctx, "合并分片失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "上传成功", userresp.UploadFileRespond{
		URL:       result.URL,
		ObjectKey: result.ObjectKey,
	})
}

// AbortMultipartUpload 取消分片上传
func (c *FileController) AbortMultipartUpload(ctx *gin.Context) {
	var req userreq.AbortMultipartUploadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error("取消分片上传参数错误: " + err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if err := c.fileService.AbortMultipartUpload(ctx.Request.Context(), req.ObjectKey, req.UploadId); err != nil {
		zlog.Error("取消分片上传失败: " + err.Error())
		resp.Error(ctx, "取消上传失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "已取消上传", nil)
}

// DeleteFile 删除 OSS 文件
func (c *FileController) DeleteFile(ctx *gin.Context) {
	var req userreq.DeleteFileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error("删除文件参数错误: " + err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if req.ObjectKey == "" && req.URL == "" {
		resp.Error(ctx, "请提供 object_key 或 url", http.StatusBadRequest)
		return
	}

	if err := c.fileService.DeleteFile(ctx.Request.Context(), req.ObjectKey, req.URL); err != nil {
		zlog.Error("删除文件失败: " + err.Error())
		resp.Error(ctx, "删除文件失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "删除成功", nil)
}
