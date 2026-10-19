package userservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/payment"
	"seekF-backend/internal/pkg/zlog"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ErrDiscoverSupportRequired 表示当前用户需要助力后才能访问作品详情及评论。
var ErrDiscoverSupportRequired = errors.New("助力创作者后才能查看此作品")

// requireDiscoverSupport 使用当前业务连接校验作品权益，事务内调用必须传入事务连接。
func requireDiscoverSupport(ctx context.Context, conn *gorm.DB, userId string, post *models.DiscoverPost) error {
	if post == nil {
		return fmt.Errorf("作品不存在")
	}
	if post.SupportOnly == 0 || post.UserId == userId {
		return nil
	}
	rows, err := userdao.NewSupportDAO(conn.WithContext(ctx)).ListEntitlements(configs.GetConfig().AlipayConfig.Environment, userId, []string{post.UserId})
	if err != nil {
		return fmt.Errorf("查询助力权益失败: %w", err)
	}
	if len(rows) > 0 && rows[0].ExpiresAt != nil && rows[0].ExpiresAt.After(time.Now()) {
		return nil
	}
	return ErrDiscoverSupportRequired
}

// SupportService 创作者助力开通、支付及权益业务接口。
type SupportService interface {
	// 提交本人订单的首次退款申请。
	RequestRefund(ctx context.Context, userId, orderNo, reason string) error
	// 查询创作者的开通状态、价格和当前用户权益。
	GetCreator(ctx context.Context, userId, creatorId string) (*userresp.SupportCreatorRespond, error)
	// 为本人开通助力或修改价格。
	SaveActivate(ctx context.Context, userId string, priceYuan int) (*userresp.SupportCreatorRespond, error)
	// 创建或复用待支付订单。
	CreateOrder(ctx context.Context, userId, creatorId, requestKey string) (*userresp.SupportOrderRespond, error)
	// 查询本人订单的服务端确认结果。
	GetOrder(ctx context.Context, userId, orderNo string) (*userresp.SupportOrderRespond, error)
	// 为本人的有效待支付订单生成网页和扫码收银台地址。
	PayOrder(ctx context.Context, userId, orderNo string) (*userresp.SupportPaymentRespond, error)
	// 查询本人订单、权益、收到的助力或创作者收益。
	ListSupport(ctx context.Context, userId, kind string, page int) (any, error)
	// 验证并处理支付宝异步通知。
	Notify(ctx context.Context, values url.Values) error
}

// SupportServiceImpl 使用现有分层组织助力开通业务。
type SupportServiceImpl struct {
	supportDAO  userdao.SupportDAO
	userInfoDAO userdao.UserInfoDAO
}

// NewSupportService 创建创作者助力服务。
func NewSupportService(supportDAO userdao.SupportDAO, userInfoDAO userdao.UserInfoDAO) SupportService {
	return &SupportServiceImpl{supportDAO: supportDAO, userInfoDAO: userInfoDAO}
}

// GetCreator 查询配置及当前用户的权益，不将查看者状态写入共享缓存。
func (s *SupportServiceImpl) GetCreator(ctx context.Context, userId, creatorId string) (*userresp.SupportCreatorRespond, error) {
	if creatorId == "" {
		creatorId = userId
	}
	user, err := s.userInfoDAO.FindUserByUuid(creatorId)
	if err != nil {
		return nil, fmt.Errorf("查询创作者失败: %w", err)
	}
	if user == nil || user.Uuid == "" || user.Status != 0 {
		return nil, fmt.Errorf("创作者不存在或账号不可用")
	}
	activate, err := s.supportDAO.FindActivate(creatorId)
	if err != nil {
		return nil, fmt.Errorf("查询助力设置失败: %w", err)
	}
	result := &userresp.SupportCreatorRespond{CreatorId: creatorId, Nickname: user.Nickname, Avatar: user.Avatar, PriceYuan: constants.SupportDefaultPriceYuan}
	if activate != nil {
		result.Activated = true
		result.PriceYuan = activate.Price.IntPart()
	}
	rows, err := s.supportDAO.ListEntitlements(configs.GetConfig().AlipayConfig.Environment, userId, []string{creatorId})
	if err != nil {
		return nil, fmt.Errorf("查询助力权益失败: %w", err)
	}
	if len(rows) > 0 && rows[0].ExpiresAt != nil {
		result.ExpiresAt = rows[0].ExpiresAt.Format(time.RFC3339)
		result.HasSupport = rows[0].ExpiresAt.After(time.Now())
	}
	return result, nil
}

