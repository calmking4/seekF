package user

import (
	"net/http"

	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// BlockController 提供用户拉黑和黑名单管理接口。
type BlockController struct{ service *userservice.BlockService }

// NewBlockController 构造黑名单控制器。
func NewBlockController(service *userservice.BlockService) *BlockController {
	return &BlockController{service: service}
}

// Set 明确设置本人对目标的拉黑状态，重复请求保持幂等。
func (c *BlockController) Set(ctx *gin.Context) {
	var input struct {
		Target  string `json:"target_user_id" binding:"required"`
		Blocked *bool  `json:"blocked" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Error(ctx, "拉黑参数错误", http.StatusBadRequest)
		return
	}
	if err := c.service.Set(ctx.Request.Context(), ctx.GetString("Uuid"), input.Target, *input.Blocked); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "设置成功", gin.H{"is_blocked": *input.Blocked})
}

// Status 查询本人及对方设置的拉黑状态。
func (c *BlockController) Status(ctx *gin.Context) {
	var input struct {
		Target string `json:"target_user_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Error(ctx, "用户参数错误", http.StatusBadRequest)
		return
	}
	blocked, by, err := c.service.Status(ctx.Request.Context(), ctx.GetString("Uuid"), input.Target)
	if err != nil {
		zlog.Error("查询拉黑状态失败，用户=" + ctx.GetString("Uuid") + ": " + err.Error())
		resp.Error(ctx, "查询拉黑状态失败", http.StatusInternalServerError)
		return
	}
	resp.Success(ctx, "查询成功", gin.H{"is_blocked": blocked, "is_blocked_by": by})
}

// List 分页读取本人设置的黑名单。
func (c *BlockController) List(ctx *gin.Context) {
	var input struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Error(ctx, "分页参数错误", http.StatusBadRequest)
		return
	}
	rows, total, err := c.service.List(ctx.Request.Context(), ctx.GetString("Uuid"), input.Page, input.PageSize)
	if err != nil {
		zlog.Error("查询黑名单失败，用户=" + ctx.GetString("Uuid") + ": " + err.Error())
		resp.Error(ctx, "查询黑名单失败", http.StatusInternalServerError)
		return
	}
	resp.Success(ctx, "查询成功", gin.H{"list": rows, "total": total})
}
