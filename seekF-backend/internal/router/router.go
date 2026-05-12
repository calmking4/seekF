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
	aichatController *user.AIChatController,
	knowledgeController *user.KnowledgeController,
	discoverController *user.DiscoverController,
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
			// AI Chat
			protectedGroup.POST("/aichat/createSession", aichatController.CreateSession)
			protectedGroup.POST("/aichat/getSessionList", aichatController.GetSessionList)
			protectedGroup.POST("/aichat/getMessageHistory", aichatController.GetMessageHistory)
			protectedGroup.POST("/aichat/sendMessage", aichatController.SendMessage)
			protectedGroup.POST("/aichat/deleteSession", aichatController.DeleteSession)
			// Knowledge Base
			protectedGroup.POST("/knowledge/add", knowledgeController.AddDocument)
			protectedGroup.POST("/knowledge/list", knowledgeController.ListDocuments)
			protectedGroup.POST("/knowledge/remove", knowledgeController.RemoveDocument)
			protectedGroup.POST("/knowledge/content", knowledgeController.GetDocumentContent)
			// Discover
			protectedGroup.POST("/discover/create", discoverController.CreatePost)
			protectedGroup.POST("/discover/list", discoverController.ListPosts)
			protectedGroup.POST("/discover/liked-list", discoverController.ListLikedPosts)
			protectedGroup.POST("/discover/detail", discoverController.GetPostDetail)
			protectedGroup.POST("/discover/like", discoverController.ToggleLike)
			protectedGroup.POST("/discover/comment/add", discoverController.AddComment)
			protectedGroup.POST("/discover/comment/list", discoverController.ListComments)
			protectedGroup.POST("/discover/comment/like", discoverController.ToggleCommentLike)
			protectedGroup.POST("/discover/comment/ai", discoverController.AddAIComment)
			// 收藏夹
			protectedGroup.POST("/discover/folder/create", discoverController.CreateFolder)
			protectedGroup.POST("/discover/folder/update", discoverController.UpdateFolder)
			protectedGroup.POST("/discover/folder/delete", discoverController.DeleteFolder)
			protectedGroup.POST("/discover/folder/list", discoverController.ListFolders)
			protectedGroup.POST("/discover/folder/detail", discoverController.GetFolderDetail)
			// 收藏
			protectedGroup.POST("/discover/collect", discoverController.CollectPost)
			protectedGroup.POST("/discover/uncollect", discoverController.UncollectPost)
			protectedGroup.POST("/discover/check-collected", discoverController.CheckCollected)
			protectedGroup.POST("/discover/collected-list", discoverController.ListCollectedPosts)
		}

	}

	return r
}
