package user

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/ai/asr"
	tool "seekF-backend/internal/pkg/ai/mcp/tool"
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
	result, err := c.aiChatService.CreateSession(userId, req)
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
	result, err := c.aiChatService.GetSessionList(userId, req.Page, req.PageSize)
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

	messageList, total, err := c.aiChatService.GetMessageHistory(req.SessionId, req.PageSize, req.Cursor, req.Direction)
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

// SendMessage 发送文本或图片消息，通过SSE返回AI回答和参考来源。
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
			zlog.Error("上传图片失败: " + err.Error())
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

	// 推送AI回答的文本片段。
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

	// 推送网页搜索来源。
	onSources := func(sources []tool.SearchSource) error {
		sourcesJSON, _ := json.Marshal(sources)
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"sources\": %s}\n\n", sourcesJSON)
		if err != nil {
			return err
		}
		ctx.Writer.Flush()
		return nil
	}

	// 推送搜索到的相关帖子。
	onPosts := func(posts []tool.DiscoverPostItem) error {
		postsJSON, _ := json.Marshal(posts)
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"posts\": %s}\n\n", postsJSON)
		if err != nil {
			return err
		}
		ctx.Writer.Flush()
		return nil
	}

	// 通知前端回答生成完成。
	onComplete := func(fullContent string) error {
		_, err := fmt.Fprintf(ctx.Writer, "data: {\"done\": true}\n\n")
		if err != nil {
			return err
		}
		ctx.Writer.Flush()
		return nil
	}

	// 与 HTTP 连接解耦：用户切换会话导致连接断开时，仍继续生成并持久化
	genCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx.Request.Context()), 15*time.Minute)
	defer cancel()

	err := c.aiChatService.SendMessageStream(genCtx, userId, req, onChunk, onSources, onPosts, onComplete)
	if err != nil {
		zlog.Info("发送消息服务错误: " + err.Error())
		fmt.Fprintf(ctx.Writer, "data: {\"error\": \"%s\"}\n\n", err.Error())
		ctx.Writer.Flush()
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

	if err := c.aiChatService.DeleteSession(req.SessionId); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除会话成功", nil)
}
