package admin

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"seekF-backend/internal/configs"
	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/pkg/auth"
	"seekF-backend/internal/pkg/jwt"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	adminservice "seekF-backend/internal/services/admin_service"
	userservice "seekF-backend/internal/services/user_service"
)

// AdminController 处理管理端认证与只读查询。
type AdminController struct {
	service *adminservice.AdminService
	auth    userservice.AuthService
}

// NewAdminController 创建管理端控制器并复用现有账号认证服务。
func NewAdminController(service *adminservice.AdminService, auth userservice.AuthService) *AdminController {
	return &AdminController{service: service, auth: auth}
}

// Login 校验密码及管理员权限后设置独立的HttpOnly登录Cookie。
func (a *AdminController) Login(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var req userservice.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, "请填写账号和密码", 400)
		return
	}
	result, err := a.auth.Login(&req)
	if err != nil {
		resp.Error(c, "账号或密码错误", 401)
		return
	}
	if result.User.IsAdmin != 1 || result.User.Status != 0 {
		if err := a.auth.Logout(result.Token); err != nil {
			zlog.Error("清理非管理员登录会话失败: " + err.Error())
		}
		resp.Error(c, "此账号无管理端访问权限", 403)
		return
	}
	cfg := configs.GetConfig()
	minutes := cfg.AuthConfig.SessionExpireMinutes
	if strings.EqualFold(cfg.AuthConfig.Mode, "jwt") {
		minutes = cfg.JWTConfig.ExpireMinutes
	}
	a.setCookie(c, result.Token, int(minutes*60))
	resp.Success(c, "管理员登录成功", adminresp.Identity{UUID: result.User.Uuid, Nickname: result.User.Nickname, Avatar: result.User.Avatar})
}

// RequireAdmin 验证独立登录会话并在数据库中重新检查权限和账号状态。
func (a *AdminController) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		token, err := c.Cookie("admin_token")
		if err != nil || token == "" {
			resp.Error(c, "请先登录管理端", 401)
			c.Abort()
			return
		}
		var uuid string
		if strings.EqualFold(configs.GetConfig().AuthConfig.Mode, "jwt") {
			claims, parseErr := jwt.ParseToken(token)
			if parseErr == nil {
				uuid = claims.UUID
			}
		} else {
			session, sessionErr := auth.GetSession(token)
			if sessionErr == nil && session != nil {
				uuid = session.UUID
			}
		}
		if uuid == "" {
			a.setCookie(c, "", -1)
			resp.Error(c, "管理端登录已过期", 401)
			c.Abort()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		user, findErr := a.service.Identity(ctx, uuid)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				a.setCookie(c, "", -1)
				resp.Error(c, "管理员账号不存在或已删除", 403)
			} else {
				zlog.Error("管理员身份校验失败，管理员=" + uuid + ": " + findErr.Error())
				resp.Error(c, "身份校验服务暂时不可用", 503)
			}
			c.Abort()
			return
		}
		if user.IsAdmin != 1 || user.Status != 0 {
			a.setCookie(c, "", -1)
			resp.Error(c, "此账号无管理端访问权限", 403)
			c.Abort()
			return
		}
		c.Set("admin_identity", adminresp.Identity{UUID: user.Uuid, Nickname: user.Nickname, Avatar: user.Avatar})
		c.Next()
	}
}

// Me 返回当前管理员身份。
func (a *AdminController) Me(c *gin.Context) {
	identity, _ := c.Get("admin_identity")
	resp.Success(c, "获取管理员信息成功", identity)
}

// Logout 撤销管理端会话，不清除用户端Cookie。
func (a *AdminController) Logout(c *gin.Context) {
	if token, err := c.Cookie("admin_token"); err == nil && token != "" {
		if err := a.auth.Logout(token); err != nil {
			zlog.Error("管理员退出登录失败: " + err.Error())
			resp.Error(c, "退出登录失败，请重试", 500)
			return
		}
	}
	a.setCookie(c, "", -1)
	resp.Success(c, "退出管理端成功", nil)
}

// Overview 返回限时聚合的业务概览。
func (a *AdminController) Overview(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	data, err := a.service.Overview(ctx)
	if err != nil {
		zlog.Error("查询管理概览失败，管理员=" + c.MustGet("admin_identity").(adminresp.Identity).UUID + ": " + err.Error())
		resp.Error(c, "概览数据暂时不可用，请重试", 503)
		return
	}
	resp.Success(c, "获取管理概览成功", data)
}

// System 返回当前进程与依赖状态。
func (a *AdminController) System(c *gin.Context) {
	resp.Success(c, "获取系统状态成功", a.service.System(c.Request.Context()))
}

// setCookie 使用同源Cookie保存管理端令牌，HTTPS部署通过环境变量开启Secure。
func (a *AdminController) setCookie(c *gin.Context, token string, age int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("admin_token", token, age, "/", "", c.Request.TLS != nil || strings.EqualFold(os.Getenv("ADMIN_COOKIE_SECURE"), "true"), true)
}
