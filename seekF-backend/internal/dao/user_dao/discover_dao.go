package userdao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"seekF-backend/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DiscoverMediaInfo 媒体展示查询结果，附带资源和帖子封面的尺寸，不对应数据库表。
type DiscoverMediaInfo struct {
	models.DiscoverMedia
	Width  int // 媒体宽度，单位像素
	Height int // 媒体高度，单位像素
}

// DiscoverDAO 发现页帖子、媒体和互动数据访问接口。
type DiscoverDAO interface {
	// 查询关注作者及公开新作品的更新边界。
	FollowingUpdates(ctx context.Context, userId string) ([]FollowingAuthorUpdate, error)
	// 在当前关注与公开权限内查询指定作者的时间流。
	ListFollowingAuthorFeed(ctx context.Context, userId, author string, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error)
	// 清理过期或当前已不可见的收件箱候选。
	CleanupFeedInbox(ctx context.Context, from time.Time, limit int) error
	// 合并收件箱与拉取候选并补查未覆盖内容。
	ListHybridFeed(ctx context.Context, userId string, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error)
	// 幂等保存当天首次点击，返回是否实际新增。
	SaveClick(click *models.DiscoverClick) (bool, error)
	// 设置帖子或作者不感兴趣状态。
	SetFeedback(feedback *models.DiscoverFeedback, active bool) error
	// 在时间流降级时同步过滤当前用户的不感兴趣目标。
	ListRecommendationFeed(ctx context.Context, userId string, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error)
	// 批量核查当前公开状态与推荐负反馈，不依赖缓存权限。
	FindRankingPosts(ctx context.Context, userId string, ids []string, mediaType *int8, recommended bool) ([]models.DiscoverPost, error)
	// 查询近期、热门或兴趣关注召回候选及窗口内真实互动分。
	ListRankingCandidates(ctx context.Context, userId, kind string, tags []string, mediaType *int8, now time.Time) ([]RankingCandidate, error)
	// 读取用户最近30天与当前公开帖子的真实互动。
	ListInterestHistory(ctx context.Context, userId string, now time.Time) ([]InterestHistory, error)
	// 读取当前用户关注的作者标识。
	ListFollowedAuthorIds(ctx context.Context, userId string) ([]string, error)
	// 批量读取候选作品的当前公开状态，不能使用ES中的旧正文。
	FindPublicPostsByUuids(ctx context.Context, uuids []string, viewerId string) ([]models.DiscoverPost, error)
	// 返回作者各状态作品数量，不受当前选中状态限制。
	CountOwnWorksByStatus(userId, keyword string, visibility *int8) (map[int8]int64, error)
	// 按作者、状态及标题筛选作品并返回真实总数。
	ListOwnWorks(userId, keyword string, status, visibility *int8, page, pageSize int) ([]models.DiscoverPost, int64, error)
	// 更新作品内容与编辑版本。
	SavePost(post *models.DiscoverPost) error
	// 软删除指定作品。
	DeletePost(post *models.DiscoverPost) error
	// 清理编辑前的媒体记录。
	DeletePostMedia(postId int64) error
	// 依赖用户请求唯一键占用发布记录，返回是否创建成功。
	ClaimPublishPost(post *models.DiscoverPost) (bool, error)
	// 包含软删除记录地锁定发布请求对应帖子。
	FindPostByPublishKeyForUpdate(userId, requestKey string) (*models.DiscoverPost, error)
	// ========== 帖子 ==========
	// 保存帖子。
	CreatePost(post *models.DiscoverPost) error
	// 按 UUID 查询帖子，不存在时返回空。
	FindPostByUuid(uuid string) (*models.DiscoverPost, error)
	// 查询并锁定帖子，须使用事务连接调用。
	FindPostByUuidForUpdate(uuid string) (*models.DiscoverPost, error)
	// 按主键查询帖子，不存在时返回空。
	FindPostById(id int64) (*models.DiscoverPost, error)
	// 查询当前可见的时间流，关注子查询避免重复关系产生重复帖子。
	ListFeed(ctx context.Context, userId string, following bool, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error)
	// 判断当前用户是否关注过博主。
	HasFollowing(ctx context.Context, userId string) (bool, error)
	// 增加帖子的评论数。
	IncrementCommentCount(postId int64) error
	// 增加帖子的浏览数。
	IncrementViewCount(postId int64) error
	// 分页查询指定用户的正常帖子。
	ListPostsByUserId(userId, viewerId string, page, pageSize int) ([]models.DiscoverPost, error)
	// 统计指定用户的正常帖子数量。
	CountPostsByUserId(userId, viewerId string) (int64, error)
	// 统计指定用户正常帖子的总获赞数。
	GetUserTotalLikes(userId, viewerId string) (int, error)

	// ========== 搜索 ==========
	// 在全文检索不可用时按关键词分页查询并返回真实总数。
	SearchPostsPage(keyword string, page, pageSize int, viewerId string) ([]models.DiscoverPost, int64, error)

	// ========== 媒体 ==========
	// 保存帖子媒体信息。
	CreateMedia(media *models.DiscoverMedia) error
	// 按展示顺序查询帖子的媒体。
	FindMediaByPostId(postId int64) ([]DiscoverMediaInfo, error)
	// 批量查询帖子媒体并按帖子及展示顺序排序。
	FindMediaByPostIds(postIds []int64) ([]DiscoverMediaInfo, error)

	// ========== 点赞 ==========
	// 统计指定目标的去重点赞用户数。
	CountLikesByTargetUuid(targetUuid string) (int, error)
	// 更新帖子的实际点赞数。
	UpdatePostLikeCount(postId int64, count int) error
	// 更新评论的实际点赞数。
	UpdateCommentLikeCount(commentId int64, count int) error
	// 保存点赞记录。
	CreateLike(like *models.DiscoverLike) error
	// 删除用户对指定目标的点赞记录。
	DeleteLike(userId, targetUuid string) error
	// 查询用户对指定目标的点赞记录。
	FindLike(userId, targetUuid string) (*models.DiscoverLike, error)
	// 批量查询用户对多个目标的点赞状态
	FindLikesByUserIdAndTargetUuids(userId string, targetUuids []string) ([]models.DiscoverLike, error)
	// 分页查询用户点赞的正常帖子。
	ListLikedPosts(userId string, page, pageSize int) ([]models.DiscoverPost, error)
	// 统计用户点赞的正常帖子数量。
	CountLikedPosts(userId string) (int64, error)
	// 增加帖子的点赞数。
	IncrementLikeCount(postId int64) error
	// 减少帖子的点赞数。
	DecrementLikeCount(postId int64) error

	// ========== 评论 ==========
	// 保存评论。
	CreateComment(comment *models.DiscoverComment) error
	// 按主键查询评论，不存在时返回空。
	FindCommentById(commentId int64) (*models.DiscoverComment, error)
	// 按 UUID 查询评论，不存在时返回空。
	FindCommentByUuid(uuid string) (*models.DiscoverComment, error)
	// 查询并锁定评论，须使用事务连接调用。
	FindCommentByUuidForUpdate(uuid string) (*models.DiscoverComment, error)
	// 按时间及 ID 稳定分页读取全部评论。
	FindCommentsByPostId(postId int64, page, pageSize int) ([]models.DiscoverComment, error)
	// 按稳定顺序分页读取顶级评论或同一父评论的回复。
	FindCommentThread(postId int64, parentUuid string, page, pageSize int) ([]models.DiscoverComment, bool, error)
	// 批量统计顶级评论的回复数量。
	CountReplies(parentUuids []string) (map[string]int64, error)
	// 增加评论的点赞数。
	IncrementCommentLikeCount(commentId int64) error
	// 减少评论的点赞数。
	DecrementCommentLikeCount(commentId int64) error

	// ========== 收藏夹 ==========
	// 创建收藏夹。
	CreateFolder(folder *models.DiscoverCollectionFolder) error
	// 更新收藏夹名称、描述和公开状态。
	UpdateFolder(uuid string, name, description string, isPublic int8) error
	// 删除指定收藏夹。
	DeleteFolder(uuid string) error
	// 按 UUID 查询收藏夹，不存在时返回空。
	FindFolderByUuid(uuid string) (*models.DiscoverCollectionFolder, error)
	// 查询用户的收藏夹列表。
	ListFoldersByUserId(userId string) ([]models.DiscoverCollectionFolder, error)
	// 查询用户的默认收藏夹。
	FindDefaultFolder(userId string) (*models.DiscoverCollectionFolder, error)

	// ========== 收藏记录 ==========
	// 保存收藏记录。
	CreateCollection(col *models.DiscoverCollection) error
	// 删除用户在指定收藏夹中的目标收藏记录。
	DeleteCollection(userId string, folderId int64, targetUuid string) error
	// 查询用户在指定收藏夹中的目标收藏记录。
	FindCollectionInFolder(userId string, folderId int64, targetUuid string) (*models.DiscoverCollection, error)
	// 查询用户对指定目标的收藏记录。
	FindCollectionByUserAndTarget(userId, targetUuid string) (*models.DiscoverCollection, error)
	// 批量查询用户对多个目标的收藏状态
	FindCollectionsByUserIdAndTargetUuids(userId string, targetUuids []string) ([]models.DiscoverCollection, error)
	// 增加收藏夹的帖子数。
	IncrementFolderPostCount(folderId int64) error
	// 减少收藏夹的帖子数。
	DecrementFolderPostCount(folderId int64) error
	// 增加帖子的收藏数。
	IncrementCollectCount(postId int64) error
	// 减少帖子的收藏数。
	DecrementCollectCount(postId int64) error
	// 分页查询收藏夹中的正常帖子。
	ListCollectedPostsByFolder(folderId int64, viewerId string, page, pageSize int) ([]models.DiscoverPost, error)
	// 统计收藏夹中的正常帖子数量。
	CountCollectedPostsByFolder(folderId int64, viewerId string) (int64, error)
}

