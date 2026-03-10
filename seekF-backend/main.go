package main

import (
	"seekF-backend/internal/router"
)

func main() {
	// 初始化路由器
	r := router.SetupRouter()

	//启动服务，监听 8080 端口
	r.Run()
}
