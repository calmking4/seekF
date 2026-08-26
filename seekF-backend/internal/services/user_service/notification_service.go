package userservice

import (
	"context"
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
)

// 通知类型常量
const (
	NotificationTypeLikePost    int8 = 1 // 点赞帖子
	NotificationTypeCommentPost int8 = 2 // 评论帖子
	NotificationTypeReply       int8 = 3 // 回复评论
	NotificationTypeFollow      int8 = 4 // 关注
)

type NotificationService interface {
	// ListNotifications 查询通知列表
	ListNotifications(ctx context.Context, userId string, page, pageSize int) ([]NotificationInfo, int64, error)
	// GetUnreadCount 获取未读通知数
	GetUnreadCount(ctx context.Context, userId string) (int64, error)
	// MarkAsRead 批量标记已读
	MarkAsRead(ctx context.Context, userId string, ids []int64) error
	// MarkAllAsRead 全部标记已读
	MarkAllAsRead(ctx context.Context, userId string) error
	// CreateNotification 创建通知（供其他 Service 调用）
	CreateNotification(ctx context.Context, userId, actorId string, notiType int8, targetUuid, content string) error
}

type NotificationInfo struct {
	Id         int64  `json:"id"`
	ActorId    string `json:"actor_id"`
	ActorName  string `json:"actor_name"`
	ActorAvatar string `json:"actor_avatar"`
	Type       int8   `json:"type"`
	TargetUuid string `json:"target_uuid"`
	Content    string `json:"content"`
	IsRead     bool   `json:"is_read"`
	CreatedAt  string `json:"created_at"`
}

type NotificationServiceImpl struct {
	notificationDAO userdao.NotificationDAO
	userInfoDAO     userdao.UserInfoDAO
}

func NewNotificationService(notificationDAO userdao.NotificationDAO, userInfoDAO userdao.UserInfoDAO) NotificationService {
	return &NotificationServiceImpl{
		notificationDAO: notificationDAO,
		userInfoDAO:     userInfoDAO,
	}
}

func (s *NotificationServiceImpl) ListNotifications(ctx context.Context, userId string, page, pageSize int) ([]NotificationInfo, int64, error) {
	notifications, err := s.notificationDAO.ListByUserId(userId, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %v", err)
	}

	total, err := s.notificationDAO.CountUnread(userId)
	if err != nil {
		return nil, 0, fmt.Errorf("查询未读数失败: %v", err)
	}

	var result []NotificationInfo
	for _, n := range notifications {
		// 查询触发者信息
		actor, _ := s.userInfoDAO.FindUserByUuid(n.ActorId)
		actorName := "未知用户"
		actorAvatar := ""
		if actor != nil {
			actorName = actor.Nickname
			actorAvatar = actor.Avatar
		}

		result = append(result, NotificationInfo{
			Id:          n.Id,
			ActorId:     n.ActorId,
			ActorName:   actorName,
			ActorAvatar: actorAvatar,
			Type:        n.Type,
			TargetUuid:  n.TargetUuid,
			Content:     n.Content,
			IsRead:      n.IsRead == 1,
			CreatedAt:   n.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

func (s *NotificationServiceImpl) GetUnreadCount(ctx context.Context, userId string) (int64, error) {
	return s.notificationDAO.CountUnread(userId)
}

func (s *NotificationServiceImpl) MarkAsRead(ctx context.Context, userId string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return s.notificationDAO.MarkAsRead(userId, ids)
}

func (s *NotificationServiceImpl) MarkAllAsRead(ctx context.Context, userId string) error {
	return s.notificationDAO.MarkAllAsRead(userId)
}

func (s *NotificationServiceImpl) CreateNotification(ctx context.Context, userId, actorId string, notiType int8, targetUuid, content string) error {
	// 不给自己发通知
	if userId == actorId {
		return nil
	}

	notification := &models.Notification{
		UserId:     userId,
		ActorId:    actorId,
		Type:       notiType,
		TargetUuid: targetUuid,
		Content:    content,
		IsRead:     0,
	}
	return s.notificationDAO.CreateNotification(notification)
}
