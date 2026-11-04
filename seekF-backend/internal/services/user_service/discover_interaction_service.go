package userservice

import (
	"context"
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	aipkg "seekF-backend/internal/pkg/ai"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/util"

	"gorm.io/gorm"
)

// DiscoverInteractionService 点赞、评论与收藏业务接口。
type DiscoverInteractionService interface {
	// 设置点赞状态。
	SetLike(ctx context.Context, userId, targetUuid string, liked bool, isComment bool) (bool, int, error)
	// 查询评论回复。
	ListCommentThread(ctx context.Context, userId, postUuid, parentUuid string, page, pageSize int) ([]CommentInfo, bool, error)
	// 查询点赞作品。
	ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error)
	// 添加评论。
	AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error)
	// 查询评论列表。
	ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error)
	// 添加评论并触发 AI 回复。
	AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error)
	// 创建收藏夹。
	CreateFolder(ctx context.Context, userId, name, description string, isPublic int8) (*FolderInfo, error)
	// 修改收藏夹。
	UpdateFolder(ctx context.Context, userId, folderUuid, name, description string, isPublic int8) error
	// 删除收藏夹。
	DeleteFolder(ctx context.Context, userId, folderUuid string) error
	// 查询收藏夹列表。
	ListFolders(ctx context.Context, userId, viewerId string) ([]FolderInfo, error)
	// 查询收藏夹详情。
	GetFolderDetail(ctx context.Context, userId, folderUuid string) (*FolderDetailInfo, error)
	// 查询收藏作品。
	ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int, offsetOverride ...int) ([]PostInfo, int64, error)
	// 收藏作品。
	CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error)
	// 取消收藏作品。
	UncollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error)
	// 查询作品收藏状态。
	CheckCollected(ctx context.Context, userId, postUuid string) (bool, string, error)
}

