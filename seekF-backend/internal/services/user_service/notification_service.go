package userservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
)

// 通知类型常量
const (
	NotificationTypeLikePost    int8 = 1 // 点赞帖子
	NotificationTypeCommentPost int8 = 2 // 评论帖子
	NotificationTypeReply       int8 = 3 // 回复评论
	NotificationTypeFollow      int8 = 4 // 关注
	NotificationTypeCollectPost int8 = 5 // 收藏帖子
	NotificationTypeLikeComment int8 = 6 // 点赞评论
)

type NotificationService interface {
	// ListNotifications 查询通知列表
	ListNotifications(ctx context.Context, userId string, page, pageSize int, category string, unreadOnly bool) ([]NotificationInfo, int64, error)
	GetUnreadCounts(ctx context.Context, userId string) (map[string]int64, error)
	// GetUnreadCount 获取未读通知数
	GetUnreadCount(ctx context.Context, userId string) (int64, error)
	// MarkAsRead 批量标记已读
	MarkAsRead(ctx context.Context, userId string, ids []int64) error
	// MarkAllAsRead 全部标记已读
	MarkAllAsRead(ctx context.Context, userId string) error
	MarkCategoryAsRead(ctx context.Context, userId, category string) (int64, error)
	// CreateNotification 创建通知（供其他 Service 调用）
	CreateNotification(ctx context.Context, userId, actorId string, notiType int8, targetUuid, content string) error
}

type NotificationInfo struct {
	Id                int64  `json:"id"`
	ActorId           string `json:"actor_id"`
	ActorName         string `json:"actor_name"`
	ActorAvatar       string `json:"actor_avatar"`
	Type              int8   `json:"type"`
	TargetUuid        string `json:"target_uuid"`
	Content           string `json:"content"`
	IsRead            bool   `json:"is_read"`
	CreatedAt         string `json:"created_at"`
	PostTitle         string `json:"post_title"`
	PostCover         string `json:"post_cover"`
	TargetAvailable   bool   `json:"target_available"`
	IsFollowing       bool   `json:"is_following"`
	CommentUuid       string `json:"comment_uuid,omitempty"`
	CommentParentUuid string `json:"comment_parent_uuid,omitempty"`
}

type NotificationServiceImpl struct {
	notificationDAO userdao.NotificationDAO
	userInfoDAO     userdao.UserInfoDAO
	discoverDAO     userdao.DiscoverDAO
	followDAO       userdao.FollowDAO
}

func NewNotificationService(notificationDAO userdao.NotificationDAO, userInfoDAO userdao.UserInfoDAO, discoverDAO userdao.DiscoverDAO, followDAO userdao.FollowDAO) NotificationService {
	return &NotificationServiceImpl{
		notificationDAO: notificationDAO,
		userInfoDAO:     userInfoDAO,
		discoverDAO:     discoverDAO,
		followDAO:       followDAO,
	}
}

