package main

import (
	"seekF-backend/internal/routers"
)

func main() {
	// 初始化路由器
	r := routers.SetupRouter()

	//启动服务，监听 8080 端口
	r.Run()
}
