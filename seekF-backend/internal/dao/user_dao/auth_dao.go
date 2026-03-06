package userdao

import (
	"errors"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

// CreateUser 创建用户
func CreateUser(user *models.UserInfo) error {
	result := db.GormDB.Create(user)
	return result.Error
}

// FindUserByTelephone 根据手机号查找用户
func FindUserByTelephone(telephone string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := db.GormDB.Where("telephone = ?", telephone).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}
