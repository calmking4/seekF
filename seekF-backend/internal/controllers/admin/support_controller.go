package admin

import (
	"net/http"

	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

// SupportWithdrawals 查询提现单和渠道核对状态。
func (a *AdminController) SupportWithdrawals(c *gin.Context) { a.supportSettlement(c, "withdrawals") }

// SupportAccounts 查询账户金额与本地差异。
func (a *AdminController) SupportAccounts(c *gin.Context) { a.supportSettlement(c, "accounts") }

// ReconcileSupportSettlement 调度原单查询或核对账户，不创建新出款。
func (a *AdminController) ReconcileSupportSettlement(c *gin.Context) {
	a.supportSettlement(c, "reconcile")
}

// supportSettlement 绑定有限查询参数并从已认证上下文提取管理员。
func (a *AdminController) supportSettlement(c *gin.Context, action string) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var req struct {
		WithdrawNo string `json:"withdraw_no"`
		AccountID  int64  `json:"account_id"`
		Page       int    `json:"page"`
	}
	if c.ShouldBindJSON(&req) != nil {
		resp.Error(c, "资金核对参数错误", 400)
		return
	}
	identity, ok := c.Get("admin_identity")
	admin, valid := identity.(adminresp.Identity)
	if !ok || !valid {
		resp.Error(c, "请先登录管理端", 401)
		return
	}
	result, err := a.service.SupportSettlement(c.Request.Context(), admin.UUID, action, req.WithdrawNo, req.AccountID, req.Page)
	if err != nil {
		resp.Error(c, err.Error(), 400)
		return
	}
	resp.Success(c, "资金核对操作成功", result)
}

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
