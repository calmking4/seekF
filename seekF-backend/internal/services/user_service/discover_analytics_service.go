package userservice

import (
	"context"
	"fmt"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// DiscoverAnalyticsService 创作者统计与分享业务接口。
type DiscoverAnalyticsService interface {
	// 查询当前认证创作者的核心数据及每日趋势。
	GetCreatorAnalytics(ctx context.Context, userId string, input userreq.CreatorAnalyticsRequest) (*userresp.CreatorAnalyticsRespond, error)
	// 记录成功分享并校验同一请求的目标一致性。
	RecordCreatorShare(ctx context.Context, userId string, input userreq.CreatorShareRequest) error
}

// DiscoverAnalyticsServiceImpl 实现创作者统计与分享业务。
type DiscoverAnalyticsServiceImpl struct{}

// NewDiscoverAnalyticsService 构造创作者统计与分享服务。
func NewDiscoverAnalyticsService() DiscoverAnalyticsService {
	return &DiscoverAnalyticsServiceImpl{}
}

// GetCreatorAnalytics 在同一只读快照内读取本人数据并为无数据日期补零。
func (s *DiscoverAnalyticsServiceImpl) GetCreatorAnalytics(ctx context.Context, userId string, input userreq.CreatorAnalyticsRequest) (*userresp.CreatorAnalyticsRespond, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录")
	}
	if (input.Days != 7 && input.Days != 30 && input.Days != 90) || (input.RevenueDays != 7 && input.RevenueDays != 30 && input.RevenueDays != 90) {
		return nil, fmt.Errorf("统计天数仅支持7、30或90天")
	}
	now := time.Now()
	start, end := creatorAnalyticsRange(now, input.Days)
	revenueStart, revenueEnd := creatorAnalyticsRange(now, input.RevenueDays)
	result := &userresp.CreatorAnalyticsRespond{UpdatedAt: now.In(start.Location()).Format("2006-01-02 15:04:05"), Environment: configs.GetConfig().AlipayConfig.Environment}
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		tracking, err := dao.CreatorAnalyticsTrackingStart(ctx)
		if err != nil {
			return fmt.Errorf("读取数据中心启用时间失败，请确认已执行统计迁移: %w", err)
		}
		result.TrackingSince = tracking.In(start.Location()).Format("2006-01-02 15:04:05")
		result.Metrics, err = dao.GetCreatorAnalyticsMetrics(ctx, userId, result.Environment, start, end)
		if err != nil {
			return fmt.Errorf("查询核心数据失败: %w", err)
		}
		clicks, err := dao.ListCreatorClickTrend(ctx, userId, start, end)
		if err != nil {
			return fmt.Errorf("查询点击趋势失败: %w", err)
		}
		revenues, err := dao.ListCreatorRevenueTrend(ctx, userId, result.Environment, revenueStart, revenueEnd)
		if err != nil {
			return err
		}
		clickMap := make(map[string]int64, len(clicks))
		for _, row := range clicks {
			clickMap[row.Date] = row.Count
		}
		for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
			date := day.Format("2006-01-02")
			result.ClickTrend = append(result.ClickTrend, userresp.CreatorAnalyticsDay{Date: date, Clicks: clickMap[date], Revenue: "0.00"})
		}
		revenueMap := make(map[string]decimal.Decimal, len(revenues))
		total := decimal.Zero
		for _, row := range revenues {
			revenueMap[row.Date] = row.Amount
			total = total.Add(row.Amount)
		}
		for day := revenueStart; day.Before(revenueEnd); day = day.AddDate(0, 0, 1) {
			date := day.Format("2006-01-02")
			result.RevenueTrend = append(result.RevenueTrend, userresp.CreatorAnalyticsDay{Date: date, Revenue: revenueMap[date].StringFixed(2)})
		}
		result.Revenue = total.StringFixed(2)
		return nil
	})
	return result, err
}

// RecordCreatorShare 校验可见性后保存成功分享，重复请求不得改为其他帖子。
func (s *DiscoverAnalyticsServiceImpl) RecordCreatorShare(ctx context.Context, userId string, input userreq.CreatorShareRequest) error {
	if userId == "" {
		return fmt.Errorf("请先登录")
	}
	if _, err := uuid.Parse(input.RequestKey); err != nil {
		return fmt.Errorf("分享请求标识无效")
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		post, err := userdao.NewDiscoverDAO(tx).FindPostByUuidForUpdate(ctx, input.PostUuid)
		if err != nil {
			return fmt.Errorf("查询分享帖子失败: %w", err)
		}
		if !canViewUnblockedDiscoverPost(ctx, tx, post, userId) || post.Visibility != 0 {
			return fmt.Errorf("帖子不存在或不可分享")
		}
		saved, err := userdao.NewDiscoverDAO(tx).SaveCreatorAnalyticsEvent(ctx, &models.CreatorAnalyticsEvent{CreatorId: post.UserId, ActorId: userId, TargetUuid: post.Uuid, Kind: constants.CreatorAnalyticsShare, Delta: 1, RequestKey: input.RequestKey, CreatedAt: time.Now()})
		if err != nil {
			return fmt.Errorf("保存分享统计失败: %w", err)
		}
		if saved.Kind != constants.CreatorAnalyticsShare || saved.TargetUuid != post.Uuid {
			return fmt.Errorf("分享请求标识已用于其他操作")
		}
		return nil
	})
}

// creatorAnalyticsRange 计算包含今天的北京时间自然日半开区间。
func creatorAnalyticsRange(now time.Time, days int) (time.Time, time.Time) {
	local := now.In(time.FixedZone("北京时间", 8*3600))
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
	return today.AddDate(0, 0, 1-days), today.AddDate(0, 0, 1)
}
