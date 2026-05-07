package userreq

type CreatePostRequest struct {
	Title     string   `json:"title" form:"title"`
	Content   string   `json:"content" form:"content"`
	MediaType int8     `json:"media_type" form:"media_type"`
	Tags      []string `json:"tags" form:"tags"`
	Urls      []string `json:"urls" form:"urls"`
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
	ParentId       int64  `json:"parent_id" form:"parent_id"`
	ReplyToUserId  string `json:"reply_to_user_id" form:"reply_to_user_id"`
	Content        string `json:"content" form:"content"`
}

type ListCommentsRequest struct {
	PostUuid string `json:"post_uuid" form:"post_uuid"`
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
}
