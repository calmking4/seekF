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
	"seekF-backend/internal/pkg/zlog"

	"gorm.io/gorm"
)

// DiscoverService 发现页内容与互动业务接口。
type DiscoverService interface {
	SetLike(ctx context.Context, userId, targetUuid string, liked bool, isComment bool) (bool, int, error)
	ListCommentThread(ctx context.Context, userId, postUuid, parentUuid string, page, pageSize int) ([]CommentInfo, bool, error)
	CreatePost(ctx context.Context, userId, title, content string, mediaType int8, tags []string, urls []string, coverUrl string, mediaSizes []MediaSize, coverSize MediaSize) (*PostInfo, error)
	ListPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error)
	AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error)
	ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error)
	AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error)
	// SearchPosts 搜索帖子（ES全文搜索）
	SearchPosts(ctx context.Context, userId string, keyword string, page, pageSize int) ([]PostInfo, int64, error)
	// GetUserProfile 获取用户主页信息
	GetUserProfile(ctx context.Context, profileUserId, currentUserId string, page, pageSize int) (*UserProfileInfo, error)

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
	discoverDAO     userdao.DiscoverDAO
	userInfoDAO     userdao.UserInfoDAO
	contactDAO      userdao.ContactDAO
	followDAO       userdao.FollowDAO
	notificationDAO userdao.NotificationDAO
}

// MediaSize 媒体宽高，单位像素，零表示未知。
type MediaSize struct {
	Width  int
	Height int
}

// PostInfo 帖子列表展示信息。
type PostInfo struct {
	Width        int
	Height       int
	Uuid         string
	UserId       string
	Nickname     string
	Avatar       string
	Title        string
	Content      string
	MediaType    int8
	CoverUrl     string
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
	CoverUrl     string
	Tags         []string
	Urls         []string
	LikeCount    int
	CommentCount int
	CollectCount int
	ViewCount    int
	IsLiked      bool
	IsCollected  bool
	CreatedAt    string
}

