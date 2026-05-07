package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/resp"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type DiscoverController struct {
	discoverService userservice.DiscoverService
}

func NewDiscoverController(discoverService userservice.DiscoverService) *DiscoverController {
	return &DiscoverController{
		discoverService: discoverService,
	}
}

// CreatePost 发布帖子
func (c *DiscoverController) CreatePost(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.CreatePostRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		resp.Error(ctx, "标题不能为空", http.StatusBadRequest)
		return
	}
	if len(req.Urls) == 0 {
		resp.Error(ctx, "请上传媒体文件", http.StatusBadRequest)
		return
	}

	postInfo, err := c.discoverService.CreatePost(ctx.Request.Context(), userId, req.Title, req.Content, req.MediaType, req.Tags, req.Urls)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "发布成功", userresp.PostItem{
		Uuid:      postInfo.Uuid,
		UserId:    postInfo.UserId,
		Nickname:  postInfo.Nickname,
		Avatar:    postInfo.Avatar,
		Title:     postInfo.Title,
		Content:   postInfo.Content,
		MediaType: postInfo.MediaType,
		Tags:      postInfo.Tags,
		FirstUrl:  postInfo.FirstUrl,
		CreatedAt: postInfo.CreatedAt,
	})
}

// ListPosts 获取帖子列表
func (c *DiscoverController) ListPosts(ctx *gin.Context) {
	var req userreq.ListPostsRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 12
	}

	posts, total, err := c.discoverService.ListPosts(ctx.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	var items []userresp.PostItem
	for _, p := range posts {
		items = append(items, userresp.PostItem{
			Uuid:         p.Uuid,
			UserId:       p.UserId,
			Nickname:     p.Nickname,
			Avatar:       p.Avatar,
			Title:        p.Title,
			Content:      p.Content,
			MediaType:    p.MediaType,
			Tags:         p.Tags,
			FirstUrl:     p.FirstUrl,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CreatedAt:    p.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListPostsRespond{
		List:  items,
		Total: total,
	})
}

// GetPostDetail 获取帖子详情
func (c *DiscoverController) GetPostDetail(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.GetPostDetailRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Uuid == "" {
		resp.Error(ctx, "uuid不能为空", http.StatusBadRequest)
		return
	}

	detail, err := c.discoverService.GetPostDetail(ctx.Request.Context(), userId, req.Uuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", userresp.PostDetailRespond{
		Uuid:         detail.Uuid,
		UserId:       detail.UserId,
		Nickname:     detail.Nickname,
		Avatar:       detail.Avatar,
		Title:        detail.Title,
		Content:      detail.Content,
		MediaType:    detail.MediaType,
		Tags:         detail.Tags,
		Urls:         detail.Urls,
		LikeCount:    detail.LikeCount,
		CommentCount: detail.CommentCount,
		IsLiked:      detail.IsLiked,
		CreatedAt:    detail.CreatedAt,
	})
}

// ToggleLike 切换点赞状态
func (c *DiscoverController) ToggleLike(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.ToggleLikeRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.TargetUuid == "" {
		resp.Error(ctx, "target_uuid不能为空", http.StatusBadRequest)
		return
	}

	isLiked, err := c.discoverService.ToggleLike(ctx.Request.Context(), userId, req.TargetUuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	if isLiked {
		resp.Success(ctx, "点赞成功", gin.H{"is_liked": true})
	} else {
		resp.Success(ctx, "取消点赞", gin.H{"is_liked": false})
	}
}

// AddComment 添加评论
func (c *DiscoverController) AddComment(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.AddCommentRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.PostUuid == "" || req.Content == "" {
		resp.Error(ctx, "缺少必要参数", http.StatusBadRequest)
		return
	}

	comment, err := c.discoverService.AddComment(ctx.Request.Context(), userId, req.PostUuid, req.ParentUuid, req.ReplyToUserId, req.Content)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "评论成功", userresp.CommentItem{
		Uuid:            comment.Uuid,
		UserId:          comment.UserId,
		Nickname:        comment.Nickname,
		Avatar:          comment.Avatar,
		ParentId:        comment.ParentId,
		ReplyToUserId:   comment.ReplyToUserId,
		ReplyToNickname: comment.ReplyToNickname,
		Content:         comment.Content,
		LikeCount:       comment.LikeCount,
		CreatedAt:       comment.CreatedAt,
	})
}

// ListComments 获取评论列表
func (c *DiscoverController) ListComments(ctx *gin.Context) {
	var req userreq.ListCommentsRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.PostUuid == "" {
		resp.Error(ctx, "post_uuid不能为空", http.StatusBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 50 {
		req.PageSize = 20
	}

	comments, err := c.discoverService.ListComments(ctx.Request.Context(), req.PostUuid, req.Page, req.PageSize)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	var items []userresp.CommentItem
	for _, c := range comments {
		items = append(items, userresp.CommentItem{
			Uuid:            c.Uuid,
			UserId:          c.UserId,
			Nickname:        c.Nickname,
			Avatar:          c.Avatar,
			ParentId:        c.ParentId,
			ReplyToUserId:   c.ReplyToUserId,
			ReplyToNickname: c.ReplyToNickname,
			Content:         c.Content,
			LikeCount:       c.LikeCount,
			CreatedAt:       c.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListCommentsRespond{
		List: items,
	})
}