// DiscoverDAOImpl 发现页数据访问实现。
type DiscoverDAOImpl struct {
	db *gorm.DB
}

// NewDiscoverDAO 创建发现页数据访问实例。
func NewDiscoverDAO(db *gorm.DB) DiscoverDAO {
	return &DiscoverDAOImpl{db: db}
}

// ========== 帖子 ==========

// CreatePost 保存帖子。
func (d *DiscoverDAOImpl) CreatePost(post *models.DiscoverPost) error {
	return d.db.Create(post).Error
}

// FindPostByUuid 按 UUID 查询帖子，不存在时返回空。
func (d *DiscoverDAOImpl) FindPostByUuid(uuid string) (*models.DiscoverPost, error) {
	var post models.DiscoverPost
	result := d.db.Where("uuid = ?", uuid).First(&post)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &post, result.Error
}

// FindPostByUuidForUpdate 查询并锁定帖子，须使用事务连接调用。
func (d *DiscoverDAOImpl) FindPostByUuidForUpdate(uuid string) (*models.DiscoverPost, error) {
	var post models.DiscoverPost
	result := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("uuid = ?", uuid).First(&post)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &post, result.Error
}

// FindPostById 按主键查询帖子，不存在时返回空。
func (d *DiscoverDAOImpl) FindPostById(id int64) (*models.DiscoverPost, error) {
	var post models.DiscoverPost
	result := d.db.First(&post, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &post, result.Error
}

// ListFeed 使用时间和主键作为边界读取公开已发布帖子，额外取一条判断是否还有内容。
func (d *DiscoverDAOImpl) ListFeed(ctx context.Context, userId string, following bool, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error) {
	posts := make([]models.DiscoverPost, 0)
	query := WithoutHiddenAuthors(d.db.WithContext(ctx), userId, "discover_post").Where("status = 0 AND visibility = 0")
	if following {
		follows := d.db.WithContext(ctx).Model(&models.UserFollow{}).Select("follow_user_id").Where("user_id = ?", userId)
		query = query.Where("user_id IN (?)", follows)
	}
	if mediaType != nil {
		query = query.Where("media_type = ?", *mediaType)
	}
	if !before.IsZero() {
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", before, before, beforeId)
	}
	err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

// HasFollowing 只读取一条关注关系，区分尚未关注与关注作者暂无公开作品。
func (d *DiscoverDAOImpl) HasFollowing(ctx context.Context, userId string) (bool, error) {
	var follows []models.UserFollow
	err := d.db.WithContext(ctx).Select("id").Where("user_id = ?", userId).Limit(1).Find(&follows).Error
	return len(follows) > 0, err
}

// IncrementCommentCount 增加帖子的评论数。
func (d *DiscoverDAOImpl) IncrementCommentCount(postId int64) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
}

// IncrementViewCount 增加帖子的浏览数。
func (d *DiscoverDAOImpl) IncrementViewCount(postId int64) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// ListPostsByUserId 分页查询指定用户的正常帖子。
func (d *DiscoverDAOImpl) ListPostsByUserId(userId, viewerId string, page, pageSize int) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	offset := (page - 1) * pageSize
	result := WithoutHiddenAuthors(d.db, viewerId, "discover_post").Where("user_id = ? AND status = ? AND (visibility = 0 OR user_id = ?)", userId, 0, viewerId).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts)
	return posts, result.Error
}

