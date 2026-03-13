package router

import (
	"seekF-backend/internal/controllers/user"
	"seekF-backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	authController *user.AuthController,
	userInfoController *user.UserInfoController,
	groupController *user.GroupController,
	contactController *user.ContactController,
) *gin.Engine {
	r := gin.Default()

	// 添加CORS中间件
	r.Use(middlewares.CORSMiddleware())

	// 不需要认证的公共接口
	publicGroup := r.Group("/user")
	{
		publicGroup.POST("/register", authController.Register)
		publicGroup.POST("/login", authController.Login)
	}

	// 需要认证的接口
	protectedGroup := r.Group("/user")
	protectedGroup.Use(middlewares.JWTAuth())
	{
		protectedGroup.POST("/logout", authController.Logout)
		protectedGroup.POST("/userinfo/getUserinfo", userInfoController.GetUserInfo)
		protectedGroup.POST("/userinfo/updateUserInfo", userInfoController.UpdateUserInfo)
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
	}

	return r
}
