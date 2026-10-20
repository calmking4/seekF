package user

import (
	"net/http"
	"seekF-backend/internal/pkg/zlog"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type SupportController struct{ supportService userservice.SupportService }

// SettlementSummary 查询登录创作者的收益账户，不接受客户端指定账户归属。
func (c *SupportController) SettlementSummary(ctx *gin.Context) {
	result, err := c.supportService.ListSupport(ctx.Request.Context(), ctx.GetString("Uuid"), "balance", 1)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "查询收益账户成功", result)
}

// SettlementLedger 分页查询登录创作者的收益流水。
func (c *SupportController) SettlementLedger(ctx *gin.Context) {
	var req struct {
		Page int `json:"page"`
	}
	if ctx.ShouldBindJSON(&req) != nil {
		resp.Error(ctx, "流水分页参数错误", http.StatusBadRequest)
		return
	}
	result, err := c.supportService.ListSupport(ctx.Request.Context(), ctx.GetString("Uuid"), "ledger", req.Page)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "查询收益流水成功", result)
}

// NewSupportController 创建创作者助力控制器。
func NewSupportController(supportService userservice.SupportService) *SupportController {
	return &SupportController{supportService: supportService}
}

// CreateOrder 为当前购买者创建或复用订单。
func (c *SupportController) CreateOrder(ctx *gin.Context) {
	var req userreq.SupportOrderRequest
	if ctx.ShouldBindJSON(&req) != nil {
		resp.Error(ctx, "订单参数错误", http.StatusBadRequest)
		return
	}
	result, err := c.supportService.CreateOrder(ctx.Request.Context(), ctx.GetString("Uuid"), req.CreatorId, req.RequestKey)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "获取助力订单成功", result)
}

// GetOrder 查询当前购买者的订单。
func (c *SupportController) GetOrder(ctx *gin.Context) {
	var req userreq.SupportOrderRequest
	if ctx.ShouldBindJSON(&req) != nil || req.OrderNo == "" {
		resp.Error(ctx, "订单号不能为空", http.StatusBadRequest)
		return
	}
	result, err := c.supportService.GetOrder(ctx.Request.Context(), ctx.GetString("Uuid"), req.OrderNo)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "查询助力订单成功", result)
}

// PayOrder 返回同一订单的支付宝网页和内嵌扫码收银台地址。
func (c *SupportController) PayOrder(ctx *gin.Context) {
	var req userreq.SupportOrderRequest
	if ctx.ShouldBindJSON(&req) != nil || req.OrderNo == "" {
		resp.Error(ctx, "订单号不能为空", http.StatusBadRequest)
		return
	}
	target, err := c.supportService.PayOrder(ctx.Request.Context(), ctx.GetString("Uuid"), req.OrderNo)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "获取支付地址成功", target)
}

// ListOrders 分页返回本人已确认付款的订单。
func (c *SupportController) ListOrders(ctx *gin.Context) { c.listSupport(ctx, "orders") }

// ListReceivedOrders 分页返回当前创作者收到的已付款助力记录。
func (c *SupportController) ListReceivedOrders(ctx *gin.Context) { c.listSupport(ctx, "received") }

// ListEntitlements 分页返回本人的权益。
func (c *SupportController) ListEntitlements(ctx *gin.Context) { c.listSupport(ctx, "entitlements") }

// GetIncome 返回当前创作者的实收、退款和净额。
func (c *SupportController) GetIncome(ctx *gin.Context) { c.listSupport(ctx, "income") }

// listSupport 统一绑定分页参数并限定当前用户查询范围。
func (c *SupportController) listSupport(ctx *gin.Context, kind string) {
	var req userreq.SupportOrderRequest
	if ctx.ShouldBindJSON(&req) != nil {
		resp.Error(ctx, "分页参数错误", http.StatusBadRequest)
		return
	}
	result, err := c.supportService.ListSupport(ctx.Request.Context(), ctx.GetString("Uuid"), kind, req.Page)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	resp.Success(ctx, "查询助力数据成功", result)
}

// AlipayNotify 使用独立公开入口验签，仅在业务事务提交后回复支付宝成功。
func (c *SupportController) AlipayNotify(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 64<<10)
	if ctx.Request.ParseForm() != nil {
		ctx.String(http.StatusBadRequest, "failure")
		return
	}
	if err := c.supportService.Notify(ctx.Request.Context(), ctx.Request.PostForm); err != nil {
		orderNo := ctx.Request.PostForm.Get("out_trade_no")
		if len(orderNo) > 64 {
			orderNo = orderNo[:64]
		}
		zlog.Error("处理支付宝助力通知失败，订单号=" + orderNo + ": " + err.Error())
		ctx.String(http.StatusBadRequest, "failure")
		return
	}
	ctx.String(http.StatusOK, "success")
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

// RequestRefund 受理当前购买者的首次退款申请。
func (c *SupportController) RequestRefund(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 8192)
	var req struct {
		OrderNo string `json:"order_no"`
		Reason  string `json:"reason"`
	}
	if ctx.ShouldBindJSON(&req) != nil {
		resp.Error(ctx, "退款申请参数错误", 400)
		return
	}
	if err := c.supportService.RequestRefund(ctx.Request.Context(), ctx.GetString("Uuid"), req.OrderNo, req.Reason); err != nil {
		resp.Error(ctx, err.Error(), 400)
		return
	}
	resp.Success(ctx, "退款申请已提交，请等待管理员审核", nil)
}
