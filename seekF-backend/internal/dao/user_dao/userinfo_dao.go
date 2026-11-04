package userdao

import (
	"context"
	"errors"
	"fmt"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserInfoDAO 提供用户资料、账号查询与关注计数的数据访问能力。
type UserInfoDAO interface {
	// 在业务事务中锁定用户，找不到时返回空。
	FindUserByUuidForUpdate(ctx context.Context, uuid string) (*models.UserInfo, error)
	// 原子维护关注及粉丝计数，拒绝减为负数。
	UpdateFollowCounts(ctx context.Context, uuid string, followingDelta, followerDelta int64) error
	// 创建用户
	CreateUser(ctx context.Context, user *models.UserInfo) error
	// 根据UUID查询用户，不存在时返回空。
	FindUserByUuid(ctx context.Context, uuid string) (*models.UserInfo, error)
	// 按 uuid 列表批量查询用户（用于避免 N+1）
	FindUsersByUuids(ctx context.Context, uuids []string) ([]models.UserInfo, error)
	// 根据手机号查找用户
	FindUserByTelephone(ctx context.Context, telephone string) (*models.UserInfo, error)
	// 根据邮箱查找用户
	FindUserByEmail(ctx context.Context, email string) (*models.UserInfo, error)
	// 根据昵称查找用户
	FindUserByNickname(ctx context.Context, nickname string) (*models.UserInfo, error)
	// 根据 GitHub ID 查找用户
	FindUserByGithubId(ctx context.Context, githubId int64) (*models.UserInfo, error)
	// 根据 Gitee ID 查找用户
	FindUserByGiteeId(ctx context.Context, giteeId int64) (*models.UserInfo, error)
	// 更新用户信息
	UpdateUserInfo(ctx context.Context, user *models.UserInfo) error
}

// UserInfoDAOImpl 使用指定数据库或事务连接访问用户资料。
type UserInfoDAOImpl struct {
	db *gorm.DB
}

// NewUserInfoDAO 构造绑定指定连接的用户资料数据访问器。
func NewUserInfoDAO(database *gorm.DB) UserInfoDAO {
	return &UserInfoDAOImpl{db: database}
}

// CreateUser 创建用户
func (d *UserInfoDAOImpl) CreateUser(ctx context.Context, user *models.UserInfo) error {
	result := d.db.WithContext(ctx).Create(user)
	return result.Error
}

// FindUserByUuid 根据UUID查询用户，不存在时返回空。
func (d *UserInfoDAOImpl) FindUserByUuid(ctx context.Context, uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.WithContext(ctx).Where("uuid = ?", uuid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUsersByUuids 根据 UUID 列表批量查找用户
func (d *UserInfoDAOImpl) FindUsersByUuids(ctx context.Context, uuids []string) ([]models.UserInfo, error) {
	if len(uuids) == 0 {
		return nil, nil
	}
	var users []models.UserInfo
	result := d.db.WithContext(ctx).Where("uuid IN ?", uuids).Find(&users)
	return users, result.Error
}

// FindUserByTelephone 根据手机号查找用户
func (d *UserInfoDAOImpl) FindUserByTelephone(ctx context.Context, telephone string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.WithContext(ctx).Where("telephone = ?", telephone).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByEmail 根据邮箱查找用户
func (d *UserInfoDAOImpl) FindUserByEmail(ctx context.Context, email string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByNickname 根据昵称查找用户
func (d *UserInfoDAOImpl) FindUserByNickname(ctx context.Context, nickname string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.WithContext(ctx).Where("nickname = ?", nickname).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByGithubId 根据 GitHub ID 查找用户
func (d *UserInfoDAOImpl) FindUserByGithubId(ctx context.Context, githubId int64) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.WithContext(ctx).Where("github_id = ?", githubId).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByGiteeId 根据 Gitee ID 查找用户
func (d *UserInfoDAOImpl) FindUserByGiteeId(ctx context.Context, giteeId int64) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.WithContext(ctx).Where("gitee_id = ?", giteeId).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// UpdateUserInfo 更新用户信息
func (d *UserInfoDAOImpl) UpdateUserInfo(ctx context.Context, user *models.UserInfo) error {
	// 计数只由关注事务维护，资料更新不得用旧快照覆盖并发计数。
	result := d.db.WithContext(ctx).Omit("following_count", "follower_count").Save(user)
	return result.Error
}

// FindUserByUuidForUpdate 在业务事务中锁定用户并读取最新计数。
func (d *UserInfoDAOImpl) FindUserByUuidForUpdate(ctx context.Context, uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("uuid = ?", uuid).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// UpdateFollowCounts 原子增减两个计数，计数不足或用户不存在时由Service回滚业务事务。
func (d *UserInfoDAOImpl) UpdateFollowCounts(ctx context.Context, uuid string, followingDelta, followerDelta int64) error {
	result := d.db.WithContext(ctx).Model(&models.UserInfo{}).Where("uuid = ? AND following_count + ? >= 0 AND follower_count + ? >= 0", uuid, followingDelta, followerDelta).Updates(map[string]any{
		"following_count": gorm.Expr("following_count + ?", followingDelta), "follower_count": gorm.Expr("follower_count + ?", followerDelta),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("用户不存在或关注计数异常，userId=%s", uuid)
	}
	return nil
}