// SaveActivate 校验整数元价格并锁定本人账号，在同一事务内保存开通配置。
func (s *SupportServiceImpl) SaveActivate(ctx context.Context, userId string, priceYuan int) (*userresp.SupportCreatorRespond, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录")
	}
	if priceYuan < constants.SupportMinPriceYuan || priceYuan > constants.SupportMaxPriceYuan {
		return nil, fmt.Errorf("助力价格必须为5至100整数元")
	}
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := userdao.NewUserInfoDAO(tx).FindUserByUuidForUpdate(userId)
		if err != nil {
			return fmt.Errorf("查询创作者账号失败: %w", err)
		}
		if user == nil || user.Status != 0 {
			return fmt.Errorf("创作者账号不存在或不可用")
		}
		return userdao.NewSupportDAO(tx).SaveActivate(&models.SupportActivate{UserId: userId, Price: decimal.NewFromInt(int64(priceYuan))})
	})
	if err != nil {
		return nil, fmt.Errorf("保存助力设置失败: %w", err)
	}
	return s.GetCreator(ctx, userId, userId)
}

// discoverSupportExpiries 为当前页批量读取助力权益，避免逐帖查询。
func discoverSupportExpiries(ctx context.Context, userId string, posts []models.DiscoverPost) (map[string]*time.Time, error) {
	creators := make([]string, 0)
	seen := make(map[string]bool)
	for _, post := range posts {
		if post.SupportOnly == 1 && post.UserId != userId && !seen[post.UserId] {
			seen[post.UserId] = true
			creators = append(creators, post.UserId)
		}
	}
	rows, err := userdao.NewSupportDAO(db.GormDB.WithContext(ctx)).ListEntitlements(configs.GetConfig().AlipayConfig.Environment, userId, creators)
	if err != nil {
		return nil, fmt.Errorf("查询助力观看权益失败: %w", err)
	}
	expires := make(map[string]*time.Time, len(rows))
	for _, row := range rows {
		expires[row.CreatorId] = row.ExpiresAt
	}
	return expires, nil
}

// discoverPlaybackAccess 在作品可见权限之后判断图文或视频观看资格，返回可观看状态、原因及权益截止时间。
func discoverPlaybackAccess(post models.DiscoverPost, userId string, expires *time.Time, now time.Time) (bool, string, string) {
	expiresAt := ""
	if expires != nil {
		expiresAt = expires.Format(time.RFC3339)
	}
	if !post.DeletedAt.Valid && post.UserId == userId {
		return true, "", expiresAt
	}
	if !CanViewDiscoverPost(&post, userId) {
		return false, "作品当前不可查看", expiresAt
	}
	if post.SupportOnly == 0 || post.UserId == userId {
		return true, "", expiresAt
	}
	if expires != nil && expires.After(now) {
		return true, "", expiresAt
	}
	if expires != nil {
		return false, "助力权益已到期", expiresAt
	}
	return false, "助力创作者后可观看", expiresAt
}

// supportTransaction 对首次权益并发插入及数据库死锁执行有限重试，所有写入随事务回滚。
func supportTransaction(ctx context.Context, run func(*gorm.DB) error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = db.GormDB.WithContext(ctx).Transaction(run, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		var sqlErr *mysql.MySQLError
		if !errors.As(err, &sqlErr) || (sqlErr.Number != 1213 && sqlErr.Number != 1205 && sqlErr.Number != 1062) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 20 * time.Millisecond):
		}
	}
	return err
}

// supportOrderRespond 转换本人可见的订单字段。
func supportOrderRespond(row *models.SupportOrder) *userresp.SupportOrderRespond {
	return &userresp.SupportOrderRespond{RefundReason: row.RefundReason, RefundRequestedAt: row.RefundRequestedAt, RefundReviewNote: row.RefundReviewNote, RefundReviewedAt: row.RefundReviewedAt, RefundedAt: row.RefundedAt, CanRefund: canRequestSupportRefund(row, time.Now()), OrderNo: row.OrderNo, RequestKey: row.RequestKey, CreatorId: row.CreatorId, Amount: row.Amount.StringFixed(2), DurationHours: row.DurationHours, Status: row.Status, RefundStatus: row.RefundStatus, ExpiresAt: row.ExpiresAt, PaidAt: row.PaidAt, ConfirmedAt: row.ConfirmedAt, CreatedAt: row.CreatedAt, HasError: row.LastError != ""}
}

