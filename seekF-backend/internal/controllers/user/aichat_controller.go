package user

import (
	"fmt"
	"net/http"
	"strings"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	aiservice "seekF-backend/internal/services/ai_service"

	"github.com/gin-gonic/gin"
)

type AIChatController struct {
	aiChatService aiservice.AIChatService
}

func NewAIChatController(aiChatService aiservice.AIChatService) *AIChatController {
	return &AIChatController{
		aiChatService: aiChatService,
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
	if err := ctx.ShouldBindQuery(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		resp.Error(ctx, "消息内容不能为空", http.StatusBadRequest)
		return
	}

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Status(http.StatusOK)
	ctx.Writer.Flush()

	userId := ctx.GetString("Uuid")

	onChunk := func(chunk string) error {
		escaped := strings.ReplaceAll(chunk, "\n", "\\n")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"content\": \"%s\"}\n\n", escaped)
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

	err := c.aiChatService.SendMessageStream(ctx.Request.Context(), userId, req, onChunk, onComplete)
	if err != nil {
		zlog.Info("SendMessageStream service err: " + err.Error())
		fmt.Fprintf(ctx.Writer, "data: {\"error\": \"%s\"}\n\n", err.Error())
		ctx.Writer.Flush()
	}
}
