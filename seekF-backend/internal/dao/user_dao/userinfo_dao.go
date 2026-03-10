package userdao

import (
	"errors"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

// FindUserByUuid 根据UUID查找用户
func FindUserByUuid(uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	result := db.GormDB.Where("uuid = ?", uuid).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}
