package main

import (
	"context"
	"seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/ai/rag"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/websocket"
	userservice "seekF-backend/internal/services/user_service"
)

// main 初始化依赖与后台任务并启动HTTP服务。
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
		app.ContactDAO,
	)
	go websocket.ChatServer.Start()

	// 初始化AI模型池
	ai.GetModelPool()

	// 启动AI消息Kafka消费者
	ai.StartAIConsumer()

	// 初始化并启动AI评论回复Kafka消费者
	ai.InitAICommentConsumer(app.DiscoverDAO, app.UserInfoDAO)
	ai.StartAICommentConsumer()

	// 初始化并启动发现页Kafka消费者
	userservice.InitDiscoverConsumer(app.ESDiscoverDAO)
	userservice.StartDiscoverConsumer()

	// 启动发现页Outbox投递器
	userservice.StartDiscoverPublisher()

	// 初始化并启动关注分发Kafka消费者，新帖分发事务完成后提交位点。
	userservice.InitDiscoverFeedConsumer(app.DiscoverDAO)
	userservice.StartDiscoverFeedConsumer()

	// 启动关注分发投递器与收件箱清理。
	userservice.StartDiscoverFeedPublisher()

	// 初始化Qdrant
	db.InitQdrant()

	// 初始化RAG
	rag.GetRAG()

	// 定期清理未被作品引用的过期上传资源。
	userservice.StartDiscoverAssetCleanup(context.Background())

	// 启动助力订单持久化查单与关单补偿，每分钟领取到期任务。
	userservice.StartSupportOrderScanner(context.Background())

	// 启动服务，监听 8080 端口
	app.Router.Run()
}
