package user

import (
	"net/http"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// GetUserInfo 获取用户信息
func GetUserInfo(c *gin.Context) {
	var req userreq.GetUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zlog.Info("GetUserInfo err: " + err.Error())
		resp.Error(c, "参数绑定失败", http.StatusBadRequest)
		return
	}

	result, err := userservice.GetUserInfo(&req)
	if err != nil {
		zlog.Info("GetUserInfo service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "获取用户信息成功", result)
}

// UpdateUserInfo 更新用户信息
func UpdateUserInfo(c *gin.Context) {
	var req userreq.UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zlog.Info("UpdateUserInfo err: " + err.Error())
		resp.Error(c, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户的 UUID
	currentUserUuid, exists := c.Get("Uuid")
	if !exists {
		zlog.Info("UpdateUserInfo err: Unable to get user UUID from context")
		resp.Error(c, "无法获取用户信息", http.StatusInternalServerError)
		return
	}

	// 验证用户只能更新自己的信息
	if req.Uuid != currentUserUuid.(string) {

		zlog.Info("UpdateUserInfo err: User can only update their own info")
		resp.Error(c, "只能更新自己的用户信息", http.StatusForbidden)
		return
	}

	err := userservice.UpdateUserInfo(&req)
	if err != nil {
		zlog.Info("UpdateUserInfo service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "更新用户信息成功", nil)
}
