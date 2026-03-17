package main

import (
	usercontroller "seekF-backend/internal/controllers/user"
	userdao "seekF-backend/internal/dao/user_dao"
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

	// 初始化 Service 层
	authService := userservice.NewAuthService(userInfoDAO)
	userInfoService := userservice.NewUserInfoService(userInfoDAO)
	contactService := userservice.NewContactService(contactDAO, sessionDAO, userInfoDAO, groupDAO, contactApplyDAO)
	groupService := userservice.NewGroupService(groupDAO, contactDAO, sessionDAO, userInfoDAO, contactApplyDAO)
	sessionService := userservice.NewSessionService(sessionDAO, userInfoDAO, groupDAO, contactDAO)

	// 初始化 Controller 层
	authController := usercontroller.NewAuthController(authService)
	userInfoController := usercontroller.NewUserInfoController(userInfoService)
	contactController := usercontroller.NewContactController(contactService)
	groupController := usercontroller.NewGroupController(groupService)
	sessionController := usercontroller.NewSessionController(sessionService)

	// 初始化路由器
	r := router.SetupRouter(authController, userInfoController, groupController, contactController, sessionController)

	//启动服务，监听 8080 端口
	r.Run()
}
