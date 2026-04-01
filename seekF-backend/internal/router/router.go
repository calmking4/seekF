package router

import (
	"seekF-backend/internal/controllers/user"
	"seekF-backend/internal/middlewares"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	authController *user.AuthController,
	userInfoController *user.UserInfoController,
	groupController *user.GroupController,
	contactController *user.ContactController,
	sessionController *user.SessionController,
	messageController *user.MessageController,
	fileController *user.FileController,
	wsController *user.WsController,
) *gin.Engine {
	r := gin.Default()

	// 访问路径http://localhost:8080/debug/pprof/
	pprof.Register(r)

	// 添加CORS中间件
	r.Use(middlewares.CORSMiddleware())

	// 用户端接口
	userGroup := r.Group("/user")
	{
		// 无需认证的接口
		userGroup.POST("/register", authController.Register)
		userGroup.POST("/login", authController.Login)
		userGroup.POST("/sendVerifyCode", authController.SendVerifyCode)
		userGroup.POST("/loginByCode", authController.LoginByCode)

		// 需要认证的接口
		protectedGroup := userGroup.Group("")
		protectedGroup.Use(middlewares.Auth())
		{
			protectedGroup.POST("/logout", authController.Logout)
			// 用户信息
			protectedGroup.POST("/userinfo/getUserinfo", userInfoController.GetUserInfo)
			protectedGroup.POST("/userinfo/getMyInfo", userInfoController.GetMyInfo)
			protectedGroup.POST("/userinfo/updateUserInfo", userInfoController.UpdateUserInfo)
			// 群组
			protectedGroup.POST("/group/createGroup", groupController.CreateGroup)
			protectedGroup.POST("/group/loadMyGroup", groupController.LoadMyGroup)
			protectedGroup.POST("/group/loadMyJoinedGroup", groupController.LoadMyJoinedGroup)
			protectedGroup.POST("/group/checkGroupAddMode", groupController.CheckGroupAddMode)
			protectedGroup.POST("/group/getGroupInfo", groupController.GetGroupInfo)
			protectedGroup.POST("/group/updateGroupInfo", groupController.UpdateGroupInfo)
			protectedGroup.POST("/group/getGroupMemberList", groupController.GetGroupMemberList)
			protectedGroup.POST("/group/removeGroupMembers", groupController.RemoveGroupMembers)
			protectedGroup.POST("/group/enterGroupDirectly", groupController.EnterGroupDirectly)
			protectedGroup.POST("/group/leaveGroup", groupController.LeaveGroup)
			protectedGroup.POST("/group/dismissGroup", groupController.DismissGroup)
			protectedGroup.POST("/group/searchGroups", groupController.SearchGroups)
			// 联系人
			protectedGroup.POST("/contact/getUserList", contactController.GetUserList)
			protectedGroup.POST("/contact/getContactInfo", contactController.GetContactInfo)
			protectedGroup.POST("/contact/deleteContact", contactController.DeleteContact)
			protectedGroup.POST("/contact/applyContact", contactController.ApplyContact)
			protectedGroup.POST("/contact/getNewContactList", contactController.GetNewContactList)
			protectedGroup.POST("/contact/passContactApply", contactController.PassContactApply)
			protectedGroup.POST("/contact/blackContact", contactController.BlackContact)
			protectedGroup.POST("/contact/cancelBlackContact", contactController.CancelBlackContact)
			protectedGroup.POST("/contact/getApplyGroupList", contactController.GetApplyGroupList)
			protectedGroup.POST("/contact/refuseContactApply", contactController.RefuseContactApply)
			protectedGroup.POST("/contact/blackApply", contactController.BlackApply)
			protectedGroup.POST("/contact/searchUsers", contactController.SearchUsers)
			protectedGroup.POST("/contact/getMyApplyList", contactController.GetMyApplyList)
			// 会话
			protectedGroup.POST("/session/openSession", sessionController.OpenSession)
			protectedGroup.POST("/session/getSessionList", sessionController.GetSessionList)
			protectedGroup.POST("/session/deleteSession", sessionController.DeleteSession)
			protectedGroup.POST("/session/checkOpenSessionAllowed", sessionController.CheckOpenSessionAllowed)
			// 消息
			protectedGroup.POST("/message/getUserMessageList", messageController.GetUserMessageList)
			protectedGroup.POST("/message/getGroupMessageList", messageController.GetGroupMessageList)
			// 文件上传
			protectedGroup.POST("/file/upload", fileController.UploadFile)
			// WebSocket
			protectedGroup.GET("/ws/login", wsController.WsLogin)
			protectedGroup.POST("/ws/logout", wsController.WsLogout)
		}

	}

	return r
}
