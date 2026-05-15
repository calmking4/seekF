package userservice

import (
	"context"
	"encoding/json"
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	aipkg "seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/util"
)

type DiscoverService interface {
	CreatePost(ctx context.Context, userId, title, content string, mediaType int8, tags []string, urls []string) (*PostInfo, error)
	ListPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error)
	ToggleLike(ctx context.Context, userId, targetUuid string) (bool, int, error)
	AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error)
	ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error)
	ToggleCommentLike(ctx context.Context, userId, commentUuid string) (bool, int, error)
	AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error)

	// 收藏夹
	CreateFolder(ctx context.Context, userId, name, description string, isPublic int8) (*FolderInfo, error)
	UpdateFolder(ctx context.Context, userId, folderUuid, name, description string, isPublic int8) error
	DeleteFolder(ctx context.Context, userId, folderUuid string) error
	ListFolders(ctx context.Context, userId string) ([]FolderInfo, error)
	GetFolderDetail(ctx context.Context, userId, folderUuid string) (*FolderDetailInfo, error)
	ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int) ([]PostInfo, int64, error)

	// 收藏
	CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error)
	UncollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error)
	CheckCollected(ctx context.Context, userId, postUuid string) (bool, string, error)
}

type DiscoverServiceImpl struct {
	discoverDAO  userdao.DiscoverDAO
	userInfoDAO  userdao.UserInfoDAO
}

type PostInfo struct {
	Uuid         string
	UserId       string
	Nickname     string
	Avatar       string
	Title        string
	Content      string
	MediaType    int8
	Tags         []string
	FirstUrl     string
	LikeCount    int
	CommentCount int
	CollectCount int
	IsLiked      bool
	IsCollected  bool
	CreatedAt    string
}

type PostDetailInfo struct {
	Uuid         string
	UserId       string
	Nickname     string
	Avatar       string
	Title        string
	Content      string
	MediaType    int8
	Tags         []string
	Urls         []string
	LikeCount    int
	CommentCount int
	CollectCount int
	IsLiked      bool
	IsCollected  bool
	CreatedAt    string
}

type CommentInfo struct {
	Uuid            string
	UserId          string
	Nickname        string
	Avatar          string
	ParentId        string
	ReplyToUserId   string
	ReplyToNickname string
	Content         string
	LikeCount       int
	IsLiked         bool
	CreatedAt       string
}

type FolderInfo struct {
	Uuid        string
	Name        string
	Description string
	IsPublic    bool
	PostCount   int
	CoverUrl    string
	CreatedAt   string
}

type FolderDetailInfo struct {
	Uuid        string
	Name        string
	Description string
	IsPublic    bool
	PostCount   int
	CreatedAt   string
}

func NewDiscoverService(discoverDAO userdao.DiscoverDAO, userInfoDAO userdao.UserInfoDAO) DiscoverService {
	return &DiscoverServiceImpl{
		discoverDAO: discoverDAO,
		userInfoDAO: userInfoDAO,
	}
}

