package userdao

import (
	"errors"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

type UserInfoDAO interface {
	CreateUser(user *models.UserInfo) error
	FindUserByUuid(uuid string) (*models.UserInfo, error)
	FindUserByTelephone(telephone string) (*models.UserInfo, error)
	UpdateUserInfo(user *models.UserInfo) error
	SearchUsers(keyword string) ([]models.UserInfo, error)
}

type UserInfoDAOImpl struct{}

func NewUserInfoDAO() UserInfoDAO {
	return &UserInfoDAOImpl{}
}

// CreateUser 创建用户
func (d *UserInfoDAOImpl) CreateUser(user *models.UserInfo) error {
	result := db.GormDB.Create(user)
	return result.Error
}

// FindUserByUuid 根据UUID查找用户
func (d *UserInfoDAOImpl) FindUserByUuid(uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := db.GormDB.Where("uuid = ?", uuid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// FindUserByTelephone 根据手机号查找用户
func (d *UserInfoDAOImpl) FindUserByTelephone(telephone string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := db.GormDB.Where("telephone = ?", telephone).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}

// UpdateUserInfo 更新用户信息
func (d *UserInfoDAOImpl) UpdateUserInfo(user *models.UserInfo) error {
	result := db.GormDB.Save(user)
	return result.Error
}

// SearchUsers 根据关键词搜索用户
func (d *UserInfoDAOImpl) SearchUsers(keyword string) ([]models.UserInfo, error) {
	var userList []models.UserInfo
	result := db.GormDB.Where("nickname LIKE ? OR telephone LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&userList)
	return userList, result.Error
}
