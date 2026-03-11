package user

import (
	"net/http"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// CreateGroup 创建群聊
func CreateGroup(c *gin.Context) {
	var createGroupReq userreq.CreateGroupRequest
	if err := c.ShouldBindJSON(&createGroupReq); err != nil {
		zlog.Info("CreateGroup err: " + err.Error())
		resp.Error(c, "参数绑定失败", http.StatusBadRequest)
		return
	}
	// 从上下文获取当前用户UUID作为群主ID
	userUuid, exists := c.Get("Uuid")
	if !exists {
		resp.Error(c, "无法获取用户信息", http.StatusUnauthorized)
		return
	}
	// 设置群主ID为当前用户UUID
	createGroupReq.OwnerId = userUuid.(string)

	err := userservice.CreateGroup(&createGroupReq)
	if err != nil {
		zlog.Info("CreateGroup service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "创建群聊成功", nil)
}

// LoadMyGroup 获取我创建的群聊
func LoadMyGroup(c *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := c.Get("Uuid")
	if !exists {
		resp.Error(c, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	groupList, err := userservice.LoadMyGroup(userUuid.(string))
	if err != nil {
		zlog.Info("LoadMyGroup service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "获取群聊成功", groupList)
}
