package userservice

import (
	"fmt"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
)

// GetUserInfo 获取用户信息
func GetUserInfo(req *userreq.GetUserInfoRequest) (*userresp.GetUserInfoRespond, error) {
	user, err := userdao.FindUserByUuid(req.Uuid)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败：%v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	userInfoRsp := &userresp.GetUserInfoRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Birthday:  user.Birthday,
		Email:     user.Email,
		Gender:    user.Gender,
		Signature: user.Signature,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}

	return userInfoRsp, nil
}
