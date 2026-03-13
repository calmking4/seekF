package user

import (
	"net/http"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService userservice.AuthService
}

func NewAuthController(authService userservice.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// Register 用户注册
func (c *AuthController) Register(ctx *gin.Context) {
	var req userservice.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("Register err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	err := c.authService.Register(&req)
	if err != nil {
		zlog.Info("Register service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "注册成功", nil)
}

// Login 用户登录
func (c *AuthController) Login(ctx *gin.Context) {
	var req userservice.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("Login err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	result, err := c.authService.Login(&req)
	if err != nil {
		zlog.Info("Login service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "登录成功", result)
}

// Logout 用户登出
func (c *AuthController) Logout(ctx *gin.Context) {
	// 从上下文获取token（由JWTAuth中间件验证后设置）
	token, exists := ctx.Get("token")
	if !exists {
		resp.Error(ctx, "未登录或 token 无效", 401)
		return
	}

	tokenString, ok := token.(string)
	if !ok {
		resp.Error(ctx, "token 格式错误", 401)
		return
	}

	// 调用service层执行登出操作
	err := c.authService.Logout(tokenString)
	if err != nil {
		zlog.Info("Logout service err: " + err.Error())
		resp.Error(ctx, "退出登录失败", http.StatusInternalServerError)
		return
	}

	resp.Success(ctx, "退出登录成功", nil)
}
