package userresp

type PostItem struct {
	Uuid         string   `json:"uuid"`
	UserId       string   `json:"user_id"`
	Nickname     string   `json:"nickname"`
	Avatar       string   `json:"avatar"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	MediaType    int8     `json:"media_type"`
	CoverUrl     string   `json:"cover_url"`
	Tags         []string `json:"tags"`
	FirstUrl     string   `json:"first_url"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	CollectCount int      `json:"collect_count"`
	IsLiked      bool     `json:"is_liked"`
	IsCollected  bool     `json:"is_collected"`
	CreatedAt    string   `json:"created_at"`
}

type ListPostsRespond struct {
	List  []PostItem `json:"list"`
	Total int64      `json:"total"`
}

type PostDetailRespond struct {
	Uuid         string   `json:"uuid"`
	UserId       string   `json:"user_id"`
	Nickname     string   `json:"nickname"`
	Avatar       string   `json:"avatar"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	MediaType    int8     `json:"media_type"`
	CoverUrl     string   `json:"cover_url"`
	Tags         []string `json:"tags"`
	Urls         []string `json:"urls"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	CollectCount int      `json:"collect_count"`
	IsLiked      bool     `json:"is_liked"`
	IsCollected  bool     `json:"is_collected"`
	CreatedAt    string   `json:"created_at"`
}

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
}

type ListCommentsRespond struct {
	List []CommentItem `json:"list"`
}

type AICommentRespond struct {
	UserComment CommentItem `json:"user_comment"`
}

type FolderItem struct {
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
