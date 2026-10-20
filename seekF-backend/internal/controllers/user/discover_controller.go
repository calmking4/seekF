package user

import (
	"errors"
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

// DiscoverController 发现页内容与互动请求控制器。
type DiscoverController struct {
	discoverService userservice.DiscoverService
}

// NewDiscoverController 创建发现页控制器。
func NewDiscoverController(discoverService userservice.DiscoverService) *DiscoverController {
	return &DiscoverController{
		discoverService: discoverService,
	}
}

// ========== 帖子 ==========

// SuggestCreation 根据已上传图片返回待用户采纳的标题和标签候选。
func (c *DiscoverController) SuggestCreation(ctx *gin.Context) {
	var req userreq.DiscoverAISuggestionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "图片创作建议参数错误", http.StatusBadRequest)
		return
	}
	result, err := c.discoverService.SuggestDiscoverCreation(ctx.Request.Context(), ctx.GetString("Uuid"), req)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "生成创作建议成功", result)
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

	if len(req.MediaSizes) != 0 && len(req.MediaSizes) != len(req.Urls) {
		resp.Error(ctx, "媒体尺寸与文件数量不一致", http.StatusBadRequest)
		return
	}
	postInfo, err := c.discoverService.CreatePost(ctx.Request.Context(), userId, req)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}

	resp.Success(ctx, "发布成功", userresp.PostItem{
		IsBlockedBy: postInfo.IsBlockedBy, SupportOnly: postInfo.SupportOnly, CanPlay: postInfo.CanPlay, LockReason: postInfo.LockReason, SupportExpiresAt: postInfo.SupportExpiresAt, Visibility: postInfo.Visibility, Version: postInfo.Version,
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
		Width:     postInfo.Width,
		Height:    postInfo.Height,
		CreatedAt: postInfo.CreatedAt,
	})
}

// ListFeed 返回公开时间流或当前登录用户的关注流。
func (c *DiscoverController) ListFeed(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	var req userreq.FeedRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "信息流参数错误", http.StatusBadRequest)
		return
	}
	result, err := c.discoverService.ListFeed(ctx.Request.Context(), ctx.GetString("Uuid"), req)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	items := make([]userresp.PostItem, 0, len(result.List))
	for _, p := range result.List {
		items = append(items, userresp.PostItem{
			IsBlockedBy: p.IsBlockedBy, SupportOnly: p.SupportOnly, CanPlay: p.CanPlay, LockReason: p.LockReason, SupportExpiresAt: p.SupportExpiresAt, Visibility: p.Visibility, Version: p.Version,
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
			Width:        p.Width,
			Height:       p.Height,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			IsLiked:      p.IsLiked,
			IsCollected:  p.IsCollected,
			CreatedAt:    p.CreatedAt,
		})
	}
	resp.Success(ctx, "获取成功", userresp.FeedRespond{List: items, NextCursor: result.NextCursor, HasMore: result.HasMore, HasFollowing: result.HasFollowing})
}

// FollowingUpdates 返回关注栏作者和有更新的作者数。
func (c *DiscoverController) FollowingUpdates(ctx *gin.Context) {
	result, err := c.discoverService.FollowingUpdates(ctx.Request.Context(), ctx.GetString("Uuid"))
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "获取关注作者更新成功", result)
}

// ReadFollowingUpdates 确认已成功查看指定作者快照内的更新。
func (c *DiscoverController) ReadFollowingUpdates(ctx *gin.Context) {
	var req userreq.FollowingReadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "作者更新参数错误", http.StatusBadRequest)
		return
	}
	if err := c.discoverService.ReadFollowingUpdates(ctx.Request.Context(), ctx.GetString("Uuid"), req.Token); err != nil {
		if errors.Is(err, userservice.ErrFollowingReadExpired) {
			resp.Error(ctx, err.Error(), http.StatusGone)
		} else {
			resp.Error(ctx, "同步作者更新状态失败，请稍后重试", http.StatusServiceUnavailable)
		}
		return
	}
	resp.Success(ctx, "已查看作者更新", nil)
}