// CreateOrder 锁定购买者串行处理幂等请求，以服务端价格创建固定期限快照。
func (s *SupportServiceImpl) CreateOrder(ctx context.Context, userId, creatorId, requestKey string) (*userresp.SupportOrderRespond, error) {
	cfg := configs.GetConfig().AlipayConfig
	if err := payment.ValidateAlipayConfig(cfg); err != nil {
		return nil, err
	}
	if userId == "" || creatorId == "" || userId == creatorId {
		return nil, fmt.Errorf("请选择其他已开通助力的创作者")
	}
	if len(requestKey) < 8 || len(requestKey) > 64 || strings.TrimSpace(requestKey) != requestKey {
		return nil, fmt.Errorf("订单请求标识必须为8至64个字符")
	}
	var result *models.SupportOrder
	err := supportTransaction(ctx, func(tx *gorm.DB) error {
		user, err := userdao.NewUserInfoDAO(tx).FindUserByUuidForUpdate(userId)
		if err != nil {
			return err
		}
		if user == nil || user.Status != 0 {
			return fmt.Errorf("购买者账号不可用")
		}
		dao := userdao.NewSupportDAO(tx)
		result, err = dao.FindRequest(cfg.Environment, userId, requestKey)
		if err != nil {
			return err
		}
		if result != nil {
			if result.CreatorId != creatorId {
				return fmt.Errorf("订单请求标识已用于其他创作者")
			}
			return nil
		}
		creator, err := userdao.NewUserInfoDAO(tx).FindUserByUuid(creatorId)
		if err != nil {
			return err
		}
		if creator == nil || creator.Uuid == "" || creator.Status != 0 {
			return fmt.Errorf("创作者账号不可用")
		}
		activate, err := userdao.NewSupportDAO(tx).FindActivate(creatorId)
		if err != nil {
			return err
		}
		if activate == nil {
			return fmt.Errorf("创作者尚未开通助力")
		}
		now := time.Now()
		result, err = dao.FindPending(cfg.Environment, userId, creatorId, now)
		if err != nil || result != nil {
			return err
		}
		result = &models.SupportOrder{Environment: cfg.Environment, OrderNo: "SO" + strings.ReplaceAll(uuid.NewString(), "-", ""), UserId: userId, CreatorId: creatorId, RequestKey: requestKey, Channel: "alipay", Amount: activate.Price, DurationHours: constants.SupportDurationHours, ExpiresAt: now.Add(constants.SupportOrderTimeout), NextAttemptAt: now.Add(time.Minute)}
		return dao.CreateOrder(result)
	})
	if err != nil {
		return nil, fmt.Errorf("创建助力订单失败: %w", err)
	}
	return supportOrderRespond(result), nil
}

// ownOrder 校验交易环境及购买者身份。
func ownOrder(ctx context.Context, userId, orderNo string) (*models.SupportOrder, error) {
	row, err := userdao.NewSupportDAO(db.GormDB.WithContext(ctx)).FindOrder(configs.GetConfig().AlipayConfig.Environment, orderNo, false)
	if err != nil {
		return nil, fmt.Errorf("查询助力订单失败: %w", err)
	}
	if row == nil || userId == "" || row.UserId != userId {
		return nil, fmt.Errorf("助力订单不存在")
	}
	return row, nil
}

// GetOrder 查询本人订单，渠道确认通过后台持久化扫描完成。
func (s *SupportServiceImpl) GetOrder(ctx context.Context, userId, orderNo string) (*userresp.SupportOrderRespond, error) {
	row, err := ownOrder(ctx, userId, orderNo)
	if err != nil {
		return nil, err
	}
	return supportOrderRespond(row), nil
}

// PayOrder 保持原订单号、金额和截止时间，不为继续支付创建新订单。
func (s *SupportServiceImpl) PayOrder(ctx context.Context, userId, orderNo string) (*userresp.SupportPaymentRespond, error) {
	row, err := ownOrder(ctx, userId, orderNo)
	if err != nil {
		return nil, err
	}
	if row.Status != constants.SupportOrderPending || !row.ExpiresAt.After(time.Now()) {
		return nil, fmt.Errorf("订单不可继续支付，请刷新支付结果")
	}
	cfg := configs.GetConfig().AlipayConfig
	pageURL, err := payment.AlipayPageURL(cfg, row.OrderNo, row.Amount, row.ExpiresAt)
	if err != nil {
		return nil, err
	}
	qrURL, err := payment.AlipayQRURL(cfg, row.OrderNo, row.Amount, row.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &userresp.SupportPaymentRespond{PayURL: pageURL, QRURL: qrURL}, nil
}

// ListSupport 限定登录用户与当前环境，分页返回交易及权益数据。
func (s *SupportServiceImpl) ListSupport(ctx context.Context, userId, kind string, page int) (any, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录")
	}
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		return nil, fmt.Errorf("页码超出范围")
	}
	cfg := configs.GetConfig().AlipayConfig
	dao := userdao.NewSupportDAO(db.GormDB.WithContext(ctx))
	switch kind {
	case "received":
		rows, total, err := dao.ListReceivedOrders(cfg.Environment, userId, page)
		if err != nil {
			return nil, fmt.Errorf("查询收到的助力记录失败: %w", err)
		}
		result := make([]userresp.SupportReceivedOrderRespond, 0, len(rows))
		for _, row := range rows {
			result = append(result, userresp.SupportReceivedOrderRespond{OrderNo: row.OrderNo, BuyerId: row.BuyerId, BuyerNickname: row.BuyerNickname, BuyerAvatar: row.BuyerAvatar, Amount: row.Amount.StringFixed(2), RefundStatus: row.RefundStatus, PaidAt: row.PaidAt, ConfirmedAt: row.ConfirmedAt})
		}
		return map[string]any{"list": result, "total": total}, nil
	case "orders":
		rows, total, err := dao.ListOrders(cfg.Environment, userId, page)
		if err != nil {
			return nil, fmt.Errorf("查询助力订单失败: %w", err)
		}
		result := make([]*userresp.SupportOrderRespond, 0, len(rows))
		for i := range rows {
			result = append(result, supportOrderRespond(&rows[i]))
		}
		return map[string]any{"list": result, "total": total}, nil
	case "entitlements":
		rows, err := dao.ListMyEntitlements(cfg.Environment, userId, page)
		if err != nil {
			return nil, fmt.Errorf("查询助力权益失败: %w", err)
		}
		result := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			result = append(result, map[string]any{"creator_id": row.CreatorId, "expires_at": row.ExpiresAt})
		}
		return result, nil
	case "income":
		row, err := dao.Income(cfg.Environment, userId)
		if err != nil {
			return nil, fmt.Errorf("查询助力收益失败: %w", err)
		}
		return map[string]any{"gross_amount": row.GrossAmount.StringFixed(2), "refund_amount": row.RefundAmount.StringFixed(2), "net_amount": row.NetAmount.StringFixed(2), "paid_orders": row.PaidOrders}, nil
	}
	return nil, fmt.Errorf("助力查询类型错误")
}

