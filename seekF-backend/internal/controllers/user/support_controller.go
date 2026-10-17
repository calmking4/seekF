package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type SupportController struct{ supportService userservice.SupportService }

func NewSupportController(supportService userservice.SupportService) *SupportController {
	return &SupportController{supportService: supportService}
}

// GetCreator 返回创作者配置与当前用户权益。
func (c *SupportController) GetCreator(ctx *gin.Context) {
	var req userreq.SupportCreatorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "创作者参数错误", http.StatusBadRequest)
		return
	}
	result, err := c.supportService.GetCreator(ctx.Request.Context(), ctx.GetString("Uuid"), req.CreatorId)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "获取助力设置成功", result)
}

// SaveActivate 只保存当前登录用户的助力开通及价格。
func (c *SupportController) SaveActivate(ctx *gin.Context) {
	var req userreq.SaveSupportActivateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "助力价格必须为5至100整数元", http.StatusBadRequest)
		return
	}
	result, err := c.supportService.SaveActivate(ctx.Request.Context(), ctx.GetString("Uuid"), req.PriceYuan)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "助力设置已保存", result)
}
