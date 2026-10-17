package userservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"

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

// SupportService 创作者助力开通及定价业务接口。
type SupportService interface {
	// 查询创作者的开通状态、价格和当前用户权益。
	GetCreator(ctx context.Context, userId, creatorId string) (*userresp.SupportCreatorRespond, error)
	// 为本人开通助力或修改价格。
	SaveActivate(ctx context.Context, userId string, priceYuan int) (*userresp.SupportCreatorRespond, error)
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
	result := &userresp.SupportCreatorRespond{CreatorId: creatorId, PriceYuan: constants.SupportDefaultPriceFen / 100}
	if activate != nil {
		result.Activated = true
		result.PriceYuan = activate.PriceFen / 100
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
		return userdao.NewSupportDAO(tx).SaveActivate(&models.SupportActivate{UserId: userId, PriceFen: int64(priceYuan) * 100})
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
