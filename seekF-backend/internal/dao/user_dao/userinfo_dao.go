package userdao

import (
	"errors"
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

type UserInfoDAO interface {
	CreateUser(user *models.UserInfo) error
	FindUserByUuid(uuid string) (*models.UserInfo, error)
	// FindUsersByUuids 按 uuid 列表批量查询用户（用于避免 N+1）
	FindUsersByUuids(uuids []string) ([]models.UserInfo, error)
	FindUserByTelephone(telephone string) (*models.UserInfo, error)
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

// UpdateUserInfo 更新用户信息
func (d *UserInfoDAOImpl) UpdateUserInfo(user *models.UserInfo) error {
	result := d.db.Save(user)
	return result.Error
}

// SearchUsers 根据关键词搜索用户
func (d *UserInfoDAOImpl) SearchUsers(keyword string) ([]models.UserInfo, error) {
	var userList []models.UserInfo
	result := d.db.Where("nickname LIKE ? OR telephone LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&userList)
	return userList, result.Error
}
