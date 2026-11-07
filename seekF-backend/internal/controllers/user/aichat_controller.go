package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/ai/asr"
	"seekF-backend/internal/pkg/ratelimit"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// AIChatController 处理AI会话、消息和语音请求。
type AIChatController struct {
	aiChatService userservice.AIChatService
	fileService   userservice.FileService
}

// NewAIChatController 创建AI聊天控制器。
func NewAIChatController(aiChatService userservice.AIChatService, fileService userservice.FileService) *AIChatController {
	return &AIChatController{
		aiChatService: aiChatService,
		fileService:   fileService,
	}
}

// CreateSession 创建用户的AI会话。
func (c *AIChatController) CreateSession(ctx *gin.Context) {
	var req userreq.CreateAISessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	userId := ctx.GetString("Uuid")
	result, err := c.aiChatService.CreateSession(ctx.Request.Context(), userId, req)
	if err != nil {
		zlog.Info("创建会话服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "创建AI会话成功", result)
}

// GetSessionList 分页获取用户的AI会话列表。
func (c *AIChatController) GetSessionList(ctx *gin.Context) {
	var req userreq.GetAISessionListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	userId := ctx.GetString("Uuid")
	result, err := c.aiChatService.GetSessionList(ctx.Request.Context(), userId, req.Page, req.PageSize)
	if err != nil {
		zlog.Info("获取会话列表服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取AI会话列表成功", result)
}

// GetMessageHistory 按游标分页获取AI会话的消息历史。
func (c *AIChatController) GetMessageHistory(ctx *gin.Context) {
	var req userreq.GetAIMessageHistoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	messageList, total, err := c.aiChatService.GetMessageHistory(ctx.Request.Context(), ctx.GetString("Uuid"), req.SessionId, req.PageSize, req.Cursor, req.Direction)
	if err != nil {
		zlog.Info("获取消息历史服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取AI消息历史成功", gin.H{
		"list":  messageList,
		"total": total,
	})
}

// aiStreamError 返回初始化或恢复失败的明确状态，不将失败伪装成正常完成。
func aiStreamError(ctx *gin.Context, err error) {
	status := http.StatusBadRequest
	var limitErr *ratelimit.Error
	switch {
	case errors.Is(err, userservice.ErrAIStreamMissing):
		status = http.StatusGone
	case errors.Is(err, userservice.ErrAIStreamForbidden):
		status = http.StatusForbidden
	case errors.As(err, &limitErr):
		status = limitErr.Status
	}
	resp.Error(ctx, err.Error(), status)
}

// StartGeneration 保存提问并启动独立生成任务，仅返回可订阅的streamId。
func (c *AIChatController) StartGeneration(ctx *gin.Context) {
	var req userreq.SendAIMessageRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "消息参数错误", http.StatusBadRequest)
		return
	}
	if file, err := ctx.FormFile("image"); err == nil {
		result, err := c.fileService.UploadFile(ctx.Request.Context(), ctx.GetString("Uuid"), file, oss.MessageImage)
		if err != nil {
			zlog.Error("上传AI消息图片失败: " + err.Error())
			resp.Error(ctx, "图片上传失败", http.StatusInternalServerError)
			return
		}
		req.ImageURL = result.URL
	}
	info, err := c.aiChatService.StartMessageStream(ctx.Request.Context(), ctx.GetString("Uuid"), req)
	if err != nil {
		aiStreamError(ctx, err)
		return
	}
	resp.Success(ctx, "AI回答任务已启动", info)
}

// GetStreamInfo 查询回答是否已落库，刷新时先决定加载历史还是恢复回放。
func (c *AIChatController) GetStreamInfo(ctx *gin.Context) {
	var req userreq.AIStreamRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Error(ctx, "回答任务参数错误", http.StatusBadRequest)
		return
	}
	info, err := c.aiChatService.GetMessageStream(ctx.Request.Context(), ctx.GetString("Uuid"), req.SessionId, req.StreamId)
	if err != nil {
		aiStreamError(ctx, err)
		return
	}
	resp.Success(ctx, "查询AI回答状态成功", info)
}

// StreamGeneration 回放Redis事件并等待后续片段，断开只终止订阅，不取消生成。
func (c *AIChatController) StreamGeneration(ctx *gin.Context) {
	var req userreq.AIStreamRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Error(ctx, "回答订阅参数错误", http.StatusBadRequest)
		return
	}
	if req.After == "" {
		req.After = "0-0"
	}
	info, err := c.aiChatService.GetMessageStream(ctx.Request.Context(), ctx.GetString("Uuid"), req.SessionId, req.StreamId)
	if err != nil {
		aiStreamError(ctx, err)
		return
	}
	// 首次读取同时验证游标，在发送SSE响应头之前返回参数和存储错误。
	events, err := c.aiChatService.ReadMessageStream(ctx.Request.Context(), req.StreamId, req.After)
	if err != nil {
		aiStreamError(ctx, err)
		return
	}
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")
	ctx.Status(http.StatusOK)
	ctx.Writer.Flush()
	writeTerminal := func(kind string, data any) {
		payload, _ := json.Marshal(data)
		fmt.Fprintf(ctx.Writer, "event: %s\ndata: %s\n\n", kind, payload)
		ctx.Writer.Flush()
	}
	for ctx.Request.Context().Err() == nil {
		for _, event := range events {
			if _, err := fmt.Fprintf(ctx.Writer, "id: %s\nevent: %s\ndata: %s\n\n", event.Id, event.Type, event.Data); err != nil {
				return
			}
			ctx.Writer.Flush()
			req.After = event.Id
			if event.Type == "done" || event.Type == "error" {
				return
			}
		}
		if len(events) == 0 {
			// 缺失终态时以MySQL补偿完成信号；心跳过期时明确报告进程中断。
			info, err = c.aiChatService.GetMessageStream(ctx.Request.Context(), ctx.GetString("Uuid"), req.SessionId, req.StreamId)
			if err != nil {
				writeTerminal("error", gin.H{"error": err.Error()})
				return
			}
			if info.Status == "finished" {
				writeTerminal("done", gin.H{"message_id": info.MessageId})
				return
			}
			if info.Status != "running" {
				writeTerminal("error", gin.H{"error": info.Error})
				return
			}
			if _, err := fmt.Fprint(ctx.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			ctx.Writer.Flush()
		}
		events, err = c.aiChatService.ReadMessageStream(ctx.Request.Context(), req.StreamId, req.After)
		if err != nil {
			if ctx.Request.Context().Err() == nil {
				writeTerminal("error", gin.H{"error": err.Error()})
			}
			return
		}
	}
}

// SpeechToText 校验上传录音并实时转发识别事件流。
func (c *AIChatController) SpeechToText(ctx *gin.Context) {
	// 在录音大小上限之外预留multipart表单开销。
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, asr.MaxAudioBytes+64*1024)
	if err := ctx.Request.ParseMultipartForm(asr.MaxAudioBytes + 64*1024); err != nil {
		resp.Error(ctx, "录音上传失败，文件过大或格式无效", http.StatusBadRequest)
		return
	}
	defer ctx.Request.MultipartForm.RemoveAll()
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		resp.Error(ctx, "请上传录音文件", http.StatusBadRequest)
		return
	}
	defer file.Close()
	audio, err := io.ReadAll(io.LimitReader(file, asr.MaxAudioBytes+1))
	if err != nil {
		resp.Error(ctx, "读取录音失败，请重新录音", http.StatusBadRequest)
		return
	}
	if err := asr.ValidateRecording(audio); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := c.aiChatService.SpeechToText(ctx.Request.Context(), audio)
	if err != nil {
		zlog.Error("语音识别失败，用户ID=" + ctx.GetString("Uuid") + ": " + err.Error())
		resp.Error(ctx, "语音识别失败，请稍后重试", http.StatusBadGateway)
		return
	}
	defer result.Body.Close()
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("X-Accel-Buffering", "no")
	ctx.Status(http.StatusOK)
	ctx.Writer.Flush()
	// 每次读取后立即刷新，取消请求时由上下文终止供应商连接。
	buf := make([]byte, 4096)
	for {
		n, readErr := result.Body.Read(buf)
		if n > 0 {
			if _, err := ctx.Writer.Write(buf[:n]); err != nil {
				return
			}
			ctx.Writer.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF && ctx.Request.Context().Err() == nil {
				zlog.Error("读取语音识别流失败，用户ID=" + ctx.GetString("Uuid") + ": " + readErr.Error())
				fmt.Fprint(ctx.Writer, "data: {\"error\":{\"message\":\"语音识别连接中断，请重试\"}}\n\n")
				ctx.Writer.Flush()
			}
			return
		}
	}
}

