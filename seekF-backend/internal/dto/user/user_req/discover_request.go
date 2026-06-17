package userreq

type CreatePostRequest struct {
	Title     string   `json:"title" form:"title"`
	Content   string   `json:"content" form:"content"`
	MediaType int8     `json:"media_type" form:"media_type"`
	Tags      []string `json:"tags" form:"tags"`
	Urls      []string `json:"urls" form:"urls"`
	CoverUrl  string   `json:"cover_url" form:"cover_url"`
}

type ListPostsRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

type GetPostDetailRequest struct {
	Uuid string `json:"uuid" form:"uuid"`
}

type ToggleLikeRequest struct {
	TargetUuid string `json:"target_uuid" form:"target_uuid"`
}

type AddCommentRequest struct {
	PostUuid       string `json:"post_uuid" form:"post_uuid"`
	ParentUuid     string `json:"parent_id" form:"parent_id"`
	ReplyToUserId  string `json:"reply_to_user_id" form:"reply_to_user_id"`
	Content        string `json:"content" form:"content"`
}

type ListCommentsRequest struct {
	PostUuid string `json:"post_uuid" form:"post_uuid"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}

type ToggleCommentLikeRequest struct {
	CommentUuid string `json:"comment_uuid" form:"comment_uuid"`
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
