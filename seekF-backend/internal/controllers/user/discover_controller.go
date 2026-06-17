package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
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

	postInfo, err := c.discoverService.CreatePost(ctx.Request.Context(), userId, req.Title, req.Content, req.MediaType, req.Tags, req.Urls, req.CoverUrl)
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
		CoverUrl:  postInfo.CoverUrl,
		Tags:      postInfo.Tags,
		FirstUrl:  postInfo.FirstUrl,
		CreatedAt: postInfo.CreatedAt,
	})
}

// ListPosts 获取帖子列表
func (c *DiscoverController) ListPosts(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")

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

	posts, total, err := c.discoverService.ListPosts(ctx.Request.Context(), userId, req.Page, req.PageSize)
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
			CoverUrl:     p.CoverUrl,
			Tags:         p.Tags,
			FirstUrl:     p.FirstUrl,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			IsLiked:      p.IsLiked,
			IsCollected:  p.IsCollected,
			CreatedAt:    p.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListPostsRespond{
		List:  items,
		Total: total,
	})
}

// SearchPosts 搜索帖子
func (c *DiscoverController) SearchPosts(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")

	var req userreq.SearchPostsRequest
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

	posts, total, err := c.discoverService.SearchPosts(ctx.Request.Context(), userId, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		zlog.Error("搜索帖子失败: " + err.Error())
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
			CoverUrl:     p.CoverUrl,
			Tags:         p.Tags,
			FirstUrl:     p.FirstUrl,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			IsLiked:      p.IsLiked,
			IsCollected:  p.IsCollected,
			CreatedAt:    p.CreatedAt,
		})
	}

	resp.Success(ctx, "搜索成功", userresp.ListPostsRespond{
		List:  items,
		Total: total,
	})
}

// ListLikedPosts 获取用户点赞的帖子列表
func (c *DiscoverController) ListLikedPosts(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

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

	posts, total, err := c.discoverService.ListLikedPosts(ctx.Request.Context(), userId, req.Page, req.PageSize)
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
			CoverUrl:     p.CoverUrl,
			Tags:         p.Tags,
			FirstUrl:     p.FirstUrl,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			IsLiked:      p.IsLiked,
			IsCollected:  p.IsCollected,
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
		CoverUrl:     detail.CoverUrl,
		Tags:         detail.Tags,
		Urls:         detail.Urls,
		LikeCount:    detail.LikeCount,
		CommentCount: detail.CommentCount,
		CollectCount: detail.CollectCount,
		IsLiked:      detail.IsLiked,
		IsCollected:  detail.IsCollected,
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

	isLiked, likeCount, err := c.discoverService.ToggleLike(ctx.Request.Context(), userId, req.TargetUuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	if isLiked {
		resp.Success(ctx, "点赞成功", gin.H{"is_liked": true, "like_count": likeCount})
	} else {
		resp.Success(ctx, "取消点赞", gin.H{"is_liked": false, "like_count": likeCount})
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

// AddAIComment 添加@AI助手评论（异步，AI稍后回复）
func (c *DiscoverController) AddAIComment(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.AICommentRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}

	comment, err := c.discoverService.AddAIComment(ctx.Request.Context(), userId, req.PostUuid, req.Content, req.AIQuestion, req.ParentUuid, req.ReplyToUserId, req.ReplyToContent)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "评论已发送，AI助手稍后回复", userresp.CommentItem{
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
	userId := ctx.GetString("Uuid")

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

	comments, err := c.discoverService.ListComments(ctx.Request.Context(), userId, req.PostUuid, req.Page, req.PageSize)
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
			IsLiked:         c.IsLiked,
			CreatedAt:       c.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListCommentsRespond{
		List: items,
	})
}

// ToggleCommentLike 切换评论点赞状态
func (c *DiscoverController) ToggleCommentLike(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.ToggleCommentLikeRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.CommentUuid == "" {
		resp.Error(ctx, "comment_uuid不能为空", http.StatusBadRequest)
		return
	}

	isLiked, likeCount, err := c.discoverService.ToggleCommentLike(ctx.Request.Context(), userId, req.CommentUuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	if isLiked {
		resp.Success(ctx, "点赞成功", gin.H{"is_liked": true, "like_count": likeCount})
	} else {
		resp.Success(ctx, "取消点赞", gin.H{"is_liked": false, "like_count": likeCount})
	}
}

// ========== 收藏夹 ==========

// CreateFolder 创建收藏夹
func (c *DiscoverController) CreateFolder(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.CreateFolderRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		resp.Error(ctx, "收藏夹名称不能为空", http.StatusBadRequest)
		return
	}

	folder, err := c.discoverService.CreateFolder(ctx.Request.Context(), userId, req.Name, req.Description, req.IsPublic)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "创建成功", userresp.FolderItem{
		Uuid:        folder.Uuid,
		Name:        folder.Name,
		Description: folder.Description,
		IsPublic:    folder.IsPublic,
		PostCount:   folder.PostCount,
		CreatedAt:   folder.CreatedAt,
	})
}

// UpdateFolder 更新收藏夹
func (c *DiscoverController) UpdateFolder(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.UpdateFolderRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Uuid == "" {
		resp.Error(ctx, "uuid不能为空", http.StatusBadRequest)
		return
	}

	if err := c.discoverService.UpdateFolder(ctx.Request.Context(), userId, req.Uuid, req.Name, req.Description, req.IsPublic); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "更新成功", nil)
}

