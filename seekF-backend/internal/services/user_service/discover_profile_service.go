package userservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/zlog"
)

// DiscoverProfileService 用户主页与作品搜索业务接口。
type DiscoverProfileService interface {
	// 搜索帖子（ES全文搜索）
	SearchPosts(ctx context.Context, userId string, keyword string, page, pageSize int) ([]PostInfo, int64, error)
	// 获取用户主页信息
	GetUserProfile(ctx context.Context, profileUserId, currentUserId string, page, pageSize int) (*UserProfileInfo, error)
}

// DiscoverProfileServiceImpl 实现用户主页与作品搜索业务。
type DiscoverProfileServiceImpl struct {
	discoverDAO userdao.DiscoverDAO
	outboxDAO   userdao.DiscoverOutboxDAO
	blockDAO    userdao.BlockDAO
	userInfoDAO userdao.UserInfoDAO
	followDAO   userdao.FollowDAO
}

// UserProfileInfo 用户主页及公开作品列表。
type UserProfileInfo struct {
	Uuid           string
	Nickname       string
	Avatar         string
	Signature      string
	PostCount      int64
	TotalLikes     int
	FollowingCount int64
	FollowerCount  int64
	IsFollowed     bool
	IsBlocked      bool
	IsBlockedBy    bool
	Posts          []PostInfo
	Total          int64
}

// NewDiscoverProfileService 构造服务并注入所需依赖。
func NewDiscoverProfileService(
	discoverDAO userdao.DiscoverDAO,
	outboxDAO userdao.DiscoverOutboxDAO,
	blockDAO userdao.BlockDAO,
	userInfoDAO userdao.UserInfoDAO,
	followDAO userdao.FollowDAO,
) DiscoverProfileService {
	return &DiscoverProfileServiceImpl{
		discoverDAO: discoverDAO,
		outboxDAO:   outboxDAO,
		blockDAO:    blockDAO,
		userInfoDAO: userInfoDAO,
		followDAO:   followDAO,
	}
}

// SearchPosts 合并全文和实时作者昵称候选后核对权限并分页，积压或ES故障时回退数据库。
func (s *DiscoverProfileServiceImpl) SearchPosts(ctx context.Context, userId, keyword string, page, pageSize int) ([]PostInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		pending, pendingErr := s.outboxDAO.HasIncomplete(ctx)
		if pendingErr == nil && !pending {
			searchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			ids, searchErr := userdao.NewESDiscoverDAO().SearchPostIDs(searchCtx, keyword)
			cancel()
			if searchErr == nil {
				authorIDs, authorErr := s.discoverDAO.SearchAuthorPostIDs(ctx, keyword, userId)
				if authorErr != nil {
					return nil, 0, authorErr
				}
				// 候选过多时不截断作者作品，改用数据库完整分页；去重在权限核对后完成。
				if len(authorIDs) <= constants.DiscoverSearchCandidateLimit && len(ids)+len(authorIDs) > 0 {
					posts, total, verifyErr := s.filterDiscoverSearch(ctx, userId, append(ids, authorIDs...), page, pageSize)
					if verifyErr != nil {
						return nil, 0, verifyErr
					}
					return buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, userId, posts, total)
				}
			}
			if searchErr != nil && ctx.Err() == nil {
				zlog.Warn("全文搜索降级为数据库查询: " + searchErr.Error())
			}
		}
	}
	posts, total, err := s.discoverDAO.SearchPostsPage(ctx, keyword, page, pageSize, userId)
	if err != nil {
		return nil, 0, err
	}
	return buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, userId, posts, total)
}

// GetUserProfile 查询用户主页及帖子列表。
func (s *DiscoverProfileServiceImpl) GetUserProfile(ctx context.Context, profileUserId, currentUserId string, page, pageSize int) (*UserProfileInfo, error) {
	// 查询用户信息
	user, err := s.userInfoDAO.FindUserByUuid(ctx, profileUserId)
	if err != nil {
		return nil, fmt.Errorf("查询用户信息失败: %v", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	blocked, blockedBy, err := s.blockDAO.Status(ctx, currentUserId, profileUserId)
	if err != nil {
		return nil, fmt.Errorf("查询主页拉黑状态失败: %w", err)
	}
	// 查询帖子列表
	posts, err := s.discoverDAO.ListPostsByUserId(ctx, profileUserId, currentUserId, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询帖子列表失败: %v", err)
	}

	total, err := s.discoverDAO.CountPostsByUserId(ctx, profileUserId, currentUserId)
	if err != nil {
		return nil, fmt.Errorf("查询帖子总数失败: %v", err)
	}

	totalLikes, err := s.discoverDAO.GetUserTotalLikes(ctx, profileUserId, currentUserId)
	if err != nil {
		totalLikes = 0
	}

	postInfos, _, err := buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, currentUserId, posts, total)
	if err != nil {
		return nil, err
	}

	// 查询当前用户是否关注了该用户
	isFollowed := false
	if currentUserId != "" && currentUserId != profileUserId {
		follow, _ := s.followDAO.FindFollow(ctx, currentUserId, profileUserId)
		if follow != nil {
			isFollowed = true
		}
	}

	return &UserProfileInfo{
		Uuid:           user.Uuid,
		Nickname:       user.Nickname,
		Avatar:         user.Avatar,
		Signature:      user.Signature,
		PostCount:      total,
		TotalLikes:     totalLikes,
		FollowingCount: user.FollowingCount,
		FollowerCount:  user.FollowerCount,
		IsFollowed:     isFollowed,
		IsBlocked:      blocked,
		IsBlockedBy:    blockedBy,
		Posts:          postInfos,
		Total:          total,
	}, nil
}

// filterDiscoverSearch 对候选先核对当前权限再分页，避免私密候选占位导致短页或错误总数。
func (s *DiscoverProfileServiceImpl) filterDiscoverSearch(ctx context.Context, userId string, ids []string, page, pageSize int) ([]models.DiscoverPost, int64, error) {
	visible := make(map[string]models.DiscoverPost, len(ids))
	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		posts, err := s.discoverDAO.FindPublicPostsByUuids(ctx, ids[start:end], userId)
		if err != nil {
			return nil, 0, fmt.Errorf("核对搜索作品公开范围失败: %w", err)
		}
		for _, post := range posts {
			visible[post.Uuid] = post
		}
	}
	ordered := make([]models.DiscoverPost, 0, len(visible))
	for _, id := range ids {
		if post, ok := visible[id]; ok {
			ordered = append(ordered, post)
			delete(visible, id)
		}
	}
	total := int64(len(ordered))
	if int64(page) > total/int64(pageSize)+1 {
		return []models.DiscoverPost{}, total, nil
	}
	offset := (int64(page) - 1) * int64(pageSize)
	if offset >= total {
		return []models.DiscoverPost{}, total, nil
	}
	end := offset + int64(pageSize)
	if end > total {
		end = total
	}
	return ordered[offset:end], total, nil
}