// CountPostsByUserId 统计指定用户的正常帖子数量。
func (d *DiscoverDAOImpl) CountPostsByUserId(userId, viewerId string) (int64, error) {
	var count int64
	result := WithoutHiddenAuthors(d.db, viewerId, "discover_post").Model(&models.DiscoverPost{}).Where("user_id = ? AND status = ? AND (visibility = 0 OR user_id = ?)", userId, 0, viewerId).Count(&count)
	return count, result.Error
}

// GetUserTotalLikes 统计指定用户正常帖子的总获赞数。
func (d *DiscoverDAOImpl) GetUserTotalLikes(userId, viewerId string) (int, error) {
	var total int
	result := WithoutHiddenAuthors(d.db, viewerId, "discover_post").Model(&models.DiscoverPost{}).Where("user_id = ? AND status = ? AND (visibility = 0 OR user_id = ?)", userId, 0, viewerId).Select("COALESCE(SUM(like_count), 0)").Scan(&total)
	return total, result.Error
}

// ========== 搜索 ==========

// SearchPostsPage 在全文检索不可用时按关键词分页查询并返回真实总数。
func (d *DiscoverDAOImpl) SearchPostsPage(keyword string, page, pageSize int, viewerId string) ([]models.DiscoverPost, int64, error) {
	query := d.db.Model(&models.DiscoverPost{}).Where("status = 0 AND visibility = 0")
	query = WithoutHiddenAuthors(query, viewerId, "discover_post")
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		tag, err := json.Marshal(keyword)
		if err != nil {
			return nil, 0, fmt.Errorf("处理搜索关键词失败: %w", err)
		}
		query = query.Where("title LIKE ? OR content LIKE ? OR JSON_CONTAINS(tags, ?)", "%"+keyword+"%", "%"+keyword+"%", string(tag))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计搜索结果失败: %w", err)
	}
	posts := make([]models.DiscoverPost, 0)
	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		return nil, 0, fmt.Errorf("查询搜索结果失败: %w", err)
	}
	return posts, total, nil
}

