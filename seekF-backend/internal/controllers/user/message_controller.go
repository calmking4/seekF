package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	userservice "seekF-backend/internal/services/user_service"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"

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

// GetMessageList 获取聊天记录
func (c *MessageController) GetMessageList(ctx *gin.Context) {
	// 绑定请求参数
	var req userreq.GetMessageListRequest
	if err := ctx.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	messageList, err := c.messageService.GetMessageList(req.UserOneId, req.UserTwoId)
	if err != nil {
		zlog.Info("GetMessageList service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取聊天记录成功", messageList)
}