// Notify 验签后统一处理渠道结果，仅在数据库事务提交后返回成功。
func (s *SupportServiceImpl) Notify(ctx context.Context, values url.Values) error {
	cfg := configs.GetConfig().AlipayConfig
	trade, err := payment.VerifyAlipayNotify(cfg, values)
	if err != nil {
		return err
	}
	return applySupportTrade(ctx, cfg.Environment, trade)
}

// applySupportTrade 统一锁定订单再锁权益，处理重复、关闭后晚到成功及并发续期。
func applySupportTrade(ctx context.Context, environment string, trade *payment.AlipayTrade) error {
	amount, err := payment.ParseAlipayAmount(trade.Amount)
	if err != nil {
		return err
	}
	if trade.TradeNo == "" || trade.OrderNo == "" {
		return fmt.Errorf("支付宝交易标识缺失")
	}
	return supportTransaction(ctx, func(tx *gorm.DB) error {
		dao := userdao.NewSupportDAO(tx)
		row, err := dao.FindOrder(environment, trade.OrderNo, true)
		if err != nil {
			return err
		}
		if row == nil {
			return fmt.Errorf("支付宝通知订单不存在")
		}
		if !amount.Equal(row.Amount) || trade.SellerID != configs.GetConfig().AlipayConfig.SellerID {
			return fmt.Errorf("支付宝交易金额或收款方不匹配")
		}
		if row.TradeNo != nil && *row.TradeNo != trade.TradeNo {
			return fmt.Errorf("支付宝交易号与订单不匹配")
		}
		now := time.Now()
		switch trade.Status {
		case "TRADE_SUCCESS", "TRADE_FINISHED":
			if row.Status == constants.SupportOrderPaid {
				return nil
			}
			paidAt, err := time.ParseInLocation("2006-01-02 15:04:05", trade.PaidTime, time.FixedZone("CST", 8*3600))
			if err != nil {
				return fmt.Errorf("支付宝付款时间格式错误")
			}
			entitlement, err := dao.LockEntitlement(environment, row.UserId, row.CreatorId)
			if err != nil {
				return err
			}
			start := now
			if entitlement.ExpiresAt != nil && entitlement.ExpiresAt.After(start) {
				start = *entitlement.ExpiresAt
			}
			if err := dao.SaveExpiry(entitlement.Id, start.Add(time.Duration(row.DurationHours)*time.Hour)); err != nil {
				return err
			}
			if err := dao.UpdateOrder(row.Id, map[string]any{"status": constants.SupportOrderPaid, "trade_no": trade.TradeNo, "paid_at": paidAt, "confirmed_at": now, "next_attempt_at": now.Add(constants.SupportRefundAuditInterval), "closed_at": nil, "processing_until": nil, "last_error": ""}); err != nil {
				return err
			}
			return userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{UserId: row.UserId, ActorId: row.CreatorId, Type: constants.NotificationSupportPaid, Content: "助力付款已确认，30天观看权益已生效。", CreatedAt: now})
		case "TRADE_CLOSED":
			if row.Status == constants.SupportOrderPaid {
				return nil
			}
			return dao.UpdateOrder(row.Id, map[string]any{"status": constants.SupportOrderClosed, "trade_no": trade.TradeNo, "closed_at": now, "processing_until": nil, "last_error": ""})
		case "WAIT_BUYER_PAY":
			return nil
		default:
			return fmt.Errorf("支付宝交易状态不支持")
		}
	})
}

