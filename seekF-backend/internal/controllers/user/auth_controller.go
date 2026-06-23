package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"seekF-backend/internal/configs"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"
	"strings"

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
		zlog.Info("注册参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	err := c.authService.Register(&req)
	if err != nil {
		zlog.Info("注册服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "注册成功", nil)
}

// Login 用户登录
func (c *AuthController) Login(ctx *gin.Context) {
	var req userservice.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("登录参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	result, err := c.authService.Login(&req)
	if err != nil {
		zlog.Info("登录服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	// 设置cookie
	cfg := configs.GetConfig()
	expireSeconds := cfg.SessionExpireMinutes * 60
	ctx.SetCookie("token", result.Token, int(expireSeconds), "/", "localhost", false, true)

	resp.Success(ctx, "登录成功", result)
}

// GithubLogin 跳转 GitHub OAuth 授权页
func (c *AuthController) GithubLogin(ctx *gin.Context) {
	authURL, err := c.authService.GithubAuthURL()
	if err != nil {
		zlog.Error("生成 GitHub 授权地址失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	ctx.Redirect(http.StatusFound, authURL)
}

// GithubCallback GitHub OAuth 回调
func (c *AuthController) GithubCallback(ctx *gin.Context) {
	cfg := configs.GetConfig()
	frontendURL := strings.TrimSpace(cfg.GithubOAuthConfig.FrontendRedirectURL)
	if frontendURL == "" {
		frontendURL = "http://localhost:3000/oauth/github/callback"
	}

	redirectWithError := func(message string) {
		target := fmt.Sprintf("%s?error=%s", frontendURL, url.QueryEscape(message))
		ctx.Redirect(http.StatusFound, target)
	}

	if errMsg := strings.TrimSpace(ctx.Query("error")); errMsg != "" {
		zlog.Info("GitHub 授权被拒绝: " + errMsg)
		redirectWithError("GitHub 授权已取消")
		return
	}

	code := strings.TrimSpace(ctx.Query("code"))
	state := strings.TrimSpace(ctx.Query("state"))

	result, err := c.authService.LoginByGithub(code, state)
	if err != nil {
		zlog.Error("GitHub 登录失败: " + err.Error())
		redirectWithError(err.Error())
		return
	}

	expireSeconds := cfg.SessionExpireMinutes * 60
	ctx.SetCookie("token", result.Token, int(expireSeconds), "/", "localhost", false, true)

	userBytes, err := json.Marshal(result.User)
	if err != nil {
		zlog.Error("序列化 GitHub 登录用户信息失败: " + err.Error())
		redirectWithError("登录成功但跳转失败")
		return
	}

	ctx.Redirect(http.StatusFound, fmt.Sprintf("%s?user=%s", frontendURL, url.QueryEscape(string(userBytes))))
}

// GiteeLogin 跳转 Gitee OAuth 授权页
func (c *AuthController) GiteeLogin(ctx *gin.Context) {
	authURL, err := c.authService.GiteeAuthURL()
	if err != nil {
		zlog.Error("生成 Gitee 授权地址失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	ctx.Redirect(http.StatusFound, authURL)
}

// GiteeCallback Gitee OAuth 回调
func (c *AuthController) GiteeCallback(ctx *gin.Context) {
	cfg := configs.GetConfig()
	frontendURL := strings.TrimSpace(cfg.GiteeOAuthConfig.FrontendRedirectURL)
	if frontendURL == "" {
		frontendURL = "http://localhost:3000/oauth/gitee/callback"
	}

	redirectWithError := func(message string) {
		target := fmt.Sprintf("%s?error=%s", frontendURL, url.QueryEscape(message))
		ctx.Redirect(http.StatusFound, target)
	}

	if errMsg := strings.TrimSpace(ctx.Query("error")); errMsg != "" {
		zlog.Info("Gitee 授权被拒绝: " + errMsg)
		redirectWithError("Gitee 授权已取消")
		return
	}

	code := strings.TrimSpace(ctx.Query("code"))
	state := strings.TrimSpace(ctx.Query("state"))

	result, err := c.authService.LoginByGitee(code, state)
	if err != nil {
		zlog.Error("Gitee 登录失败: " + err.Error())
		redirectWithError(err.Error())
		return
	}

	expireSeconds := cfg.SessionExpireMinutes * 60
	ctx.SetCookie("token", result.Token, int(expireSeconds), "/", "localhost", false, true)

	userBytes, err := json.Marshal(result.User)
	if err != nil {
		zlog.Error("序列化 Gitee 登录用户信息失败: " + err.Error())
		redirectWithError("登录成功但跳转失败")
		return
	}

	ctx.Redirect(http.StatusFound, fmt.Sprintf("%s?user=%s", frontendURL, url.QueryEscape(string(userBytes))))
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
		zlog.Info("登出服务错误: " + err.Error())
		resp.Error(ctx, "退出登录失败", http.StatusInternalServerError)
		return
	}

	// 清除 token cookie
	ctx.SetCookie("token", "", -1, "/", "localhost", false, true)

	resp.Success(ctx, "退出登录成功", nil)
}

// SendVerifyCode 发送验证码
func (c *AuthController) SendVerifyCode(ctx *gin.Context) {
	var req userreq.SendVerifyCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("发送验证码参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 调用service层发送验证码
	err := c.authService.SendVerifyCode(req.Telephone)
	if err != nil {
		zlog.Info("发送验证码服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "验证码发送成功", nil)
}

// LoginByCode 验证码登录
func (c *AuthController) LoginByCode(ctx *gin.Context) {
	var req userreq.LoginByCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("验证码登录参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 调用service层执行验证码登录
	result, err := c.authService.LoginByCode(req.Telephone, req.Code)
	if err != nil {
		zlog.Info("验证码登录服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	// 设置cookie
	cfg := configs.GetConfig()
	expireSeconds := cfg.SessionExpireMinutes * 60
	ctx.SetCookie("token", result.Token, int(expireSeconds), "/", "localhost", false, true)

	resp.Success(ctx, "登录成功", result)
}
