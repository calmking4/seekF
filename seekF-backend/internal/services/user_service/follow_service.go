package userservice

import (
	"context"
	"fmt"
	"sort"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

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
	IsFriend  bool
}

type FollowServiceImpl struct {
	followDAO       userdao.FollowDAO
	userInfoDAO     userdao.UserInfoDAO
	contactDAO      userdao.ContactDAO
	notificationDAO userdao.NotificationDAO
}

// NewFollowService 构造关注服务并注入列表查询依赖。
func NewFollowService(followDAO userdao.FollowDAO, userInfoDAO userdao.UserInfoDAO, contactDAO userdao.ContactDAO, notificationDAO userdao.NotificationDAO) FollowService {
	return &FollowServiceImpl{
		followDAO:       followDAO,
		userInfoDAO:     userInfoDAO,
		contactDAO:      contactDAO,
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
		for _, id := range ids {
			user, err := users.FindUserByUuidForUpdate(id)
			if err != nil {
				return err
			}
			if user == nil {
				return fmt.Errorf("用户不存在")
			}
		}
		follows := userdao.NewFollowDAO(tx)
		existing, err := follows.FindFollowForUpdate(userId, followUserId)
		if err != nil {
			return err
		}
		delta := int64(1)
		if existing != nil {
			delta = -1
			if err := follows.DeleteFollow(userId, followUserId); err != nil {
				return err
			}
		} else {
			if err := follows.CreateFollow(&models.UserFollow{UserId: userId, FollowUserId: followUserId}); err != nil {
				return err
			}
		}
		if err := users.UpdateFollowCounts(userId, delta, 0); err != nil {
			return err
		}
		if err := users.UpdateFollowCounts(followUserId, 0, delta); err != nil {
			return err
		}
		if delta > 0 {
			if err := userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{UserId: followUserId, ActorId: userId, Type: NotificationTypeFollow, TargetUuid: userId, Content: "关注了你"}); err != nil {
				return err
			}
		}
		followed = delta > 0
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("切换关注失败: %w", err)
	}
	return followed, nil
}

// GetFollowCounts 读取用户资料中同事务维护的关注和粉丝计数。
func (s *FollowServiceImpl) GetFollowCounts(ctx context.Context, userId string) (int64, int64, error) {
	user, err := s.userInfoDAO.FindUserByUuid(userId)
	if err != nil {
		return 0, 0, fmt.Errorf("查询关注粉丝数失败: %w", err)
	}
	if user == nil || user.Uuid == "" {
		return 0, 0, fmt.Errorf("用户不存在")
	}
	return user.FollowingCount, user.FollowerCount, nil
}

// ListFollowing 分页装配关注用户及当前好友状态。
func (s *FollowServiceImpl) ListFollowing(ctx context.Context, userId string, page, pageSize int) ([]FollowUserInfo, int64, error) {
	follows, err := s.followDAO.ListFollowing(userId, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询关注列表失败: %v", err)
	}

	total, err := s.followDAO.CountFollowing(userId)
	if err != nil {
		return nil, 0, fmt.Errorf("查询关注总数失败: %v", err)
	}

	var result []FollowUserInfo
	for _, f := range follows {
		user, _ := s.userInfoDAO.FindUserByUuid(f.FollowUserId)
		if user == nil {
			continue
		}

		// 检查是否为好友
		isFriend := false
		contact, _ := s.contactDAO.GetUserContactByUserIdAndContactId(userId, f.FollowUserId)
		if contact != nil && contact.Status == 0 {
			isFriend = true
		}

		result = append(result, FollowUserInfo{
			Uuid:      user.Uuid,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Signature: user.Signature,
			IsFriend:  isFriend,
		})
	}

	return result, total, nil
}

// ListFollowers 分页装配粉丝及当前好友状态。
func (s *FollowServiceImpl) ListFollowers(ctx context.Context, userId string, page, pageSize int) ([]FollowUserInfo, int64, error) {
	follows, err := s.followDAO.ListFollowers(userId, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询粉丝列表失败: %v", err)
	}

	total, err := s.followDAO.CountFollowers(userId)
	if err != nil {
		return nil, 0, fmt.Errorf("查询粉丝总数失败: %v", err)
	}

	var result []FollowUserInfo
	for _, f := range follows {
		user, _ := s.userInfoDAO.FindUserByUuid(f.UserId)
		if user == nil {
			continue
		}

		// 检查是否为好友
		isFriend := false
		contact, _ := s.contactDAO.GetUserContactByUserIdAndContactId(userId, f.UserId)
		if contact != nil && contact.Status == 0 {
			isFriend = true
		}

		// 检查我是否关注了这个人（互相关注标记）
		result = append(result, FollowUserInfo{
			Uuid:      user.Uuid,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Signature: user.Signature,
			IsFriend:  isFriend,
		})
	}

	return result, total, nil
}

// IsFollowing 查询当前账号是否关注目标用户。
func (s *FollowServiceImpl) IsFollowing(ctx context.Context, userId, targetUserId string) (bool, error) {
	existing, err := s.followDAO.FindFollow(userId, targetUserId)
	if err != nil {
		return false, fmt.Errorf("查询关注状态失败: %v", err)
	}
	return existing != nil, nil
}
