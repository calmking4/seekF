package userdao

import (
	"errors"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

type DiscoverDAO interface {
	CreatePost(post *models.DiscoverPost) error
	FindPostByUuid(uuid string) (*models.DiscoverPost, error)
	ListPosts(page, pageSize int) ([]models.DiscoverPost, error)
	SearchPostsByKeyword(keyword string, limit int) ([]models.DiscoverPost, error)
	ListLikedPosts(userId string, page, pageSize int) ([]models.DiscoverPost, error)
	CountPosts() (int64, error)
	CountLikedPosts(userId string) (int64, error)
	IncrementLikeCount(postId int64) error
	DecrementLikeCount(postId int64) error
	IncrementCommentCount(postId int64) error

	CreateMedia(media *models.DiscoverMedia) error
	FindMediaByPostId(postId int64) ([]models.DiscoverMedia, error)

	CreateLike(like *models.DiscoverLike) error
	DeleteLike(userId, targetUuid string) error
	FindLike(userId, targetUuid string) (*models.DiscoverLike, error)

	CreateComment(comment *models.DiscoverComment) error
	FindCommentById(commentId int64) (*models.DiscoverComment, error)
	FindCommentByUuid(uuid string) (*models.DiscoverComment, error)
	FindCommentsByPostId(postId int64, page, pageSize int) ([]models.DiscoverComment, error)
	IncrementCommentLikeCount(commentId int64) error
	DecrementCommentLikeCount(commentId int64) error

	// 收藏夹
	CreateFolder(folder *models.DiscoverCollectionFolder) error
	UpdateFolder(uuid string, name, description string, isPublic int8) error
	DeleteFolder(uuid string) error
	FindFolderByUuid(uuid string) (*models.DiscoverCollectionFolder, error)
	ListFoldersByUserId(userId string) ([]models.DiscoverCollectionFolder, error)
	FindDefaultFolder(userId string) (*models.DiscoverCollectionFolder, error)

	// 收藏记录
	CreateCollection(col *models.DiscoverCollection) error
	DeleteCollection(userId string, folderId int64, targetUuid string) error
	FindCollectionInFolder(userId string, folderId int64, targetUuid string) (*models.DiscoverCollection, error)
	FindCollectionByUserAndTarget(userId, targetUuid string) (*models.DiscoverCollection, error)
	IncrementFolderPostCount(folderId int64) error
	DecrementFolderPostCount(folderId int64) error
	IncrementCollectCount(postId int64) error
	DecrementCollectCount(postId int64) error
	ListCollectedPostsByFolder(folderId int64, page, pageSize int) ([]models.DiscoverPost, error)
	CountCollectedPostsByFolder(folderId int64) (int64, error)
}

type DiscoverDAOImpl struct{}

func NewDiscoverDAO() DiscoverDAO {
	return &DiscoverDAOImpl{}
}

func (d *DiscoverDAOImpl) CreatePost(post *models.DiscoverPost) error {
	return db.GormDB.Create(post).Error
}

func (d *DiscoverDAOImpl) FindPostByUuid(uuid string) (*models.DiscoverPost, error) {
	var post models.DiscoverPost
	result := db.GormDB.Where("uuid = ?", uuid).First(&post)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &post, result.Error
}

func (d *DiscoverDAOImpl) ListPosts(page, pageSize int) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	offset := (page - 1) * pageSize
	result := db.GormDB.Where("status = ?", 0).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts)
	return posts, result.Error
}

func (d *DiscoverDAOImpl) SearchPostsByKeyword(keyword string, limit int) ([]models.DiscoverPost, error) {
	if keyword == "" {
		return d.ListPosts(1, limit)
	}
	var posts []models.DiscoverPost
	likePattern := "%" + keyword + "%"
	tagMatch := `"` + keyword + `"`
	result := db.GormDB.Where(
		"status = 0 AND (title LIKE ? OR content LIKE ? OR JSON_CONTAINS(tags, ?))",
		likePattern, likePattern, tagMatch,
	).Order("created_at DESC").Limit(limit).Find(&posts)
	return posts, result.Error
}

func (d *DiscoverDAOImpl) ListLikedPosts(userId string, page, pageSize int) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	offset := (page - 1) * pageSize
	// 子查询：获取用户点赞的帖子 UUID
	likedUuids := db.GormDB.Model(&models.DiscoverLike{}).
		Where("user_id = ?", userId).
		Select("target_uuid")
	// 查询帖子
	result := db.GormDB.Where("status = ? AND uuid IN (?)", 0, likedUuids).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&posts)
	return posts, result.Error
}

