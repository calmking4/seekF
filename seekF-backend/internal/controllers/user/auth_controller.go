package user

import (
	"net/http"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// Register 用户注册
func Register(c *gin.Context) {
	var req userservice.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zlog.Info("Register err: " + err.Error())
		resp.Error(c, "参数绑定失败", http.StatusBadRequest)
		return
	}

	err := userservice.Register(&req)
	if err != nil {
		zlog.Info("Register service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "注册成功", nil)
}

// Login 用户登录
func Login(c *gin.Context) {
	var req userservice.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zlog.Info("Login err: " + err.Error())
		resp.Error(c, "参数绑定失败", http.StatusBadRequest)
		return
	}

	token, err := userservice.Login(&req)
	if err != nil {
		zlog.Info("Login service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "登录成功", gin.H{
		"token": token,
	})
}
