package userdao

import (
	"errors"
	"fmt"
	"seekF-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserInfoDAO interface {
	// 在业务事务中锁定用户，找不到时返回空。
	FindUserByUuidForUpdate(uuid string) (*models.UserInfo, error)
	// 原子维护关注及粉丝计数，拒绝减为负数。
	UpdateFollowCounts(uuid string, followingDelta, followerDelta int64) error
	CreateUser(user *models.UserInfo) error
	FindUserByUuid(uuid string) (*models.UserInfo, error)
	// FindUsersByUuids 按 uuid 列表批量查询用户（用于避免 N+1）
	FindUsersByUuids(uuids []string) ([]models.UserInfo, error)
	FindUserByTelephone(telephone string) (*models.UserInfo, error)
	FindUserByEmail(email string) (*models.UserInfo, error)
	FindUserByNickname(nickname string) (*models.UserInfo, error)
	FindUserByGithubId(githubId int64) (*models.UserInfo, error)
	FindUserByGiteeId(giteeId int64) (*models.UserInfo, error)
	UpdateUserInfo(user *models.UserInfo) error
	SearchUsers(keyword string) ([]models.UserInfo, error)
}

type UserInfoDAOImpl struct {
	db *gorm.DB
}

func NewUserInfoDAO(db *gorm.DB) UserInfoDAO {
	return &UserInfoDAOImpl{db: db}
}

// CreateUser 创建用户
func (d *UserInfoDAOImpl) CreateUser(user *models.UserInfo) error {
	result := d.db.Create(user)
	return result.Error
}

// FindUserByUuid 根据UUID查找用户
func (d *UserInfoDAOImpl) FindUserByUuid(uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.Where("uuid = ?", uuid).Find(&user)
	return &user, result.Error
}

// FindUsersByUuids 根据 UUID 列表批量查找用户
func (d *UserInfoDAOImpl) FindUsersByUuids(uuids []string) ([]models.UserInfo, error) {
	if len(uuids) == 0 {
		return nil, nil
	}
	var users []models.UserInfo
	result := d.db.Where("uuid IN ?", uuids).Find(&users)
	return users, result.Error
}

// FindUserByTelephone 根据手机号查找用户
func (d *UserInfoDAOImpl) FindUserByTelephone(telephone string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.Where("telephone = ?", telephone).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByEmail 根据邮箱查找用户
func (d *UserInfoDAOImpl) FindUserByEmail(email string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.Where("email = ?", email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByNickname 根据昵称查找用户
func (d *UserInfoDAOImpl) FindUserByNickname(nickname string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.Where("nickname = ?", nickname).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByGithubId 根据 GitHub ID 查找用户
func (d *UserInfoDAOImpl) FindUserByGithubId(githubId int64) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.Where("github_id = ?", githubId).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByGiteeId 根据 Gitee ID 查找用户
func (d *UserInfoDAOImpl) FindUserByGiteeId(giteeId int64) (*models.UserInfo, error) {
	var user models.UserInfo
	result := d.db.Where("gitee_id = ?", giteeId).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// UpdateUserInfo 更新用户信息
func (d *UserInfoDAOImpl) UpdateUserInfo(user *models.UserInfo) error {
	// 计数只由关注事务维护，资料更新不得用旧快照覆盖并发计数。
	result := d.db.Omit("following_count", "follower_count").Save(user)
	return result.Error
}

// FindUserByUuidForUpdate 在业务事务中锁定用户并读取最新计数。
func (d *UserInfoDAOImpl) FindUserByUuidForUpdate(uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("uuid = ?", uuid).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// UpdateFollowCounts 原子增减两个计数，计数不足或用户不存在时由Service回滚业务事务。
func (d *UserInfoDAOImpl) UpdateFollowCounts(uuid string, followingDelta, followerDelta int64) error {
	result := d.db.Model(&models.UserInfo{}).Where("uuid = ? AND following_count + ? >= 0 AND follower_count + ? >= 0", uuid, followingDelta, followerDelta).Updates(map[string]any{
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

// SearchUsers 根据关键词搜索用户
func (d *UserInfoDAOImpl) SearchUsers(keyword string) ([]models.UserInfo, error) {
	var userList []models.UserInfo
	result := d.db.Where("nickname LIKE ? OR telephone LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&userList)
	return userList, result.Error
}