// ========== 媒体 ==========

// CreateMedia 保存帖子媒体信息。
func (d *DiscoverDAOImpl) CreateMedia(media *models.DiscoverMedia) error {
	return d.db.Create(media).Error
}

// FindMediaByPostId 按展示顺序查询帖子的媒体。
func (d *DiscoverDAOImpl) FindMediaByPostId(postId int64) ([]DiscoverMediaInfo, error) {
	var mediaList []DiscoverMediaInfo
	result := d.mediaWithAssets().Where("discover_media.post_id = ?", postId).Order("discover_media.sort_order ASC").Find(&mediaList)
	return mediaList, result.Error
}

// FindMediaByPostIds 批量查询帖子媒体并按帖子及展示顺序排序。
func (d *DiscoverDAOImpl) FindMediaByPostIds(postIds []int64) ([]DiscoverMediaInfo, error) {
	if len(postIds) == 0 {
		return nil, nil
	}
	var mediaList []DiscoverMediaInfo
	result := d.mediaWithAssets().Where("discover_media.post_id IN ?", postIds).Order("discover_media.post_id ASC, discover_media.sort_order ASC").Find(&mediaList)
	return mediaList, result.Error
}

// ========== 点赞 ==========

// CountLikesByTargetUuid 统计指定目标的去重点赞用户数。
func (d *DiscoverDAOImpl) CountLikesByTargetUuid(targetUuid string) (int, error) {
	var total int64
	err := d.db.Model(&models.DiscoverLike{}).Where("target_uuid = ?", targetUuid).Distinct("user_id").Count(&total).Error
	return int(total), err
}

// UpdatePostLikeCount 更新帖子的实际点赞数。
func (d *DiscoverDAOImpl) UpdatePostLikeCount(postId int64, count int) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("like_count", count).Error
}

