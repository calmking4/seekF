package user

import (
	"errors"
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// FollowingUpdates 返回关注栏作者和有更新的作者数。
func (c *DiscoverController) FollowingUpdates(ctx *gin.Context) {
	result, err := c.discoverService.FollowingUpdates(ctx.Request.Context(), ctx.GetString("Uuid"))
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "获取关注作者更新成功", result)
}

// ReadFollowingUpdates 确认已成功查看指定作者快照内的更新。
func (c *DiscoverController) ReadFollowingUpdates(ctx *gin.Context) {
	var req userreq.FollowingReadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "作者更新参数错误", http.StatusBadRequest)
		return
	}
	if err := c.discoverService.ReadFollowingUpdates(ctx.Request.Context(), ctx.GetString("Uuid"), req.Token); err != nil {
		if errors.Is(err, userservice.ErrFollowingReadExpired) {
			resp.Error(ctx, err.Error(), http.StatusGone)
		} else {
			resp.Error(ctx, "同步作者更新状态失败，请稍后重试", http.StatusServiceUnavailable)
		}
		return
	}
	resp.Success(ctx, "已查看作者更新", nil)
}
