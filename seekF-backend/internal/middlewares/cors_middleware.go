package middlewares

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware 允许用户端和管理端明确的本机开发来源，兼容域名与回环地址访问。
func CORSMiddleware() gin.HandlerFunc {
	// 配置CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://[::1]:3000",
		"http://localhost:3002",
		"http://127.0.0.1:3002",
		"http://[::1]:3002",
	}
	config.AllowCredentials = true // 允许携带凭证
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	config.ExposeHeaders = []string{"Content-Range", "Accept-Ranges"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Range"}

	return cors.New(config)
}
