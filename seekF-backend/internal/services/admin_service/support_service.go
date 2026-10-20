package adminservice

import (
	"context"
	"fmt"
	"strings"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	adminresp "seekF-backend/internal/dto/admin"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/db"
	userservice "seekF-backend/internal/services/user_service"
)

// SupportSettlement 查询或调度资金核对，业务层再次检查管理员资格。
func (s *AdminService) SupportSettlement(ctx context.Context, adminId, action, no string, id int64, page int) (any, error) {
	return userservice.AdminSupportSettlement(ctx, adminId, action, no, id, page)
}

// ListSupportOrders 分页查询当前环境的订单及退款申请。
func (s *AdminService) ListSupportOrders(ctx context.Context, orderNo string, status *int8, page int) (any, error) {
	if page < 1 {
		page = 1
	}
	if page > 100000 || len(orderNo) > 64 || (status != nil && (*status < 0 || *status > 5)) {
		return nil, fmt.Errorf("查询参数无效")
	}
	rows, total, err := userdao.NewSupportDAO(db.GormDB.WithContext(ctx)).ListAdminOrders(configs.GetConfig().AlipayConfig.Environment, strings.TrimSpace(orderNo), status, page)
	if err != nil {
		return nil, fmt.Errorf("查询助力订单失败: %w", err)
	}
	result := make([]adminresp.SupportOrder, 0, len(rows))
	for _, row := range rows {
		result = append(result, adminresp.SupportOrder{SupportOrderRespond: &userresp.SupportOrderRespond{OrderNo: row.OrderNo, CreatorId: row.CreatorId, Amount: row.Amount.StringFixed(2), DurationHours: row.DurationHours, Status: row.Status, RefundStatus: row.RefundStatus, PaidAt: row.PaidAt, ConfirmedAt: row.ConfirmedAt, CreatedAt: row.CreatedAt, RefundRequestedAt: row.RefundRequestedAt, RefundReason: row.RefundReason, RefundReviewNote: row.RefundReviewNote, RefundReviewedAt: row.RefundReviewedAt, RefundedAt: row.RefundedAt}, UserId: row.UserId, ReviewerId: row.RefundReviewerId, LastError: row.LastError})
	}
	return map[string]any{"list": result, "total": total}, nil
}

// ReviewSupportRefund 记录审核操作者及说明，异步处理渠道退款。
func (s *AdminService) ReviewSupportRefund(ctx context.Context, adminId, orderNo, action, note string) error {
	return userservice.ReviewSupportRefund(ctx, adminId, orderNo, action, note)
}
