package router

import (
	"seekF-backend/internal/controllers/user"
	"seekF-backend/internal/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 添加CORS中间件
	r.Use(middlewares.CORSMiddleware())

	// 不需要认证的公共接口
	publicGroup := r.Group("/user")
	{
		publicGroup.POST("/register", user.Register)
		publicGroup.POST("/login", user.Login)
	}

	// 需要认证的接口
	protectedGroup := r.Group("/user")
	protectedGroup.Use(middlewares.JWTAuth())
	{
		protectedGroup.POST("/logout", user.Logout)
		protectedGroup.POST("/userinfo/getUserinfo", user.GetUserInfo)
		protectedGroup.POST("/userinfo/updateUserInfo", user.UpdateUserInfo)
		protectedGroup.POST("/group/createGroup", user.CreateGroup)
		protectedGroup.POST("/group/loadMyGroup", user.LoadMyGroup)
		protectedGroup.POST("/group/checkGroupAddMode", user.CheckGroupAddMode)
		protectedGroup.POST("/group/getGroupInfo", user.GetGroupInfo)
		protectedGroup.POST("/group/updateGroupInfo", user.UpdateGroupInfo)
		protectedGroup.POST("/group/getGroupMemberList", user.GetGroupMemberList)
		protectedGroup.POST("/group/removeGroupMembers", user.RemoveGroupMembers)
		protectedGroup.POST("/group/enterGroupDirectly", user.EnterGroupDirectly)
		protectedGroup.POST("/group/leaveGroup", user.LeaveGroup)
		protectedGroup.POST("/group/dismissGroup", user.DismissGroup)
	}

	return r
}
