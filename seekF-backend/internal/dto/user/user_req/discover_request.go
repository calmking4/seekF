package userreq

// CreatePostRequest 发布帖子及可选媒体尺寸信息。
type CreatePostRequest struct {
	RequestKey string             `json:"request_key"`
	Visibility int8               `json:"visibility"`
	Version    uint64             `json:"version"`
	Title      string             `json:"title" form:"title"`
	Content    string             `json:"content" form:"content"`
	MediaType  int8               `json:"media_type" form:"media_type"`
	Tags       []string           `json:"tags" form:"tags"`
	Urls       []string           `json:"urls" form:"urls"`
	CoverSize  *MediaSizeRequest  `json:"cover_size"`
	MediaSizes []MediaSizeRequest `json:"media_sizes"`
	CoverUrl   string             `json:"cover_url" form:"cover_url"`
}

// UpdatePostRequest 更新已有帖子，UUID来自帖子详情，创建时由后端生成。
type UpdatePostRequest struct {
	CreatePostRequest
	PostUuid string `json:"post_uuid" binding:"required"` // 需要更新的已有帖子UUID
}

// FeedRequest 发现页游标分页，区分时间流、关注流、推荐与热榜。
type FeedRequest struct {
	AuthorUuid string `json:"author_uuid" binding:"max=20"`
	Source     string `json:"source" binding:"omitempty,oneof=all following recommended hot"`
	MediaType  *int8  `json:"media_type" binding:"omitempty,oneof=0 1"`
	Cursor     string `json:"cursor" binding:"max=2048"`
	PageSize   int    `json:"page_size" binding:"min=0,max=20"`
}

// FollowingReadRequest 使用服务端签发的作者更新快照确认查看。
type FollowingReadRequest struct {
	Token string `json:"token" binding:"required,max=2048"`
}

// DiscoverClickRequest 只用于用户主动打开详情，不由详情查询自动触发。
type DiscoverClickRequest struct {
	PostUuid string `json:"post_uuid" binding:"required,max=20"`
	Source   string `json:"source" binding:"required,oneof=recommended hot following image video search profile collection liked"`
}

// DiscoverFeedbackRequest 根据帖子确定目标作者，明确设置不感兴趣状态。
type DiscoverFeedbackRequest struct {
	PostUuid   string `json:"post_uuid" binding:"required,max=20"`
	TargetType string `json:"target_type" binding:"required,oneof=post author"`
	Active     *bool  `json:"active" binding:"required"`
}

// ListPostsRequest 点赞作品的页码分页参数。
type ListPostsRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

type GetPostDetailRequest struct {
	Uuid string `json:"uuid" form:"uuid"`
}

type AddCommentRequest struct {
	PostUuid      string `json:"post_uuid" form:"post_uuid"`
	ParentUuid    string `json:"parent_id" form:"parent_id"`
	ReplyToUserId string `json:"reply_to_user_id" form:"reply_to_user_id"`
	Content       string `json:"content" form:"content"`
}

type ListCommentsRequest struct {
	PostUuid string `json:"post_uuid" form:"post_uuid"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}

type AICommentRequest struct {
	PostUuid       string `json:"post_uuid" binding:"required"`
	Content        string `json:"content" binding:"required"` // 完整内容（含@AI助手前缀），存数据库
	AIQuestion     string `json:"ai_question"`                // 去掉前缀的问题，给AI处理
	ParentUuid     string `json:"parent_id"`
	ReplyToUserId  string `json:"reply_to_user_id"`
	ReplyToContent string `json:"reply_to_content"` // 被回复评论的内容，加入AI上下文
}

type CreateFolderRequest struct {
	Name        string `json:"name" form:"name"`
	Description string `json:"description" form:"description"`
	IsPublic    int8   `json:"is_public" form:"is_public"`
}

type UpdateFolderRequest struct {
	Uuid        string `json:"uuid" form:"uuid"`
	Name        string `json:"name" form:"name"`
	Description string `json:"description" form:"description"`
	IsPublic    int8   `json:"is_public" form:"is_public"`
}

type DeleteFolderRequest struct {
	Uuid string `json:"uuid" form:"uuid"`
}

type GetFolderDetailRequest struct {
	Uuid     string `json:"uuid" form:"uuid"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}

type ListFoldersRequest struct {
	UserId string `json:"user_id" form:"user_id"` // 为空时查看自己的收藏夹
}

type CollectPostRequest struct {
	PostUuid   string `json:"post_uuid" form:"post_uuid"`
	FolderUuid string `json:"folder_uuid" form:"folder_uuid"`
}

type UncollectPostRequest struct {
	PostUuid   string `json:"post_uuid" form:"post_uuid"`
	FolderUuid string `json:"folder_uuid" form:"folder_uuid"`
}

type CheckCollectedRequest struct {
	PostUuid string `json:"post_uuid" form:"post_uuid"`
}

type ListCollectedPostsRequest struct {
	FolderUuid string `json:"folder_uuid" form:"folder_uuid"`
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"page_size" form:"page_size"`
}

type SearchPostsRequest struct {
	Keyword  string `json:"keyword" binding:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type GetUserProfileRequest struct {
	UserId   string `json:"user_id" form:"user_id"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}

// MediaSizeRequest 媒体宽高，单位像素；按 urls 的顺序传递，零表示未知。
type MediaSizeRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SetDiscoverLikeRequest 明确设置帖子或评论的点赞状态。
type SetDiscoverLikeRequest struct {
	TargetUuid string `json:"target_uuid" binding:"required"`
	IsLiked    *bool  `json:"is_liked" binding:"required"`
}

// CommentThreadRequest 按顶级评论或指定父评论分页查询。
type CommentThreadRequest struct {
	PostUuid   string `json:"post_uuid" binding:"required"`
	ParentUuid string `json:"parent_id"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// ListOwnWorksRequest 作者作品分页与筛选参数。
type ListOwnWorksRequest struct {
	Status     *int8  `json:"status"` // 0已发布、1已下架、2审核中、3未通过
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Keyword    string `json:"keyword"`
	Visibility *int8  `json:"visibility"`
}

// SetPostVisibilityRequest 仅修改作品可见性，并校验编辑版本。
type SetPostVisibilityRequest struct {
	PostUuid   string `json:"post_uuid" binding:"required"`
	Visibility *int8  `json:"visibility" binding:"required"`
	Version    uint64 `json:"version" binding:"required"`
}