// SearchPosts 搜索帖子
func (c *DiscoverController) SearchPosts(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
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
		respondDiscoverError(ctx, err)
		return
	}

	var items []userresp.PostItem
	for _, p := range posts {
		items = append(items, userresp.PostItem{
			IsBlockedBy: p.IsBlockedBy, SupportOnly: p.SupportOnly, CanPlay: p.CanPlay, LockReason: p.LockReason, SupportExpiresAt: p.SupportExpiresAt, Visibility: p.Visibility, Version: p.Version,
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
			Width:        p.Width,
			Height:       p.Height,
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
	ctx.Header("Cache-Control", "no-store")
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
		respondDiscoverError(ctx, err)
		return
	}

	var items []userresp.PostItem
	for _, p := range posts {
		items = append(items, userresp.PostItem{
			IsBlockedBy: p.IsBlockedBy, SupportOnly: p.SupportOnly, CanPlay: p.CanPlay, LockReason: p.LockReason, SupportExpiresAt: p.SupportExpiresAt, Visibility: p.Visibility, Version: p.Version,
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
			Width:        p.Width,
			Height:       p.Height,
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
	ctx.Header("Cache-Control", "no-store")
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
		respondDiscoverError(ctx, err)
		return
	}

	resp.Success(ctx, "获取成功", userresp.PostDetailRespond{
		SupportOnly: detail.SupportOnly, CanPlay: detail.CanPlay, LockReason: detail.LockReason, SupportExpiresAt: detail.SupportExpiresAt, Visibility: detail.Visibility, Version: detail.Version,
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
		ViewCount:    detail.ViewCount,
		IsLiked:      detail.IsLiked,
		IsCollected:  detail.IsCollected,
		CreatedAt:    detail.CreatedAt,
	})
}

// ========== 点赞 ==========

// SetPostLike 明确设置帖子的点赞状态。
func (c *DiscoverController) SetPostLike(ctx *gin.Context) { c.setDiscoverLike(ctx, false) }

// SetCommentLike 明确设置评论的点赞状态。
func (c *DiscoverController) SetCommentLike(ctx *gin.Context) { c.setDiscoverLike(ctx, true) }

// setDiscoverLike 校验目标状态并返回事务内确定的点赞结果。
func (c *DiscoverController) setDiscoverLike(ctx *gin.Context, isComment bool) {
	userId := ctx.GetString("Uuid")
	if userId == "" {
		resp.Error(ctx, "请先登录", http.StatusUnauthorized)
		return
	}
	var req userreq.SetDiscoverLikeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "请提供目标和明确的点赞状态", http.StatusBadRequest)
		return
	}
	liked, count, err := c.discoverService.SetLike(ctx.Request.Context(), userId, req.TargetUuid, *req.IsLiked, isComment)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "点赞状态已更新", gin.H{"is_liked": liked, "like_count": count})
}

// ========== 评论 ==========

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
		respondDiscoverError(ctx, err)
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
		respondDiscoverError(ctx, err)
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
		respondDiscoverError(ctx, err)
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

// ListCommentThreads 分页读取顶级评论及各自回复数量。
func (c *DiscoverController) ListCommentThreads(ctx *gin.Context) {
	c.listDiscoverCommentThread(ctx, false)
}

// ListCommentReplies 分页读取指定顶级评论的回复。
func (c *DiscoverController) ListCommentReplies(ctx *gin.Context) {
	c.listDiscoverCommentThread(ctx, true)
}

