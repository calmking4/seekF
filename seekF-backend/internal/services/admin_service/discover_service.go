package adminservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	userdao "seekF-backend/internal/dao/user_dao"
	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/upload/oss"
	"seekF-backend/internal/pkg/zlog"
)

// ErrDiscoverReviewConflict 提示管理员刷新后按最新内容重新审核。
var ErrDiscoverReviewConflict = errors.New("帖子内容或状态已变更，请刷新后重新审核")

// requireDiscoverReviewer 校验管理员实时权限，避免业务方法被绕过调用。
func (s *AdminService) requireDiscoverReviewer(ctx context.Context, adminId string) error {
	user, err := s.Identity(ctx, adminId)
	if err != nil || user == nil || user.IsAdmin != 1 || user.Status != 0 {
		return fmt.Errorf("管理员账号不可用")
	}
	return nil
}

// discoverReviewMediaURL 使用同源管理端媒体入口，保持私有图片与视频鉴权。
func discoverReviewMediaURL(postUuid, source string) string {
	if source == "" {
		return ""
	}
	return "/api/admin/discover/media?" + url.Values{"post_uuid": {postUuid}, "url": {source}}.Encode()
}

// discoverReviewItem 构建不含正文的帖子列表摘要。
func discoverReviewItem(post models.DiscoverPost, author string) adminresp.DiscoverPostReview {
	return adminresp.DiscoverPostReview{Uuid: post.Uuid, UserId: post.UserId, AuthorName: author, Title: post.Title, MediaType: post.MediaType, Visibility: post.Visibility, SupportOnly: post.SupportOnly, Status: post.Status, Version: post.Version, ReviewNote: post.ReviewNote, CoverURL: discoverReviewMediaURL(post.Uuid, post.CoverUrl), CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt}
}

// ListDiscoverReview 返回帖子列表及真实状态计数。
func (s *AdminService) ListDiscoverReview(ctx context.Context, adminId, keyword string, status *int8, page int) (any, error) {
	if err := s.requireDiscoverReviewer(ctx, adminId); err != nil {
		return nil, err
	}
	keyword = strings.TrimSpace(keyword)
	if utf8.RuneCountInString(keyword) > 100 || page > 100000 || (status != nil && (*status < 0 || *status > 3)) {
		return nil, fmt.Errorf("帖子筛选参数无效")
	}
	if page < 1 {
		page = 1
	}
	rows, total, counts, err := s.dao.ListDiscoverReview(ctx, keyword, status, page)
	if err != nil {
		return nil, fmt.Errorf("查询审核帖子失败: %w", err)
	}
	list := make([]adminresp.DiscoverPostReview, 0, len(rows))
	for _, row := range rows {
		item := discoverReviewItem(row.DiscoverPost, row.AuthorName)
		item.CoverURL = discoverReviewMediaURL(row.Uuid, row.ReviewCoverURL)
		list = append(list, item)
	}
	return map[string]any{"list": list, "total": total, "counts": counts}, nil
}

// DiscoverReviewDetail 允许管理员预览待审、驳回、下架及助力内容，不增加浏览数。
func (s *AdminService) DiscoverReviewDetail(ctx context.Context, adminId, postUuid string) (*adminresp.DiscoverPostReview, error) {
	if err := s.requireDiscoverReviewer(ctx, adminId); err != nil {
		return nil, err
	}
	if len(postUuid) > 20 || postUuid == "" {
		return nil, fmt.Errorf("帖子标识无效")
	}
	dao := userdao.NewDiscoverDAO(db.GormDB.WithContext(ctx))
	post, err := dao.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}
	user, err := userdao.NewUserInfoDAO(db.GormDB.WithContext(ctx)).FindUserByUuid(post.UserId)
	if err != nil {
		return nil, err
	}
	author := "已删除用户"
	if user != nil {
		author = user.Nickname
	}
	result := discoverReviewItem(*post, author)
	result.Content = post.Content
	if len(post.Tags) > 0 && json.Unmarshal(post.Tags, &result.Tags) != nil {
		return nil, fmt.Errorf("读取帖子标签失败")
	}
	result.URLs = make([]string, 0)
	media, err := dao.FindMediaByPostId(post.Id)
	if err != nil {
		return nil, err
	}
	for _, m := range media {
		result.URLs = append(result.URLs, discoverReviewMediaURL(post.Uuid, m.Url))
	}
	return &result, nil
}

