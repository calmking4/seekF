package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/zlog"

	"github.com/mark3labs/mcp-go/mcp"
)

// DiscoverPostItem 帖子结构化数据（供前端渲染）
type DiscoverPostItem struct {
	ID           string   `json:"id"`
	Src          string   `json:"src"`
	Title        string   `json:"title"`
	Avatar       string   `json:"avatar"`
	Nickname     string   `json:"nickname"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
}

// DiscoverPostsTool 查询平台发现页帖子的 MCP 工具
type DiscoverPostsTool struct {
	discoverDAO userdao.DiscoverDAO
	userInfoDAO userdao.UserInfoDAO
}

// NewDiscoverPostsTool 创建帖子查询工具实例
func NewDiscoverPostsTool() *DiscoverPostsTool {
	return &DiscoverPostsTool{
		discoverDAO: userdao.NewDiscoverDAO(),
		userInfoDAO: userdao.NewUserInfoDAO(),
	}
}

// GetDiscoverPostsTool 获取帖子查询工具定义
func (t *DiscoverPostsTool) GetDiscoverPostsTool() mcp.Tool {
	return mcp.NewTool(
		"get_discover_posts",
		mcp.WithDescription("查询平台发现页的帖子。当用户询问有什么新鲜帖子、推荐帖子、社区动态、特定话题/标签的帖子时使用此工具。搜索会同时匹配帖子标题、正文内容和标签。"),
		mcp.WithString("keyword",
			mcp.Description("搜索关键词，会匹配帖子标题、正文和标签。如：'旅行'、'美食'、'编程'。留空则返回最新帖子。"),
		),
		mcp.WithString("limit",
			mcp.Description("返回帖子数量，默认5，最大10。"),
		),
	)
}

// HandleDiscoverPostsRequest 处理帖子查询请求
// 返回 TextContent（文本摘要给 AI）+ EmbeddedResource（结构化 JSON 给前端）
func (t *DiscoverPostsTool) HandleDiscoverPostsRequest(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return mcp.NewToolResultText("参数解析失败"), nil
	}

	// 解析 keyword 参数
	keyword := ""
	if kw, ok := arguments["keyword"].(string); ok {
		keyword = strings.TrimSpace(kw)
	}

	// 解析 limit 参数，默认5，最大10
	limit := 5
	if lv, ok := arguments["limit"]; ok {
		switch v := lv.(type) {
		case float64:
			limit = int(v)
		case string:
			fmt.Sscanf(v, "%d", &limit)
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 10 {
			limit = 10
		}
	}

	// 查询帖子
	posts, err := t.discoverDAO.SearchPostsByKeyword(keyword, limit)
	if err != nil {
		zlog.Error("search discover posts failed: " + err.Error())
		return mcp.NewToolResultText("查询帖子失败: " + err.Error()), nil
	}

	if len(posts) == 0 {
		return mcp.NewToolResultText("未找到相关帖子"), nil
	}

	// 批量拉取媒体与用户，避免对每个帖子单独查库（N+1）
	postIDs := make([]int64, len(posts))
	userUUIDs := make([]string, 0, len(posts))
	seenUser := make(map[string]struct{}, len(posts))
	for i, post := range posts {
		postIDs[i] = post.Id
		if post.UserId == "" {
			continue
		}
		if _, ok := seenUser[post.UserId]; ok {
			continue
		}
		seenUser[post.UserId] = struct{}{}
		userUUIDs = append(userUUIDs, post.UserId)
	}

	firstMediaURLByPostID := make(map[int64]string, len(posts))
	mediaRows, mErr := t.discoverDAO.FindMediaByPostIds(postIDs)
	if mErr != nil {
		zlog.Error("batch load discover media failed: " + mErr.Error())
	} else {
		for _, row := range mediaRows {
			if _, exists := firstMediaURLByPostID[row.PostId]; !exists {
				firstMediaURLByPostID[row.PostId] = row.Url
			}
		}
	}

	userByUUID := make(map[string]*models.UserInfo, len(userUUIDs))
	if len(userUUIDs) > 0 {
		users, uErr := t.userInfoDAO.FindUsersByUuids(userUUIDs)
		if uErr != nil {
			zlog.Error("batch load user info failed: " + uErr.Error())
		} else {
			for i := range users {
				u := &users[i]
				userByUUID[u.Uuid] = u
			}
		}
	}

	// 构建文本摘要（给 AI 模型）和结构化数据（给前端）
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个相关帖子：\n\n", len(posts)))

	postItems := make([]DiscoverPostItem, 0, len(posts))
	for i, post := range posts {
		// 获取首图 URL
		src := firstMediaURLByPostID[post.Id]

		// 获取用户信息
		var avatar, nickname string
		if u := userByUUID[post.UserId]; u != nil {
			avatar = u.Avatar
			nickname = u.Nickname
		}

		// 解析标签
		var tags []string
		if len(post.Tags) > 0 {
			if jerr := json.Unmarshal(post.Tags, &tags); jerr != nil {
				zlog.Error("unmarshal post tags failed: " + jerr.Error())
			}
		}

		// 截断正文摘要
		contentSnippet := post.Content
		if len(contentSnippet) > 100 {
			contentSnippet = contentSnippet[:100] + "..."
		}

		// 文本摘要
		tagStr := ""
		if len(tags) > 0 {
			tagStr = "，标签：" + strings.Join(tags, "、")
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s%s\n   点赞：%d，评论：%d\n\n",
			i+1, post.Title, contentSnippet, tagStr, post.LikeCount, post.CommentCount))

		// 结构化数据
		postItems = append(postItems, DiscoverPostItem{
			ID:           post.Uuid,
			Src:          src,
			Title:        post.Title,
			Avatar:       avatar,
			Nickname:     nickname,
			Content:      contentSnippet,
			Tags:         tags,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
		})
	}

	// 返回 TextContent（AI 摘要）+ StructuredContent（前端结构化数据）
	return mcp.NewToolResultStructured(postItems, sb.String()), nil
}
