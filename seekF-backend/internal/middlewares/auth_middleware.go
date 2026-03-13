package middlewares

import (
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/auth"
	"seekF-backend/internal/pkg/jwt"
	"seekF-backend/internal/pkg/resp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Auth 根据配置选择认证方案
func Auth() gin.HandlerFunc {
	cfg := configs.GetConfig()
	mode := strings.ToLower(strings.TrimSpace(cfg.AuthConfig.Mode))
	if mode == "jwt" {
		return JWTAuth()
	}
	return TokenRedisAuth()
}

// TokenRedisAuth 中间件：校验不透明 token，并从 Redis 会话中恢复用户信息
func TokenRedisAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			resp.Error(c, "未登录或 token 缺失", 401)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			resp.Error(c, "Authorization 格式错误", 401)
			c.Abort()
			return
		}

		tokenString := parts[1]
		sess, err := auth.GetSession(tokenString)
		if err != nil {
			resp.Error(c, "token 无效或已过期", 401)
			c.Abort()
			return
		}
		if sess == nil {
			resp.Error(c, "token 已失效或用户已登出", 401)
			c.Abort()
			return
		}

		c.Set("ID", sess.Id)
		c.Set("Phone", sess.Phone)
		c.Set("Nickname", sess.Nickname)
		c.Set("Uuid", sess.UUID)
		c.Set("token", tokenString)

		c.Next()
	}
}

// JWTAuth 中间件：校验 JWT，失败直接返回
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 里取 Authorization: Bearer xxx
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			resp.Error(c, "未登录或 token 缺失", 401)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			resp.Error(c, "Authorization 格式错误", 401)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 首先验证 JWT token 本身的有效性
		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			resp.Error(c, "token 无效或已过期", 401)
			c.Abort()
			return
		}

		// 在 Gin 框架中，控制器（controller）可以通过上下文（context）获取中间件中存储的数据。通过c.Get("key")获取
		c.Set("ID", claims.Id)
		c.Set("Phone", claims.Phone)
		c.Set("Nickname", claims.Nickname)
		c.Set("Uuid", claims.UUID) //用户UUID
		c.Set("token", tokenString)

		c.Next()
	}
}
