package routers

import (
	"seekF-backend/internal/controllers/user"
	"seekF-backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 添加CORS中间件
	r.Use(middlewares.CORSMiddleware())

	// 用户认证相关路由
	userGroup := r.Group("/user")
	{
		userGroup.POST("/register", user.Register)
		userGroup.POST("/login", user.Login)
	}

	return r
}
