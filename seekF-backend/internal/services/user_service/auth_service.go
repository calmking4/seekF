package userservice

import (
	"fmt"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/util"

	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Nickname  string `json:"nickname" binding:"required"`
	Telephone string `json:"telephone" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// Register 用户注册
func Register(req *RegisterRequest) error {
	// 检查手机号是否已存在
	existingUser, err := userdao.FindUserByTelephone(req.Telephone)
	if err != nil {
		return err
	}
	if existingUser != nil {
		return fmt.Errorf("该手机号已被注册")
	}

	// 创建新用户
	password, err := encryptPassword(req.Password)
	if err != nil {
		return err
	}

	// 生成UUID
	userUUID := "U" + util.GetNowAndLenRandomString(11)

	user := &models.UserInfo{
		Uuid:      userUUID,
		Nickname:  req.Nickname,
		Telephone: req.Telephone,
		Password:  password,
	}

	err = userdao.CreateUser(user)
	if err != nil {
		return err
	}

	return nil
}

// 加密密码
func encryptPassword(password string) (string, error) {
	// 使用bcrypt加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