// ListNotifications 分页查询分类通知并装配触发者、评论和所属帖子信息。
func (s *NotificationServiceImpl) ListNotifications(ctx context.Context, userId string, page, pageSize int, category string, unreadOnly bool) ([]NotificationInfo, int64, error) {
	types := NotificationCategoryTypes(category)
	notifications, err := s.notificationDAO.ListByUserId(userId, page, pageSize, types, unreadOnly)
	if err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %v", err)
	}

	total, err := s.notificationDAO.CountByUserId(userId, types, unreadOnly)
	if err != nil {
		return nil, 0, fmt.Errorf("查询通知总数失败: %w", err)
	}

	result := make([]NotificationInfo, 0, len(notifications))
	posts := make(map[string]*models.DiscoverPost)
	covers := make(map[string]string)
	actors := make(map[string]*models.UserInfo)
	following := make(map[string]bool)
	for _, n := range notifications {
		// 评论互动通知记录评论 UUID，解析所属帖子供通知页面打开；旧通知仍可使用帖子 UUID。
		targetUuid := n.TargetUuid
		var notificationComment *models.DiscoverComment
		if (n.Type == NotificationTypeCommentPost || n.Type == NotificationTypeReply || n.Type == NotificationTypeLikeComment) && strings.HasPrefix(targetUuid, "C") {
			notificationComment, err = s.discoverDAO.FindCommentByUuid(targetUuid)
			if err != nil {
				return nil, 0, fmt.Errorf("查询通知评论失败: %w", err)
			}
			targetUuid = ""
			if notificationComment != nil {
				post, err := s.discoverDAO.FindPostById(notificationComment.PostId)
				if err != nil {
					return nil, 0, fmt.Errorf("查询评论所属帖子失败: %w", err)
				}
				if post != nil {
					targetUuid = post.Uuid
					posts[targetUuid] = post
				}
			}
		}
		// 查询触发者信息
		actor, cached := actors[n.ActorId]
		if !cached {
			actor, err = s.userInfoDAO.FindUserByUuid(n.ActorId)
			if err != nil {
				return nil, 0, fmt.Errorf("查询通知用户失败: %w", err)
			}
			actors[n.ActorId] = actor
		}
		actorName := "未知用户"
		actorAvatar := ""
		if actor != nil {
			actorName = actor.Nickname
			actorAvatar = actor.Avatar
		}

		info := NotificationInfo{
			Id:          n.Id,
			ActorId:     n.ActorId,
			ActorName:   actorName,
			ActorAvatar: actorAvatar,
			Type:        n.Type,
			TargetUuid:  targetUuid,
			Content:     n.Content,
			IsRead:      n.IsRead == 1,
			CreatedAt:   n.CreatedAt.Format(time.RFC3339),
		}
		if notificationComment != nil {
			info.CommentUuid = notificationComment.Uuid
			info.CommentParentUuid = notificationComment.ParentId
			info.Content = notificationComment.Content
		}
		if n.Type == NotificationTypeFollow {
			if _, ok := following[n.ActorId]; !ok {
				follow, err := s.followDAO.FindFollow(userId, n.ActorId)
				if err != nil {
					return nil, 0, fmt.Errorf("查询关注状态失败: %w", err)
				}
				following[n.ActorId] = follow != nil
			}
			info.IsFollowing = following[n.ActorId]
			info.TargetAvailable = actor != nil
		} else if targetUuid != "" {
			post, cached := posts[targetUuid]
			if !cached {
				post, err = s.discoverDAO.FindPostByUuid(targetUuid)
				if err != nil {
					return nil, 0, fmt.Errorf("查询通知帖子失败: %w", err)
				}
				posts[targetUuid] = post
			}
			if _, cached := covers[targetUuid]; !cached {
				if CanViewDiscoverPost(post, userId) {
					covers[targetUuid] = post.CoverUrl
					if post.MediaType == 0 && post.CoverUrl == "" {
						media, err := s.discoverDAO.FindMediaByPostId(post.Id)
						if err != nil {
							return nil, 0, fmt.Errorf("查询通知封面失败: %w", err)
						}
						if len(media) > 0 {
							covers[targetUuid] = media[0].Url
						}
					}
				}
			}
			if CanViewDiscoverPost(post, userId) {
				info.PostTitle, info.PostCover, info.TargetAvailable = post.Title, covers[targetUuid], true
			}
		}
		if n.Type != NotificationTypeFollow && !info.TargetAvailable {
			info.Content = ""
			info.TargetUuid = ""
			info.CommentUuid = ""
			info.CommentParentUuid = ""
		}
		result = append(result, info)
	}

	return result, total, nil
}

// NotificationCategoryTypes 将页面分类映射到通知类型，空分类兼容旧客户端。
func NotificationCategoryTypes(category string) []int8 {
	switch category {
	case "comments":
		return []int8{NotificationTypeCommentPost, NotificationTypeReply}
	case "likes":
		return []int8{NotificationTypeLikePost, NotificationTypeCollectPost, NotificationTypeLikeComment}
	case "follows":
		return []int8{NotificationTypeFollow}
	default:
		return nil
	}
}

// GetUnreadCounts 统计评论、赞和收藏、关注各分类的未读数量。
func (s *NotificationServiceImpl) GetUnreadCounts(ctx context.Context, userId string) (map[string]int64, error) {
	counts, err := s.notificationDAO.CountUnreadByType(userId)
	if err != nil {
		return nil, fmt.Errorf("查询分类未读数失败: %w", err)
	}
	return map[string]int64{
		"comments": counts[NotificationTypeCommentPost] + counts[NotificationTypeReply],
		"likes":    counts[NotificationTypeLikePost] + counts[NotificationTypeCollectPost] + counts[NotificationTypeLikeComment],
		"follows":  counts[NotificationTypeFollow],
	}, nil
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

func (s *NotificationServiceImpl) MarkCategoryAsRead(ctx context.Context, userId, category string) (int64, error) {
	types := NotificationCategoryTypes(category)
	if len(types) == 0 {
		return 0, fmt.Errorf("通知分类无效")
	}
	return s.notificationDAO.MarkCategoryAsRead(userId, types)
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
