package userservice

import (
	"encoding/json"
	"errors"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/constants"
	contacttypeenum "seekF-backend/internal/pkg/enum/contact_enum/contact_type_enum"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/zlog"

	"github.com/redis/go-redis/v9"
)

type ContactService interface {
	GetUserList(ownerId string) ([]userresp.MyUserListRespond, error)
}

type ContactServiceImpl struct {
	contactDAO  userdao.ContactDAO
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
}

func NewContactService(
	contactDAO userdao.ContactDAO,
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
) ContactService {
	return &ContactServiceImpl{
		contactDAO:  contactDAO,
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
	}
}

// GetUserList 获取联系人列表
func (s *ContactServiceImpl) GetUserList(ownerId string) ([]userresp.MyUserListRespond, error) {
	rspString, err := myredis.GetKeyNilIsErr("contact_user_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// dao 层获取联系人列表
			contactList, err := s.contactDAO.GetUserContactList(ownerId)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}

			// dto 转换
			var userListRsp []userresp.MyUserListRespond
			for _, contact := range contactList {
				// 联系人中是用户的
				if contact.ContactType == contacttypeenum.USER {
					// 获取用户信息
					user, err := s.userInfoDAO.FindUserByUuid(contact.ContactId)
					if err != nil {
						zlog.Error(err.Error())
						return nil, err
					}
					if user == nil {
						continue
					}
					userListRsp = append(userListRsp, userresp.MyUserListRespond{
						UserId:   user.Uuid,
						UserName: user.Nickname,
						Avatar:   user.Avatar,
					})
				}
			}
			rspString, err := json.Marshal(userListRsp)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}
			if err := myredis.SetKeyEx("contact_user_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
			}
			return userListRsp, nil
		} else {
			zlog.Error(err.Error())
			return nil, err
		}
	}
	var rsp []userresp.MyUserListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
		return nil, err
	}
	return rsp, nil
}
