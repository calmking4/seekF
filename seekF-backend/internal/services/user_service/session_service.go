package userservice

import (
	"errors"
	"fmt"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	groupstatusenum "seekF-backend/internal/pkg/enum/group_enum/group_status_enum"
	userstatusenum "seekF-backend/internal/pkg/enum/user_enum/user_status_enum"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"gorm.io/gorm"
)

type SessionService interface {
	OpenSession(sendId string, receiveId string) (string, error)
	CreateSession(sendId string, receiveId string) (string, error)
}

type SessionServiceImpl struct {
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
	groupDAO    userdao.GroupDAO
}

func NewSessionService(
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
	groupDAO userdao.GroupDAO,
) SessionService {
	return &SessionServiceImpl{
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
		groupDAO:    groupDAO,
	}
}

// OpenSession 打开会话
func (s *SessionServiceImpl) OpenSession(sendId string, receiveId string) (string, error) {
	// 直接查询数据库
	session, err := s.sessionDAO.GetSessionBySendAndReceiveId(sendId, receiveId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 会话不存在，创建新会话
			return s.CreateSession(sendId, receiveId)
		}
		zlog.Error(err.Error())
		return "", fmt.Errorf("系统错误")
	}
	return session.Uuid, nil
}

// CreateSession 创建会话
func (s *SessionServiceImpl) CreateSession(sendId string, receiveId string) (string, error) {
	// 检查发送者是否存在
	user, err := s.userInfoDAO.FindUserByUuid(sendId)
	if err != nil {
		zlog.Error(err.Error())
		return "", fmt.Errorf("系统错误")
	}
	if user == nil {
		return "", fmt.Errorf("用户不存在")
	}

	// 创建会话
	session := models.Session{
		Uuid:      fmt.Sprintf("S%s", util.GetNowAndLenRandomString(11)),
		SendId:    sendId,
		ReceiveId: receiveId,
		CreatedAt: time.Now(),
	}

	// 根据接收者类型设置会话信息
	if receiveId[0] == 'U' {
		// 接收者是用户
		receiveUser, err := s.userInfoDAO.FindUserByUuid(receiveId)
		if err != nil {
			zlog.Error(err.Error())
			return "", fmt.Errorf("系统错误")
		}
		if receiveUser == nil {
			return "", fmt.Errorf("用户不存在")
		}
		if receiveUser.Status == userstatusenum.DISABLE {
			zlog.Error("该用户被禁用了")
			return "", fmt.Errorf("该用户被禁用了")
		}
		session.ReceiveName = receiveUser.Nickname
		session.Avatar = receiveUser.Avatar
	} else {
		// 接收者是群聊
		receiveGroup, err := s.groupDAO.GetGroupInfoByUuid(receiveId)
		if err != nil {
			zlog.Error(err.Error())
			return "", fmt.Errorf("系统错误")
		}
		if receiveGroup.Status == groupstatusenum.DISABLE {
			zlog.Error("该群聊被禁用了")
			return "", fmt.Errorf("该群聊被禁用了")
		}
		session.ReceiveName = receiveGroup.Name
		session.Avatar = receiveGroup.Avatar
	}

	// 直接保存会话到数据库
	if err := s.sessionDAO.CreateSession(&session); err != nil {
		zlog.Error(err.Error())
		return "", fmt.Errorf("系统错误")
	}

	return session.Uuid, nil
}
