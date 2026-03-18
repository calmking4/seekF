package user

import (
	"net/http"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type UserInfoController struct {
	userInfoService userservice.UserInfoService
}

func NewUserInfoController(userInfoService userservice.UserInfoService) *UserInfoController {
	return &UserInfoController{
		userInfoService: userInfoService,
	}
}

// GetUserInfo 获取用户信息
func (c *UserInfoController) GetUserInfo(ctx *gin.Context) {
	var req userreq.GetUserInfoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("GetUserInfo err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	result, err := c.userInfoService.GetUserInfo(&req)
	if err != nil {
		zlog.Info("GetUserInfo service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取用户信息成功", result)
}

// GetMyInfo 获取当前登录用户的信息
func (c *UserInfoController) GetMyInfo(ctx *gin.Context) {
	// 从上下文获取当前用户的 UUID
	currentUserUuid, exists := ctx.Get("Uuid")
	if !exists {
		zlog.Info("GetMyInfo err: Unable to get user UUID from context")
		resp.Error(ctx, "无法获取用户信息", http.StatusInternalServerError)
		return
	}

	// 创建请求对象
	req := &userreq.GetUserInfoRequest{
		Uuid: currentUserUuid.(string),
	}

	result, err := c.userInfoService.GetUserInfo(req)
	if err != nil {
		zlog.Info("GetMyInfo service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取用户信息成功", result)
}

// UpdateUserInfo 更新用户信息
func (c *UserInfoController) UpdateUserInfo(ctx *gin.Context) {
	var req userreq.UpdateUserInfoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Info("UpdateUserInfo err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户的 UUID
	currentUserUuid, exists := ctx.Get("Uuid")
	if !exists {
		zlog.Info("UpdateUserInfo err: Unable to get user UUID from context")
		resp.Error(ctx, "无法获取用户信息", http.StatusInternalServerError)
		return
	}

	// 验证用户只能更新自己的信息
	if req.Uuid != currentUserUuid.(string) {
		zlog.Info("UpdateUserInfo err: User can only update their own info")
		resp.Error(ctx, "只能更新自己的用户信息", http.StatusForbidden)
		return
	}

	err := c.userInfoService.UpdateUserInfo(&req)
	if err != nil {
		zlog.Info("UpdateUserInfo service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "更新用户信息成功", nil)
}