// TextToSpeech 将文字转换为语音，流式返回PCM音频。
func (c *AIChatController) TextToSpeech(ctx *gin.Context) {
	var req userreq.TTSRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	streamResult, err := c.aiChatService.TextToSpeech(ctx.Request.Context(), req.Content, req.Voice)
	if err != nil {
		zlog.Error("语音合成失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusInternalServerError)
		return
	}
	defer streamResult.Body.Close()

	// 设置流式响应头
	ctx.Header("Content-Type", "audio/pcm")
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Status(http.StatusOK)

	// 逐块转发 PCM 数据
	buf := make([]byte, 4096)
	for {
		n, readErr := streamResult.Body.Read(buf)
		if n > 0 {
			if _, writeErr := ctx.Writer.Write(buf[:n]); writeErr != nil {
				zlog.Error("写入音频数据块失败: " + writeErr.Error())
				return
			}
			ctx.Writer.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				zlog.Error("读取音频数据流失败: " + readErr.Error())
			}
			break
		}
	}
}

// DeleteSession 删除指定的AI会话。
func (c *AIChatController) DeleteSession(ctx *gin.Context) {
	var req userreq.DeleteAISessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if err := c.aiChatService.DeleteSession(ctx.Request.Context(), ctx.GetString("Uuid"), req.SessionId); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除会话成功", nil)
}
