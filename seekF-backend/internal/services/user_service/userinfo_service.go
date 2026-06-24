package userservice

import (
	"database/sql"
	"fmt"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
)

type UserInfoService interface {
	GetUserInfo(req *userreq.GetUserInfoRequest) (*userresp.GetUserInfoRespond, error)
	UpdateUserInfo(req *userreq.UpdateUserInfoRequest) error
}

type UserInfoServiceImpl struct {
	userInfoDAO userdao.UserInfoDAO
}

func NewUserInfoService(userInfoDAO userdao.UserInfoDAO) UserInfoService {
	return &UserInfoServiceImpl{
		userInfoDAO: userInfoDAO,
	}
}

// GetUserInfo 获取用户信息
func (s *UserInfoServiceImpl) GetUserInfo(req *userreq.GetUserInfoRequest) (*userresp.GetUserInfoRespond, error) {
	user, err := s.userInfoDAO.FindUserByUuid(req.Uuid)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败：%v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	birthday := ""
	if user.Birthday.Valid {
		birthday = user.Birthday.String
	}

	userInfoRsp := &userresp.GetUserInfoRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Birthday:  birthday,
		Email:     user.Email,
		Gender:    user.Gender,
		Signature: user.Signature,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}

	return userInfoRsp, nil
}

// UpdateUserInfo 更新用户信息
func (s *UserInfoServiceImpl) UpdateUserInfo(req *userreq.UpdateUserInfoRequest) error {
	// 根据UUID查找用户
	user, err := s.userInfoDAO.FindUserByUuid(req.Uuid)
	if err != nil {
		return fmt.Errorf("查找用户失败：%v", err)
	}

	if user == nil {
		return fmt.Errorf("用户不存在")
	}

	// 更新用户信息
	if req.Email != "" {
		// 检查邮箱是否已被其他用户使用
		existingUser, err := s.userInfoDAO.FindUserByEmail(req.Email)
		if err != nil {
			return fmt.Errorf("检查邮箱失败：%v", err)
		}
		if existingUser != nil && existingUser.Uuid != user.Uuid {
			return fmt.Errorf("该邮箱已被其他用户使用")
		}
		user.Email = req.Email
	}
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Birthday != "" {
		user.Birthday = sql.NullString{String: req.Birthday, Valid: true}
	}
	if req.Signature != "" {
		user.Signature = req.Signature
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	// 保存更新
	err = s.userInfoDAO.UpdateUserInfo(user)
	if err != nil {
		return fmt.Errorf("更新用户信息失败：%v", err)
	}

	return nil
}
