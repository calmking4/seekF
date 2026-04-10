package main

import (
	usercontroller "seekF-backend/internal/controllers/user"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/websocket"
	"seekF-backend/internal/router"
	userservice "seekF-backend/internal/services/user_service"
)

func main() {
	// 初始化 DAO 层
	userInfoDAO := userdao.NewUserInfoDAO()
	contactDAO := userdao.NewContactDAO()
	sessionDAO := userdao.NewSessionDAO()
	groupDAO := userdao.NewGroupDAO()
	contactApplyDAO := userdao.NewContactApplyDAO()
	messageDAO := userdao.NewMessageDAO()

	// 初始化 Service 层
	authService := userservice.NewAuthService(userInfoDAO)
	userInfoService := userservice.NewUserInfoService(userInfoDAO)
	contactService := userservice.NewContactService(contactDAO, sessionDAO, userInfoDAO, groupDAO, contactApplyDAO)
	groupService := userservice.NewGroupService(groupDAO, contactDAO, sessionDAO, userInfoDAO, contactApplyDAO)
	sessionService := userservice.NewSessionService(sessionDAO, userInfoDAO, groupDAO, contactDAO)
	messageService := userservice.NewMessageService(messageDAO)
	fileService := userservice.NewFileService()

	// 初始化 AI Service 层（复用 sessionDAO 和 messageDAO）
	aiChatService := userservice.NewAIChatService(sessionDAO, messageDAO, userInfoDAO)

	// 初始化 Controller 层
	authController := usercontroller.NewAuthController(authService)
	userInfoController := usercontroller.NewUserInfoController(userInfoService)
	contactController := usercontroller.NewContactController(contactService)
	groupController := usercontroller.NewGroupController(groupService)
	sessionController := usercontroller.NewSessionController(sessionService)
	messageController := usercontroller.NewMessageController(messageService)
	fileController := usercontroller.NewFileController(fileService)
	wsController := usercontroller.NewWsController()
	aichatController := usercontroller.NewAIChatController(aiChatService, fileService)

	// 初始化Kafka并启动WebSocket服务器
	kafka.KafkaServiceInstance.Init()
	websocket.ChatServer = websocket.NewServer(sessionService, messageDAO, sessionDAO, groupDAO)
	go websocket.ChatServer.Start()

	// 初始化AI模型池
	ai.GetModelPool()

	// 启动AI消息Kafka消费者
	go ai.StartAIConsumer()

	// 初始化路由器
	r := router.SetupRouter(authController, userInfoController, groupController, contactController, sessionController, messageController, fileController, wsController, aichatController)

	//启动服务，监听 8080 端口
	r.Run()
}