// listDiscoverCommentThread 统一处理评论线程分页与响应装配。
func (c *DiscoverController) listDiscoverCommentThread(ctx *gin.Context, replies bool) {
	var req userreq.CommentThreadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "评论分页参数错误", http.StatusBadRequest)
		return
	}
	if replies && req.ParentUuid == "" {
		resp.Error(ctx, "请选择需要查看回复的评论", http.StatusBadRequest)
		return
	}
	if !replies {
		req.ParentUuid = ""
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 50 {
		req.PageSize = 20
	}
	comments, hasMore, err := c.discoverService.ListCommentThread(ctx.Request.Context(), ctx.GetString("Uuid"), req.PostUuid, req.ParentUuid, req.Page, req.PageSize)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	items := make([]userresp.CommentItem, 0, len(comments))
	for _, comment := range comments {
		items = append(items, userresp.CommentItem{Uuid: comment.Uuid, UserId: comment.UserId, Nickname: comment.Nickname, Avatar: comment.Avatar, ParentId: comment.ParentId, ReplyToUserId: comment.ReplyToUserId, ReplyToNickname: comment.ReplyToNickname, Content: comment.Content, LikeCount: comment.LikeCount, IsLiked: comment.IsLiked, ReplyCount: comment.ReplyCount, CreatedAt: comment.CreatedAt})
	}
	resp.Success(ctx, "获取成功", userresp.CommentThreadRespond{List: items, HasMore: hasMore})
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
		respondDiscoverError(ctx, err)
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
		respondDiscoverError(ctx, err)
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
		respondDiscoverError(ctx, err)
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

	var req userreq.ListFoldersRequest
	// 原有调用可能没有请求体，继续默认读取自己的收藏夹。
	if ctx.Request.ContentLength != 0 {
		if err := ctx.ShouldBind(&req); err != nil {
			resp.Error(ctx, "参数错误", http.StatusBadRequest)
			return
		}
	}
	ownerId := req.UserId
	if ownerId == "" {
		ownerId = userId
	}
	folders, err := c.discoverService.ListFolders(ctx.Request.Context(), ownerId, userId)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}

	var items []userresp.FolderItem
	for _, f := range folders {
		if ownerId != userId && !f.IsPublic {
			continue
		}
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
		respondDiscoverError(ctx, err)
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

// ========== 收藏记录 ==========

// ListCollectedPosts 获取收藏夹中的帖子列表
func (c *DiscoverController) ListCollectedPosts(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
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
		respondDiscoverError(ctx, err)
		return
	}

	var items []userresp.PostItem
	for _, p := range posts {
		items = append(items, userresp.PostItem{
			IsBlockedBy: p.IsBlockedBy, SupportOnly: p.SupportOnly, CanPlay: p.CanPlay, LockReason: p.LockReason, SupportExpiresAt: p.SupportExpiresAt, Visibility: p.Visibility, Version: p.Version,
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
			Width:        p.Width,
			Height:       p.Height,
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
		respondDiscoverError(ctx, err)
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
		respondDiscoverError(ctx, err)
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
		respondDiscoverError(ctx, err)
		return
	}

	resp.Success(ctx, "获取成功", gin.H{"is_collected": isCollected, "folder_uuid": folderUuid})
}

// ========== 用户主页 ==========

// GetUserProfile 获取用户主页信息
func (c *DiscoverController) GetUserProfile(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	userId := ctx.GetString("Uuid")

	var req userreq.GetUserProfileRequest
	if err := ctx.ShouldBind(&req); err != nil {
		resp.Error(ctx, "参数错误", http.StatusBadRequest)
		return
	}
	// 如果未指定用户ID，默认查自己
	if req.UserId == "" {
		req.UserId = userId
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 12
	}

	profile, err := c.discoverService.GetUserProfile(ctx.Request.Context(), req.UserId, userId, req.Page, req.PageSize)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}

	var items []userresp.PostItem
	for _, p := range profile.Posts {
		items = append(items, userresp.PostItem{
			IsBlockedBy: p.IsBlockedBy, SupportOnly: p.SupportOnly, CanPlay: p.CanPlay, LockReason: p.LockReason, SupportExpiresAt: p.SupportExpiresAt, Visibility: p.Visibility, Version: p.Version,
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
			Width:        p.Width,
			Height:       p.Height,
			LikeCount:    p.LikeCount,
			CommentCount: p.CommentCount,
			CollectCount: p.CollectCount,
			IsLiked:      p.IsLiked,
			IsCollected:  p.IsCollected,
			CreatedAt:    p.CreatedAt,
		})
	}

	resp.Success(ctx, "获取成功", userresp.UserProfileRespond{
		Uuid:           profile.Uuid,
		Nickname:       profile.Nickname,
		Avatar:         profile.Avatar,
		Signature:      profile.Signature,
		PostCount:      profile.PostCount,
		TotalLikes:     profile.TotalLikes,
		FollowingCount: profile.FollowingCount,
		FollowerCount:  profile.FollowerCount,
		IsFollowed:     profile.IsFollowed,
		IsBlocked:      profile.IsBlocked,
		IsBlockedBy:    profile.IsBlockedBy,
		Posts:          items,
		Total:          profile.Total,
	})
}

// ========== 作品管理 ==========

// UpdatePost 保存作者编辑的作品，版本冲突返回409。
func (c *DiscoverController) UpdatePost(ctx *gin.Context) {
	var req userreq.UpdatePostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "作品参数错误", http.StatusBadRequest)
		return
	}
	post, err := c.discoverService.UpdatePost(ctx.Request.Context(), ctx.GetString("Uuid"), req.PostUuid, req.CreatePostRequest)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "作品已更新", gin.H{"uuid": post.Uuid, "version": post.Version})
}