// UpdateCommentLikeCount 更新评论的实际点赞数。
func (d *DiscoverDAOImpl) UpdateCommentLikeCount(commentId int64, count int) error {
	return d.db.Model(&models.DiscoverComment{}).Where("id = ?", commentId).UpdateColumn("like_count", count).Error
}

// CreateLike 保存点赞记录。
func (d *DiscoverDAOImpl) CreateLike(like *models.DiscoverLike) error {
	return d.db.Create(like).Error
}

// DeleteLike 删除用户对指定目标的点赞记录。
func (d *DiscoverDAOImpl) DeleteLike(userId, targetUuid string) error {
	return d.db.Where("user_id = ? AND target_uuid = ?", userId, targetUuid).Delete(&models.DiscoverLike{}).Error
}

// FindLike 查询用户对指定目标的点赞记录。
func (d *DiscoverDAOImpl) FindLike(userId, targetUuid string) (*models.DiscoverLike, error) {
	var like models.DiscoverLike
	result := d.db.Where("user_id = ? AND target_uuid = ?", userId, targetUuid).Find(&like)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &like, nil
}

// FindLikesByUserIdAndTargetUuids 批量查询用户对多个目标的点赞状态
func (d *DiscoverDAOImpl) FindLikesByUserIdAndTargetUuids(userId string, targetUuids []string) ([]models.DiscoverLike, error) {
	if len(targetUuids) == 0 {
		return nil, nil
	}
	var likes []models.DiscoverLike
	result := d.db.Where("user_id = ? AND target_uuid IN ?", userId, targetUuids).Find(&likes)
	return likes, result.Error
}

// ListLikedPosts 分页查询用户点赞的正常帖子。
func (d *DiscoverDAOImpl) ListLikedPosts(userId string, page, pageSize int) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	offset := (page - 1) * pageSize
	// 子查询：获取用户点赞的帖子 UUID
	likedUuids := d.db.Model(&models.DiscoverLike{}).
		Where("user_id = ?", userId).
		Select("target_uuid")
	// 查询帖子
	result := WithoutHiddenAuthors(d.db, userId, "discover_post").Where("status = ? AND (visibility = 0 OR user_id = ?) AND uuid IN (?)", 0, userId, likedUuids).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&posts)
	return posts, result.Error
}

// CountLikedPosts 统计用户点赞的正常帖子数量。
func (d *DiscoverDAOImpl) CountLikedPosts(userId string) (int64, error) {
	var count int64
	// 子查询：获取用户点赞的帖子 UUID
	likedUuids := d.db.Model(&models.DiscoverLike{}).
		Where("user_id = ?", userId).
		Select("target_uuid")
	result := WithoutHiddenAuthors(d.db, userId, "discover_post").Model(&models.DiscoverPost{}).
		Where("status = ? AND (visibility = 0 OR user_id = ?) AND uuid IN (?)", 0, userId, likedUuids).
		Count(&count)
	return count, result.Error
}

// IncrementLikeCount 增加帖子的点赞数。
func (d *DiscoverDAOImpl) IncrementLikeCount(postId int64) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}

// DecrementLikeCount 减少帖子的点赞数。
func (d *DiscoverDAOImpl) DecrementLikeCount(postId int64) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
}

// ========== 评论 ==========

// CreateComment 保存评论。
func (d *DiscoverDAOImpl) CreateComment(comment *models.DiscoverComment) error {
	return d.db.Create(comment).Error
}

// FindCommentById 按主键查询评论，不存在时返回空。
func (d *DiscoverDAOImpl) FindCommentById(commentId int64) (*models.DiscoverComment, error) {
	var comment models.DiscoverComment
	result := d.db.Where("id = ?", commentId).First(&comment)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &comment, result.Error
}

// FindCommentByUuid 按 UUID 查询评论，不存在时返回空。
func (d *DiscoverDAOImpl) FindCommentByUuid(uuid string) (*models.DiscoverComment, error) {
	var comment models.DiscoverComment
	result := d.db.Where("uuid = ?", uuid).First(&comment)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &comment, result.Error
}

// FindCommentByUuidForUpdate 查询并锁定评论，须使用事务连接调用。
func (d *DiscoverDAOImpl) FindCommentByUuidForUpdate(uuid string) (*models.DiscoverComment, error) {
	var comment models.DiscoverComment
	result := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("uuid = ?", uuid).First(&comment)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &comment, result.Error
}

