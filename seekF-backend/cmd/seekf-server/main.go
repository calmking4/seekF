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

	// 初始化其他业务Kafka与直接推送的WebSocket服务器
	kafka.KafkaServiceInstance.Init()
	websocket.ChatServer = websocket.NewServer(
		app.SessionService,
	)
	go websocket.ChatServer.Start()

	// 初始化AI模型池
	ai.GetModelPool()

	// 初始化并启动AI评论回复Kafka消费者
	ai.InitAICommentConsumer(app.DiscoverDAO, app.UserInfoDAO)
	ai.StartAICommentConsumer()

	// 初始化并启动发现页Kafka消费者
	userservice.InitDiscoverConsumer(app.ESDiscoverDAO)
	userservice.StartDiscoverConsumer()

	// 启动发现页Outbox投递器
	userservice.StartDiscoverPublisher()

	// 初始化并启动关注分发Kafka消费者，Redis写入及事件状态提交完成后提交位点。
	userservice.InitDiscoverFeedConsumer()
	userservice.StartDiscoverFeedConsumer()

	// 启动关注分发Outbox投递器，Redis收件箱使用TTL自动过期。
	userservice.StartDiscoverFeedPublisher()

	// 初始化Qdrant
	db.InitQdrant()

	// 初始化RAG
	rag.GetRAG()

	// 定期清理未被作品引用的过期上传资源。
	userservice.StartDiscoverAssetCleanup(context.Background())

	// 启动助力订单查单、关单与退款补偿，并分批核对每日渠道退款账单。
	userservice.StartSupportOrderScanner(context.Background())

	// 启动服务，监听 8080 端口
	app.Router.Run()
}