func (s *DiscoverServiceImpl) CreatePost(ctx context.Context, userId, title, content string, mediaType int8, tags []string, urls []string) (*PostInfo, error) {
	postUUID := "D" + util.GetNowAndLenRandomString(11)

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	post := &models.DiscoverPost{
		Uuid:      postUUID,
		UserId:    userId,
		Title:     title,
		Content:   content,
		MediaType: mediaType,
		Tags:      tagsJSON,
		Status:    0,
	}

	if err := s.discoverDAO.CreatePost(post); err != nil {
		return nil, fmt.Errorf("创建帖子失败: %v", err)
	}

	for i, url := range urls {
		media := &models.DiscoverMedia{
			PostId:    post.Id,
			Type:      mediaType,
			Url:       url,
			SortOrder: i,
		}
		if err := s.discoverDAO.CreateMedia(media); err != nil {
			return nil, fmt.Errorf("保存媒体失败: %v", err)
		}
	}

	user, _ := s.userInfoDAO.FindUserByUuid(userId)

	firstUrl := ""
	if len(urls) > 0 {
		firstUrl = urls[0]
	}

	return &PostInfo{
		Uuid:      postUUID,
		UserId:    userId,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Title:     title,
		Content:   content,
		MediaType: mediaType,
		Tags:      tags,
		FirstUrl:  firstUrl,
		CreatedAt: post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) ListPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error) {
	posts, err := s.discoverDAO.ListPosts(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountPosts()
	if err != nil {
		return nil, 0, err
	}

	if len(posts) == 0 {
		return []PostInfo{}, total, nil
	}

	// 收集所有帖子ID和用户ID，用于批量查询
	postIds := make([]int64, 0, len(posts))
	userIds := make([]string, 0, len(posts))
	postUuids := make([]string, 0, len(posts))
	for _, post := range posts {
		postIds = append(postIds, post.Id)
		userIds = append(userIds, post.UserId)
		postUuids = append(postUuids, post.Uuid)
	}

	// 批量查询媒体，构建 postId -> firstUrl 映射
	mediaList, _ := s.discoverDAO.FindMediaByPostIds(postIds)
	mediaMap := make(map[int64]string)
	for _, media := range mediaList {
		if _, exists := mediaMap[media.PostId]; !exists {
			mediaMap[media.PostId] = media.Url
		}
	}

	// 批量查询用户信息，构建 userId -> 用户信息映射
	users, _ := s.userInfoDAO.FindUsersByUuids(userIds)
	userMap := make(map[string]*models.UserInfo)
	for i := range users {
		userMap[users[i].Uuid] = &users[i]
	}

	// 批量查询点赞状态，构建 targetUuid -> 是否点赞映射
	likedMap := make(map[string]bool)
	if userId != "" {
		likes, _ := s.discoverDAO.FindLikesByUserIdAndTargetUuids(userId, postUuids)
		for _, like := range likes {
			likedMap[like.TargetUuid] = true
		}
	}

	// 批量查询收藏状态，构建 targetUuid -> 是否收藏映射
	collectedMap := make(map[string]bool)
	if userId != "" {
		collections, _ := s.discoverDAO.FindCollectionsByUserIdAndTargetUuids(userId, postUuids)
		for _, col := range collections {
			collectedMap[col.TargetUuid] = true
		}
	}

	var result []PostInfo
	for _, post := range posts {
		firstUrl := mediaMap[post.Id]

		var tags []string
		if len(post.Tags) > 0 {
			json.Unmarshal(post.Tags, &tags)
		}

		nickname := ""
		avatar := ""
		if user, exists := userMap[post.UserId]; exists {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		result = append(result, PostInfo{
			Uuid:         post.Uuid,
			UserId:       post.UserId,
			Nickname:     nickname,
			Avatar:       avatar,
			Title:        post.Title,
			Content:      post.Content,
			MediaType:    post.MediaType,
			Tags:         tags,
			FirstUrl:     firstUrl,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			CollectCount: post.CollectCount,
			IsLiked:      likedMap[post.Uuid],
			IsCollected:  collectedMap[post.Uuid],
			CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

func (s *DiscoverServiceImpl) ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error) {
	posts, err := s.discoverDAO.ListLikedPosts(userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountLikedPosts(userId)
	if err != nil {
		return nil, 0, err
	}

	var result []PostInfo
	for _, post := range posts {
		mediaList, _ := s.discoverDAO.FindMediaByPostId(post.Id)
		firstUrl := ""
		if len(mediaList) > 0 {
			firstUrl = mediaList[0].Url
		}

		var tags []string
		if len(post.Tags) > 0 {
			json.Unmarshal(post.Tags, &tags)
		}

		user, _ := s.userInfoDAO.FindUserByUuid(post.UserId)
		nickname := ""
		avatar := ""
		if user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		result = append(result, PostInfo{
			Uuid:         post.Uuid,
			UserId:       post.UserId,
			Nickname:     nickname,
			Avatar:       avatar,
			Title:        post.Title,
			Content:      post.Content,
			MediaType:    post.MediaType,
			Tags:         tags,
			FirstUrl:     firstUrl,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			CollectCount: post.CollectCount,
			IsLiked:      true, // 点赞列表中的帖子都是已点赞的
			CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

func (s *DiscoverServiceImpl) GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(uuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}

	mediaList, _ := s.discoverDAO.FindMediaByPostId(post.Id)
	var urls []string
	for _, m := range mediaList {
		urls = append(urls, m.Url)
	}

	var tags []string
	if len(post.Tags) > 0 {
		json.Unmarshal(post.Tags, &tags)
	}

	user, _ := s.userInfoDAO.FindUserByUuid(post.UserId)
	nickname := ""
	avatar := ""
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	isLiked := false
	like, _ := s.discoverDAO.FindLike(userId, uuid)
	if like != nil {
		isLiked = true
	}

	isCollected := false
	col, _ := s.discoverDAO.FindCollectionByUserAndTarget(userId, uuid)
	if col != nil {
		isCollected = true
	}

	return &PostDetailInfo{
		Uuid:         post.Uuid,
		UserId:       post.UserId,
		Nickname:     nickname,
		Avatar:       avatar,
		Title:        post.Title,
		Content:      post.Content,
		MediaType:    post.MediaType,
		Tags:         tags,
		Urls:         urls,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		CollectCount: post.CollectCount,
		IsLiked:      isLiked,
		IsCollected:  isCollected,
		CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) ToggleLike(ctx context.Context, userId, targetUuid string) (bool, int, error) {
	existing, _ := s.discoverDAO.FindLike(userId, targetUuid)
	if existing != nil {
		if err := s.discoverDAO.DeleteLike(userId, targetUuid); err != nil {
			return false, 0, err
		}
		post, _ := s.discoverDAO.FindPostByUuid(targetUuid)
		if post != nil {
			s.discoverDAO.DecrementLikeCount(post.Id)
			// 重新获取帖子以获取更新后的点赞数
			updatedPost, _ := s.discoverDAO.FindPostByUuid(targetUuid)
			if updatedPost != nil {
				return false, updatedPost.LikeCount, nil
			}
		}
		return false, 0, nil
	}

	like := &models.DiscoverLike{
		UserId:     userId,
		TargetUuid: targetUuid,
	}
	if err := s.discoverDAO.CreateLike(like); err != nil {
		return false, 0, err
	}

	post, _ := s.discoverDAO.FindPostByUuid(targetUuid)
	if post != nil {
		s.discoverDAO.IncrementLikeCount(post.Id)
		// 重新获取帖子以获取更新后的点赞数
		updatedPost, _ := s.discoverDAO.FindPostByUuid(targetUuid)
		if updatedPost != nil {
			return true, updatedPost.LikeCount, nil
		}
	}
	return true, 0, nil
}

func (s *DiscoverServiceImpl) AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}

	commentUUID := "C" + util.GetNowAndLenRandomString(11)
	comment := &models.DiscoverComment{
		Uuid:          commentUUID,
		PostId:        post.Id,
		UserId:        userId,
		ParentId:      parentUuid,
		ReplyToUserId: replyToUserId,
		Content:       content,
	}

	if err := s.discoverDAO.CreateComment(comment); err != nil {
		return nil, fmt.Errorf("评论失败: %v", err)
	}

	s.discoverDAO.IncrementCommentCount(post.Id)

	user, _ := s.userInfoDAO.FindUserByUuid(userId)
	nickname := ""
	avatar := ""
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	return &CommentInfo{
		Uuid:          commentUUID,
		UserId:        userId,
		Nickname:      nickname,
		Avatar:        avatar,
		ParentId:      parentUuid,
		ReplyToUserId: replyToUserId,
		Content:       content,
		CreatedAt:     comment.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("帖子不存在")
	}

	comments, err := s.discoverDAO.FindCommentsByPostId(post.Id, page, pageSize)
	if err != nil {
		return nil, err
	}

	var result []CommentInfo
	for _, c := range comments {
		user, _ := s.userInfoDAO.FindUserByUuid(c.UserId)
		nickname := ""
		avatar := ""
		if user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		replyToNickname := ""
		if c.ReplyToUserId != "" {
			replyToUser, _ := s.userInfoDAO.FindUserByUuid(c.ReplyToUserId)
			if replyToUser != nil {
				replyToNickname = replyToUser.Nickname
			}
		}

		// 查询当前用户是否点赞了该评论
		isLiked := false
		if userId != "" {
			like, _ := s.discoverDAO.FindLike(userId, c.Uuid)
			if like != nil {
				isLiked = true
			}
		}

		result = append(result, CommentInfo{
			Uuid:            c.Uuid,
			UserId:          c.UserId,
			Nickname:        nickname,
			Avatar:          avatar,
			ParentId:        c.ParentId,
			ReplyToUserId:   c.ReplyToUserId,
			ReplyToNickname: replyToNickname,
			Content:         c.Content,
			LikeCount:       c.LikeCount,
			IsLiked:         isLiked,
			CreatedAt:       c.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

func (s *DiscoverServiceImpl) ToggleCommentLike(ctx context.Context, userId, commentUuid string) (bool, int, error) {
	existing, _ := s.discoverDAO.FindLike(userId, commentUuid)
	if existing != nil {
		if err := s.discoverDAO.DeleteLike(userId, commentUuid); err != nil {
			return false, 0, err
		}
		// 通过 UUID 查询评论
		comment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
		if comment != nil {
			s.discoverDAO.DecrementCommentLikeCount(comment.Id)
			// 重新查询获取更新后的点赞数
			updatedComment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
			if updatedComment != nil {
				return false, updatedComment.LikeCount, nil
			}
		}
		return false, 0, nil
	}

	like := &models.DiscoverLike{
		UserId:     userId,
		TargetUuid: commentUuid,
	}
	if err := s.discoverDAO.CreateLike(like); err != nil {
		return false, 0, err
	}
	// 通过 UUID 查询评论
	comment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
	if comment != nil {
		s.discoverDAO.IncrementCommentLikeCount(comment.Id)
		// 重新查询获取更新后的点赞数
		updatedComment, _ := s.discoverDAO.FindCommentByUuid(commentUuid)
		if updatedComment != nil {
			return true, updatedComment.LikeCount, nil
		}
	}
	return true, 0, nil
}

func (s *DiscoverServiceImpl) AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error) {
	// 1. 保存用户评论（完整内容含@AI助手前缀）
	userComment, err := s.AddComment(ctx, userId, postUuid, parentUuid, replyToUserId, content)
	if err != nil {
		return nil, err
	}

	// 2. 确定AI评论的父级和回复目标
	aiParentUuid := parentUuid
	aiReplyToUserId := ""
	if aiParentUuid == "" {
		// 用户发的是顶级评论，AI回复嵌套在用户评论下，不显示"回复 XXX:"
		aiParentUuid = userComment.Uuid
	} else {
		// 用户回复别人时@AI，AI回复显示"回复 [用户昵称]:"
		aiReplyToUserId = userId
	}

	// 3. 发送Kafka任务
	aipkg.SendAICommentTask(aipkg.AICommentPayload{
		PostUuid:       postUuid,
		Content:        aiQuestion,
		ParentUuid:     aiParentUuid,
		ReplyToUserId:  aiReplyToUserId,
		ReplyToContent: replyToContent,
	})

	return userComment, nil
}

// ========== 收藏夹 ==========

func (s *DiscoverServiceImpl) CreateFolder(ctx context.Context, userId, name, description string, isPublic int8) (*FolderInfo, error) {
	folderUUID := "F" + util.GetNowAndLenRandomString(11)
	folder := &models.DiscoverCollectionFolder{
		Uuid:        folderUUID,
		UserId:      userId,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
	}
	if err := s.discoverDAO.CreateFolder(folder); err != nil {
		return nil, fmt.Errorf("创建收藏夹失败: %v", err)
	}
	return &FolderInfo{
		Uuid:        folderUUID,
		Name:        name,
		Description: description,
		IsPublic:    isPublic == 1,
		PostCount:   0,
		CreatedAt:   folder.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) UpdateFolder(ctx context.Context, userId, folderUuid, name, description string, isPublic int8) error {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return err
	}
	if folder == nil {
		return fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return fmt.Errorf("无权修改此收藏夹")
	}
	return s.discoverDAO.UpdateFolder(folderUuid, name, description, isPublic)
}

func (s *DiscoverServiceImpl) DeleteFolder(ctx context.Context, userId, folderUuid string) error {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return err
	}
	if folder == nil {
		return fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return fmt.Errorf("无权删除此收藏夹")
	}
	return s.discoverDAO.DeleteFolder(folderUuid)
}

func (s *DiscoverServiceImpl) ListFolders(ctx context.Context, userId string) ([]FolderInfo, error) {
	folders, err := s.discoverDAO.ListFoldersByUserId(userId)
	if err != nil {
		return nil, err
	}
	var result []FolderInfo
	for _, f := range folders {
		// 获取收藏夹第一张图片作为封面
		coverUrl := ""
		posts, _ := s.discoverDAO.ListCollectedPostsByFolder(f.Id, 1, 1)
		if len(posts) > 0 {
			mediaList, _ := s.discoverDAO.FindMediaByPostId(posts[0].Id)
			if len(mediaList) > 0 {
				coverUrl = mediaList[0].Url
			}
		}
		result = append(result, FolderInfo{
			Uuid:        f.Uuid,
			Name:        f.Name,
			Description: f.Description,
			IsPublic:    f.IsPublic == 1,
			PostCount:   f.PostCount,
			CoverUrl:    coverUrl,
			CreatedAt:   f.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

func (s *DiscoverServiceImpl) GetFolderDetail(ctx context.Context, userId, folderUuid string) (*FolderDetailInfo, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, fmt.Errorf("收藏夹不存在")
	}
	return &FolderDetailInfo{
		Uuid:        folder.Uuid,
		Name:        folder.Name,
		Description: folder.Description,
		IsPublic:    folder.IsPublic == 1,
		PostCount:   folder.PostCount,
		CreatedAt:   folder.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *DiscoverServiceImpl) ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int) ([]PostInfo, int64, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return nil, 0, err
	}
	if folder == nil {
		return nil, 0, fmt.Errorf("收藏夹不存在")
	}

	posts, err := s.discoverDAO.ListCollectedPostsByFolder(folder.Id, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountCollectedPostsByFolder(folder.Id)
	if err != nil {
		return nil, 0, err
	}

	var result []PostInfo
	for _, post := range posts {
		mediaList, _ := s.discoverDAO.FindMediaByPostId(post.Id)
		firstUrl := ""
		if len(mediaList) > 0 {
			firstUrl = mediaList[0].Url
		}
		var tags []string
		if len(post.Tags) > 0 {
			json.Unmarshal(post.Tags, &tags)
		}
		user, _ := s.userInfoDAO.FindUserByUuid(post.UserId)
		nickname := ""
		avatar := ""
		if user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}
		result = append(result, PostInfo{
			Uuid:         post.Uuid,
			UserId:       post.UserId,
			Nickname:     nickname,
			Avatar:       avatar,
			Title:        post.Title,
			Content:      post.Content,
			MediaType:    post.MediaType,
			Tags:         tags,
			FirstUrl:     firstUrl,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			CollectCount: post.CollectCount,
			IsLiked:      true,
			IsCollected:  true,
			CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, total, nil
}

// ========== 收藏/取消收藏 ==========

func (s *DiscoverServiceImpl) CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return false, 0, err
	}
	if post == nil {
		return false, 0, fmt.Errorf("帖子不存在")
	}

	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return false, 0, err
	}
	if folder == nil {
		return false, 0, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return false, 0, fmt.Errorf("无权操作此收藏夹")
	}

	// 检查是否已在该收藏夹中收藏
	existing, _ := s.discoverDAO.FindCollectionInFolder(userId, folder.Id, postUuid)
	if existing != nil {
		return false, 0, fmt.Errorf("该帖子已在此收藏夹中")
	}

	col := &models.DiscoverCollection{
		UserId:     userId,
		FolderId:   folder.Id,
		TargetUuid: postUuid,
	}
	if err := s.discoverDAO.CreateCollection(col); err != nil {
		return false, 0, err
	}

	s.discoverDAO.IncrementFolderPostCount(folder.Id)
	s.discoverDAO.IncrementCollectCount(post.Id)

	updatedPost, _ := s.discoverDAO.FindPostByUuid(postUuid)
	collectCount := 0
	if updatedPost != nil {
		collectCount = updatedPost.CollectCount
	}
	return true, collectCount, nil
}

func (s *DiscoverServiceImpl) UncollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return false, 0, err
	}
	if folder == nil {
		return false, 0, fmt.Errorf("收藏夹不存在")
	}

	existing, _ := s.discoverDAO.FindCollectionInFolder(userId, folder.Id, postUuid)
	if existing == nil {
		return false, 0, fmt.Errorf("未收藏该帖子")
	}

	if err := s.discoverDAO.DeleteCollection(userId, folder.Id, postUuid); err != nil {
		return false, 0, err
	}

	s.discoverDAO.DecrementFolderPostCount(folder.Id)

	post, _ := s.discoverDAO.FindPostByUuid(postUuid)
	if post != nil {
		s.discoverDAO.DecrementCollectCount(post.Id)
		updatedPost, _ := s.discoverDAO.FindPostByUuid(postUuid)
		if updatedPost != nil {
			return false, updatedPost.CollectCount, nil
		}
	}
	return false, 0, nil
}

func (s *DiscoverServiceImpl) CheckCollected(ctx context.Context, userId, postUuid string) (bool, string, error) {
	col, _ := s.discoverDAO.FindCollectionByUserAndTarget(userId, postUuid)
	if col == nil {
		return false, "", nil
	}
	// 通过 folderId 查找收藏夹
	var folder models.DiscoverCollectionFolder
	result := db.GormDB.Where("id = ?", col.FolderId).First(&folder)
	if result.Error != nil {
		return true, "", nil
	}
	return true, folder.Uuid, nil
}
