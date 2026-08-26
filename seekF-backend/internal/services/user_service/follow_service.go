package userservice

import (
	"context"
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/zlog"
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

func NewFollowService(followDAO userdao.FollowDAO, userInfoDAO userdao.UserInfoDAO, contactDAO userdao.ContactDAO, notificationDAO userdao.NotificationDAO) FollowService {
	return &FollowServiceImpl{
		followDAO:       followDAO,
		userInfoDAO:     userInfoDAO,
		contactDAO:      contactDAO,
		notificationDAO: notificationDAO,
	}
}

func (s *FollowServiceImpl) ToggleFollow(ctx context.Context, userId, followUserId string) (bool, error) {
	if userId == followUserId {
		return false, fmt.Errorf("不能关注自己")
	}

	// 检查目标用户是否存在
	targetUser, err := s.userInfoDAO.FindUserByUuid(followUserId)
	if err != nil {
		return false, fmt.Errorf("查询用户失败: %v", err)
	}
	if targetUser == nil {
		return false, fmt.Errorf("用户不存在")
	}

	// 查询是否已关注
	existing, err := s.followDAO.FindFollow(userId, followUserId)
	if err != nil {
		return false, fmt.Errorf("查询关注状态失败: %v", err)
	}

	if existing != nil {
		// 已关注，取消关注
		if err := s.followDAO.DeleteFollow(userId, followUserId); err != nil {
			return false, fmt.Errorf("取消关注失败: %v", err)
		}
		return false, nil
	}

	// 未关注，创建关注
	follow := &models.UserFollow{
		Id:           0,
		UserId:       userId,
		FollowUserId: followUserId,
	}
	// 生成一个 id（GORM 自增不需要手动设）
	if err := s.followDAO.CreateFollow(follow); err != nil {
		return false, fmt.Errorf("关注失败: %v", err)
	}

	// 异步创建关注通知
	go func() {
		notification := &models.Notification{
			UserId:     followUserId,
			ActorId:    userId,
			Type:       NotificationTypeFollow,
			TargetUuid: userId,
			Content:    "关注了你",
		}
		if err := s.notificationDAO.CreateNotification(notification); err != nil {
			zlog.Error("创建关注通知失败: " + err.Error())
		}
	}()

	return true, nil
}

func (s *FollowServiceImpl) GetFollowCounts(ctx context.Context, userId string) (int64, int64, error) {
	following, err := s.followDAO.CountFollowing(userId)
	if err != nil {
		return 0, 0, fmt.Errorf("查询关注数失败: %v", err)
	}
	followers, err := s.followDAO.CountFollowers(userId)
	if err != nil {
		return 0, 0, fmt.Errorf("查询粉丝数失败: %v", err)
	}
	return following, followers, nil
}

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

func (s *FollowServiceImpl) IsFollowing(ctx context.Context, userId, targetUserId string) (bool, error) {
	existing, err := s.followDAO.FindFollow(userId, targetUserId)
	if err != nil {
		return false, fmt.Errorf("查询关注状态失败: %v", err)
	}
	return existing != nil, nil
}
