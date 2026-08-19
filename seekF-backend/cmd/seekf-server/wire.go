//go:build wireinject
// +build wireinject

package main

import (
	usercontroller "seekF-backend/internal/controllers/user"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/router"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/google/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DAO ProviderSet — 所有数据访问层依赖
var daoProviderSet = wire.NewSet(
	userdao.NewUserInfoDAO,
	userdao.NewContactDAO,
	userdao.NewSessionDAO,
	userdao.NewGroupDAO,
	userdao.NewContactApplyDAO,
	userdao.NewMessageDAO,
	userdao.NewKnowledgeDAO,
	userdao.NewDiscoverDAO,
)

// ServiceProviderSet — 所有业务逻辑层依赖
var serviceProviderSet = wire.NewSet(
	userservice.NewAuthService,
	userservice.NewUserInfoService,
	userservice.NewContactService,
	userservice.NewGroupService,
	userservice.NewSessionService,
	userservice.NewMessageService,
	userservice.NewFileService,
	userservice.NewAIChatService,
	userservice.NewKnowledgeService,
	userservice.NewDiscoverService,
)

// controllerProviderSet — 所有控制器依赖
var controllerProviderSet = wire.NewSet(
	usercontroller.NewAuthController,
	usercontroller.NewUserInfoController,
	usercontroller.NewContactController,
	usercontroller.NewGroupController,
	usercontroller.NewSessionController,
	usercontroller.NewMessageController,
	usercontroller.NewFileController,
	usercontroller.NewWsController,
	usercontroller.NewAIChatController,
	usercontroller.NewKnowledgeController,
	usercontroller.NewDiscoverController,
)

// App 包含所有需要暴露的依赖
// 全局单例（websocket、ai 等）需要通过 App 暴露给 main.go 使用
type App struct {
	Router         *gin.Engine
	SessionService userservice.SessionService
	MessageDAO     userdao.MessageDAO
	SessionDAO     userdao.SessionDAO
	GroupDAO       userdao.GroupDAO
	DiscoverDAO    userdao.DiscoverDAO
	UserInfoDAO    userdao.UserInfoDAO
}

// initApp 是 Wire 注入器函数
// Wire 会在编译期分析依赖关系，自动生成完整的初始化代码
func initApp(db *gorm.DB) App {
	wire.Build(
		daoProviderSet,
		serviceProviderSet,
		controllerProviderSet,
		router.SetupRouter,
		wire.Struct(new(App), "*"),
	)
	return App{}
}
