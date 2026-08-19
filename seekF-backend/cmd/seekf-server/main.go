package main

import (
	"seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/ai/rag"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/websocket"
)

func main() {
	// 使用 Wire 生成的注入器初始化 DAO/Service/Controller/Router
	app := initApp(db.GormDB)

	// 以下为全局单例初始化，不适合 Wire 管理

	// 初始化Kafka并启动WebSocket服务器
	kafka.KafkaServiceInstance.Init()
	websocket.ChatServer = websocket.NewServer(
		app.SessionService,
		app.MessageDAO,
		app.SessionDAO,
		app.GroupDAO,
	)
	go websocket.ChatServer.Start()

	// 初始化AI模型池
	ai.GetModelPool()

	// 启动AI消息Kafka消费者
	ai.StartAIConsumer()

	// 初始化并启动AI评论回复Kafka消费者
	ai.InitAICommentConsumer(app.DiscoverDAO, app.UserInfoDAO)
	ai.StartAICommentConsumer()

	// 初始化Qdrant
	db.InitQdrant()

	// 初始化RAG
	rag.GetRAG()

	// 启动服务，监听 8080 端口
	app.Router.Run()
}
