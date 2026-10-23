package user

import (
	"net/http"
	"time"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type SessionController struct {
	sessionService userservice.SessionService
}

// Pin 设置本人私聊置顶状态并返回实际保存的时间。
func (c *SessionController) Pin(ctx *gin.Context) {
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}
	var request userreq.SessionPinRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		resp.Error(ctx, "会话编号或置顶状态无效", http.StatusBadRequest)
		return
	}
	pinnedAt, err := c.sessionService.SetPinned(ctx.Request.Context(), userId.(string), request.SessionId, *request.Pinned)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	value := ""
	if pinnedAt != nil {
		value = pinnedAt.Format(time.RFC3339Nano)
	}
	resp.Success(ctx, "设置置顶成功", gin.H{"is_pinned": pinnedAt != nil, "pinned_at": value})
}

// Unread 获取当前用户的私聊未读统计。
func (c *SessionController) Unread(ctx *gin.Context) {
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}
	result, err := c.sessionService.GetUnreadSessions(userId.(string))
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "获取成功", result)
}

// Read 标记当前用户已展示的私聊消息为已读。
func (c *SessionController) Read(ctx *gin.Context) {
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}
	var request userreq.SessionReadRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		resp.Error(ctx, "会话编号或已读消息边界无效", http.StatusBadRequest)
		return
	}
	if err := c.sessionService.ReadSession(ctx.Request.Context(), userId.(string), request.SessionId, request.MessageId); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "标记已读成功", nil)
}

func NewSessionController(sessionService userservice.SessionService) *SessionController {
	return &SessionController{
		sessionService: sessionService,
	}
}

// OpenSession 打开会话
func (c *SessionController) OpenSession(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var openSessionReq userreq.OpenSessionRequest
	if err := ctx.BindJSON(&openSessionReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 打开无需好友关系的用户私聊。
	result, err := c.sessionService.OpenSessionWithInfo(userUuid.(string), openSessionReq.ReceiveId)
	if err != nil {
		zlog.Info("打开会话服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "会话创建成功", result)
}

// GetSessionList 获取用户私聊会话列表。
func (c *SessionController) GetSessionList(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 调用服务层方法
	sessionList, err := c.sessionService.GetSessionList(userUuid.(string))
	if err != nil {
		zlog.Info("获取会话列表服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", sessionList)
}

// DeleteSession 删除会话
func (c *SessionController) DeleteSession(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var deleteSessionReq userreq.DeleteSessionRequest
	if err := ctx.BindJSON(&deleteSessionReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	err := c.sessionService.DeleteSession(userUuid.(string), deleteSessionReq.SessionId)
	if err != nil {
		zlog.Info("删除会话服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除成功", nil)
}

// CheckOpenSessionAllowed 检查是否可以打开会话
func (c *SessionController) CheckOpenSessionAllowed(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var openSessionReq userreq.OpenSessionRequest
	if err := ctx.BindJSON(&openSessionReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	allowed, err := c.sessionService.CheckOpenSessionAllowed(userUuid.(string), openSessionReq.ReceiveId)
	if err != nil {
		zlog.Info("检查会话权限服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "检查成功", allowed)
}
