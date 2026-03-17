package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type SessionController struct {
	sessionService userservice.SessionService
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

	// 调用服务层方法
	sessionId, err := c.sessionService.OpenSession(userUuid.(string), openSessionReq.ReceiveId)
	if err != nil {
		zlog.Info("OpenSession service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "会话创建成功", sessionId)
}

// GetSessionList 获取会话列表（用户和群聊）
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
		zlog.Info("GetSessionList service err: " + err.Error())
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
		zlog.Info("DeleteSession service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除成功", nil)
}
