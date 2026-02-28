package main

// 导入 Gin 包
import "github.com/gin-gonic/gin"

func main() {
	// 1. 创建 Gin 引擎实例（默认模式，包含日志和恢复中间件）
	// 如果是生产环境，可使用 gin.ReleaseMode：
	// gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 2. 定义路由：GET 请求，路径为 /hello
	r.GET("/hello", func(c *gin.Context) {
		// 返回 JSON 响应
		c.JSON(200, gin.H{
			"message": "Hello Gin!",
			"status":  "success",
		})
	})

	// 3. 启动服务，监听 8080 端口
	// 若想指定其他端口，如 9090：r.Run(":9090")
	r.Run()
}