// ReviewDiscoverPost 锁定当前版本并原子保存状态、原因和搜索及关注同步事件。
func (s *AdminService) ReviewDiscoverPost(ctx context.Context, adminId string, req adminresp.DiscoverReviewRequest) error {
	if err := s.requireDiscoverReviewer(ctx, adminId); err != nil {
		return err
	}
	req.Note = strings.TrimSpace(req.Note)
	if req.PostUuid == "" || len(req.PostUuid) > 20 || req.Version == 0 || utf8.RuneCountInString(req.Note) > 300 || (req.Status != constants.DiscoverPostPublished && req.Status != constants.DiscoverPostOffline && req.Status != constants.DiscoverPostRejected) {
		return fmt.Errorf("审核参数无效")
	}
	if req.Status != constants.DiscoverPostPublished && req.Note == "" {
		return fmt.Errorf("请填写驳回或下架原因")
	}
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewDiscoverDAO(tx)
		post, err := dao.FindPostByUuidForUpdate(req.PostUuid)
		if err != nil {
			return err
		}
		if post == nil {
			return fmt.Errorf("帖子不存在或已删除")
		}
		if post.Version != req.Version {
			return ErrDiscoverReviewConflict
		}
		allowed := (post.Status == constants.DiscoverPostPending && (req.Status == constants.DiscoverPostPublished || req.Status == constants.DiscoverPostRejected)) || (post.Status == constants.DiscoverPostPublished && req.Status == constants.DiscoverPostOffline) || (post.Status == constants.DiscoverPostOffline && req.Status == constants.DiscoverPostPublished)
		if !allowed {
			return fmt.Errorf("当前状态不能执行此审核操作，请刷新列表")
		}
		post.Status = req.Status
		post.ReviewNote = req.Note
		post.Version++
		if req.Status == constants.DiscoverPostPublished {
			post.ReviewNote = ""
			post.CreatedAt = time.Now()
		}
		if err := dao.SavePost(post); err != nil {
			return err
		}
		return userdao.NewDiscoverOutboxDAO(tx).Create(post, post.Status == constants.DiscoverPostPublished && post.Visibility == 0)
	})
	if err == nil {
		zlog.Info(fmt.Sprintf("管理员审核帖子成功，管理员=%s，帖子=%s，原版本=%d，目标状态=%d", adminId, req.PostUuid, req.Version, req.Status))
	}
	return err
}

// OpenDiscoverReviewMedia 只读取指定帖子引用的本站资源，避免任意URL代理。
func (s *AdminService) OpenDiscoverReviewMedia(ctx context.Context, adminId, postUuid, source, rangeHeader string) (*oss.ResourceStream, error) {
	if err := s.requireDiscoverReviewer(ctx, adminId); err != nil {
		return nil, err
	}
	if len(postUuid) > 20 || len(source) > 2000 {
		return nil, fmt.Errorf("媒体参数无效")
	}
	dao := userdao.NewDiscoverDAO(db.GormDB.WithContext(ctx))
	post, err := dao.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}
	matched := source != "" && post.CoverUrl == source
	if !matched {
		media, err := dao.FindMediaByPostId(post.Id)
		if err != nil {
			return nil, err
		}
		for _, m := range media {
			if m.Url == source {
				matched = true
				break
			}
		}
	}
	key := oss.ObjectKeyFromURL(source)
	if !matched || !strings.HasPrefix(key, "discover/") || source != oss.FileURL(key) {
		return nil, fmt.Errorf("媒体不属于该帖子或不在本站存储")
	}
	return oss.OpenResource(ctx, key, rangeHeader)
}
