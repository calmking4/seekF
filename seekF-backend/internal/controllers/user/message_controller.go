package user

import (
	"net/http"
	"seekF-backend/internal/pkg/constants"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/websocket"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// MessageController 消息控制器
type MessageController struct {
	messageService userservice.MessageService
}

// NewMessageController 创建消息控制器实例
func NewMessageController(messageService userservice.MessageService) *MessageController {
	return &MessageController{
		messageService: messageService,
	}
}

// GetUserMessageList 获取用户聊天记录
func (c *MessageController) GetUserMessageList(ctx *gin.Context) {
	// 绑定请求参数
	var req userreq.GetUserMessageListRequest
	if err := ctx.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 私聊记录只允许会话参与者读取，发送给服务层的用户身份来自登录态。
	currentUser := ctx.GetString("Uuid")
	if currentUser == "" || (req.UserOneId != currentUser && req.UserTwoId != currentUser) {
		resp.Error(ctx, "无权查看其他用户的私聊记录", http.StatusForbidden)
		return
	}
	// 调用服务层方法
	peerId := req.UserTwoId
	if peerId == currentUser {
		peerId = req.UserOneId
	}
	if req.Refresh {
		added, updated, more, err := c.messageService.RefreshPrivateMessages(ctx.Request.Context(), currentUser, peerId, req.AfterId, req.KnownIds)
		if err != nil {
			resp.Error(ctx, err.Error(), http.StatusBadRequest)
			return
		}
		resp.Success(ctx, "刷新聊天记录成功", gin.H{"list": added, "updated": updated, "has_more": more})
		return
	}
	messageList, total, err := c.messageService.GetUserMessageList(ctx.Request.Context(), currentUser, peerId, req.Page, req.PageSize, req.BeforeMessageId)
	if err != nil {
		zlog.Info("获取用户消息列表服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取聊天记录成功", gin.H{
		"list":  messageList,
		"total": total,
	})
}

// SearchMessages 搜索聊天消息
func (c *MessageController) SearchMessages(ctx *gin.Context) {
	var req userreq.SearchMessageRequest
	if err := ctx.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	messageList, total, err := c.messageService.SearchMessages(ctx.Request.Context(), ctx.GetString("Uuid"), req.SessionId, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		zlog.Info("搜索消息服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "搜索成功", gin.H{
		"list":  messageList,
		"total": total,
	})
}

// SearchMessageSuggestions 搜索消息联想（跨会话）
func (c *MessageController) SearchMessageSuggestions(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	var req userreq.SearchMessageSuggestionsRequest
	if err := ctx.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	messageList, err := c.messageService.SearchMessageSuggestions(ctx.Request.Context(), userId, req.Keyword, req.PageSize)
	if err != nil {
		zlog.Info("搜索消息联想服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "搜索成功", gin.H{
		"list": messageList,
	})
}

// RecallMessage 提交撤回后向双方推送事件，离线通过历史读取最终状态。
func (c *MessageController) RecallMessage(ctx *gin.Context) {
	var req struct {
		MessageId string `json:"message_id" binding:"required,max=20"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "消息编号无效", http.StatusBadRequest)
		return
	}
	message, err := c.messageService.RecallMessage(ctx.Request.Context(), ctx.GetString("Uuid"), req.MessageId)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	event := gin.H{"type": "message_recalled", "uuid": message.Uuid, "send_id": message.SendId, "receive_id": message.ReceiveId}
	for _, userId := range []string{message.SendId, message.ReceiveId} {
		myredis.DelKeysWithPattern(constants.SessionListKeyPrefix + userId)
		if websocket.ChatServer != nil {
			websocket.ChatServer.PushNotification(userId, event)
		}
	}
	resp.Success(ctx, "撤回成功", event)
}

// DeleteMessage 只隐藏当前用户的私聊记录，并仅向该用户同步删除事件。
func (c *MessageController) DeleteMessage(ctx *gin.Context) {
	var req struct {
		MessageId string `json:"message_id" binding:"required,max=20"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "消息编号无效", http.StatusBadRequest)
		return
	}
	userId := ctx.GetString("Uuid")
	if err := c.messageService.DeleteMessage(ctx.Request.Context(), userId, req.MessageId); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	myredis.DelKeysWithPattern(constants.SessionListKeyPrefix + userId)
	event := gin.H{"type": "message_deleted", "uuid": req.MessageId, "user_id": userId}
	if websocket.ChatServer != nil {
		websocket.ChatServer.PushNotification(userId, event)
	}
	resp.Success(ctx, "删除成功", event)
}
