package userservice

import (
	"context"
	"fmt"
	"sort"
	"time"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowService interface {
	// ToggleFollow 关注/取消关注
	ToggleFollow(ctx context.Context, userId, followUserId string) (bool, error)
	// GetFollowCounts 获取关注/粉丝数
	GetFollowCounts(ctx context.Context, userId string) (following int64, followers int64, err error)
	// ListFollowing 关注列表
	ListFollowing(ctx context.Context, userId string, page, pageSize int) ([]FollowUserInfo, int64, error)
	// ListFollowers 粉丝列表
	ListFollowers(ctx context.Context, userId string, page, pageSize int) ([]FollowUserInfo, int64, error)
	// IsFollowing 查询是否关注了某人
	IsFollowing(ctx context.Context, userId, targetUserId string) (bool, error)
}

type FollowUserInfo struct {
	Uuid      string
	Nickname  string
	Avatar    string
	Signature string
}

type FollowServiceImpl struct {
	followDAO       userdao.FollowDAO
	userInfoDAO     userdao.UserInfoDAO
	notificationDAO userdao.NotificationDAO
}

// NewFollowService 构造关注服务并注入列表查询依赖。
func NewFollowService(followDAO userdao.FollowDAO, userInfoDAO userdao.UserInfoDAO, notificationDAO userdao.NotificationDAO) FollowService {
	return &FollowServiceImpl{
		followDAO:       followDAO,
		userInfoDAO:     userInfoDAO,
		notificationDAO: notificationDAO,
	}
}

// ToggleFollow 按固定顺序锁定双方用户，在同一事务内切换关注、维护计数及保存通知。
func (s *FollowServiceImpl) ToggleFollow(ctx context.Context, userId, followUserId string) (bool, error) {
	if userId == followUserId {
		return false, fmt.Errorf("不能关注自己")
	}

	followed := false
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := userdao.NewUserInfoDAO(tx)
		// 相互关注也以相同顺序加锁，避免两个事务反向等待。
		ids := []string{userId, followUserId}
		sort.Strings(ids)
		var author *models.UserInfo
		for _, id := range ids {
			user, err := users.FindUserByUuidForUpdate(ctx, id)
			if err != nil {
				return err
			}
			if user == nil {
				return fmt.Errorf("用户不存在")
			}
			if id == followUserId {
				author = user
			}
		}
		follows := userdao.NewFollowDAO(tx)
		existing, err := follows.FindFollowForUpdate(ctx, userId, followUserId)
		if err != nil {
			return err
		}
		delta := int64(1)
		if existing != nil {
			delta = -1
			if err := follows.DeleteFollow(ctx, userId, followUserId); err != nil {
				return err
			}
		} else {
			latest, err := userdao.NewFollowingReadDAO(tx).LatestPostID(ctx, followUserId)
			if err != nil {
				return err
			}
			if err := follows.CreateFollow(ctx, &models.UserFollow{UserId: userId, FollowUserId: followUserId, LastSeenPostId: latest}); err != nil {
				return err
			}
		}
		if err := users.UpdateFollowCounts(ctx, userId, delta, 0); err != nil {
			return err
		}
		if err := users.UpdateFollowCounts(ctx, followUserId, 0, delta); err != nil {
			return err
		}
		// 达到阈值即升级；取消关注只减少计数，不清除已经生效的拉模式。
		if author.FeedMode != constants.DiscoverFeedPull && author.FollowerCount+delta >= discoverDistributionThreshold(configs.GetConfig().DiscoverFeedConfig) {
			if err := users.SetFeedModePull(ctx, followUserId); err != nil {
				return fmt.Errorf("升级作者关注流分发模式失败: %w", err)
			}
		}
		if delta > 0 {
			if err := userdao.NewNotificationDAO(tx).CreateNotification(ctx, &models.Notification{UserId: followUserId, ActorId: userId, Type: NotificationTypeFollow, TargetUuid: userId, Content: "关注了你"}); err != nil {
				return err
			}
		}
		// 与关系、计数同事务保存增减事件，取消关注后仍可还原净增。
		if _, err := userdao.NewDiscoverDAO(tx).SaveCreatorAnalyticsEvent(ctx, &models.CreatorAnalyticsEvent{CreatorId: followUserId, ActorId: userId, Kind: constants.CreatorAnalyticsFollower, Delta: delta, RequestKey: uuid.NewString(), CreatedAt: time.Now()}); err != nil {
			return fmt.Errorf("保存粉丝变化失败: %w", err)
		}
		followed = delta > 0
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("切换关注失败: %w", err)
	}
	// 关系已提交后使旧索引及在途重建失效；失败不回滚关注，新关注补查和MySQL权限过滤继续兜底。
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), constants.DiscoverInboxReadTimeout)
	defer cancel()
	if err := invalidateDiscoverInbox(cacheCtx, userId, uuid.NewString()); err != nil {
		zlog.Warn("关注关系已更新但收件箱失效失败，userId=" + userId + ": " + err.Error())
	}
	return followed, nil
}

// GetFollowCounts 读取用户资料中同事务维护的关注和粉丝计数。
func (s *FollowServiceImpl) GetFollowCounts(ctx context.Context, userId string) (int64, int64, error) {
	user, err := s.userInfoDAO.FindUserByUuid(ctx, userId)
	if err != nil {
		return 0, 0, fmt.Errorf("查询关注粉丝数失败: %w", err)
	}
	if user == nil || user.Uuid == "" {
		return 0, 0, fmt.Errorf("用户不存在")
	}
	return user.FollowingCount, user.FollowerCount, nil
}

// ListFollowing 分页装配关注用户的公开信息。
func (s *FollowServiceImpl) ListFollowing(ctx context.Context, userId string, page, pageSize int) ([]FollowUserInfo, int64, error) {
	follows, err := s.followDAO.ListFollowing(ctx, userId, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询关注列表失败: %v", err)
	}

	total, err := s.followDAO.CountFollowing(ctx, userId)
	if err != nil {
		return nil, 0, fmt.Errorf("查询关注总数失败: %v", err)
	}

	var result []FollowUserInfo
	for _, f := range follows {
		user, _ := s.userInfoDAO.FindUserByUuid(ctx, f.FollowUserId)
		if user == nil {
			continue
		}

		result = append(result, FollowUserInfo{
			Uuid:      user.Uuid,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Signature: user.Signature,
		})
	}

	return result, total, nil
}

// ListFollowers 分页装配粉丝的公开信息。
func (s *FollowServiceImpl) ListFollowers(ctx context.Context, userId string, page, pageSize int) ([]FollowUserInfo, int64, error) {
	follows, err := s.followDAO.ListFollowers(ctx, userId, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询粉丝列表失败: %v", err)
	}

	total, err := s.followDAO.CountFollowers(ctx, userId)
	if err != nil {
		return nil, 0, fmt.Errorf("查询粉丝总数失败: %v", err)
	}

	var result []FollowUserInfo
	for _, f := range follows {
		user, _ := s.userInfoDAO.FindUserByUuid(ctx, f.UserId)
		if user == nil {
			continue
		}

		result = append(result, FollowUserInfo{
			Uuid:      user.Uuid,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Signature: user.Signature,
		})
	}

	return result, total, nil
}

// IsFollowing 查询当前账号是否关注目标用户。
func (s *FollowServiceImpl) IsFollowing(ctx context.Context, userId, targetUserId string) (bool, error) {
	existing, err := s.followDAO.FindFollow(ctx, userId, targetUserId)
	if err != nil {
		return false, fmt.Errorf("查询关注状态失败: %v", err)
	}
	return existing != nil, nil
}
