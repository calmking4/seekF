package userservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	contactstatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_status_enum"
	groupstatusenum "seekF-backend/internal/pkg/enum/group_enum/group_status_enum"
	userstatusenum "seekF-backend/internal/pkg/enum/user_enum/user_status_enum"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SessionService interface {
	OpenSession(sendId string, receiveId string) (string, error)
	CreateSession(sendId string, receiveId string) (string, error)
	GetSessionList(userId string) ([]userresp.GetSessionListRespond, error)
	DeleteSession(userId string, sessionId string) error
	CheckOpenSessionAllowed(sendId string, receiveId string) (bool, error)
}

type SessionServiceImpl struct {
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
	groupDAO    userdao.GroupDAO
	contactDAO  userdao.ContactDAO
}

func NewSessionService(
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
	groupDAO userdao.GroupDAO,
	contactDAO userdao.ContactDAO,
) SessionService {
	return &SessionServiceImpl{
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
		groupDAO:    groupDAO,
		contactDAO:  contactDAO,
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
	}

	// 直接保存会话到数据库
	if err := s.sessionDAO.CreateSession(&session); err != nil {
		zlog.Error(err.Error())
		return "", fmt.Errorf("系统错误")
	}

	// 清除用户会话列表缓存
	if err := myredis.DelKeysWithPattern("session_list_" + sendId); err != nil {
		zlog.Error(err.Error())
	}

	return session.Uuid, nil
}

// GetSessionList 获取会话列表（用户和群聊）
func (s *SessionServiceImpl) GetSessionList(userId string) ([]userresp.GetSessionListRespond, error) {
	// 从 Redis 获取会话列表
	rspString, err := myredis.GetKeyNilIsErr("session_list_" + userId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Redis 中不存在，从数据库获取
			sessionList, err := s.sessionDAO.GetSessionListBySendId(userId)
			if err != nil {
				zlog.Error(err.Error())
				return nil, fmt.Errorf("系统错误")
			}

			// 构建响应
			var sessionListRsp []userresp.GetSessionListRespond
			for _, session := range sessionList {
				// 根据 ReceiveId 获取最新的头像和名称信息
				var avatar, name string

				if session.ReceiveId[0] == 'U' {
					// ReceiveId 是用户ID，获取用户最新信息
					user, err := s.userInfoDAO.FindUserByUuid(session.ReceiveId)
					if err != nil {
						zlog.Error("获取用户信息失败: " + err.Error())
						// 用户不存在，跳过这个会话
						continue
					} else if user == nil {
						// 用户不存在，跳过这个会话
						continue
					} else {
						avatar = user.Avatar
						name = user.Nickname
					}
				} else {
					// ReceiveId 是群聊ID，获取群聊最新信息
					group, err := s.groupDAO.GetGroupInfoByUuid(session.ReceiveId)
					if err != nil {
						zlog.Error("获取群聊信息失败: " + err.Error())
						// 群聊不存在，跳过这个会话
						continue
					} else {
						avatar = group.Avatar
						name = group.Name
					}
				}

				// 构建会话响应
				sessionRsp := userresp.GetSessionListRespond{
					SessionId: session.Uuid,
					Avatar:    avatar,
					Id:        session.ReceiveId,
					Name:      name,
				}
				sessionListRsp = append(sessionListRsp, sessionRsp)
			}

			// 缓存到 Redis
			rspString, err := json.Marshal(sessionListRsp)
			if err != nil {
				zlog.Error(err.Error())
			}
			if err := myredis.SetKeyEx("session_list_"+userId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
			}

			return sessionListRsp, nil
		} else {
			zlog.Error(err.Error())
			return nil, fmt.Errorf("系统错误")
		}
	}

	// 从 Redis 缓存中获取
	var sessionListRsp []userresp.GetSessionListRespond
	if err := json.Unmarshal([]byte(rspString), &sessionListRsp); err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	return sessionListRsp, nil
}

// DeleteSession 删除会话
func (s *SessionServiceImpl) DeleteSession(userId string, sessionId string) error {
	// 从数据库中删除会话
	if err := s.sessionDAO.DeleteSession(sessionId); err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	// 清除用户会话列表缓存
	if err := myredis.DelKeysWithPattern("session_list_" + userId); err != nil {
		zlog.Error(err.Error())
	}

	return nil
}

// CheckOpenSessionAllowed 检查是否可以打开会话
func (s *SessionServiceImpl) CheckOpenSessionAllowed(sendId string, receiveId string) (bool, error) {
	// 检查发送者和接收者之间的联系人关系
	contact, err := s.contactDAO.GetUserContactByUserIdAndContactId(sendId, receiveId)
	if err != nil {
		zlog.Error(err.Error())
		return false, fmt.Errorf("系统错误")
	}

	// 检查联系人状态
	if contact.Status == contactstatusenum.BE_BLACK {
		return false, fmt.Errorf("已被对方拉黑，无法发起会话")
	} else if contact.Status == contactstatusenum.BLACK {
		return false, fmt.Errorf("已拉黑对方，先解除拉黑状态才能发起会话")
	}

	// 检查接收者状态
	if receiveId[0] == 'U' {
		// 接收者是用户
		user, err := s.userInfoDAO.FindUserByUuid(receiveId)
		if err != nil {
			zlog.Error(err.Error())
			return false, fmt.Errorf("系统错误")
		}
		if user.Status == userstatusenum.DISABLE {
			return false, fmt.Errorf("对方已被禁用，无法发起会话")
		}
	} else {
		// 接收者是群聊
		group, err := s.groupDAO.GetGroupInfoByUuid(receiveId)
		if err != nil {
			zlog.Error(err.Error())
			return false, fmt.Errorf("系统错误")
		}
		if group.Status == groupstatusenum.DISABLE {
			return false, fmt.Errorf("对方已被禁用，无法发起会话")
		}
	}

	return true, nil
}