// DeleteFolder 删除收藏夹
func (c *DiscoverController) DeleteFolder(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.DeleteFolderRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Uuid == "" {
		resp.Error(ctx, "uuid不能为空", http.StatusBadRequest)
		return
	}

	if err := c.discoverService.DeleteFolder(ctx.Request.Context(), userId, req.Uuid); err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除成功", nil)
}

// ListFolders 获取收藏夹列表
func (c *DiscoverController) ListFolders(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	folders, err := c.discoverService.ListFolders(ctx.Request.Context(), userId)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	var items []userresp.FolderItem
	for _, f := range folders {
		items = append(items, userresp.FolderItem{
			Uuid:        f.Uuid,
			Name:        f.Name,
			Description: f.Description,
			IsPublic:    f.IsPublic,
			PostCount:   f.PostCount,
			CoverUrl:    f.CoverUrl,
			CreatedAt:   f.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListFoldersRespond{List: items})
}

// GetFolderDetail 获取收藏夹详情
func (c *DiscoverController) GetFolderDetail(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")

	var req userreq.GetFolderDetailRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.Uuid == "" {
		resp.Error(ctx, "uuid不能为空", http.StatusBadRequest)
		return
	}

	folder, err := c.discoverService.GetFolderDetail(ctx.Request.Context(), userId, req.Uuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", gin.H{
		"uuid":        folder.Uuid,
		"name":        folder.Name,
		"description": folder.Description,
		"is_public":   folder.IsPublic,
		"post_count":  folder.PostCount,
		"created_at":  folder.CreatedAt,
	})
}

// ListCollectedPosts 获取收藏夹中的帖子列表
func (c *DiscoverController) ListCollectedPosts(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")

	var req userreq.ListCollectedPostsRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.FolderUuid == "" {
		resp.Error(ctx, "folder_uuid不能为空", http.StatusBadRequest)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 12
	}

	posts, total, err := c.discoverService.ListCollectedPosts(ctx.Request.Context(), userId, req.FolderUuid, req.Page, req.PageSize)
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
			CoverUrl:     p.CoverUrl,
			Tags:         p.Tags,
			FirstUrl:     p.FirstUrl,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			IsLiked:      p.IsLiked,
			IsCollected:  p.IsCollected,
			CreatedAt:    p.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.ListPostsRespond{
		List:  items,
		Total: total,
	})
}

// CollectPost 收藏帖子
func (c *DiscoverController) CollectPost(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.CollectPostRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.PostUuid == "" || req.FolderUuid == "" {
		resp.Error(ctx, "缺少必要参数", http.StatusBadRequest)
		return
	}

	isCollected, collectCount, err := c.discoverService.CollectPost(ctx.Request.Context(), userId, req.PostUuid, req.FolderUuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "收藏成功", gin.H{"is_collected": isCollected, "collect_count": collectCount})
}

// UncollectPost 取消收藏
func (c *DiscoverController) UncollectPost(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.UncollectPostRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.PostUuid == "" || req.FolderUuid == "" {
		resp.Error(ctx, "缺少必要参数", http.StatusBadRequest)
		return
	}

	isCollected, collectCount, err := c.discoverService.UncollectPost(ctx.Request.Context(), userId, req.PostUuid, req.FolderUuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "取消收藏成功", gin.H{"is_collected": isCollected, "collect_count": collectCount})
}

// CheckCollected 检查帖子是否已收藏
func (c *DiscoverController) CheckCollected(ctx *gin.Context) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "获取用户信息失败", http.StatusBadRequest)
		return
	}

	var req userreq.CheckCollectedRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	if req.PostUuid == "" {
		resp.Error(ctx, "post_uuid不能为空", http.StatusBadRequest)
		return
	}

	isCollected, folderUuid, err := c.discoverService.CheckCollected(ctx.Request.Context(), userId, req.PostUuid)
	if err != nil {
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", gin.H{"is_collected": isCollected, "folder_uuid": folderUuid})
}
