package user

import (
	"net/http"

	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type ContactController struct {
	contactService userservice.ContactService
}

func NewContactController(contactService userservice.ContactService) *ContactController {
	return &ContactController{
		contactService: contactService,
	}
}

// GetUserList 获取联系人列表
func (c *ContactController) GetUserList(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	userList, err := c.contactService.GetUserList(userUuid.(string))
	if err != nil {
		zlog.Info("GetUserList service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取用户列表成功", userList)
}
