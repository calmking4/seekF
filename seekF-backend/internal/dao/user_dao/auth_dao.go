package userdao

import (
	"errors"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

type AuthDAO interface {
	CreateUser(user *models.UserInfo) error
	FindUserByTelephone(telephone string) (*models.UserInfo, error)
}

type AuthDAOImpl struct{}

func NewAuthDAO() AuthDAO {
	return &AuthDAOImpl{}
}

// CreateUser 创建用户
func (d *AuthDAOImpl) CreateUser(user *models.UserInfo) error {
	result := db.GormDB.Create(user)
	return result.Error
}

// FindUserByTelephone 根据手机号查找用户
func (d *AuthDAOImpl) FindUserByTelephone(telephone string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := db.GormDB.Where("telephone = ?", telephone).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}