// reconcileSupportOrder 在事务外先查单再关单，未知或网络失败始终保持待确认并继续补偿。
func reconcileSupportOrder(ctx context.Context, row models.SupportOrder) error {
	cfg := configs.GetConfig().AlipayConfig
	trade, err := payment.AlipayRequest(ctx, cfg, "alipay.trade.query", row.OrderNo)
	if err != nil {
		return err
	}
	if trade.Code == "10000" {
		if trade.Status != "WAIT_BUYER_PAY" {
			return applySupportTrade(ctx, row.Environment, trade)
		}
		amount, err := payment.ParseAlipayAmount(trade.Amount)
		if err != nil || !amount.Equal(row.Amount) || trade.OrderNo != row.OrderNo || trade.SellerID != cfg.SellerID {
			return fmt.Errorf("支付宝待付款交易信息不匹配")
		}
	} else if trade.SubCode != "ACQ.TRADE_NOT_EXIST" {
		return fmt.Errorf("支付宝查单未成功，业务代码%s", trade.SubCode)
	}
	if row.ExpiresAt.After(time.Now()) {
		return nil
	}
	closed, err := payment.AlipayRequest(ctx, cfg, "alipay.trade.close", row.OrderNo)
	if err != nil {
		return err
	}
	if closed.Code != "10000" && closed.SubCode != "ACQ.TRADE_NOT_EXIST" && closed.SubCode != "ACQ.TRADE_HAS_CLOSE" {
		// 关单可能与付款竞争，再查一次，支付成功走同一幂等事务。
		latest, err := payment.AlipayRequest(ctx, cfg, "alipay.trade.query", row.OrderNo)
		if err != nil {
			return err
		}
		if latest.Code == "10000" && (latest.Status == "TRADE_SUCCESS" || latest.Status == "TRADE_FINISHED") {
			return applySupportTrade(ctx, row.Environment, latest)
		}
		return fmt.Errorf("支付宝关单结果待确认，业务代码%s", closed.SubCode)
	}
	if closed.Code == "10000" && closed.OrderNo != row.OrderNo {
		return fmt.Errorf("支付宝关单订单号不匹配")
	}
	return supportTransaction(ctx, func(tx *gorm.DB) error {
		dao := userdao.NewSupportDAO(tx)
		current, err := dao.FindOrder(row.Environment, row.OrderNo, true)
		if err != nil {
			return err
		}
		if current == nil || current.Status == constants.SupportOrderPaid {
			return nil
		}
		return dao.UpdateOrder(row.Id, map[string]any{"status": constants.SupportOrderClosed, "closed_at": time.Now(), "processing_until": nil, "last_error": ""})
	})
}

