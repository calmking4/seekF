package adminresp

import "time"

// DiscoverPostReview 管理端帖子摘要及待审核内容，媒体通过管理端鉴权入口读取。
type DiscoverPostReview struct {
	Uuid        string    `json:"uuid"`
	UserId      string    `json:"user_id"`
	AuthorName  string    `json:"author_name"`
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	MediaType   int8      `json:"media_type"`
	Visibility  int8      `json:"visibility"`
	SupportOnly int8      `json:"support_only"`
	Status      int8      `json:"status"`
	Version     uint64    `json:"version"`
	ReviewNote  string    `json:"review_note"`
	CoverURL    string    `json:"cover_url"`
	URLs        []string  `json:"urls,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DiscoverReviewRequest 提交当前版本和审核目标，原因最多300字。
type DiscoverReviewRequest struct {
	PostUuid string `json:"post_uuid"`
	Version  uint64 `json:"version"`
	Status   int8   `json:"status"`
	Note     string `json:"note"`
}
