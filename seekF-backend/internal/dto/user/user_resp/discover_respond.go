package userresp

type PostItem struct {
	Uuid        string   `json:"uuid"`
	UserId      string   `json:"user_id"`
	Nickname    string   `json:"nickname"`
	Avatar      string   `json:"avatar"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	MediaType   int8     `json:"media_type"`
	Tags        []string `json:"tags"`
	FirstUrl    string   `json:"first_url"`
	LikeCount   int      `json:"like_count"`
	CommentCount int     `json:"comment_count"`
	CreatedAt   string   `json:"created_at"`
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
	Tags         []string `json:"tags"`
	Urls         []string `json:"urls"`
	LikeCount    int      `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	IsLiked      bool     `json:"is_liked"`
	CreatedAt    string   `json:"created_at"`
}

type CommentItem struct {
	Uuid          string `json:"uuid"`
	UserId        string `json:"user_id"`
	Nickname      string `json:"nickname"`
	Avatar        string `json:"avatar"`
	ParentId      int64  `json:"parent_id"`
	ReplyToUserId string `json:"reply_to_user_id"`
	Content       string `json:"content"`
	LikeCount     int    `json:"like_count"`
	CreatedAt     string `json:"created_at"`
}

type ListCommentsRespond struct {
	List []CommentItem `json:"list"`
}
