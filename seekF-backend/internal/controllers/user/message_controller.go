package user

import (
	"net/http"

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

	// 调用服务层方法
	messageList, total, err := c.messageService.GetUserMessageList(req.UserOneId, req.UserTwoId, req.Page, req.PageSize)
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

// GetGroupMessageList 获取群聊消息记录
func (c *MessageController) GetGroupMessageList(ctx *gin.Context) {
	// 绑定请求参数
	var req userreq.GetGroupMessageListRequest
	if err := ctx.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	messageList, total, err := c.messageService.GetGroupMessageList(req.GroupId, req.Page, req.PageSize)
	if err != nil {
		zlog.Info("获取群组消息列表服务错误: " + err.Error())
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

	messageList, total, err := c.messageService.SearchMessages(req.SessionId, req.Keyword, req.Page, req.PageSize)
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

	messageList, err := c.messageService.SearchMessageSuggestions(userId, req.Keyword, req.PageSize)
	if err != nil {
		zlog.Info("搜索消息联想服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "搜索成功", gin.H{
		"list": messageList,
	})
}