// FindCommentsByPostId 按时间及 ID 稳定分页读取全部评论。
func (d *DiscoverDAOImpl) FindCommentsByPostId(postId int64, page, pageSize int) ([]models.DiscoverComment, error) {
	var comments []models.DiscoverComment
	offset := (page - 1) * pageSize
	result := d.db.Where("post_id = ?", postId).Order("created_at ASC, id ASC").Offset(offset).Limit(pageSize).Find(&comments)
	return comments, result.Error
}

// FindCommentThread 按稳定顺序分页读取顶级评论或同一父评论的回复。
func (d *DiscoverDAOImpl) FindCommentThread(postId int64, parentUuid string, page, pageSize int) ([]models.DiscoverComment, bool, error) {
	query := d.db.Where("post_id = ?", postId)
	if parentUuid == "" {
		query = query.Where("parent_id = '' OR parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentUuid)
	}
	comments := make([]models.DiscoverComment, 0)
	err := query.Order("created_at ASC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize + 1).Find(&comments).Error
	if err != nil {
		return nil, false, fmt.Errorf("查询评论分页失败: %w", err)
	}
	hasMore := len(comments) > pageSize
	if hasMore {
		comments = comments[:pageSize]
	}
	return comments, hasMore, nil
}

// CountReplies 批量统计顶级评论的回复数量。
func (d *DiscoverDAOImpl) CountReplies(parentUuids []string) (map[string]int64, error) {
	counts := make(map[string]int64)
	if len(parentUuids) == 0 {
		return counts, nil
	}
	var rows []struct {
		ParentId string
		Total    int64
	}
	err := d.db.Model(&models.DiscoverComment{}).Select("parent_id, COUNT(*) AS total").Where("parent_id IN ?", parentUuids).Group("parent_id").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("统计回复数量失败: %w", err)
	}
	for _, row := range rows {
		counts[row.ParentId] = row.Total
	}
	return counts, nil
}

// IncrementCommentLikeCount 增加评论的点赞数。
func (d *DiscoverDAOImpl) IncrementCommentLikeCount(commentId int64) error {
	return d.db.Model(&models.DiscoverComment{}).Where("id = ?", commentId).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}

// DecrementCommentLikeCount 减少评论的点赞数。
func (d *DiscoverDAOImpl) DecrementCommentLikeCount(commentId int64) error {
	return d.db.Model(&models.DiscoverComment{}).Where("id = ?", commentId).UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
}

// ========== 收藏夹 ==========

// CreateFolder 创建收藏夹。
func (d *DiscoverDAOImpl) CreateFolder(folder *models.DiscoverCollectionFolder) error {
	return d.db.Create(folder).Error
}

// UpdateFolder 更新收藏夹名称、描述和公开状态。
func (d *DiscoverDAOImpl) UpdateFolder(uuid string, name, description string, isPublic int8) error {
	return d.db.Model(&models.DiscoverCollectionFolder{}).
		Where("uuid = ?", uuid).
		Updates(map[string]interface{}{
			"name":        name,
			"description": description,
			"is_public":   isPublic,
		}).Error
}

// DeleteFolder 删除指定收藏夹。
func (d *DiscoverDAOImpl) DeleteFolder(uuid string) error {
	return d.db.Where("uuid = ?", uuid).Delete(&models.DiscoverCollectionFolder{}).Error
}

// FindFolderByUuid 按 UUID 查询收藏夹，不存在时返回空。
func (d *DiscoverDAOImpl) FindFolderByUuid(uuid string) (*models.DiscoverCollectionFolder, error) {
	var folder models.DiscoverCollectionFolder
	result := d.db.Where("uuid = ?", uuid).First(&folder)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &folder, result.Error
}

// ListFoldersByUserId 查询用户的收藏夹列表。
func (d *DiscoverDAOImpl) ListFoldersByUserId(userId string) ([]models.DiscoverCollectionFolder, error) {
	var folders []models.DiscoverCollectionFolder
	result := d.db.Where("user_id = ?", userId).Order("created_at DESC").Find(&folders)
	return folders, result.Error
}

// FindDefaultFolder 查询用户的默认收藏夹。
func (d *DiscoverDAOImpl) FindDefaultFolder(userId string) (*models.DiscoverCollectionFolder, error) {
	var folder models.DiscoverCollectionFolder
	result := d.db.Where("user_id = ? AND name = ?", userId, "默认收藏夹").First(&folder)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &folder, result.Error
}

// ========== 收藏记录 ==========

// CreateCollection 保存收藏记录。
func (d *DiscoverDAOImpl) CreateCollection(col *models.DiscoverCollection) error {
	return d.db.Create(col).Error
}

// DeleteCollection 删除用户在指定收藏夹中的目标收藏记录。
func (d *DiscoverDAOImpl) DeleteCollection(userId string, folderId int64, targetUuid string) error {
	return d.db.Where("user_id = ? AND folder_id = ? AND target_uuid = ?", userId, folderId, targetUuid).
		Delete(&models.DiscoverCollection{}).Error
}

// FindCollectionInFolder 查询用户在指定收藏夹中的目标收藏记录。
func (d *DiscoverDAOImpl) FindCollectionInFolder(userId string, folderId int64, targetUuid string) (*models.DiscoverCollection, error) {
	var col models.DiscoverCollection
	result := d.db.Where("user_id = ? AND folder_id = ? AND target_uuid = ?", userId, folderId, targetUuid).Find(&col)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &col, result.Error
}

// FindCollectionByUserAndTarget 查询用户对指定目标的收藏记录。
func (d *DiscoverDAOImpl) FindCollectionByUserAndTarget(userId, targetUuid string) (*models.DiscoverCollection, error) {
	var col models.DiscoverCollection
	result := d.db.Where("user_id = ? AND target_uuid = ?", userId, targetUuid).Find(&col)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &col, result.Error
}

// FindCollectionsByUserIdAndTargetUuids 批量查询用户对多个目标的收藏状态
func (d *DiscoverDAOImpl) FindCollectionsByUserIdAndTargetUuids(userId string, targetUuids []string) ([]models.DiscoverCollection, error) {
	if len(targetUuids) == 0 {
		return nil, nil
	}
	var collections []models.DiscoverCollection
	result := d.db.Where("user_id = ? AND target_uuid IN ?", userId, targetUuids).Find(&collections)
	return collections, result.Error
}

// IncrementFolderPostCount 增加收藏夹的帖子数。
func (d *DiscoverDAOImpl) IncrementFolderPostCount(folderId int64) error {
	return d.db.Model(&models.DiscoverCollectionFolder{}).Where("id = ?", folderId).UpdateColumn("post_count", gorm.Expr("post_count + 1")).Error
}

// DecrementFolderPostCount 减少收藏夹的帖子数。
func (d *DiscoverDAOImpl) DecrementFolderPostCount(folderId int64) error {
	return d.db.Model(&models.DiscoverCollectionFolder{}).Where("id = ?", folderId).UpdateColumn("post_count", gorm.Expr("post_count - 1")).Error
}

// IncrementCollectCount 增加帖子的收藏数。
func (d *DiscoverDAOImpl) IncrementCollectCount(postId int64) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("collect_count", gorm.Expr("collect_count + 1")).Error
}