// StartSupportOrderScanner 启动每分钟支付及退款补偿，进程重启后按领取租约恢复。
func StartSupportOrderScanner(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(constants.SupportOrderScanInterval)
		defer ticker.Stop()
		for {
			scanSupportOrders(ctx)
			scanSupportRefunds(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// scanSupportOrders 分批领取任务并在事务外查询渠道，失败保留中文上下文。
func scanSupportOrders(ctx context.Context) {
	cfg := configs.GetConfig().AlipayConfig
	// 尚未创建支付订单的环境无需配置渠道；配置缺失不会阻止其他服务启动。
	var rows []models.SupportOrder
	err := supportTransaction(ctx, func(tx *gorm.DB) error {
		var err error
		rows, err = userdao.NewSupportDAO(tx).ClaimOrders(cfg.Environment, time.Now())
		return err
	})
	if err != nil {
		zlog.Error("领取助力订单补偿失败: " + err.Error())
		return
	}
	var workers sync.WaitGroup
	for _, row := range rows {
		workers.Add(1)
		// 每批最多20个有界请求，并行处理以确保网络超时不会拖过领取租约。
		go func(row models.SupportOrder) {
			defer workers.Done()
			attemptCtx, cancel := context.WithTimeout(ctx, constants.SupportOrderAttemptTimeout)
			defer cancel()
			err := reconcileSupportOrder(attemptCtx, row)
			message := ""
			if err != nil {
				message = err.Error()
				zlog.Error("助力订单查单补偿失败，订单号=" + row.OrderNo + ": " + message)
			}
			if len([]rune(message)) > 300 {
				message = string([]rune(message)[:300])
			}
			if err := userdao.NewSupportDAO(db.GormDB.WithContext(ctx)).FinishAttempt(row, message); err != nil {
				zlog.Error("保存助力订单补偿结果失败，订单号=" + row.OrderNo + ": " + err.Error())
			}
		}(row)
	}
	workers.Wait()
}

// canRequestSupportRefund 判断渠道付款后连续168小时窗口，截止瞬间不再接受。
func canRequestSupportRefund(row *models.SupportOrder, now time.Time) bool {
	return row.Status == constants.SupportOrderPaid && row.RefundStatus == constants.SupportRefundNone && row.RefundRequestedAt == nil && row.PaidAt != nil && !now.Before(*row.PaidAt) && now.Before(row.PaidAt.Add(constants.SupportRefundWindow))
}

// RequestRefund 锁定订单后校验首次申请、所有权和时间窗口并生成固定请求号。
func (s *SupportServiceImpl) RequestRefund(ctx context.Context, userId, orderNo, reason string) error {
	reason = strings.TrimSpace(reason)
	if userId == "" || reason == "" || len([]rune(reason)) > 500 {
		return fmt.Errorf("请填写1至500字的退款原因")
	}
	return supportTransaction(ctx, func(tx *gorm.DB) error {
		dao := userdao.NewSupportDAO(tx)
		row, err := dao.FindOrder(configs.GetConfig().AlipayConfig.Environment, orderNo, true)
		if err != nil {
			return err
		}
		if row == nil || row.UserId != userId {
			return fmt.Errorf("助力订单不存在")
		}
		now := time.Now()
		if !canRequestSupportRefund(row, now) {
			return fmt.Errorf("该订单已申请退款或不在付款后7天内")
		}
		return dao.UpdateOrder(row.Id, map[string]any{"refund_status": constants.SupportRefundPending, "refund_requested_at": now, "refund_reason": reason, "refund_request_no": "SR" + strings.ReplaceAll(uuid.NewString(), "-", "")})
	})
}

// ReviewSupportRefund 复核管理员身份并锁定订单审批，渠道调用在后台事务外执行。
func ReviewSupportRefund(ctx context.Context, adminId, orderNo, action, note string) error {
	note = strings.TrimSpace(note)
	if adminId == "" || note == "" || len([]rune(note)) > 500 {
		return fmt.Errorf("请填写1至500字的审核说明")
	}
	return supportTransaction(ctx, func(tx *gorm.DB) error {
		admin, err := userdao.NewUserInfoDAO(tx).FindUserByUuid(adminId)
		if err != nil {
			return err
		}
		if admin == nil || admin.IsAdmin != 1 || admin.Status != 0 {
			return fmt.Errorf("管理员账号不可用")
		}
		dao := userdao.NewSupportDAO(tx)
		row, err := dao.FindOrder(configs.GetConfig().AlipayConfig.Environment, orderNo, true)
		if err != nil {
			return err
		}
		if row == nil || row.Status != constants.SupportOrderPaid || row.RefundRequestNo == nil {
			return fmt.Errorf("退款申请不存在")
		}
		status := constants.SupportRefundProcessing
		switch action {
		case "approve", "reject":
			if row.RefundStatus != constants.SupportRefundPending {
				return fmt.Errorf("退款申请已处理，请刷新列表")
			}
			if action == "reject" {
				status = constants.SupportRefundRejected
			}
		case "retry":
			if row.RefundStatus != constants.SupportRefundFailed {
				return fmt.Errorf("仅允许重试已明确失败的退款")
			}
			if strings.HasPrefix(row.LastError, "需人工核对") {
				return fmt.Errorf("资金存在差异，请先在支付宝侧核对处理，系统将每日复核")
			}
		default:
			return fmt.Errorf("审核操作无效")
		}
		return dao.UpdateOrder(row.Id, map[string]any{"refund_status": status, "refund_reviewer_id": adminId, "refund_reviewed_at": time.Now(), "refund_review_note": note, "processing_until": nil, "next_attempt_at": time.Now(), "last_error": ""})
	})
}

// supportRefundExpiry 按确认时间及ID升序的有效订单重算权益，不扣减其他订单。
func supportRefundExpiry(rows []models.SupportOrder) *time.Time {
	var expires *time.Time
	for _, row := range rows {
		if row.Status != constants.SupportOrderPaid || row.RefundStatus == constants.SupportRefundSucceeded || row.ConfirmedAt == nil {
			continue
		}
		start := *row.ConfirmedAt
		if expires != nil && expires.After(start) {
			start = *expires
		}
		end := start.Add(time.Duration(row.DurationHours) * time.Hour)
		expires = &end
	}
	return expires
}

// confirmSupportRefund 在订单和权益锁内幂等确认退款、重算权益并保存通知。
func confirmSupportRefund(ctx context.Context, row models.SupportOrder, refundedAt time.Time) error {
	return supportTransaction(ctx, func(tx *gorm.DB) error {
		dao := userdao.NewSupportDAO(tx)
		current, err := dao.FindOrder(row.Environment, row.OrderNo, true)
		if err != nil {
			return err
		}
		if current == nil || current.Status != constants.SupportOrderPaid {
			return fmt.Errorf("退款订单付款状态异常")
		}
		if current.RefundStatus == constants.SupportRefundSucceeded {
			return nil
		}
		entitlement, err := dao.LockEntitlement(current.Environment, current.UserId, current.CreatorId)
		if err != nil {
			return err
		}
		if err := dao.UpdateOrder(current.Id, map[string]any{"refund_status": constants.SupportRefundSucceeded, "refunded_at": refundedAt, "processing_until": nil, "last_error": ""}); err != nil {
			return err
		}
		rows, err := dao.PaidPairOrders(current.Environment, current.UserId, current.CreatorId)
		if err != nil {
			return err
		}
		if err := dao.SetExpiry(entitlement.Id, supportRefundExpiry(rows)); err != nil {
			return err
		}
		return userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{UserId: current.UserId, ActorId: current.CreatorId, Type: constants.NotificationSupportRefund, Content: "助力订单" + current.OrderNo + "已全额退款，观看权益已重新计算。", CreatedAt: time.Now()})
	})
}

// supportRefundFailure 记录明确失败或人工核对原因，未知结果保持处理中。
func supportRefundFailure(ctx context.Context, row models.SupportOrder, message string) error {
	zlog.Warn("助力退款失败或需要人工核对，订单号=" + row.OrderNo + ": " + message)
	return userdao.NewSupportDAO(db.GormDB.WithContext(ctx)).FailRefund(row, message)
}

// reconcileSupportRefund 校验原交易后按固定请求查询退款，日常账单核对识别站外退款。
func reconcileSupportRefund(ctx context.Context, row models.SupportOrder) error {
	cfg := configs.GetConfig().AlipayConfig
	trade, err := payment.AlipayRequest(ctx, cfg, "alipay.trade.query", row.OrderNo)
	if err != nil {
		return err
	}
	if trade.Code != "10000" {
		return fmt.Errorf("核对支付宝已付款订单失败，业务代码%s", trade.SubCode)
	}
	amount, err := payment.ParseAlipayAmount(trade.Amount)
	if err != nil || trade.OrderNo != row.OrderNo || row.TradeNo == nil || trade.TradeNo != *row.TradeNo || !amount.Equal(row.Amount) || trade.SellerID != cfg.SellerID {
		return supportRefundFailure(ctx, row, "需人工核对：渠道订单、交易号或付款金额不匹配")
	}
	if row.RefundStatus != constants.SupportRefundProcessing {
		if err := auditSupportRefundBill(ctx, &row); err != nil {
			return err
		}
		if row.ChannelRefundAmount.Equal(row.Amount) {
			return confirmSupportRefund(ctx, row, time.Now())
		}
		if !row.ChannelRefundAmount.IsZero() {
			return supportRefundFailure(ctx, row, "需人工核对：渠道存在部分退款或累计退款金额异常")
		}
		return nil
	}
	if !row.ChannelRefundAmount.IsZero() {
		return supportRefundFailure(ctx, row, "需人工核对：渠道已存在退款金额，请核对后处理")
	}
	if row.RefundRequestNo == nil {
		return supportRefundFailure(ctx, row, "需人工核对：退款请求号缺失")
	}
	result, err := payment.AlipayRefundRequest(ctx, cfg, "alipay.trade.fastpay.refund.query", row.OrderNo, *row.RefundRequestNo, row.Amount)
	if err != nil {
		return err
	}
	if result.Code == "10000" && result.RefundStatus == "REFUND_SUCCESS" {
		return applySupportRefundResult(ctx, row, result, true)
	}
	// 未收到或尚未成功的请求只按原号重放，未知及系统异常继续查询。
	if result.Code != "10000" && result.SubCode != "ACQ.TRADE_NOT_EXIST" {
		return fmt.Errorf("查询支付宝退款结果待确认，业务代码%s", result.SubCode)
	}
	if result.Code == "10000" && result.RefundStatus != "" {
		return fmt.Errorf("支付宝退款结果待确认")
	}
	result, err = payment.AlipayRefundRequest(ctx, cfg, "alipay.trade.refund", row.OrderNo, *row.RefundRequestNo, row.Amount)
	if err != nil {
		return err
	}
	if result.Code == "10000" {
		return applySupportRefundResult(ctx, row, result, false)
	}
	switch result.SubCode {
	case "ACQ.REFUND_AMT_NOT_EQUAL_TOTAL", "ACQ.REASON_TRADE_REFUND_FEE_ERR":
		return supportRefundFailure(ctx, row, "需人工核对：退款金额与渠道剩余可退金额不一致，业务代码"+result.SubCode)
	case "ACQ.SELLER_BALANCE_NOT_ENOUGH", "ACQ.REASON_TRADE_BEEN_FREEZEN", "ACQ.TRADE_HAS_FINISHED":
		return supportRefundFailure(ctx, row, "支付宝退款失败，业务代码"+result.SubCode)
	default:
		return fmt.Errorf("支付宝退款结果待确认，业务代码%s", result.SubCode)
	}
}

// applySupportRefundResult 核对退款金额及请求关联，仅接受明确全额成功。
func applySupportRefundResult(ctx context.Context, row models.SupportOrder, result *payment.AlipayTrade, query bool) error {
	if result.OrderNo != row.OrderNo || row.TradeNo == nil || result.TradeNo != *row.TradeNo {
		return fmt.Errorf("退款响应订单或交易号不匹配")
	}
	amount := result.RefundFee
	if query {
		if row.RefundRequestNo == nil || result.RefundRequestNo != *row.RefundRequestNo {
			return fmt.Errorf("退款响应请求号不匹配")
		}
		amount = result.RefundAmount
	}
	amountValue, err := payment.ParseAlipayAmount(amount)
	if err != nil || !amountValue.Equal(row.Amount) {
		return fmt.Errorf("退款响应金额不匹配")
	}
	refundedAt := time.Now()
	if result.RefundTime != "" {
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", result.RefundTime, time.FixedZone("CST", 8*3600))
		if err != nil {
			return fmt.Errorf("支付宝退款时间格式错误")
		}
		refundedAt = parsed
	}
	return confirmSupportRefund(ctx, row, refundedAt)
}

// scanSupportRefunds 分批领取退款及资金核对任务，通过租约支持重启恢复。
func scanSupportRefunds(ctx context.Context) {
	var rows []models.SupportOrder
	err := supportTransaction(ctx, func(tx *gorm.DB) error {
		var err error
		rows, err = userdao.NewSupportDAO(tx).ClaimRefundOrders(configs.GetConfig().AlipayConfig.Environment, time.Now())
		return err
	})
	if err != nil {
		zlog.Error("领取助力退款核对任务失败: " + err.Error())
		return
	}
	var workers sync.WaitGroup
	for _, row := range rows {
		workers.Add(1)
		go func(row models.SupportOrder) {
			defer workers.Done()
			attempt, cancel := context.WithTimeout(ctx, constants.SupportOrderAttemptTimeout)
			defer cancel()
			err := reconcileSupportRefund(attempt, row)
			next := time.Now().Add(constants.SupportRefundAuditInterval)
			message := ""
			if row.RefundStatus == constants.SupportRefundProcessing || err != nil {
				next = time.Now().Add(constants.SupportOrderScanInterval)
			}
			if err != nil {
				message = err.Error()
				zlog.Error("助力退款处理或资金核对失败，订单号=" + row.OrderNo + ": " + message)
				if row.RefundStatus == constants.SupportRefundFailed && row.LastError != "" {
					// 后续核对的网络错误不能覆盖原退款失败或资金差异原因。
					message = row.LastError
				}
			} else if row.RefundStatus == constants.SupportRefundFailed {
				message = row.LastError
			}
			if len([]rune(message)) > 300 {
				message = string([]rune(message)[:300])
			}
			if err := userdao.NewSupportDAO(db.GormDB.WithContext(ctx)).FinishRefundAttempt(row, message, next); err != nil {
				zlog.Error("保存助力退款核对结果失败，订单号=" + row.OrderNo + ": " + err.Error())
			}
		}(row)
	}
	workers.Wait()
}

// auditSupportRefundBill 每次推进一个自然日，持久化日期避免重启或重复核对累计两次。
func auditSupportRefundBill(ctx context.Context, row *models.SupportOrder) error {
	if row.PaidAt == nil || row.TradeNo == nil {
		return fmt.Errorf("已付款订单缺少渠道付款信息")
	}
	zone := time.FixedZone("CST", 8*3600)
	paid := row.PaidAt.In(zone)
	day := time.Date(paid.Year(), paid.Month(), paid.Day(), 0, 0, 0, 0, zone)
	if row.RefundCheckedDate != nil {
		checked := row.RefundCheckedDate
		day = time.Date(checked.Year(), checked.Month(), checked.Day(), 0, 0, 0, 0, zone).AddDate(0, 0, 1)
	}
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	if !day.Before(today) {
		return nil
	}
	amount, err := payment.AlipayBillRefund(ctx, configs.GetConfig().AlipayConfig, day, row.OrderNo, *row.TradeNo)
	if err != nil {
		return err
	}
	err = supportTransaction(ctx, func(tx *gorm.DB) error {
		dao := userdao.NewSupportDAO(tx)
		current, err := dao.FindOrder(row.Environment, row.OrderNo, true)
		if err != nil {
			return err
		}
		if current == nil {
			return fmt.Errorf("核对账单的订单不存在")
		}
		if current.RefundCheckedDate != nil && current.RefundCheckedDate.Format("2006-01-02") >= day.Format("2006-01-02") {
			row.ChannelRefundAmount = current.ChannelRefundAmount
			return nil
		}
		row.ChannelRefundAmount = current.ChannelRefundAmount.Add(amount)
		return dao.UpdateOrder(row.Id, map[string]any{"channel_refund_amount": row.ChannelRefundAmount, "refund_checked_date": day})
	})
	if err != nil {
		return err
	}
	if day.AddDate(0, 0, 1).Before(today) && row.ChannelRefundAmount.IsZero() {
		return fmt.Errorf("渠道历史退款账单核对尚未完成")
	}
	return nil
}
