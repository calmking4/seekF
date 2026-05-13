package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	userreq "seekF-backend/internal/dto/user/user_req"
	tool "seekF-backend/internal/pkg/ai/mcp/tool"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type AIChatController struct {
	aiChatService userservice.AIChatService
	fileService   userservice.FileService
}

func NewAIChatController(aiChatService userservice.AIChatService, fileService userservice.FileService) *AIChatController {
	return &AIChatController{
		aiChatService: aiChatService,
		fileService:   fileService,
	}
}

func (c *AIChatController) CreateSession(ctx *gin.Context) {
	var req userreq.CreateAISessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	userId := ctx.GetString("Uuid")
	result, err := c.aiChatService.CreateSession(userId, req)
	if err != nil {
		zlog.Info("CreateSession service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "创建AI会话成功", result)
}

func (c *AIChatController) GetSessionList(ctx *gin.Context) {
	var req userreq.GetAISessionListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	userId := ctx.GetString("Uuid")
	result, err := c.aiChatService.GetSessionList(userId, req.Page, req.PageSize)
	if err != nil {
		zlog.Info("GetSessionList service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取AI会话列表成功", result)
}

func (c *AIChatController) GetMessageHistory(ctx *gin.Context) {
	var req userreq.GetAIMessageHistoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	messageList, total, err := c.aiChatService.GetMessageHistory(req.SessionId, req.Page, req.PageSize)
	if err != nil {
		zlog.Info("GetMessageHistory service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取AI消息历史成功", gin.H{
		"list":  messageList,
		"total": total,
	})
}

func (c *AIChatController) SendMessage(ctx *gin.Context) {
	var req userreq.SendAIMessageRequest
	if err := ctx.ShouldBind(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	// 如果有图片文件，上传到 OSS
	if file, err := ctx.FormFile("image"); err == nil {
		result, err := c.fileService.UploadFile(ctx.Request.Context(), file, oss.MessageImage)
		if err != nil {
			zlog.Error("upload image failed: " + err.Error())
			resp.Error(ctx, "图片上传失败", http.StatusInternalServerError)
			return
		}
		req.ImageURL = result.URL
	}

	// 设置SSE响应头，启用服务器发送事件
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Status(http.StatusOK)
	ctx.Writer.Flush()

	userId := ctx.GetString("Uuid")

	onChunk := func(chunk string) error {
		// 对内容进行转义处理，防止换行符和引号破坏JSON格式
		escaped := strings.ReplaceAll(chunk, "\n", "\\n")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		// 将数据块以SSE格式写入响应流  \n\n（两个连续的换行符）是SSE协议中消息边界标记
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"content\": \"%s\"}\n\n", escaped)
		if err != nil {
			return err
		}
		// 立即刷新响应缓冲区，确保数据实时发送到前端
		ctx.Writer.Flush()
		return nil
	}

	onSources := func(sources []tool.SearchSource) error {
		sourcesJSON, _ := json.Marshal(sources)
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"sources\": %s}\n\n", sourcesJSON)
		if err != nil {
			return err
		}
		ctx.Writer.Flush()
		return nil
	}

	onComplete := func(fullContent string) error {
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"done\": true}\n\n")
		if err != nil {
			return err
		}
		ctx.Writer.Flush()
		return nil
	}

	err := c.aiChatService.SendMessageStream(ctx.Request.Context(), userId, req, onChunk, onSources, onComplete)
	if err != nil {
		zlog.Info("SendMessageStream service err: " + err.Error())
		fmt.Fprintf(ctx.Writer, "data: {\"error\": \"%s\"}\n\n", err.Error())
		ctx.Writer.Flush()
	}
}

func (c *AIChatController) DeleteSession(ctx *gin.Context) {
	var req userreq.DeleteAISessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if err := c.aiChatService.DeleteSession(req.SessionId); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除会话成功", nil)
}
