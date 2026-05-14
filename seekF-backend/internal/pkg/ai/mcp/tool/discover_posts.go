package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	userdao "seekF-backend/internal/dao/user_dao"
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

// DiscoverPostsSentinel 帖子数据哨兵标记，Service 层通过此标记提取结构化数据
const DiscoverPostsSentinel = "__DISCOVER_POSTS_JSON__"

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

	// 构建文本摘要和结构化数据
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个相关帖子：\n\n", len(posts)))

	postItems := make([]DiscoverPostItem, 0, len(posts))
	for i, post := range posts {
		// 获取首图 URL
		var src string
		mediaList, err := t.discoverDAO.FindMediaByPostId(post.Id)
		if err == nil && len(mediaList) > 0 {
			src = mediaList[0].Url
		}

		// 获取用户信息
		var avatar, nickname string
		user, err := t.userInfoDAO.FindUserByUuid(post.UserId)
		if err == nil && user != nil {
			avatar = user.Avatar
			nickname = user.Nickname
		}

		// 解析标签
		var tags []string
		if len(post.Tags) > 0 {
			json.Unmarshal(post.Tags, &tags)
		}

		// 截断正文摘要
		contentSnippet := post.Content
		if len(contentSnippet) > 100 {
			contentSnippet = contentSnippet[:100] + "..."
		}

		// 文本摘要给 AI 模型
		tagStr := ""
		if len(tags) > 0 {
			tagStr = "，标签：" + strings.Join(tags, "、")
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s%s\n   点赞：%d，评论：%d\n\n",
			i+1, post.Title, contentSnippet, tagStr, post.LikeCount, post.CommentCount))

		// 结构化数据给前端
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

	// 序列化结构化数据
	postsJSON, _ := json.Marshal(postItems)

	// 追加哨兵标记和 JSON，Service 层通过此标记提取结构化数据
	sb.WriteString("\n")
	sb.WriteString(DiscoverPostsSentinel)
	sb.WriteString(string(postsJSON))

	return mcp.NewToolResultText(sb.String()), nil
}
