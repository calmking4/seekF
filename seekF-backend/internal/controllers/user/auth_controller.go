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

	result, err := userservice.Login(&req)
	if err != nil {
		zlog.Info("Login service err: " + err.Error())
		resp.Error(c, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(c, "登录成功", result)
}

// Logout 用户登出
func Logout(c *gin.Context) {
	// 从上下文获取token（由JWTAuth中间件验证后设置）
	token, exists := c.Get("token")
	if !exists {
		resp.Error(c, "未登录或 token 无效", 401)
		return
	}

	tokenString, ok := token.(string)
	if !ok {
		resp.Error(c, "token 格式错误", 401)
		return
	}

	// 调用service层执行登出操作
	err := userservice.Logout(tokenString)
	if err != nil {
		zlog.Info("Logout service err: " + err.Error())
		resp.Error(c, "退出登录失败", http.StatusInternalServerError)
		return
	}

	resp.Success(c, "退出登录成功", nil)
}
