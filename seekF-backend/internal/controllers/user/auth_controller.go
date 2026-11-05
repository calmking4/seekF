package user

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"seekF-backend/internal/configs"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/auth"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService userservice.AuthService
}

// NewAuthController 构造认证控制器并注入业务服务。
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

	err := c.authService.Register(ctx.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			ctx.Set(constants.AuthCredentialFailed, true)
		}
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

	result, err := c.authService.Login(ctx.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			ctx.Set(constants.AuthCredentialFailed, true)
		}
		zlog.Info("登录服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	// 设置cookie
	cfg := configs.GetConfig()
	expireSeconds := cfg.SessionExpireMinutes * 60
	c.setLoginCookie(ctx, result.Token, int(expireSeconds))

	resp.Success(ctx, "登录成功", result)
}

// GithubLogin 跳转 GitHub OAuth 授权页
func (c *AuthController) GithubLogin(ctx *gin.Context) {
	authURL, state, err := c.authService.GithubAuthURL(ctx.Request.Context())
	if err != nil {
		zlog.Error("生成 GitHub 授权地址失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	c.setOAuthCookie(ctx, constants.OAuthProviderGithub, state, int(constants.OAuthStateTTL.Seconds()))
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

	state := ctx.Query("state")
	browserState, _ := ctx.Cookie(constants.OAuthStateCookiePrefix + constants.OAuthProviderGithub)
	c.clearMatchingOAuthCookie(ctx, constants.OAuthProviderGithub, state, browserState)
	if errMsg := strings.TrimSpace(ctx.Query("error")); errMsg != "" {
		if err := c.authService.CancelOAuthLogin(ctx.Request.Context(), constants.OAuthProviderGithub, state, browserState); err != nil {
			redirectWithError(err.Error())
			return
		}
		zlog.Info("GitHub 授权被拒绝: " + errMsg)
		redirectWithError("GitHub 授权已取消")
		return
	}

	code := strings.TrimSpace(ctx.Query("code"))

	result, err := c.authService.LoginByGithub(ctx.Request.Context(), code, state, browserState)
	if err != nil {
		zlog.Error("GitHub 登录失败: " + err.Error())
		redirectWithError(err.Error())
		return
	}

	expireSeconds := cfg.SessionExpireMinutes * 60
	c.setLoginCookie(ctx, result.Token, int(expireSeconds))

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
	authURL, state, err := c.authService.GiteeAuthURL(ctx.Request.Context())
	if err != nil {
		zlog.Error("生成 Gitee 授权地址失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	c.setOAuthCookie(ctx, constants.OAuthProviderGitee, state, int(constants.OAuthStateTTL.Seconds()))
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

	state := ctx.Query("state")
	browserState, _ := ctx.Cookie(constants.OAuthStateCookiePrefix + constants.OAuthProviderGitee)
	c.clearMatchingOAuthCookie(ctx, constants.OAuthProviderGitee, state, browserState)
	if errMsg := strings.TrimSpace(ctx.Query("error")); errMsg != "" {
		if err := c.authService.CancelOAuthLogin(ctx.Request.Context(), constants.OAuthProviderGitee, state, browserState); err != nil {
			redirectWithError(err.Error())
			return
		}
		zlog.Info("Gitee 授权被拒绝: " + errMsg)
		redirectWithError("Gitee 授权已取消")
		return
	}

	code := strings.TrimSpace(ctx.Query("code"))

	result, err := c.authService.LoginByGitee(ctx.Request.Context(), code, state, browserState)
	if err != nil {
		zlog.Error("Gitee 登录失败: " + err.Error())
		redirectWithError(err.Error())
		return
	}

	expireSeconds := cfg.SessionExpireMinutes * 60
	c.setLoginCookie(ctx, result.Token, int(expireSeconds))

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
	c.setLoginCookie(ctx, "", -1)

	resp.Success(ctx, "退出登录成功", nil)
}

// SendVerifyCode 发送验证码（支持手机号和邮箱）
func (c *AuthController) SendVerifyCode(ctx *gin.Context) {
	var req userreq.SendVerifyCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("发送验证码参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 调用service层发送验证码
	err := c.authService.SendVerifyCode(req.Telephone, req.Email)
	if err != nil {
		zlog.Info("发送验证码服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "验证码发送成功", nil)
}

// LoginByCode 验证码登录（支持手机号和邮箱）
func (c *AuthController) LoginByCode(ctx *gin.Context) {
	var req userreq.LoginByCodeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("验证码登录参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 调用service层执行验证码登录
	result, err := c.authService.LoginByCode(ctx.Request.Context(), req.Telephone, req.Email, req.Code)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			ctx.Set(constants.AuthCredentialFailed, true)
		}
		zlog.Info("验证码登录服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	// 设置cookie
	cfg := configs.GetConfig()
	expireSeconds := cfg.SessionExpireMinutes * 60
	c.setLoginCookie(ctx, result.Token, int(expireSeconds))

	resp.Success(ctx, "登录成功", result)
}

// userCookieSecure 使用显式部署配置或直接HTTPS连接决定是否仅通过HTTPS发送Cookie。
func userCookieSecure(ctx *gin.Context) bool {
	return ctx.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(os.Getenv("USER_COOKIE_SECURE")), "true")
}

// setLoginCookie 统一设置或删除用户登录Cookie，默认仅绑定当前主机且禁止脚本读取。
func (c *AuthController) setLoginCookie(ctx *gin.Context, token string, age int) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("token", token, age, "/", "", userCookieSecure(ctx), true)
}

// setOAuthCookie 设置或删除平台独立的临时授权Cookie，Lax允许授权平台顶层GET返回。
func (c *AuthController) setOAuthCookie(ctx *gin.Context, provider, state string, age int) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(constants.OAuthStateCookiePrefix+provider, state, age, "/", "", userCookieSecure(ctx), true)
}

// clearMatchingOAuthCookie 仅清除当前回调对应的Cookie，错误或旧回调不干扰后发起的登录。
func (c *AuthController) clearMatchingOAuthCookie(ctx *gin.Context, provider, state, browserState string) {
	if len(state) == constants.OAuthStateLength && subtle.ConstantTimeCompare([]byte(state), []byte(browserState)) == 1 {
		c.setOAuthCookie(ctx, provider, "", -1)
	}
}