// DecrementCollectCount 减少帖子的收藏数。
func (d *DiscoverDAOImpl) DecrementCollectCount(postId int64) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("collect_count", gorm.Expr("collect_count - 1")).Error
}

// ListCollectedPostsByFolder 分页查询收藏夹中的正常帖子。
func (d *DiscoverDAOImpl) ListCollectedPostsByFolder(folderId int64, viewerId string, page, pageSize int) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	offset := (page - 1) * pageSize
	// 子查询：获取该收藏夹中的帖子 UUID
	collectedUuids := d.db.Model(&models.DiscoverCollection{}).
		Where("folder_id = ?", folderId).
		Select("target_uuid")
	result := WithoutHiddenAuthors(d.db, viewerId, "discover_post").Where("status = ? AND (visibility = 0 OR user_id = ?) AND uuid IN (?)", 0, viewerId, collectedUuids).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&posts)
	return posts, result.Error
}

// CountCollectedPostsByFolder 统计收藏夹中的正常帖子数量。
func (d *DiscoverDAOImpl) CountCollectedPostsByFolder(folderId int64, viewerId string) (int64, error) {
	var count int64
	collectedUuids := d.db.Model(&models.DiscoverCollection{}).
		Where("folder_id = ?", folderId).
		Select("target_uuid")
	result := WithoutHiddenAuthors(d.db, viewerId, "discover_post").Model(&models.DiscoverPost{}).
		Where("status = ? AND (visibility = 0 OR user_id = ?) AND uuid IN (?)", 0, viewerId, collectedUuids).
		Count(&count)
	return count, result.Error
}

