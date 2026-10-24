package userresp

// PostItem 帖子列表信息及首个媒体的像素尺寸。
type PostItem struct {
	IsUnavailable    bool     `json:"is_unavailable"` // 失效收藏仅展示占位，不能打开或互动。
	IsBlockedBy      bool     `json:"is_blocked_by"`
	ReviewNote       string   `json:"review_note,omitempty"`
	SupportOnly      int8     `json:"support_only"`
	CanPlay          bool     `json:"can_play"`
	LockReason       string   `json:"lock_reason"`
	SupportExpiresAt string   `json:"support_expires_at"`
	Status           int8     `json:"status"`
	ViewCount        int      `json:"view_count"`
	Visibility       int8     `json:"visibility"`
	Version          uint64   `json:"version"`
	Uuid             string   `json:"uuid"`
	UserId           string   `json:"user_id"`
	Nickname         string   `json:"nickname"`
	Avatar           string   `json:"avatar"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	MediaType        int8     `json:"media_type"`
	CoverUrl         string   `json:"cover_url"`
	Tags             []string `json:"tags"`
	FirstUrl         string   `json:"first_url"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
	LikeCount        int      `json:"like_count"`
	CommentCount     int      `json:"comment_count"`
	CollectCount     int      `json:"collect_count"`
	IsLiked          bool     `json:"is_liked"`
	IsCollected      bool     `json:"is_collected"`
	CreatedAt        string   `json:"created_at"`
}

type ListPostsRespond struct {
	List  []PostItem `json:"list"`
	Total int64      `json:"total"`
}

type PostDetailRespond struct {
	SupportOnly      int8     `json:"support_only"`
	CanPlay          bool     `json:"can_play"`
	LockReason       string   `json:"lock_reason"`
	SupportExpiresAt string   `json:"support_expires_at"`
	Visibility       int8     `json:"visibility"`
	Version          uint64   `json:"version"`
	Uuid             string   `json:"uuid"`
	UserId           string   `json:"user_id"`
	Nickname         string   `json:"nickname"`
	Avatar           string   `json:"avatar"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	MediaType        int8     `json:"media_type"`
	CoverUrl         string   `json:"cover_url"`
	Tags             []string `json:"tags"`
	Urls             []string `json:"urls"`
	LikeCount        int      `json:"like_count"`
	CommentCount     int      `json:"comment_count"`
	CollectCount     int      `json:"collect_count"`
	ViewCount        int      `json:"view_count"`
	IsLiked          bool     `json:"is_liked"`
	IsCollected      bool     `json:"is_collected"`
	CreatedAt        string   `json:"created_at"`
}

// CommentItem 评论信息及其回复数量。
type CommentItem struct {
	Uuid            string `json:"uuid"`
	UserId          string `json:"user_id"`
	Nickname        string `json:"nickname"`
	Avatar          string `json:"avatar"`
	ParentId        string `json:"parent_id"`
	ReplyToUserId   string `json:"reply_to_user_id"`
	ReplyToNickname string `json:"reply_to_nickname"`
	Content         string `json:"content"`
	LikeCount       int    `json:"like_count"`
	IsLiked         bool   `json:"is_liked"`
	CreatedAt       string `json:"created_at"`
	ReplyCount      int64  `json:"reply_count"`
}

type ListCommentsRespond struct {
	List []CommentItem `json:"list"`
}

type AICommentRespond struct {
	UserComment CommentItem `json:"user_comment"`
}

type FolderItem struct {
	IsBlockedBy bool   `json:"is_blocked_by"` // 夹主拉黑查看者时仅展示锁定占位。
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	PostCount   int    `json:"post_count"`
	CoverUrl    string `json:"cover_url"`
	CreatedAt   string `json:"created_at"`
}

type ListFoldersRespond struct {
	List []FolderItem `json:"list"`
}

type UserProfileRespond struct {
	Uuid           string     `json:"uuid"`
	Nickname       string     `json:"nickname"`
	Avatar         string     `json:"avatar"`
	Signature      string     `json:"signature"`
	PostCount      int64      `json:"post_count"`
	TotalLikes     int        `json:"total_likes"`
	FollowingCount int64      `json:"following_count"`
	FollowerCount  int64      `json:"follower_count"`
	IsFollowed     bool       `json:"is_followed"`
	IsBlocked      bool       `json:"is_blocked"`
	IsBlockedBy    bool       `json:"is_blocked_by"`
	Posts          []PostItem `json:"posts"`
	Total          int64      `json:"total"`
}

// CommentThreadRespond 评论或回复分页结果，has_more 表示仍有后续内容。
type CommentThreadRespond struct {
	List    []CommentItem `json:"list"`
	HasMore bool          `json:"has_more"`
}

// FeedRespond 游标分页结果，关注关系标识用于展示不同空状态。
type FeedRespond struct {
	List         []PostItem `json:"list"`
	NextCursor   string     `json:"next_cursor"`
	HasMore      bool       `json:"has_more"`
	HasFollowing bool       `json:"has_following"`
}

// ========== 创作者数据中心 ==========

// CreatorAnalyticsMetrics 表示所选期间的七项核心指标，净增粉丝允许负数。
type CreatorAnalyticsMetrics struct {
	Clicks       int64 `json:"clicks"`
	NetFollowers int64 `json:"net_followers"`
	Likes        int64 `json:"likes"`
	Collections  int64 `json:"collections"`
	Supports     int64 `json:"supports"`
	Comments     int64 `json:"comments"`
	Shares       int64 `json:"shares"`
}

// CreatorAnalyticsDay 表示北京时间自然日的点击及元金额。
type CreatorAnalyticsDay struct {
	Date    string `json:"date"`
	Clicks  int64  `json:"clicks"`
	Revenue string `json:"revenue"`
}

// CreatorAnalyticsRespond 返回本人数据中心汇总、补零后的趋势及统计时间边界。
type CreatorAnalyticsRespond struct {
	Metrics       CreatorAnalyticsMetrics `json:"metrics"`
	ClickTrend    []CreatorAnalyticsDay   `json:"click_trend"`
	RevenueTrend  []CreatorAnalyticsDay   `json:"revenue_trend"`
	Revenue       string                  `json:"revenue"`
	TrackingSince string                  `json:"tracking_since"`
	UpdatedAt     string                  `json:"updated_at"`
	Environment   string                  `json:"environment"`
}
