package userservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	userstatusenum "seekF-backend/internal/pkg/enum/user_enum/user_status_enum"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// OpenSessionResult 打开会话结果
type OpenSessionResult struct {
	SessionId string `json:"session_id"`
}

type SessionService interface {
	OpenSession(sendId string, receiveId string) (string, error)
	OpenSessionWithInfo(sendId string, receiveId string) (*OpenSessionResult, error)
	CreateSession(sendId string, receiveId string) (string, error)
	GetSessionList(userId string) ([]userresp.GetSessionListRespond, error)
	DeleteSession(userId string, sessionId string) error
	CheckOpenSessionAllowed(sendId string, receiveId string) (bool, error)
}

type SessionServiceImpl struct {
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
}

// NewSessionService 构造无需好友关系的用户私聊服务。
func NewSessionService(
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
) SessionService {
	return &SessionServiceImpl{
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
	}
}

// OpenSession 打开会话
func (s *SessionServiceImpl) OpenSession(sendId string, receiveId string) (string, error) {
	if _, err := s.CheckOpenSessionAllowed(sendId, receiveId); err != nil {
		return "", err
	}
	// 先查询是否存在正常会话
	session, err := s.sessionDAO.GetSessionBySendAndReceiveId(sendId, receiveId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 会话不存在，查找是否有已删除的会话
			deletedSession, err := s.sessionDAO.GetDeletedSessionBySendAndReceiveId(sendId, receiveId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// 没有已删除的会话，创建新会话
					return s.CreateSession(sendId, receiveId)
				}
				zlog.Error(err.Error())
				return "", fmt.Errorf("系统错误")
			}
			// 恢复已删除的会话
			if err := s.sessionDAO.RestoreSession(deletedSession.Uuid); err != nil {
				zlog.Error(err.Error())
				return "", fmt.Errorf("系统错误")
			}
			// 清除缓存
			myredis.DelKeysWithPattern(constants.SessionListKeyPrefix + sendId)
			return deletedSession.Uuid, nil
		}
		zlog.Error(err.Error())
		return "", fmt.Errorf("系统错误")
	}
	return session.Uuid, nil
}

// OpenSessionWithInfo 打开用户私聊并返回会话编号。
func (s *SessionServiceImpl) OpenSessionWithInfo(sendId string, receiveId string) (*OpenSessionResult, error) {
	sessionId, err := s.OpenSession(sendId, receiveId)
	if err != nil {
		return nil, err
	}

	return &OpenSessionResult{
		SessionId: sessionId,
	}, nil
}

// CreateSession 创建会话
func (s *SessionServiceImpl) CreateSession(sendId string, receiveId string) (string, error) {
	if _, err := s.CheckOpenSessionAllowed(sendId, receiveId); err != nil {
		return "", err
	}

	// 创建会话
	session := models.Session{
		Uuid:      fmt.Sprintf("S%s", util.GetNowAndLenRandomString(11)),
		SendId:    sendId,
		ReceiveId: receiveId,
		CreatedAt: time.Now(),
	}

	// 直接保存会话到数据库
	if err := s.sessionDAO.CreateSession(&session); err != nil {
		zlog.Error(err.Error())
		return "", fmt.Errorf("系统错误")
	}

	// 清除用户会话列表缓存
	if err := myredis.DelKeysWithPattern(constants.SessionListKeyPrefix + sendId); err != nil {
		zlog.Error(err.Error())
	}

	return session.Uuid, nil
}

// GetSessionList 获取用户私聊列表，过滤旧缓存中的群聊与 AI 会话。
func (s *SessionServiceImpl) GetSessionList(userId string) ([]userresp.GetSessionListRespond, error) {
	key := constants.SessionListKeyPrefix + userId
	cached, err := myredis.GetKeyNilIsErr(key)
	if err == nil {
		var list []userresp.GetSessionListRespond
		if err := json.Unmarshal([]byte(cached), &list); err != nil {
			return nil, fmt.Errorf("解析私聊会话缓存失败: %w", err)
		}
		result := make([]userresp.GetSessionListRespond, 0, len(list))
		for _, item := range list {
			if strings.HasPrefix(item.Id, "U") {
				result = append(result, item)
			}
		}
		return result, nil
	}
	if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("读取私聊会话缓存失败: %w", err)
	}
	sessions, err := s.sessionDAO.GetSessionListBySendId(userId)
	if err != nil {
		return nil, fmt.Errorf("查询私聊会话失败: %w", err)
	}
	var ids []string
	for _, session := range sessions {
		if strings.HasPrefix(session.ReceiveId, "U") {
			ids = append(ids, session.ReceiveId)
		}
	}
	users := make(map[string]models.UserInfo)
	if len(ids) > 0 {
		list, err := s.userInfoDAO.FindUsersByUuids(ids)
		if err != nil {
			return nil, fmt.Errorf("批量查询私聊用户失败: %w", err)
		}
		for _, user := range list {
			users[user.Uuid] = user
		}
	}
	result := make([]userresp.GetSessionListRespond, 0, len(ids))
	for _, session := range sessions {
		user, ok := users[session.ReceiveId]
		if !ok {
			continue
		}
		var lastMessageAt string
		if session.LastMessageAt.Valid {
			lastMessageAt = session.LastMessageAt.Time.Format("2006-01-02 15:04:05")
		}
		result = append(result, userresp.GetSessionListRespond{
			SessionId: session.Uuid, Id: session.ReceiveId, Name: user.Nickname,
			Avatar: user.Avatar, LastMessage: session.LastMessage, LastMessageAt: lastMessageAt,
		})
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化私聊会话失败: %w", err)
	}
	if err := myredis.SetKeyEx(key, string(payload), time.Minute*constants.REDIS_TIMEOUT); err != nil {
		zlog.Error("缓存私聊会话失败: " + err.Error())
	}
	return result, nil
}

// DeleteSession 删除会话
func (s *SessionServiceImpl) DeleteSession(userId string, sessionId string) error {
	// 从数据库中删除会话
	if err := s.sessionDAO.DeleteSession(sessionId); err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	// 清除用户会话列表缓存
	if err := myredis.DelKeysWithPattern(constants.SessionListKeyPrefix + userId); err != nil {
		zlog.Error(err.Error())
	}

	return nil
}

// CheckOpenSessionAllowed 检查双方用户状态及私聊目标，不依赖联系人关系。
func (s *SessionServiceImpl) CheckOpenSessionAllowed(sendId string, receiveId string) (bool, error) {
	if !strings.HasPrefix(sendId, "U") || !strings.HasPrefix(receiveId, "U") || sendId == receiveId {
		return false, fmt.Errorf("只能与其他用户发起私聊")
	}
	for _, id := range []string{sendId, receiveId} {
		user, err := s.userInfoDAO.FindUserByUuid(id)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && user == nil) {
			return false, fmt.Errorf("私聊用户不存在")
		}
		if err != nil {
			return false, fmt.Errorf("查询私聊用户失败: %w", err)
		}
		if user.Status == userstatusenum.DISABLE {
			return false, fmt.Errorf("用户已被禁用，无法私聊")
		}
	}
	return true, nil
}
