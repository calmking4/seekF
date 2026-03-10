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