// DeletePost 删除当前作者的作品。
func (c *DiscoverController) DeletePost(ctx *gin.Context) {
	var req struct {
		PostUuid string `json:"post_uuid" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "请选择需要删除的作品", http.StatusBadRequest)
		return
	}
	if err := c.discoverService.DeletePost(ctx.Request.Context(), ctx.GetString("Uuid"), req.PostUuid); err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "作品已删除", nil)
}

// respondDiscoverError 区分参数错误、编辑冲突和推荐会话过期。
func respondDiscoverError(ctx *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, userservice.ErrDiscoverConflict) {
		status = http.StatusConflict
	}
	if errors.Is(err, userservice.ErrDiscoverSessionExpired) {
		status = http.StatusGone
	}
	resp.Error(ctx, err.Error(), status)
}

// RecordClick 记录用户主动打开帖子详情的点击，详情读取不自动产生该行为。
func (c *DiscoverController) RecordClick(ctx *gin.Context) {
	var input userreq.DiscoverClickRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Error(ctx, "帖子点击参数错误", http.StatusBadRequest)
		return
	}
	if err := c.discoverService.RecordDiscoverClick(ctx.Request.Context(), ctx.GetString("Uuid"), input); err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "点击已记录", nil)
}

// SetFeedback 设置或撤销当前用户对帖子或作者的不感兴趣。
func (c *DiscoverController) SetFeedback(ctx *gin.Context) {
	var input userreq.DiscoverFeedbackRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Error(ctx, "不感兴趣参数错误", http.StatusBadRequest)
		return
	}
	authorId, err := c.discoverService.SetDiscoverFeedback(ctx.Request.Context(), ctx.GetString("Uuid"), input)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "推荐偏好已更新", gin.H{"post_uuid": input.PostUuid, "author_uuid": authorId, "target_type": input.TargetType, "active": *input.Active})
}

// ListOwnWorks 返回当前登录作者的作品及筛选分页总数。
func (c *DiscoverController) ListOwnWorks(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	var req userreq.ListOwnWorksRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "作品筛选参数错误", http.StatusBadRequest)
		return
	}
	posts, total, counts, err := c.discoverService.ListOwnWorks(ctx.Request.Context(), ctx.GetString("Uuid"), req)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	items := make([]userresp.PostItem, 0, len(posts))
	for _, p := range posts {
		items = append(items, userresp.PostItem{ReviewNote: p.ReviewNote, Status: p.Status, Uuid: p.Uuid, UserId: p.UserId, Nickname: p.Nickname, Avatar: p.Avatar, Title: p.Title, Content: p.Content, MediaType: p.MediaType, CoverUrl: p.CoverUrl, Tags: p.Tags, FirstUrl: p.FirstUrl, Width: p.Width, Height: p.Height, ViewCount: p.ViewCount, LikeCount: p.LikeCount, CommentCount: p.CommentCount, CollectCount: p.CollectCount, IsLiked: p.IsLiked, IsCollected: p.IsCollected, IsBlockedBy: p.IsBlockedBy, SupportOnly: p.SupportOnly, CanPlay: p.CanPlay, LockReason: p.LockReason, SupportExpiresAt: p.SupportExpiresAt, Visibility: p.Visibility, Version: p.Version, CreatedAt: p.CreatedAt})
	}
	resp.Success(ctx, "获取作品成功", gin.H{"list": items, "total": total, "counts": counts})
}

// SetPostVisibility 修改作者作品的公开范围，版本冲突返回409。
func (c *DiscoverController) SetPostVisibility(ctx *gin.Context) {
	var req userreq.SetPostVisibilityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Error(ctx, "作品可见性参数错误", http.StatusBadRequest)
		return
	}
	version, err := c.discoverService.SetPostVisibility(ctx.Request.Context(), ctx.GetString("Uuid"), req)
	if err != nil {
		respondDiscoverError(ctx, err)
		return
	}
	resp.Success(ctx, "可见性已更新", gin.H{"visibility": *req.Visibility, "version": version})
}
