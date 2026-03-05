package middlewares

import (
	"strings"

	"seekF-backend/internal/pkg/jwt"
	"seekF-backend/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

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

		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			resp.Error(c, "token 无效或已过期", 401)
			c.Abort()
			return
		}

		// 把解析出的用户信息存到 Context，后面的 controller 可以用
		c.Set("userID", claims.UserID)
		c.Set("userPhone", claims.Phone)
		c.Set("userNickname", claims.Nickname)

		c.Next()
	}
}
