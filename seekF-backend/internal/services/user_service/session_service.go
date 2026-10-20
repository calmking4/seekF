package userservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
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
	// 获取本人有效私聊会话的未读数。
	GetUnreadSessions(userId string) (*userresp.SessionUnreadRespond, error)
	// 校验实际入站消息后单调推进已读边界。
	ReadSession(ctx context.Context, userId string, sessionId string, messageId int64) error
	// 锁定双方用户并复核拉黑状态后保存私信。
	SavePrivateMessage(ctx context.Context, message *models.Message) error
	OpenSession(sendId string, receiveId string) (string, error)
	OpenSessionWithInfo(sendId string, receiveId string) (*OpenSessionResult, error)
	CreateSession(sendId string, receiveId string) (string, error)
	GetSessionList(userId string) ([]userresp.GetSessionListRespond, error)
	DeleteSession(userId string, sessionId string) error
	CheckOpenSessionAllowed(sendId string, receiveId string) (bool, error)
}

// GetUnreadSessions 获取私聊未读统计，不读取会话列表缓存，避免返回过期计数。
func (s *SessionServiceImpl) GetUnreadSessions(userId string) (*userresp.SessionUnreadRespond, error) {
	rows, err := s.sessionDAO.GetUnreadSessions(userId)
	if err != nil {
		return nil, fmt.Errorf("查询私聊未读数失败: %w", err)
	}
	result := &userresp.SessionUnreadRespond{List: make([]userresp.SessionUnreadItem, 0, len(rows))}
	for _, row := range rows {
		result.List = append(result.List, userresp.SessionUnreadItem{SessionId: row.SessionId, PeerId: row.PeerId, UnreadCount: row.UnreadCount, LatestMessageId: row.LatestMessageId, LastReadMessageId: row.LastReadMessageId})
		result.Total += row.UnreadCount
	}
	return result, nil
}

// ReadSession 在事务内锁定本人会话并核验消息范围，只确认客户端实际展示的边界。
func (s *SessionServiceImpl) ReadSession(ctx context.Context, userId string, sessionId string, messageId int64) error {
	if sessionId == "" || messageId <= 0 {
		return fmt.Errorf("已读会话或消息边界无效")
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewSessionDAO(tx)
		session, err := dao.GetOwnedSessionForUpdate(userId, sessionId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("私聊会话不存在或无权访问")
		}
		if err != nil {
			return fmt.Errorf("查询私聊会话失败: %w", err)
		}
		if !strings.HasPrefix(session.ReceiveId, "U") {
			return fmt.Errorf("只能标记用户私聊为已读")
		}
		valid, err := dao.HasIncomingMessage(userId, session.ReceiveId, messageId)
		if err != nil {
			return fmt.Errorf("核验私聊已读消息失败: %w", err)
		}
		if !valid {
			return fmt.Errorf("已读消息不属于该会话的入站私信")
		}
		if err := dao.AdvanceRead(userId, sessionId, messageId); err != nil {
			return fmt.Errorf("更新私聊阅读进度失败: %w", err)
		}
		return nil
	})
}

// SavePrivateMessage 与拉黑操作采用相同用户锁顺序，防止校验后并发拉黑仍写入新私信。
func (s *SessionServiceImpl) SavePrivateMessage(ctx context.Context, message *models.Message) error {
	if message == nil || !strings.HasPrefix(message.SendId, "U") || !strings.HasPrefix(message.ReceiveId, "U") || message.SendId == message.ReceiveId {
		return fmt.Errorf("私信目标无效")
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := userdao.NewUserInfoDAO(tx)
		ids := []string{message.SendId, message.ReceiveId}
		sort.Strings(ids)
		for _, id := range ids {
			user, err := users.FindUserByUuidForUpdate(id)
			if err != nil {
				return fmt.Errorf("查询私信用户失败: %w", err)
			}
			if user == nil || user.Status == userstatusenum.DISABLE {
				return fmt.Errorf("私信用户不存在或已禁用")
			}
		}
		if err := checkUserBlock(tx, message.SendId, message.ReceiveId); err != nil {
			return err
		}
		if err := userdao.NewMessageDAO(tx).CreateMessage(message); err != nil {
			return fmt.Errorf("保存私信失败: %w", err)
		}
		return nil
	})
}

type SessionServiceImpl struct {
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
	blockDAO    *userdao.BlockDAO
}

// NewSessionService 构造无需好友关系的用户私聊服务。
func NewSessionService(
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
	blockDAO *userdao.BlockDAO,
) SessionService {
	return &SessionServiceImpl{
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
		blockDAO:    blockDAO,
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
	// 仅软删除本人私聊会话，保留对方会话和历史消息。
	if err := s.sessionDAO.DeleteSession(userId, sessionId); err != nil {
		zlog.Error("删除私聊会话失败，用户=" + userId + "，会话=" + sessionId + ": " + err.Error())
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
	blocked, blockedBy, err := s.blockDAO.Status(sendId, receiveId)
	if err != nil {
		return false, fmt.Errorf("查询私聊拉黑状态失败: %w", err)
	}
	if blocked || blockedBy {
		return false, fmt.Errorf("存在拉黑关系，无法发送私信")
	}
	return true, nil
}
