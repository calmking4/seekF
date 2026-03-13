package main

import (
	usercontroller "seekF-backend/internal/controllers/user"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/router"
	userservice "seekF-backend/internal/services/user_service"
)

func main() {
	// 初始化 DAO 层
	authDAO := userdao.NewAuthDAO()
	userInfoDAO := userdao.NewUserInfoDAO()
	contactDAO := userdao.NewContactDAO()
	sessionDAO := userdao.NewSessionDAO()
	groupDAO := userdao.NewGroupDAO()

	// 初始化 Service 层
	authService := userservice.NewAuthService(authDAO)
	userInfoService := userservice.NewUserInfoService(userInfoDAO)
	contactService := userservice.NewContactService(contactDAO, sessionDAO, userInfoDAO)
	groupService := userservice.NewGroupService(groupDAO, contactDAO, sessionDAO, userInfoDAO)

	// 初始化 Controller 层
	authController := usercontroller.NewAuthController(authService)
	userInfoController := usercontroller.NewUserInfoController(userInfoService)
	contactController := usercontroller.NewContactController(contactService)
	groupController := usercontroller.NewGroupController(groupService)

	// 初始化路由器
	r := router.SetupRouter(authController, userInfoController, groupController, contactController)

	//启动服务，监听 8080 端口
	r.Run()
}
