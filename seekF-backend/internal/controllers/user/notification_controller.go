package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notificationService userservice.NotificationService
}

func NewNotificationController(notificationService userservice.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

// ListNotifications 获取通知列表
func (c *NotificationController) ListNotifications(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.ListNotificationRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 20
	}

	notifications, total, err := c.notificationService.ListNotifications(ctx.Request.Context(), userId, req.Page, req.PageSize)
	if err != nil {
		zlog.Error("获取通知列表失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "获取成功", gin.H{
		"list":       notifications,
		"total":      total,
		"page":       req.Page,
		"page_size":  req.PageSize,
	})
}

// GetUnreadCount 获取未读通知数
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	count, err := c.notificationService.GetUnreadCount(ctx.Request.Context(), userId)
	if err != nil {
		zlog.Error("获取未读通知数失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "获取成功", gin.H{"unread_count": count})
}

// MarkAsRead 标记通知已读
func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.MarkNotificationReadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if len(req.Ids) == 0 {
		resp.Error(ctx, "通知ID不能为空", http.StatusBadRequest)
		return
	}

	if err := c.notificationService.MarkAsRead(ctx.Request.Context(), userId, req.Ids); err != nil {
		zlog.Error("标记通知已读失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "标记成功", nil)
}

// MarkAllAsRead 标记所有通知已读
func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	if err := c.notificationService.MarkAllAsRead(ctx.Request.Context(), userId); err != nil {
		zlog.Error("标记所有通知已读失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "标记成功", nil)
}