func (d *DiscoverDAOImpl) CountPosts() (int64, error) {
	var count int64
	result := db.GormDB.Model(&models.DiscoverPost{}).Where("status = ?", 0).Count(&count)
	return count, result.Error
}

func (d *DiscoverDAOImpl) CountLikedPosts(userId string) (int64, error) {
	var count int64
	// 子查询：获取用户点赞的帖子 UUID
	likedUuids := db.GormDB.Model(&models.DiscoverLike{}).
		Where("user_id = ?", userId).
		Select("target_uuid")
	result := db.GormDB.Model(&models.DiscoverPost{}).
		Where("status = ? AND uuid IN (?)", 0, likedUuids).
		Count(&count)
	return count, result.Error
}

func (d *DiscoverDAOImpl) IncrementLikeCount(postId int64) error {
	return db.GormDB.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}

func (d *DiscoverDAOImpl) DecrementLikeCount(postId int64) error {
	return db.GormDB.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
}

func (d *DiscoverDAOImpl) IncrementCommentCount(postId int64) error {
	return db.GormDB.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
}

func (d *DiscoverDAOImpl) CreateMedia(media *models.DiscoverMedia) error {
	return db.GormDB.Create(media).Error
}

func (d *DiscoverDAOImpl) FindMediaByPostId(postId int64) ([]models.DiscoverMedia, error) {
	var mediaList []models.DiscoverMedia
	result := db.GormDB.Where("post_id = ?", postId).Order("sort_order ASC").Find(&mediaList)
	return mediaList, result.Error
}

func (d *DiscoverDAOImpl) CreateLike(like *models.DiscoverLike) error {
	return db.GormDB.Create(like).Error
}

func (d *DiscoverDAOImpl) DeleteLike(userId, targetUuid string) error {
	return db.GormDB.Where("user_id = ? AND target_uuid = ?", userId, targetUuid).Delete(&models.DiscoverLike{}).Error
}

func (d *DiscoverDAOImpl) FindLike(userId, targetUuid string) (*models.DiscoverLike, error) {
	var like models.DiscoverLike
	result := db.GormDB.Where("user_id = ? AND target_uuid = ?", userId, targetUuid).Find(&like)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &like, nil
}

func (d *DiscoverDAOImpl) FindCommentById(commentId int64) (*models.DiscoverComment, error) {
	var comment models.DiscoverComment
	result := db.GormDB.Where("id = ?", commentId).First(&comment)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &comment, result.Error
}

func (d *DiscoverDAOImpl) FindCommentByUuid(uuid string) (*models.DiscoverComment, error) {
	var comment models.DiscoverComment
	result := db.GormDB.Where("uuid = ?", uuid).First(&comment)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &comment, result.Error
}

func (d *DiscoverDAOImpl) CreateComment(comment *models.DiscoverComment) error {
	return db.GormDB.Create(comment).Error
}

func (d *DiscoverDAOImpl) FindCommentsByPostId(postId int64, page, pageSize int) ([]models.DiscoverComment, error) {
	var comments []models.DiscoverComment
	offset := (page - 1) * pageSize
	result := db.GormDB.Where("post_id = ?", postId).Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&comments)
	return comments, result.Error
}

func (d *DiscoverDAOImpl) IncrementCommentLikeCount(commentId int64) error {
	return db.GormDB.Model(&models.DiscoverComment{}).Where("id = ?", commentId).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error
}

func (d *DiscoverDAOImpl) DecrementCommentLikeCount(commentId int64) error {
	return db.GormDB.Model(&models.DiscoverComment{}).Where("id = ?", commentId).UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error
}

// ========== 收藏夹 ==========

func (d *DiscoverDAOImpl) CreateFolder(folder *models.DiscoverCollectionFolder) error {
	return db.GormDB.Create(folder).Error
}

func (d *DiscoverDAOImpl) UpdateFolder(uuid string, name, description string, isPublic int8) error {
	return db.GormDB.Model(&models.DiscoverCollectionFolder{}).
		Where("uuid = ?", uuid).
		Updates(map[string]interface{}{
			"name":        name,
			"description": description,
			"is_public":   isPublic,
		}).Error
}

