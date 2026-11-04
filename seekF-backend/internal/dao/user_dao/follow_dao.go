package userdao

import (
	"context"
	"errors"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FollowDAO 提供关注关系和关注计数的数据访问能力。
type FollowDAO interface {
	// 在事务内读取并锁定当前关注关系。
	FindFollowForUpdate(ctx context.Context, userId, followUserId string) (*models.UserFollow, error)
	// 创建唯一关注关系。
	CreateFollow(ctx context.Context, follow *models.UserFollow) error
	// 删除关注关系。
	DeleteFollow(ctx context.Context, userId, followUserId string) error
	// 查询关注关系。
	FindFollow(ctx context.Context, userId, followUserId string) (*models.UserFollow, error)
	// 读取用户资料保存的关注数。
	CountFollowing(ctx context.Context, userId string) (int64, error)
	// 读取用户资料保存的粉丝数。
	CountFollowers(ctx context.Context, userId string) (int64, error)
	// 分页查询关注列表。
	ListFollowing(ctx context.Context, userId string, page, pageSize int) ([]models.UserFollow, error)
	// 分页查询粉丝列表。
	ListFollowers(ctx context.Context, userId string, page, pageSize int) ([]models.UserFollow, error)
}

// FollowDAOImpl 使用指定数据库或事务连接访问关注关系。
type FollowDAOImpl struct {
	db *gorm.DB
}

// NewFollowDAO 构造绑定数据库或业务事务的关注DAO。
func NewFollowDAO(database *gorm.DB) FollowDAO {
	return &FollowDAOImpl{db: database}
}

// CreateFollow 保存唯一关注关系。
func (d *FollowDAOImpl) CreateFollow(ctx context.Context, follow *models.UserFollow) error {
	return d.db.WithContext(ctx).Create(follow).Error
}

// DeleteFollow 删除指定关注关系，事务由Service管理。
func (d *FollowDAOImpl) DeleteFollow(ctx context.Context, userId, followUserId string) error {
	return d.db.WithContext(ctx).Where("user_id = ? AND follow_user_id = ?", userId, followUserId).Delete(&models.UserFollow{}).Error
}

// FindFollow 查询当前关注关系，不存在时返回空。
func (d *FollowDAOImpl) FindFollow(ctx context.Context, userId, followUserId string) (*models.UserFollow, error) {
	var follow models.UserFollow
	result := d.db.WithContext(ctx).Where("user_id = ? AND follow_user_id = ?", userId, followUserId).First(&follow)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &follow, result.Error
}

// CountFollowing 读取用户资料中保存的关注数。
func (d *FollowDAOImpl) CountFollowing(ctx context.Context, userId string) (int64, error) {
	var count int64
	result := d.db.WithContext(ctx).Model(&models.UserInfo{}).Select("following_count").Where("uuid = ?", userId).Scan(&count)
	return count, result.Error
}

// CountFollowers 读取用户资料中保存的粉丝数。
func (d *FollowDAOImpl) CountFollowers(ctx context.Context, userId string) (int64, error) {
	var count int64
	result := d.db.WithContext(ctx).Model(&models.UserInfo{}).Select("follower_count").Where("uuid = ?", userId).Scan(&count)
	return count, result.Error
}

// ListFollowing 分页查询用户关注的作者。
func (d *FollowDAOImpl) ListFollowing(ctx context.Context, userId string, page, pageSize int) ([]models.UserFollow, error) {
	var follows []models.UserFollow
	offset := (page - 1) * pageSize
	result := d.db.WithContext(ctx).Where("user_id = ?", userId).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows)
	return follows, result.Error
}

// ListFollowers 分页查询用户的粉丝。
func (d *FollowDAOImpl) ListFollowers(ctx context.Context, userId string, page, pageSize int) ([]models.UserFollow, error) {
	var follows []models.UserFollow
	offset := (page - 1) * pageSize
	result := d.db.WithContext(ctx).Where("follow_user_id = ?", userId).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows)
	return follows, result.Error
}

// FindFollowForUpdate 在锁定双方用户后读取最新关注关系，保证并发切换串行执行。
func (d *FollowDAOImpl) FindFollowForUpdate(ctx context.Context, userId, followUserId string) (*models.UserFollow, error) {
	var row models.UserFollow
	err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND follow_user_id = ?", userId, followUserId).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}
