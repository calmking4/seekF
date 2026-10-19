package admin

import (
	"net/http"

	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

// ListSupportOrders 分页查询当前环境的订单及退款申请。
func (a *AdminController) ListSupportOrders(c *gin.Context) {
	var req struct {
		OrderNo      string `json:"order_no"`
		RefundStatus *int8  `json:"refund_status"`
		Page         int    `json:"page"`
	}
	if c.ShouldBindJSON(&req) != nil {
		resp.Error(c, "订单查询参数错误", 400)
		return
	}
	result, err := a.service.ListSupportOrders(c.Request.Context(), req.OrderNo, req.RefundStatus, req.Page)
	if err != nil {
		resp.Error(c, err.Error(), 400)
		return
	}
	resp.Success(c, "查询助力订单成功", result)
}

// ReviewSupportRefund 记录审核操作者及说明，异步处理渠道退款。
func (a *AdminController) ReviewSupportRefund(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var req struct {
		OrderNo string `json:"order_no"`
		Action  string `json:"action"`
		Note    string `json:"note"`
	}
	if c.ShouldBindJSON(&req) != nil {
		resp.Error(c, "退款审核参数错误", 400)
		return
	}
	identity, ok := c.Get("admin_identity")
	admin, valid := identity.(adminresp.Identity)
	if !ok || !valid {
		resp.Error(c, "请先登录管理端", 401)
		return
	}
	if err := a.service.ReviewSupportRefund(c.Request.Context(), admin.UUID, req.OrderNo, req.Action, req.Note); err != nil {
		resp.Error(c, err.Error(), 400)
		return
	}
	resp.Success(c, "退款审核操作已保存，请刷新查看渠道结果", nil)
}