func (d *DiscoverDAOImpl) DeleteFolder(uuid string) error {
	return db.GormDB.Where("uuid = ?", uuid).Delete(&models.DiscoverCollectionFolder{}).Error
}

func (d *DiscoverDAOImpl) FindFolderByUuid(uuid string) (*models.DiscoverCollectionFolder, error) {
	var folder models.DiscoverCollectionFolder
	result := db.GormDB.Where("uuid = ?", uuid).First(&folder)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &folder, result.Error
}

func (d *DiscoverDAOImpl) ListFoldersByUserId(userId string) ([]models.DiscoverCollectionFolder, error) {
	var folders []models.DiscoverCollectionFolder
	result := db.GormDB.Where("user_id = ?", userId).Order("created_at DESC").Find(&folders)
	return folders, result.Error
}

func (d *DiscoverDAOImpl) FindDefaultFolder(userId string) (*models.DiscoverCollectionFolder, error) {
	var folder models.DiscoverCollectionFolder
	result := db.GormDB.Where("user_id = ? AND name = ?", userId, "默认收藏夹").First(&folder)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &folder, result.Error
}

// ========== 收藏记录 ==========

func (d *DiscoverDAOImpl) CreateCollection(col *models.DiscoverCollection) error {
	return db.GormDB.Create(col).Error
}

func (d *DiscoverDAOImpl) DeleteCollection(userId string, folderId int64, targetUuid string) error {
	return db.GormDB.Where("user_id = ? AND folder_id = ? AND target_uuid = ?", userId, folderId, targetUuid).
		Delete(&models.DiscoverCollection{}).Error
}

func (d *DiscoverDAOImpl) FindCollectionInFolder(userId string, folderId int64, targetUuid string) (*models.DiscoverCollection, error) {
	var col models.DiscoverCollection
	result := db.GormDB.Where("user_id = ? AND folder_id = ? AND target_uuid = ?", userId, folderId, targetUuid).Find(&col)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &col, result.Error
}

func (d *DiscoverDAOImpl) FindCollectionByUserAndTarget(userId, targetUuid string) (*models.DiscoverCollection, error) {
	var col models.DiscoverCollection
	result := db.GormDB.Where("user_id = ? AND target_uuid = ?", userId, targetUuid).Find(&col)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &col, result.Error
}

func (d *DiscoverDAOImpl) IncrementFolderPostCount(folderId int64) error {
	return db.GormDB.Model(&models.DiscoverCollectionFolder{}).Where("id = ?", folderId).UpdateColumn("post_count", gorm.Expr("post_count + 1")).Error
}

func (d *DiscoverDAOImpl) DecrementFolderPostCount(folderId int64) error {
	return db.GormDB.Model(&models.DiscoverCollectionFolder{}).Where("id = ?", folderId).UpdateColumn("post_count", gorm.Expr("post_count - 1")).Error
}

func (d *DiscoverDAOImpl) IncrementCollectCount(postId int64) error {
	return db.GormDB.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("collect_count", gorm.Expr("collect_count + 1")).Error
}

func (d *DiscoverDAOImpl) DecrementCollectCount(postId int64) error {
	return db.GormDB.Model(&models.DiscoverPost{}).Where("id = ?", postId).UpdateColumn("collect_count", gorm.Expr("collect_count - 1")).Error
}

func (d *DiscoverDAOImpl) ListCollectedPostsByFolder(folderId int64, page, pageSize int) ([]models.DiscoverPost, error) {
	var posts []models.DiscoverPost
	offset := (page - 1) * pageSize
	// 子查询：获取该收藏夹中的帖子 UUID
	collectedUuids := db.GormDB.Model(&models.DiscoverCollection{}).
		Where("folder_id = ?", folderId).
		Select("target_uuid")
	result := db.GormDB.Where("status = ? AND uuid IN (?)", 0, collectedUuids).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&posts)
	return posts, result.Error
}

func (d *DiscoverDAOImpl) CountCollectedPostsByFolder(folderId int64) (int64, error) {
	var count int64
	collectedUuids := db.GormDB.Model(&models.DiscoverCollection{}).
		Where("folder_id = ?", folderId).
		Select("target_uuid")
	result := db.GormDB.Model(&models.DiscoverPost{}).
		Where("status = ? AND uuid IN (?)", 0, collectedUuids).
		Count(&count)
	return count, result.Error
}
