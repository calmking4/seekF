package userdao

import (
	"errors"
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

type FollowDAO interface {
	// CreateFollow 创建关注关系
	CreateFollow(follow *models.UserFollow) error
	// DeleteFollow 删除关注关系
	DeleteFollow(userId, followUserId string) error
	// FindFollow 查询关注关系
	FindFollow(userId, followUserId string) (*models.UserFollow, error)
	// CountFollowing 我关注了多少人
	CountFollowing(userId string) (int64, error)
	// CountFollowers 多少人关注了我
	CountFollowers(userId string) (int64, error)
	// ListFollowing 关注列表
	ListFollowing(userId string, page, pageSize int) ([]models.UserFollow, error)
	// ListFollowers 粉丝列表
	ListFollowers(userId string, page, pageSize int) ([]models.UserFollow, error)
}

type FollowDAOImpl struct {
	db *gorm.DB
}

func NewFollowDAO(db *gorm.DB) FollowDAO {
	return &FollowDAOImpl{db: db}
}

func (d *FollowDAOImpl) CreateFollow(follow *models.UserFollow) error {
	return d.db.Create(follow).Error
}

func (d *FollowDAOImpl) DeleteFollow(userId, followUserId string) error {
	return d.db.Where("user_id = ? AND follow_user_id = ?", userId, followUserId).Delete(&models.UserFollow{}).Error
}

func (d *FollowDAOImpl) FindFollow(userId, followUserId string) (*models.UserFollow, error) {
	var follow models.UserFollow
	result := d.db.Where("user_id = ? AND follow_user_id = ?", userId, followUserId).First(&follow)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &follow, result.Error
}

func (d *FollowDAOImpl) CountFollowing(userId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.UserFollow{}).Where("user_id = ?", userId).Count(&count)
	return count, result.Error
}

func (d *FollowDAOImpl) CountFollowers(userId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.UserFollow{}).Where("follow_user_id = ?", userId).Count(&count)
	return count, result.Error
}

func (d *FollowDAOImpl) ListFollowing(userId string, page, pageSize int) ([]models.UserFollow, error) {
	var follows []models.UserFollow
	offset := (page - 1) * pageSize
	result := d.db.Where("user_id = ?", userId).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows)
	return follows, result.Error
}

func (d *FollowDAOImpl) ListFollowers(userId string, page, pageSize int) ([]models.UserFollow, error) {
	var follows []models.UserFollow
	offset := (page - 1) * pageSize
	result := d.db.Where("follow_user_id = ?", userId).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows)
	return follows, result.Error
}