// DiscoverInteractionServiceImpl 实现点赞、评论与收藏业务。
type DiscoverInteractionServiceImpl struct {
	discoverDAO userdao.DiscoverDAO
	blockDAO    userdao.BlockDAO
	userInfoDAO userdao.UserInfoDAO
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

// FolderInfo 收藏夹列表展示信息。
type FolderInfo struct {
	// 夹主是否拉黑了当前查看者，此时只返回锁定占位。
	IsBlockedBy bool
	Uuid        string
	Name        string
	Description string
	IsPublic    bool
	PostCount   int
	CoverUrl    string
	CreatedAt   string
}

// FolderDetailInfo 收藏夹详情。
type FolderDetailInfo struct {
	Uuid        string
	Name        string
	Description string
	IsPublic    bool
	PostCount   int
	CreatedAt   string
}

// NewDiscoverInteractionService 构造服务并注入所需依赖。
func NewDiscoverInteractionService(
	discoverDAO userdao.DiscoverDAO,
	blockDAO userdao.BlockDAO,
	userInfoDAO userdao.UserInfoDAO,
) DiscoverInteractionService {
	return &DiscoverInteractionServiceImpl{
		discoverDAO: discoverDAO,
		blockDAO:    blockDAO,
		userInfoDAO: userInfoDAO,
	}
}

// ListLikedPosts 查询用户点赞的帖子及媒体信息。
func (s *DiscoverInteractionServiceImpl) ListLikedPosts(ctx context.Context, userId string, page, pageSize int) ([]PostInfo, int64, error) {
	posts, err := s.discoverDAO.ListLikedPosts(ctx, userId, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.discoverDAO.CountLikedPosts(ctx, userId)
	if err != nil {
		return nil, 0, err
	}

	return buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, userId, posts, total)
}

// SetLike 在同一事务中设置点赞状态、校准计数并写入通知，重复请求保持幂等。
func (s *DiscoverInteractionServiceImpl) SetLike(ctx context.Context, userId, targetUuid string, liked bool, isComment bool) (bool, int, error) {
	var count int
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		var targetId int64
		var owner string
		if isComment {
			comment, err := txDiscoverDAO.FindCommentByUuidForUpdate(ctx, targetUuid)
			if err != nil {
				return fmt.Errorf("查询评论失败: %w", err)
			}
			if comment == nil {
				return fmt.Errorf("评论不存在")
			}
			post, err := txDiscoverDAO.FindPostById(ctx, comment.PostId)
			if err != nil {
				return fmt.Errorf("查询评论所属帖子失败: %w", err)
			}
			if !canViewUnblockedDiscoverPost(ctx, tx, post, userId) {
				return fmt.Errorf("评论所属帖子不可用")
			}
			if err := requireDiscoverSupport(ctx, tx, userId, post); err != nil {
				return err
			}
			targetId, owner = comment.Id, comment.UserId
		} else {
			post, err := txDiscoverDAO.FindPostByUuidForUpdate(ctx, targetUuid)
			if err != nil {
				return fmt.Errorf("查询帖子失败: %w", err)
			}
			if !canViewUnblockedDiscoverPost(ctx, tx, post, userId) {
				return fmt.Errorf("帖子不存在或已下架")
			}
			targetId, owner = post.Id, post.UserId
		}
		// 先锁定目标再读取当前状态，避免同一目标并发请求重复增删和通知。
		existing, err := txDiscoverDAO.FindLike(ctx, userId, targetUuid)
		if err != nil {
			return fmt.Errorf("查询点赞状态失败: %w", err)
		}
		changed := liked != (existing != nil)
		if changed {
			if liked {
				if err := txDiscoverDAO.CreateLike(ctx, &models.DiscoverLike{UserId: userId, TargetUuid: targetUuid}); err != nil {
					return fmt.Errorf("保存点赞失败: %w", err)
				}
			} else {
				if err := txDiscoverDAO.DeleteLike(ctx, userId, targetUuid); err != nil {
					return fmt.Errorf("取消点赞失败: %w", err)
				}
			}
		}
		// 保留按真实唯一用户数校准计数的行为，兼容遗留重复记录。
		count, err = txDiscoverDAO.CountLikesByTargetUuid(ctx, targetUuid)
		if err != nil {
			return fmt.Errorf("查询点赞数量失败: %w", err)
		}
		if isComment {
			err = txDiscoverDAO.UpdateCommentLikeCount(ctx, targetId, count)
		} else {
			err = txDiscoverDAO.UpdatePostLikeCount(ctx, targetId, count)
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
			if err := userdao.NewNotificationDAO(tx).CreateNotification(ctx, &models.Notification{UserId: owner, ActorId: userId, Type: notificationType, TargetUuid: targetUuid, Content: content}); err != nil {
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
func (s *DiscoverInteractionServiceImpl) AddComment(ctx context.Context, userId, postUuid string, parentUuid, replyToUserId, content string) (*CommentInfo, error) {
	var post *models.DiscoverPost
	replyRecipient := replyToUserId

	commentUUID := "C" + util.GetNowAndLenRandomString(11)
	comment := &models.DiscoverComment{
		Uuid:          commentUUID,
		UserId:        userId,
		ParentId:      parentUuid,
		ReplyToUserId: replyToUserId,
		Content:       content,
	}

	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		var err error
		post, err = txDiscoverDAO.FindPostByUuidForUpdate(ctx, postUuid)
		if err != nil {
			return err
		}
		if !canViewUnblockedDiscoverPost(ctx, tx, post, userId) {
			return fmt.Errorf("帖子不存在或已下架")
		}
		if err := requireDiscoverSupport(ctx, tx, userId, post); err != nil {
			return err
		}
		if parentUuid != "" {
			parent, err := txDiscoverDAO.FindCommentByUuid(ctx, parentUuid)
			if err != nil {
				return fmt.Errorf("查询被回复评论失败: %w", err)
			}
			if parent == nil || parent.PostId != post.Id {
				return fmt.Errorf("被回复的评论不存在或不属于当前帖子")
			}
			if replyRecipient == "" {
				replyRecipient = parent.UserId
			}
		}
		comment.PostId = post.Id
		if err := txDiscoverDAO.CreateComment(ctx, comment); err != nil {
			return fmt.Errorf("评论失败: %v", err)
		}
		if err := txDiscoverDAO.IncrementCommentCount(ctx, post.Id); err != nil {
			return err
		}
		recipient, notificationType, text := post.UserId, NotificationTypeCommentPost, "评论了你的帖子："
		if parentUuid != "" {
			recipient, notificationType, text = replyRecipient, NotificationTypeReply, "回复了你的评论："
		}
		if recipient != "" && recipient != userId {
			return userdao.NewNotificationDAO(tx).CreateNotification(ctx, &models.Notification{UserId: recipient, ActorId: userId, Type: notificationType, TargetUuid: commentUUID, Content: text + truncateContent(content, 50)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	user, _ := s.userInfoDAO.FindUserByUuid(ctx, userId)
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

// ListComments 兼容旧客户端的全部评论分页。
func (s *DiscoverInteractionServiceImpl) ListComments(ctx context.Context, userId, postUuid string, page, pageSize int) ([]CommentInfo, error) {
	post, err := s.discoverDAO.FindPostByUuid(ctx, postUuid)
	if err != nil {
		return nil, err
	}
	if !canViewUnblockedDiscoverPost(ctx, db.GormDB.WithContext(ctx), post, userId) {
		return nil, fmt.Errorf("帖子不存在或已下架")
	}
	if err := requireDiscoverSupport(ctx, db.GormDB, userId, post); err != nil {
		return nil, err
	}
	comments, err := s.discoverDAO.FindCommentsByPostId(ctx, post.Id, page, pageSize)
	if err != nil {
		return nil, err
	}
	return s.buildCommentInfoList(ctx, userId, comments)
}

// ListCommentThread 分页查询顶级评论或指定评论的回复，校验父评论归属。
func (s *DiscoverInteractionServiceImpl) ListCommentThread(ctx context.Context, userId, postUuid, parentUuid string, page, pageSize int) ([]CommentInfo, bool, error) {
	post, err := s.discoverDAO.FindPostByUuid(ctx, postUuid)
	if err != nil {
		return nil, false, err
	}
	if !canViewUnblockedDiscoverPost(ctx, db.GormDB.WithContext(ctx), post, userId) {
		return nil, false, fmt.Errorf("帖子不存在或已下架")
	}
	if err := requireDiscoverSupport(ctx, db.GormDB, userId, post); err != nil {
		return nil, false, err
	}
	if parentUuid != "" {
		parent, err := s.discoverDAO.FindCommentByUuid(ctx, parentUuid)
		if err != nil {
			return nil, false, err
		}
		if parent == nil || parent.PostId != post.Id || parent.ParentId != "" {
			return nil, false, fmt.Errorf("父评论不存在或不属于当前帖子")
		}
	}
	comments, hasMore, err := s.discoverDAO.FindCommentThread(ctx, post.Id, parentUuid, page, pageSize)
	if err != nil {
		return nil, false, err
	}
	infos, err := s.buildCommentInfoList(ctx, userId, comments)
	return infos, hasMore, err
}

// AddAIComment 保存用户评论并触发 AI 回复任务。
func (s *DiscoverInteractionServiceImpl) AddAIComment(ctx context.Context, userId, postUuid, content, aiQuestion, parentUuid, replyToUserId, replyToContent string) (*CommentInfo, error) {
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
		RequestUserID:  userId,
		PostUuid:       postUuid,
		Content:        aiQuestion,
		ParentUuid:     aiParentUuid,
		ReplyToUserId:  aiReplyToUserId,
		ReplyToContent: replyToContent,
	})

	return userComment, nil
}

func (s *DiscoverInteractionServiceImpl) CreateFolder(ctx context.Context, userId, name, description string, isPublic int8) (*FolderInfo, error) {
	folderUUID := "F" + util.GetNowAndLenRandomString(11)
	folder := &models.DiscoverCollectionFolder{
		Uuid:        folderUUID,
		UserId:      userId,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
	}
	if err := s.discoverDAO.CreateFolder(ctx, folder); err != nil {
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

func (s *DiscoverInteractionServiceImpl) UpdateFolder(ctx context.Context, userId, folderUuid, name, description string, isPublic int8) error {
	folder, err := s.discoverDAO.FindFolderByUuid(ctx, folderUuid)
	if err != nil {
		return err
	}
	if folder == nil {
		return fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return fmt.Errorf("无权修改此收藏夹")
	}
	return s.discoverDAO.UpdateFolder(ctx, folderUuid, name, description, isPublic)
}

func (s *DiscoverInteractionServiceImpl) DeleteFolder(ctx context.Context, userId, folderUuid string) error {
	folder, err := s.discoverDAO.FindFolderByUuid(ctx, folderUuid)
	if err != nil {
		return err
	}
	if folder == nil {
		return fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId {
		return fmt.Errorf("无权删除此收藏夹")
	}
	return s.discoverDAO.DeleteFolder(ctx, folderUuid)
}

// ListFolders 按查看者权限装配收藏夹封面与列表。
func (s *DiscoverInteractionServiceImpl) ListFolders(ctx context.Context, userId, viewerId string) ([]FolderInfo, error) {
	blockedBy := false
	if userId != viewerId {
		blocked, by, err := s.blockDAO.Status(ctx, viewerId, userId)
		if err != nil {
			return nil, fmt.Errorf("查询收藏夹拉黑权限失败: %w", err)
		}
		if blocked {
			return []FolderInfo{}, nil
		}
		blockedBy = by
	}
	folders, err := s.discoverDAO.ListFoldersByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	var result []FolderInfo
	for _, f := range folders {
		if userId != viewerId && f.IsPublic != 1 {
			continue
		}
		if blockedBy {
			result = append(result, FolderInfo{Uuid: f.Uuid, Name: "已被拉黑", IsPublic: true, IsBlockedBy: true})
			continue
		}
		// 获取收藏夹第一张图片作为封面
		coverUrl := ""
		posts, _ := s.discoverDAO.ListCollectedPostsByFolder(ctx, f.Id, viewerId, 1, 1)
		if len(posts) > 0 && canViewUnblockedDiscoverPost(ctx, db.GormDB.WithContext(ctx), &posts[0], viewerId) {
			coverUrl = posts[0].CoverUrl
			if coverUrl == "" && posts[0].MediaType != 1 {
				mediaList, _ := s.discoverDAO.FindMediaByPostId(ctx, posts[0].Id)
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

// GetFolderDetail 按收藏夹可见性及夹主拉黑状态检查详情读取权限。
func (s *DiscoverInteractionServiceImpl) GetFolderDetail(ctx context.Context, userId, folderUuid string) (*FolderDetailInfo, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(ctx, folderUuid)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId && folder.IsPublic != 1 {
		return nil, fmt.Errorf("无权查看私密收藏夹")
	}
	if err := checkFolderBlockAccess(ctx, userId, folder.UserId); err != nil {
		return nil, err
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
func (s *DiscoverInteractionServiceImpl) ListCollectedPosts(ctx context.Context, userId, folderUuid string, page, pageSize int, offsetOverride ...int) ([]PostInfo, int64, error) {
	folder, err := s.discoverDAO.FindFolderByUuid(ctx, folderUuid)
	if err != nil {
		return nil, 0, err
	}
	if folder == nil {
		return nil, 0, fmt.Errorf("收藏夹不存在")
	}
	if folder.UserId != userId && folder.IsPublic != 1 {
		return nil, 0, fmt.Errorf("无权查看私密收藏夹")
	}
	if err := checkFolderBlockAccess(ctx, userId, folder.UserId); err != nil {
		return nil, 0, err
	}

	entries, err := s.discoverDAO.ListCollectionEntries(ctx, folder.Id, userId, page, pageSize, offsetOverride...)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.discoverDAO.CountCollectionEntries(ctx, folder.Id)
	if err != nil {
		return nil, 0, err
	}
	visible := make([]models.DiscoverPost, 0, len(entries))
	items := make(map[string]PostInfo, len(entries))
	for _, entry := range entries {
		reason := collectionUnavailableReason(entry, userId)
		if reason == "" {
			visible = append(visible, entry.DiscoverPost)
			continue
		}
		items[entry.CollectionUuid] = PostInfo{Uuid: entry.CollectionUuid, Title: reason, IsUnavailable: true, LockReason: reason, IsCollected: true, Width: 4, Height: 3, CreatedAt: entry.CollectedAt.Format("2006-01-02 15:04:05")}
	}
	posts, _, err := buildPostInfoList(ctx, s.discoverDAO, s.userInfoDAO, s.blockDAO, userId, visible, total)
	if err != nil {
		return nil, 0, err
	}
	for _, post := range posts {
		items[post.Uuid] = post
	}
	result := make([]PostInfo, 0, len(entries))
	for _, entry := range entries {
		result = append(result, items[entry.CollectionUuid])
	}
	return result, total, nil
}

// CollectPost 保存收藏及通知，并在事务内维护收藏夹和帖子计数。
func (s *DiscoverInteractionServiceImpl) CollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	post, err := s.discoverDAO.FindPostByUuid(ctx, postUuid)
	if err != nil {
		return false, 0, err
	}
	if !canViewUnblockedDiscoverPost(ctx, db.GormDB.WithContext(ctx), post, userId) {
		return false, 0, fmt.Errorf("帖子不存在")
	}

	folder, err := s.discoverDAO.FindFolderByUuid(ctx, folderUuid)
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
	existing, _ := s.discoverDAO.FindCollectionInFolder(ctx, userId, folder.Id, postUuid)
	if existing != nil {
		return false, 0, fmt.Errorf("该帖子已在此收藏夹中")
	}
	previousCollection, err := s.discoverDAO.FindCollectionByUserAndTarget(ctx, userId, postUuid)
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
		if err := txDiscoverDAO.CreateCollection(ctx, col); err != nil {
			return err
		}
		// 收藏多个文件夹时只在首次收藏发送通知，与收藏写入保持同一事务。
		if previousCollection == nil && post.UserId != userId {
			if err := userdao.NewNotificationDAO(tx).CreateNotification(ctx, &models.Notification{
				UserId: post.UserId, ActorId: userId, Type: NotificationTypeCollectPost,
				TargetUuid: postUuid, Content: "收藏了你的帖子：" + truncateContent(post.Title, 50),
			}); err != nil {
				return fmt.Errorf("创建收藏通知失败: %w", err)
			}
		}

		if err := txDiscoverDAO.IncrementFolderPostCount(ctx, folder.Id); err != nil {
			return fmt.Errorf("更新收藏夹帖子数失败: %w", err)
		}
		if err := txDiscoverDAO.IncrementCollectCount(ctx, post.Id); err != nil {
			return fmt.Errorf("更新帖子收藏数失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	updatedPost, _ := s.discoverDAO.FindPostByUuid(ctx, postUuid)
	collectCount := 0
	if updatedPost != nil {
		collectCount = updatedPost.CollectCount
	}
	return true, collectCount, nil
}

// UncollectPost 允许移除失效收藏，事务内加锁并幂等更新收藏夹和帖子计数。
func (s *DiscoverInteractionServiceImpl) UncollectPost(ctx context.Context, userId, postUuid, folderUuid string) (bool, int, error) {
	if userId == "" || postUuid == "" || folderUuid == "" {
		return false, 0, fmt.Errorf("缺少取消收藏所需的信息")
	}
	collectCount := 0
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDiscoverDAO := userdao.NewDiscoverDAO(tx)
		// 与收藏写入的收藏夹计数、帖子计数顺序一致，避免交叉持锁。
		folder, err := txDiscoverDAO.FindFolderForUpdate(ctx, folderUuid)
		if err != nil {
			return fmt.Errorf("查询收藏夹失败: %w", err)
		}
		if folder == nil {
			return fmt.Errorf("收藏夹不存在")
		}
		if folder.UserId != userId {
			return fmt.Errorf("只能取消自己收藏夹中的收藏")
		}
		post, err := txDiscoverDAO.FindCollectionPostForUpdate(ctx, postUuid)
		if err != nil {
			return fmt.Errorf("查询收藏帖子失败: %w", err)
		}
		existing, err := txDiscoverDAO.FindCollectionInFolder(ctx, userId, folder.Id, postUuid)
		if err != nil {
			return fmt.Errorf("查询收藏记录失败: %w", err)
		}
		if post != nil {
			collectCount = post.CollectCount
		}
		if existing != nil {
			if err := txDiscoverDAO.DeleteCollection(ctx, userId, folder.Id, postUuid); err != nil {
				return fmt.Errorf("删除收藏记录失败: %w", err)
			}
			if err := txDiscoverDAO.DecrementFolderPostCount(ctx, folder.Id); err != nil {
				return fmt.Errorf("更新收藏夹帖子数失败: %w", err)
			}
			if post != nil {
				if err := txDiscoverDAO.DecrementCollectionTargetCount(ctx, post.Id); err != nil {
					return fmt.Errorf("更新帖子收藏数失败: %w", err)
				}
				if collectCount > 0 {
					collectCount--
				}
			}
		}
		// 无权查看的帖子只返回取消结果，不暴露真实互动计数。
		if post == nil || !CanViewDiscoverPost(post, userId) {
			collectCount = 0
			return nil
		}
		blocked, by, err := userdao.NewBlockDAO(tx).Status(ctx, userId, post.UserId)
		if err != nil {
			return fmt.Errorf("查询收藏帖子权限失败: %w", err)
		}
		if blocked || by {
			collectCount = 0
		}
		return nil
	})
	if err != nil {
		return false, 0, err
	}

	return false, collectCount, nil
}

// CheckCollected 仅对当前身份可访问的作品查询收藏状态。
func (s *DiscoverInteractionServiceImpl) CheckCollected(ctx context.Context, userId, postUuid string) (bool, string, error) {
	post, err := s.discoverDAO.FindPostByUuid(ctx, postUuid)
	if err != nil {
		return false, "", err
	}
	if !canViewUnblockedDiscoverPost(ctx, db.GormDB.WithContext(ctx), post, userId) {
		return false, "", fmt.Errorf("作品不存在或不可见")
	}
	col, _ := s.discoverDAO.FindCollectionByUserAndTarget(ctx, userId, postUuid)
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

// truncateContent 截断内容，超过 maxLen 加省略号
func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// buildCommentInfoList 批量装配评论作者、回复数量与当前用户点赞状态。
func (s *DiscoverInteractionServiceImpl) buildCommentInfoList(ctx context.Context, userId string, comments []models.DiscoverComment) ([]CommentInfo, error) {
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
	users, err := s.userInfoDAO.FindUsersByUuids(ctx, userIds)
	if err != nil {
		return nil, fmt.Errorf("查询评论用户失败: %w", err)
	}
	userMap := make(map[string]models.UserInfo)
	for _, u := range users {
		userMap[u.Uuid] = u
	}
	likes, err := s.discoverDAO.FindLikesByUserIdAndTargetUuids(ctx, userId, uuids)
	if err != nil {
		return nil, fmt.Errorf("查询评论点赞失败: %w", err)
	}
	liked := make(map[string]bool)
	for _, like := range likes {
		liked[like.TargetUuid] = true
	}
	replies, err := s.discoverDAO.CountReplies(ctx, uuids)
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		user, replyUser := userMap[c.UserId], userMap[c.ReplyToUserId]
		result = append(result, CommentInfo{Uuid: c.Uuid, UserId: c.UserId, Nickname: user.Nickname, Avatar: user.Avatar, ParentId: c.ParentId, ReplyToUserId: c.ReplyToUserId, ReplyToNickname: replyUser.Nickname, Content: c.Content, LikeCount: c.LikeCount, IsLiked: liked[c.Uuid], ReplyCount: replies[c.Uuid], CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	return result, nil
}

// checkFolderBlockAccess 限制拉黑双方读取对方收藏夹详情及夹内帖子，不影响本人管理。
func checkFolderBlockAccess(ctx context.Context, viewerId, ownerId string) error {
	if viewerId == ownerId {
		return nil
	}
	blocked, by, err := userdao.NewBlockDAO(db.GormDB.WithContext(ctx)).Status(ctx, viewerId, ownerId)
	if err != nil {
		return fmt.Errorf("查询收藏夹拉黑权限失败: %w", err)
	}
	if by {
		return fmt.Errorf("已被拉黑")
	}
	if blocked {
		return fmt.Errorf("你已拉黑该用户")
	}
	return nil
}

// collectionUnavailableReason 为失效收藏生成不含原正文和媒体的状态说明。
func collectionUnavailableReason(entry userdao.CollectionPostRow, viewerId string) string {
	if entry.Id == 0 || entry.DeletedAt.Valid {
		return "帖子已删除"
	}
	if entry.IsAuthorBlocked {
		return "你已拉黑该作者"
	}
	if entry.Status == 1 {
		return "帖子已下架"
	}
	if entry.Status != 0 {
		return "帖子暂不可见"
	}
	if entry.Visibility != 0 && entry.UserId != viewerId {
		return "帖子已设为私密"
	}
	return ""
}
