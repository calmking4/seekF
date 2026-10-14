package userservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	userreq "seekF-backend/internal/dto/user/user_req"
)

// FeedInfo 信息流展示数据及下一页边界，不对时间流计算总数。
type FeedInfo struct {
	List         []PostInfo
	NextCursor   string
	HasMore      bool
	HasFollowing bool
}

// feedCursor 保存排序边界和查询条件，仅用于翻页，不作为权限依据。
type feedCursor struct {
	Source    string    `json:"source"`
	MediaType int8      `json:"media_type"`
	UserId    string    `json:"user_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Id        int64     `json:"id"`
}

// encodeFeedCursor 将排序边界和查询条件编码为客户端可传递的游标。
func encodeFeedCursor(value feedCursor) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("生成信息流游标失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// decodeFeedCursor 检查游标格式、排序边界及查询条件，实际权限仍由数据库查询保证。
func decodeFeedCursor(token string, expected feedCursor) (feedCursor, error) {
	invalid := fmt.Errorf("信息流游标无效或与当前筛选不匹配，请刷新列表")
	if len(token) > 2048 {
		return feedCursor{}, invalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return feedCursor{}, invalid
	}
	var value feedCursor
	if json.Unmarshal(raw, &value) != nil || value.Source != expected.Source || value.MediaType != expected.MediaType || value.UserId != expected.UserId || value.CreatedAt.IsZero() || value.Id <= 0 {
		return feedCursor{}, invalid
	}
	return value, nil
}

// ListFeed 按当前公开范围查询下一批帖子，失败时不返回新游标。
func (s *DiscoverServiceImpl) ListFeed(ctx context.Context, userId string, input userreq.FeedRequest) (*FeedInfo, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录后查看信息流")
	}
	if input.Source == "" {
		input.Source = "all"
	}
	if input.Source != "all" && input.Source != "following" {
		return nil, fmt.Errorf("信息流数据源无效")
	}
	if input.PageSize == 0 {
		input.PageSize = 12
	}
	if input.PageSize < 1 || input.PageSize > 20 {
		return nil, fmt.Errorf("信息流每页数量须为1至20条")
	}
	boundary := feedCursor{Source: input.Source, MediaType: -1}
	if input.MediaType != nil {
		if *input.MediaType != 0 && *input.MediaType != 1 {
			return nil, fmt.Errorf("信息流媒体类型无效")
		}
		boundary.MediaType = *input.MediaType
	}
	following := input.Source == "following"
	if following {
		boundary.UserId = userId
	}
	if input.Cursor != "" {
		var err error
		boundary, err = decodeFeedCursor(input.Cursor, boundary)
		if err != nil {
			return nil, err
		}
	}
	posts, err := s.discoverDAO.ListFeed(ctx, userId, following, input.MediaType, boundary.CreatedAt, boundary.Id, input.PageSize+1)
	if err != nil {
		return nil, fmt.Errorf("查询信息流失败: %w", err)
	}
	result := &FeedInfo{HasMore: len(posts) > input.PageSize}
	if result.HasMore {
		posts = posts[:input.PageSize]
	}
	if following {
		result.HasFollowing = len(posts) > 0
		// 已返回关注作品便可确定存在关注关系，只有空列表需要额外查询。
		if !result.HasFollowing {
			result.HasFollowing, err = s.discoverDAO.HasFollowing(ctx, userId)
			if err != nil {
				return nil, fmt.Errorf("查询关注关系失败: %w", err)
			}
		}
	}
	result.List, _, err = s.buildPostInfoList(ctx, userId, posts, 0)
	if err != nil {
		return nil, err
	}
	if result.HasMore {
		last := posts[len(posts)-1]
		boundary.CreatedAt, boundary.Id = last.CreatedAt, last.Id
		result.NextCursor, err = encodeFeedCursor(boundary)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