// ========== 作品管理与发布幂等 ==========

// SavePost 更新作品内容与编辑版本。
func (d *DiscoverDAOImpl) SavePost(post *models.DiscoverPost) error { return d.db.Save(post).Error }

// DeletePost 软删除指定作品，保留业务记录供审计和幂等重试识别。
func (d *DiscoverDAOImpl) DeletePost(post *models.DiscoverPost) error { return d.db.Delete(post).Error }

// DeletePostMedia 清理编辑前的媒体记录。
func (d *DiscoverDAOImpl) DeletePostMedia(postId int64) error {
	return d.db.Where("post_id = ?", postId).Delete(&models.DiscoverMedia{}).Error
}

// ClaimPublishPost 依赖唯一键占用发布记录，冲突时不覆盖原帖子。
func (d *DiscoverDAOImpl) ClaimPublishPost(post *models.DiscoverPost) (bool, error) {
	result := d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(post)
	return result.RowsAffected == 1, result.Error
}

// FindPostByPublishKeyForUpdate 查询并锁定发布对应帖子，软删除记录也用于防重。
func (d *DiscoverDAOImpl) FindPostByPublishKeyForUpdate(userId, requestKey string) (*models.DiscoverPost, error) {
	var post models.DiscoverPost
	err := d.db.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND publish_request_key = ?", userId, requestKey).First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &post, err
}

// mediaWithAssets 从资源表读取媒体原尺寸，帖子封面尺寸由帖子字段提供。
func (d *DiscoverDAOImpl) mediaWithAssets() *gorm.DB {
	return d.db.Model(&models.DiscoverMedia{}).Select("discover_media.*, COALESCE(a.width, 0) AS width, COALESCE(a.height, 0) AS height").Joins("LEFT JOIN file_asset a ON a.id = discover_media.asset_id")
}

// ListOwnWorks 按作者、作品状态及搜索条件分页查询，软删除记录由GORM排除。
func (d *DiscoverDAOImpl) ListOwnWorks(userId, keyword string, status, visibility *int8, page, pageSize int) ([]models.DiscoverPost, int64, error) {
	query := d.db.Model(&models.DiscoverPost{}).Where("user_id = ?", userId)
	state := int8(0)
	if status != nil {
		state = *status
	}
	query = query.Where("status = ?", state)
	if visibility != nil {
		query = query.Where("visibility = ?", *visibility)
	}
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计作者作品失败: %w", err)
	}
	var posts []models.DiscoverPost
	err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error
	return posts, total, err
}

// CountOwnWorksByStatus 按作者、标题和公开范围统计所有状态，补齐无记录状态的零值。
func (d *DiscoverDAOImpl) CountOwnWorksByStatus(userId, keyword string, visibility *int8) (map[int8]int64, error) {
	query := d.db.Model(&models.DiscoverPost{}).Where("user_id = ? AND status IN (0,1,2,3)", userId)
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}
	if visibility != nil {
		query = query.Where("visibility = ?", *visibility)
	}
	var rows []struct {
		Status int8
		Total  int64
	}
	if err := query.Select("status, COUNT(*) AS total").Group("status").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计作品状态数量失败: %w", err)
	}
	counts := map[int8]int64{0: 0, 1: 0, 2: 0, 3: 0}
	for _, row := range rows {
		counts[row.Status] = row.Total
	}
	return counts, nil
}

// FindPublicPostsByUuids 统一过滤已下架、私密及软删除作品，返回当前数据库内容。
func (d *DiscoverDAOImpl) FindPublicPostsByUuids(ctx context.Context, uuids []string, viewerId string) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	if len(uuids) == 0 {
		return posts, nil
	}
	query := d.db.WithContext(ctx).Where("uuid IN ? AND status=0 AND visibility=0", uuids)
	query = WithoutHiddenAuthors(query, viewerId, "discover_post")
	err := query.Find(&posts).Error
	return posts, err
}
