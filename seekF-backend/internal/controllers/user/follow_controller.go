package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type FollowController struct {
	followService userservice.FollowService
}

func NewFollowController(followService userservice.FollowService) *FollowController {
	return &FollowController{
		followService: followService,
	}
}

// ToggleFollow 关注/取消关注
func (c *FollowController) ToggleFollow(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.ToggleFollowRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.FollowUserId == "" {
		resp.Error(ctx, "关注目标不能为空", http.StatusBadRequest)
		return
	}
	if req.FollowUserId == userId {
		resp.Error(ctx, "不能关注自己", http.StatusBadRequest)
		return
	}

	isFollowed, err := c.followService.ToggleFollow(ctx.Request.Context(), userId, req.FollowUserId)
	if err != nil {
		zlog.Error("关注操作失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	if isFollowed {
		resp.Success(ctx, "关注成功", gin.H{"is_followed": true})
	} else {
		resp.Success(ctx, "取消关注", gin.H{"is_followed": false})
	}
}

// GetFollowCounts 获取关注/粉丝数
func (c *FollowController) GetFollowCounts(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req struct {
		UserId string `json:"user_id" form:"user_id"`
	}
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	// 如果未指定用户ID，默认查自己
	if req.UserId == "" {
		req.UserId = userId
	}

	following, followers, err := c.followService.GetFollowCounts(ctx.Request.Context(), req.UserId)
	if err != nil {
		zlog.Error("获取关注粉丝数失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查是否关注了该用户
	isFollowing := false
	if userId != req.UserId {
		isFollowing, _ = c.followService.IsFollowing(ctx.Request.Context(), userId, req.UserId)
	}

	resp.Success(ctx, "获取成功", gin.H{
		"following":    following,
		"followers":    followers,
		"is_following": isFollowing,
	})
}

// ListFollowing 关注列表
func (c *FollowController) ListFollowing(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.ListFollowRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.UserId == "" {
		req.UserId = userId
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 20
	}

	users, total, err := c.followService.ListFollowing(ctx.Request.Context(), req.UserId, req.Page, req.PageSize)
	if err != nil {
		zlog.Error("获取关注列表失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	var items []userresp.FollowUserItem
	for _, u := range users {
		items = append(items, userresp.FollowUserItem{
			Uuid:      u.Uuid,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Signature: u.Signature,
			IsFriend:  u.IsFriend,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListFollowRespond{
		List:  items,
		Total: total,
	})
}

// ListFollowers 粉丝列表
func (c *FollowController) ListFollowers(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.ListFollowRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.UserId == "" {
		req.UserId = userId
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 20
	}

	users, total, err := c.followService.ListFollowers(ctx.Request.Context(), req.UserId, req.Page, req.PageSize)
	if err != nil {
		zlog.Error("获取粉丝列表失败: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	var items []userresp.FollowUserItem
	for _, u := range users {
		items = append(items, userresp.FollowUserItem{
			Uuid:      u.Uuid,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Signature: u.Signature,
			IsFriend:  u.IsFriend,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListFollowRespond{
		List:  items,
		Total: total,
	})
}