// CommentInfo 评论展示及回复数量信息。
type CommentInfo struct {
	ReplyCount      int64
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

func NewDiscoverService(discoverDAO userdao.DiscoverDAO, userInfoDAO userdao.UserInfoDAO, contactDAO userdao.ContactDAO, followDAO userdao.FollowDAO, notificationDAO userdao.NotificationDAO) DiscoverService {
	return &DiscoverServiceImpl{
		discoverDAO:     discoverDAO,
		userInfoDAO:     userInfoDAO,
		contactDAO:      contactDAO,
		followDAO:       followDAO,
		notificationDAO: notificationDAO,
	}
}

// CreatePost 保存帖子与媒体尺寸，并异步建立搜索索引。
func (s *DiscoverServiceImpl) CreatePost(ctx context.Context, userId, title, content string, mediaType int8, tags []string, urls []string, coverUrl string, mediaSizes []MediaSize, coverSize MediaSize) (*PostInfo, error) {
	postUUID := "D" + util.GetNowAndLenRandomString(11)

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	post := &models.DiscoverPost{
		Uuid:       postUUID,
		UserId:     userId,
		Title:      title,
		Content:    content,
		MediaType:  mediaType,
		CoverUrl:   coverUrl,
		CoverWidth: coverSize.Width, CoverHeight: coverSize.Height,
		Tags:   tagsJSON,
		Status: 0,
	}

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txDiscoverDAO := userdao.NewDiscoverDAO(tx)

		if err := txDiscoverDAO.CreatePost(post); err != nil {
			return fmt.Errorf("创建帖子失败: %v", err)
		}

		for i, url := range urls {
			var size MediaSize
			if i < len(mediaSizes) {
				size = mediaSizes[i]
			}
			media := &models.DiscoverMedia{
				PostId:    post.Id,
				Width:     size.Width,
				Height:    size.Height,
				Type:      mediaType,
				Url:       url,
				SortOrder: i,
			}
			if err := txDiscoverDAO.CreateMedia(media); err != nil {
				return fmt.Errorf("保存媒体失败: %v", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 异步索引帖子到ES（搜索用）
	go func() {
		if db.ESClient != nil {
			esDAO := userdao.NewESDiscoverDAO()
			if err := esDAO.IndexPost(post); err != nil {
				zlog.Error("索引帖子到ES失败: " + err.Error())
			}
		}
	}()

	user, _ := s.userInfoDAO.FindUserByUuid(userId)
	nickname, avatar := "", ""
	if user != nil {
		nickname, avatar = user.Nickname, user.Avatar
	}

	firstUrl := ""
	if len(urls) > 0 {
		firstUrl = urls[0]
	}

	var firstSize MediaSize
	if len(mediaSizes) > 0 {
		firstSize = mediaSizes[0]
	}
	if mediaType == 1 && coverUrl != "" && coverSize.Width > 0 && coverSize.Height > 0 {
		firstSize = coverSize
	}
	return &PostInfo{
		Width:     firstSize.Width,
		Height:    firstSize.Height,
		Uuid:      postUUID,
		UserId:    userId,
		Nickname:  nickname,
		Avatar:    avatar,
		Title:     title,
		Content:   content,
		MediaType: mediaType,
		CoverUrl:  coverUrl,
		Tags:      tags,
		FirstUrl:  firstUrl,
		CreatedAt: post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ListPosts 查询公共帖子并批量装配媒体与互动信息。
func (s *DiscoverServiceImpl) ListPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error) {
	posts, err := s.discoverDAO.ListPosts(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.discoverDAO.CountPosts()
	if err != nil {
		return nil, 0, err
	}
	return s.buildPostInfoList(ctx, userId, posts, total)
}

// ListLikedPosts 查询用户点赞的帖子及媒体信息。
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
		var firstMedia models.DiscoverMedia
		if len(mediaList) > 0 {
			firstMedia = mediaList[0]
			firstUrl = firstMedia.Url
		}
		cardSize := discoverCardSize(post, firstMedia)

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
			CoverUrl:     post.CoverUrl,
			Tags:         tags,
			FirstUrl:     firstUrl,
			Width:        cardSize.Width,
			Height:       cardSize.Height,
			LikeCount:    post.LikeCount,
			CommentCount: post.CommentCount,
			CollectCount: post.CollectCount,
			IsLiked:      true, // 点赞列表中的帖子都是已点赞的
			CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

// GetPostDetail 查询可用帖子详情及当前用户互动状态。
func (s *DiscoverServiceImpl) GetPostDetail(ctx context.Context, userId, uuid string) (*PostDetailInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(uuid)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != 0 {
		return nil, fmt.Errorf("帖子不存在")
	}

	// 异步增加浏览量
	go func() {
		if err := s.discoverDAO.IncrementViewCount(post.Id); err != nil {
			zlog.Error("增加帖子浏览量失败: " + err.Error())
		}
	}()

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
		CoverUrl:     post.CoverUrl,
		Tags:         tags,
		Urls:         urls,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		CollectCount: post.CollectCount,
		ViewCount:    post.ViewCount,
		IsLiked:      isLiked,
		IsCollected:  isCollected,
		CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// SetLike 在同一事务中设置点赞状态、校准计数并写入通知，重复请求保持幂等。
func (s *DiscoverServiceImpl) SetLike(ctx context.Context, userId, targetUuid string, liked bool, isComment bool) (bool, int, error) {
	var count int
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		var targetId int64
		var owner string
		if isComment {
			comment, err := txDiscoverDAO.FindCommentByUuidForUpdate(targetUuid)
			if err != nil {
				return fmt.Errorf("查询评论失败: %w", err)
			}
			if comment == nil {
				return fmt.Errorf("评论不存在")
			}
			post, err := txDiscoverDAO.FindPostById(comment.PostId)
			if err != nil {
				return fmt.Errorf("查询评论所属帖子失败: %w", err)
			}
			if post == nil || post.Status != 0 {
				return fmt.Errorf("评论所属帖子不可用")
			}
			targetId, owner = comment.Id, comment.UserId
		} else {
			post, err := txDiscoverDAO.FindPostByUuidForUpdate(targetUuid)
			if err != nil {
				return fmt.Errorf("查询帖子失败: %w", err)
			}
			if post == nil || post.Status != 0 {
				return fmt.Errorf("帖子不存在或已下架")
			}
			targetId, owner = post.Id, post.UserId
		}
		// 先锁定目标再读取当前状态，避免同一目标并发请求重复增删和通知。
		existing, err := txDiscoverDAO.FindLike(userId, targetUuid)
		if err != nil {
			return fmt.Errorf("查询点赞状态失败: %w", err)
		}
		changed := liked != (existing != nil)
		if changed {
			if liked {
				if err := txDiscoverDAO.CreateLike(&models.DiscoverLike{UserId: userId, TargetUuid: targetUuid}); err != nil {
					return fmt.Errorf("保存点赞失败: %w", err)
				}
			} else {
				if err := txDiscoverDAO.DeleteLike(userId, targetUuid); err != nil {
					return fmt.Errorf("取消点赞失败: %w", err)
				}
			}
		}
		// 保留按真实唯一用户数校准计数的行为，兼容遗留重复记录。
		count, err = txDiscoverDAO.CountLikesByTargetUuid(targetUuid)
		if err != nil {
			return fmt.Errorf("查询点赞数量失败: %w", err)
		}
		if isComment {
			err = txDiscoverDAO.UpdateCommentLikeCount(targetId, count)
		} else {
			err = txDiscoverDAO.UpdatePostLikeCount(targetId, count)
		}
		if err != nil {
			return fmt.Errorf("更新点赞数量失败: %w", err)
		}
		if changed && liked && owner != userId {
			notificationType, content := NotificationTypeLikePost, "赞了你的帖子"
			if isComment {
				notificationType, content = NotificationTypeLikeComment, "赞了你的评论"
			}
			// 通知写入失败时一起回滚点赞与计数，避免业务成功但通知丢失。
			if err := userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{UserId: owner, ActorId: userId, Type: notificationType, TargetUuid: targetUuid, Content: content}); err != nil {
				return fmt.Errorf("创建点赞通知失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}
	return liked, count, nil
}

// AddComment 保存评论并通知作者或被回复者。
func (s *DiscoverServiceImpl) AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != 0 {
		return nil, fmt.Errorf("帖子不存在")
	}
	// 回复顶级评论时客户端不传回复用户，从父评论确定通知接收者。
	replyRecipient := replyToUserId
	if parentUuid != "" {
		parent, err := s.discoverDAO.FindCommentByUuid(parentUuid)
		if err != nil {
			return nil, fmt.Errorf("查询被回复评论失败: %w", err)
		}
		if parent == nil || parent.PostId != post.Id {
			return nil, fmt.Errorf("被回复的评论不存在或不属于当前帖子")
		}
		if replyRecipient == "" {
			replyRecipient = parent.UserId
		}
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

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txDiscoverDAO := userdao.NewDiscoverDAO(tx)

		if err := txDiscoverDAO.CreateComment(comment); err != nil {
			return fmt.Errorf("评论失败: %v", err)
		}

		return txDiscoverDAO.IncrementCommentCount(post.Id)
	})
	if err != nil {
		return nil, err
	}

	user, _ := s.userInfoDAO.FindUserByUuid(userId)
	nickname := ""
	avatar := ""
	if user != nil {
		nickname = user.Nickname
		avatar = user.Avatar
	}

	// 异步创建评论/回复通知
	go func() {
		if parentUuid != "" {
			// 回复评论，通知被回复者
			if replyRecipient != "" && replyRecipient != userId {
				notification := &models.Notification{
					UserId:     replyRecipient,
					ActorId:    userId,
					Type:       NotificationTypeReply,
					TargetUuid: commentUUID,
					Content:    "回复了你的评论：" + truncateContent(content, 50),
				}
				if err := s.notificationDAO.CreateNotification(notification); err != nil {
					zlog.Error("创建回复通知失败: " + err.Error())
				}
			}
		} else {
			// 评论帖子，通知帖子作者
			if post.UserId != userId {
				notification := &models.Notification{
					UserId:     post.UserId,
					ActorId:    userId,
					Type:       NotificationTypeCommentPost,
					TargetUuid: commentUUID,
					Content:    "评论了你的帖子：" + truncateContent(content, 50),
				}
				if err := s.notificationDAO.CreateNotification(notification); err != nil {
					zlog.Error("创建评论通知失败: " + err.Error())
				}
			}
		}
	}()

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

// truncateContent 截断内容，超过 maxLen 加省略号
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// ListComments 兼容旧客户端的全部评论分页。
func (s *DiscoverServiceImpl) ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != 0 {
		return nil, fmt.Errorf("帖子不存在或已下架")
	}
	comments, err := s.discoverDAO.FindCommentsByPostId(post.Id, page, pageSize)
	if err != nil {
		return nil, err
	}
	return s.buildCommentInfoList(userId, comments)
}

// ListCommentThread 分页查询顶级评论或指定评论的回复，校验父评论归属。
func (s *DiscoverServiceImpl) ListCommentThread(ctx context.Context, userId, postUuid, parentUuid string, page, pageSize int) ([]CommentInfo, bool, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return nil, false, err
	}
	if post == nil || post.Status != 0 {
		return nil, false, fmt.Errorf("帖子不存在或已下架")
	}
	if parentUuid != "" {
		parent, err := s.discoverDAO.FindCommentByUuid(parentUuid)
		if err != nil {
			return nil, false, err
		}
		if parent == nil || parent.PostId != post.Id || parent.ParentId != "" {
			return nil, false, fmt.Errorf("父评论不存在或不属于当前帖子")
		}
	}
	comments, hasMore, err := s.discoverDAO.FindCommentThread(post.Id, parentUuid, page, pageSize)
	if err != nil {
		return nil, false, err
	}
	infos, err := s.buildCommentInfoList(userId, comments)
	return infos, hasMore, err
}

// buildCommentInfoList 批量装配评论作者、回复数量与当前用户点赞状态。
func (s *DiscoverServiceImpl) buildCommentInfoList(userId string, comments []models.DiscoverComment) ([]CommentInfo, error) {
	result := make([]CommentInfo, 0, len(comments))
	if len(comments) == 0 {
		return result, nil
	}
	userIds, uuids := make([]string, 0), make([]string, 0, len(comments))
	for _, c := range comments {
		userIds = append(userIds, c.UserId)
		if c.ReplyToUserId != "" {
			userIds = append(userIds, c.ReplyToUserId)
		}
		uuids = append(uuids, c.Uuid)
	}
	users, err := s.userInfoDAO.FindUsersByUuids(userIds)
	if err != nil {
		return nil, fmt.Errorf("查询评论用户失败: %w", err)
	}
	userMap := make(map[string]models.UserInfo)
	for _, u := range users {
		userMap[u.Uuid] = u
	}
	likes, err := s.discoverDAO.FindLikesByUserIdAndTargetUuids(userId, uuids)
	if err != nil {
		return nil, fmt.Errorf("查询评论点赞失败: %w", err)
	}
	liked := make(map[string]bool)
	for _, like := range likes {
		liked[like.TargetUuid] = true
	}
	replies, err := s.discoverDAO.CountReplies(uuids)
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		user, replyUser := userMap[c.UserId], userMap[c.ReplyToUserId]
		result = append(result, CommentInfo{Uuid: c.Uuid, UserId: c.UserId, Nickname: user.Nickname, Avatar: user.Avatar, ParentId: c.ParentId, ReplyToUserId: c.ReplyToUserId, ReplyToNickname: replyUser.Nickname, Content: c.Content, LikeCount: c.LikeCount, IsLiked: liked[c.Uuid], ReplyCount: replies[c.Uuid], CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	return result, nil
}

// AddAIComment 保存用户评论并触发 AI 回复任务。
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
			coverUrl = posts[0].CoverUrl
			if coverUrl == "" && posts[0].MediaType != 1 {
				mediaList, _ := s.discoverDAO.FindMediaByPostId(posts[0].Id)
				if len(mediaList) > 0 {
					coverUrl = mediaList[0].Url
				}
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
	if folder.UserId != userId && folder.IsPublic != 1 {
		return nil, fmt.Errorf("无权查看私密收藏夹")
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

// ListCollectedPosts 查询收藏夹内的帖子及互动状态。
func (s *DiscoverServiceImpl) ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int) ([]PostInfo, int64, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(folderUuid)
	if err != nil {
		return nil, 0, err
	}
	if folder == nil {
		return nil, 0, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId && folder.IsPublic != 1 {
		return nil, 0, fmt.Errorf("无权查看私密收藏夹")
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
	postUuids := make([]string, 0, len(posts))
	for _, post := range posts {
		postUuids = append(postUuids, post.Uuid)
	}
	likedMap := make(map[string]bool)
	collectedMap := make(map[string]bool)
	if len(postUuids) > 0 && userId != "" {
		likes, err := s.discoverDAO.FindLikesByUserIdAndTargetUuids(userId, postUuids)
		if err != nil {
			return nil, 0, err
		}
		for _, like := range likes {
			likedMap[like.TargetUuid] = true
		}
		collections, err := s.discoverDAO.FindCollectionsByUserIdAndTargetUuids(userId, postUuids)
		if err != nil {
			return nil, 0, err
		}
		for _, collection := range collections {
			collectedMap[collection.TargetUuid] = true
		}
	}
	for _, post := range posts {
		mediaList, _ := s.discoverDAO.FindMediaByPostId(post.Id)
		firstUrl := ""
		var firstMedia models.DiscoverMedia
		if len(mediaList) > 0 {
			firstMedia = mediaList[0]
			firstUrl = firstMedia.Url
		}
		cardSize := discoverCardSize(post, firstMedia)
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
			CoverUrl:     post.CoverUrl,
			Tags:         tags,
			FirstUrl:     firstUrl,
			Width:        cardSize.Width,
			Height:       cardSize.Height,
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

// ========== 收藏/取消收藏 ==========

// CollectPost 保存收藏及通知，并在事务内维护收藏夹和帖子计数。
func (s *DiscoverServiceImpl) CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	post, err := s.discoverDAO.FindPostByUuid(postUuid)
	if err != nil {
		return false, 0, err
	}
	if post == nil || post.Status != 0 {
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
	previousCollection, err := s.discoverDAO.FindCollectionByUserAndTarget(userId, postUuid)
	if err != nil {
		return false, 0, fmt.Errorf("查询收藏状态失败: %w", err)
	}

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txDiscoverDAO := userdao.NewDiscoverDAO(tx)

		col := &models.DiscoverCollection{
			UserId:     userId,
			FolderId:   folder.Id,
			TargetUuid: postUuid,
		}
		if err := txDiscoverDAO.CreateCollection(col); err != nil {
			return err
		}
		// 收藏多个文件夹时只在首次收藏发送通知，与收藏写入保持同一事务。
		if previousCollection == nil && post.UserId != userId {
			if err := userdao.NewNotificationDAO(tx).CreateNotification(&models.Notification{
				UserId: post.UserId, ActorId: userId, Type: NotificationTypeCollectPost,
				TargetUuid: postUuid, Content: "收藏了你的帖子：" + truncateContent(post.Title, 50),
			}); err != nil {
				return fmt.Errorf("创建收藏通知失败: %w", err)
			}
		}

		if err := txDiscoverDAO.IncrementFolderPostCount(folder.Id); err != nil {
			return fmt.Errorf("更新收藏夹帖子数失败: %w", err)
		}
		if err := txDiscoverDAO.IncrementCollectCount(post.Id); err != nil {
			return fmt.Errorf("更新帖子收藏数失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	updatedPost, _ := s.discoverDAO.FindPostByUuid(postUuid)
	collectCount := 0
	if updatedPost != nil {
		collectCount = updatedPost.CollectCount
	}
	return true, collectCount, nil
}

// UncollectPost 删除收藏并在事务内维护收藏夹和帖子计数。
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

	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txDiscoverDAO := userdao.NewDiscoverDAO(tx)

		if err := txDiscoverDAO.DeleteCollection(userId, folder.Id, postUuid); err != nil {
			return err
		}

		if err := txDiscoverDAO.DecrementFolderPostCount(folder.Id); err != nil {
			return fmt.Errorf("更新收藏夹帖子数失败: %w", err)
		}

		post, err := txDiscoverDAO.FindPostByUuid(postUuid)
		if err != nil {
			return fmt.Errorf("查询收藏帖子失败: %w", err)
		}
		if post != nil {
			if err := txDiscoverDAO.DecrementCollectCount(post.Id); err != nil {
				return fmt.Errorf("更新帖子收藏数失败: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	updatedPost, _ := s.discoverDAO.FindPostByUuid(postUuid)
	if updatedPost != nil {
		return false, updatedPost.CollectCount, nil
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

// SearchPosts 搜索帖子（ES全文搜索，ES不可用时降级到MySQL）
// SearchPosts 优先全文检索，降级时保留数据库分页与真实总数。
func (s *DiscoverServiceImpl) SearchPosts(ctx context.Context, userId string, keyword string, page, pageSize int) ([]PostInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	// 尝试ES搜索
	if db.ESClient != nil {
		esDAO := userdao.NewESDiscoverDAO()
		posts, total, err := esDAO.SearchPosts(keyword, page, pageSize)
		if err == nil {
			return s.buildPostInfoList(ctx, userId, posts, total)
		}
		zlog.Error("ES搜索帖子失败，降级到MySQL: " + err.Error())
	}

	// 降级到MySQL LIKE搜索
	posts, total, err := s.discoverDAO.SearchPostsPage(keyword, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return s.buildPostInfoList(ctx, userId, posts, total)
}

// buildPostInfoList 将帖子模型列表转换为PostInfo列表
func (s *DiscoverServiceImpl) buildPostInfoList(ctx context.Context, userId string, posts []models.DiscoverPost, total int64) ([]PostInfo, int64, error) {
	if len(posts) == 0 {
		return []PostInfo{}, total, nil
	}

	postIds := make([]int64, 0, len(posts))
	userIds := make([]string, 0, len(posts))
	postUuids := make([]string, 0, len(posts))
	for _, post := range posts {
		postIds = append(postIds, post.Id)
		userIds = append(userIds, post.UserId)
		postUuids = append(postUuids, post.Uuid)
	}

	// 批量查询媒体
	mediaList, err := s.discoverDAO.FindMediaByPostIds(postIds)
	if err != nil {
		return nil, 0, fmt.Errorf("查询帖子媒体失败: %w", err)
	}
	mediaMap := make(map[int64]models.DiscoverMedia)
	for _, media := range mediaList {
		if _, exists := mediaMap[media.PostId]; !exists {
			mediaMap[media.PostId] = media
		}
	}

	// 批量查询用户信息
	users, err := s.userInfoDAO.FindUsersByUuids(userIds)
	if err != nil {
		return nil, 0, fmt.Errorf("查询帖子作者失败: %w", err)
	}
	userMap := make(map[string]*models.UserInfo)
	for i := range users {
		userMap[users[i].Uuid] = &users[i]
	}

	// 批量查询点赞状态
	likedMap := make(map[string]bool)
	if userId != "" {
		likes, err := s.discoverDAO.FindLikesByUserIdAndTargetUuids(userId, postUuids)
		if err != nil {
			return nil, 0, fmt.Errorf("查询帖子点赞状态失败: %w", err)
		}
		for _, like := range likes {
			likedMap[like.TargetUuid] = true
		}
	}

	// 批量查询收藏状态
	collectedMap := make(map[string]bool)
	if userId != "" {
		collections, err := s.discoverDAO.FindCollectionsByUserIdAndTargetUuids(userId, postUuids)
		if err != nil {
			return nil, 0, fmt.Errorf("查询帖子收藏状态失败: %w", err)
		}
		for _, col := range collections {
			collectedMap[col.TargetUuid] = true
		}
	}

	var result []PostInfo
	for _, post := range posts {
		firstMedia := mediaMap[post.Id]
		firstUrl := firstMedia.Url
		cardSize := discoverCardSize(post, firstMedia)

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
			CoverUrl:     post.CoverUrl,
			Tags:         tags,
			FirstUrl:     firstUrl,
			Width:        cardSize.Width,
			Height:       cardSize.Height,
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

// ========== 用户主页 ==========

type UserProfileInfo struct {
	Uuid           string
	Nickname       string
	Avatar         string
	Signature      string
	PostCount      int64
	TotalLikes     int
	FollowingCount int64
	FollowerCount  int64
	IsFollowed     bool
	IsFriend       bool
	Posts          []PostInfo
	Total          int64
}

// GetUserProfile 查询用户主页及帖子列表。
func (s *DiscoverServiceImpl) GetUserProfile(ctx context.Context, profileUserId, currentUserId string, page, pageSize int) (*UserProfileInfo, error) {
	// 查询用户信息
	user, err := s.userInfoDAO.FindUserByUuid(profileUserId)
	if err != nil {
		return nil, fmt.Errorf("查询用户信息失败: %v", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 查询帖子列表
	posts, err := s.discoverDAO.ListPostsByUserId(profileUserId, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询帖子列表失败: %v", err)
	}

	total, err := s.discoverDAO.CountPostsByUserId(profileUserId)
	if err != nil {
		return nil, fmt.Errorf("查询帖子总数失败: %v", err)
	}

	totalLikes, err := s.discoverDAO.GetUserTotalLikes(profileUserId)
	if err != nil {
		totalLikes = 0
	}

	// 查询是否为好友
	isFriend := false
	if currentUserId != "" && currentUserId != profileUserId {
		contact, _ := s.contactDAO.GetUserContactByUserIdAndContactId(currentUserId, profileUserId)
		if contact != nil && contact.Status == 0 {
			isFriend = true
		}
	}

	// 批量查询帖子的媒体、点赞、收藏状态
	var postInfos []PostInfo
	if len(posts) > 0 {
		postIds := make([]int64, 0, len(posts))
		postUuids := make([]string, 0, len(posts))
		for _, post := range posts {
			postIds = append(postIds, post.Id)
			postUuids = append(postUuids, post.Uuid)
		}

		mediaList, _ := s.discoverDAO.FindMediaByPostIds(postIds)
		mediaMap := make(map[int64]models.DiscoverMedia)
		for _, media := range mediaList {
			if _, exists := mediaMap[media.PostId]; !exists {
				mediaMap[media.PostId] = media
			}
		}

		likedMap := make(map[string]bool)
		collectedMap := make(map[string]bool)
		if currentUserId != "" {
			likes, _ := s.discoverDAO.FindLikesByUserIdAndTargetUuids(currentUserId, postUuids)
			for _, like := range likes {
				likedMap[like.TargetUuid] = true
			}
			collections, _ := s.discoverDAO.FindCollectionsByUserIdAndTargetUuids(currentUserId, postUuids)
			for _, col := range collections {
				collectedMap[col.TargetUuid] = true
			}
		}

		for _, post := range posts {
			firstMedia := mediaMap[post.Id]
			firstUrl := firstMedia.Url
			cardSize := discoverCardSize(post, firstMedia)
			var tags []string
			if len(post.Tags) > 0 {
				json.Unmarshal(post.Tags, &tags)
			}
			postInfos = append(postInfos, PostInfo{
				Uuid:         post.Uuid,
				UserId:       post.UserId,
				Nickname:     user.Nickname,
				Avatar:       user.Avatar,
				Title:        post.Title,
				Content:      post.Content,
				MediaType:    post.MediaType,
				CoverUrl:     post.CoverUrl,
				Tags:         tags,
				FirstUrl:     firstUrl,
				Width:        cardSize.Width,
				Height:       cardSize.Height,
				LikeCount:    post.LikeCount,
				CommentCount: post.CommentCount,
				CollectCount: post.CollectCount,
				IsLiked:      likedMap[post.Uuid],
				IsCollected:  collectedMap[post.Uuid],
				CreatedAt:    post.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 查询关注/粉丝数
	followingCount, _ := s.followDAO.CountFollowing(profileUserId)
	followerCount, _ := s.followDAO.CountFollowers(profileUserId)

	// 查询当前用户是否关注了该用户
	isFollowed := false
	if currentUserId != "" && currentUserId != profileUserId {
		follow, _ := s.followDAO.FindFollow(currentUserId, profileUserId)
		if follow != nil {
			isFollowed = true
		}
	}

	return &UserProfileInfo{
		Uuid:           user.Uuid,
		Nickname:       user.Nickname,
		Avatar:         user.Avatar,
		Signature:      user.Signature,
		PostCount:      total,
		TotalLikes:     totalLikes,
		FollowingCount: followingCount,
		FollowerCount:  followerCount,
		IsFollowed:     isFollowed,
		IsFriend:       isFriend,
		Posts:          postInfos,
		Total:          total,
	}, nil
}

// discoverCardSize 返回卡片展示尺寸，视频自定义封面优先，媒体原尺寸仍独立保存。
func discoverCardSize(post models.DiscoverPost, media models.DiscoverMedia) MediaSize {
	if post.MediaType == 1 && post.CoverUrl != "" && post.CoverWidth > 0 && post.CoverHeight > 0 {
		return MediaSize{Width: post.CoverWidth, Height: post.CoverHeight}
	}
	return MediaSize{Width: media.Width, Height: media.Height}
}
