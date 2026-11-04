package userservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	userstatusenum "seekF-backend/internal/pkg/enum/user_enum/user_status_enum"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"gorm.io/gorm"
)

// OpenSessionResult 打开会话结果
type OpenSessionResult struct {
	SessionId string `json:"session_id"`
}

type SessionService interface {
	// 在事务内设置本人私聊置顶状态，返回实际保存的置顶时间。
	SetPinned(ctx context.Context, userId, sessionId string, pinned bool) (*time.Time, error)
	// 获取本人有效私聊会话的未读数。
	GetUnreadSessions(ctx context.Context, userId string) (*userresp.SessionUnreadRespond, error)
	// 校验实际入站消息后单调推进已读边界。
	ReadSession(ctx context.Context, userId string, sessionId string, messageId int64) error
	// 锁定双方用户并在同一事务中去重、保存消息及更新双方会话。
	SendPrivateMessage(ctx context.Context, userId string, req userreq.ChatMessageRequest) (*PrivateMessageResult, error)
	OpenSession(ctx context.Context, sendId string, receiveId string) (string, error)
	OpenSessionWithInfo(ctx context.Context, sendId string, receiveId string) (*OpenSessionResult, error)
	CreateSession(ctx context.Context, sendId string, receiveId string) (string, error)
	GetSessionList(ctx context.Context, userId string) ([]userresp.GetSessionListRespond, error)
	DeleteSession(ctx context.Context, userId string, sessionId string) error
	CheckOpenSessionAllowed(ctx context.Context, sendId string, receiveId string) (bool, error)
}

// GetUnreadSessions 获取私聊未读统计，不读取会话列表缓存，避免返回过期计数。
func (s *SessionServiceImpl) GetUnreadSessions(ctx context.Context, userId string) (*userresp.SessionUnreadRespond, error) {
	rows, err := s.sessionDAO.GetUnreadSessions(ctx, userId)
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
		session, err := dao.GetOwnedSessionForUpdate(ctx, userId, sessionId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("私聊会话不存在或无权访问")
		}
		if err != nil {
			return fmt.Errorf("查询私聊会话失败: %w", err)
		}
		if !strings.HasPrefix(session.ReceiveId, "U") {
			return fmt.Errorf("只能标记用户私聊为已读")
		}
		valid, err := dao.HasIncomingMessage(ctx, userId, session.ReceiveId, messageId)
		if err != nil {
			return fmt.Errorf("核验私聊已读消息失败: %w", err)
		}
		if !valid {
			return fmt.Errorf("已读消息不属于该会话的入站私信")
		}
		if err := dao.AdvanceRead(ctx, userId, sessionId, messageId); err != nil {
			return fmt.Errorf("更新私聊阅读进度失败: %w", err)
		}
		return nil
	})
}

// PrivateMessageResult 保存发送事务结果，不对应数据库表。
type PrivateMessageResult struct {
	Message           models.Message
	ReceiverSessionId string
	Duplicate         bool
}

// SendPrivateMessage 复用消息唯一索引去重，提交后由连接层直接推送，不依赖Kafka或ES。
func (s *SessionServiceImpl) SendPrivateMessage(ctx context.Context, userId string, req userreq.ChatMessageRequest) (*PrivateMessageResult, error) {
	if !strings.HasPrefix(userId, "U") || !strings.HasPrefix(req.ReceiveId, "U") || userId == req.ReceiveId {
		return nil, fmt.Errorf("私信目标无效")
	}
	if req.Type != 0 && req.Type != 2 {
		return nil, fmt.Errorf("私聊仅支持文字和文件消息")
	}
	if len(req.ClientMessageId) > constants.PrivateChatRequestIDLimit {
		return nil, fmt.Errorf("消息请求编号过长")
	}
	if req.Type == 0 {
		if strings.TrimSpace(req.Content) == "" || utf8.RuneCountInString(req.Content) > constants.PrivateChatContentLimit {
			return nil, fmt.Errorf("私信内容不能为空且不能超过%d个字符", constants.PrivateChatContentLimit)
		}
		req.Url, req.FileName, req.FileSize, req.FileType = "", "", "0B", ""
	} else {
		parsed, err := url.Parse(req.Url)
		if err != nil || len(req.Url) > 255 || !(strings.HasPrefix(req.Url, "/static/") || (parsed.Host != "" && (parsed.Scheme == "https" || parsed.Scheme == "http"))) {
			return nil, fmt.Errorf("文件地址无效")
		}
		if utf8.RuneCountInString(req.FileName) > 50 || len(req.FileType) > 10 || len(req.FileSize) > 20 {
			return nil, fmt.Errorf("文件描述超过长度限制")
		}
		req.Content = ""
	}
	uuid := "M" + util.GetNowAndLenRandomString(11)
	if req.ClientMessageId != "" {
		// 认证用户和请求编号共同决定既有20字节UUID，重试复用唯一索引，无需改表。
		sum := sha256.Sum256([]byte(userId + ":" + req.ClientMessageId))
		uuid = "M" + hex.EncodeToString(sum[:])[:19]
	}
	result := &PrivateMessageResult{}
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := userdao.NewUserInfoDAO(tx)
		ids := []string{userId, req.ReceiveId}
		sort.Strings(ids)
		profiles := make(map[string]*models.UserInfo, 2)
		for _, id := range ids {
			user, err := users.FindUserByUuidForUpdate(ctx, id)
			if err != nil {
				return fmt.Errorf("查询私信用户失败: %w", err)
			}
			if user == nil {
				return fmt.Errorf("私信用户不存在")
			}
			profiles[id] = user
		}
		messages := userdao.NewMessageDAO(tx)
		existing, err := messages.GetMessageByUuid(ctx, uuid)
		if err == nil {
			if existing.SendId != userId || existing.ReceiveId != req.ReceiveId {
				return fmt.Errorf("消息请求编号冲突")
			}
			if existing.Type != constants.MessageRecalled && existing.SendDeletedAt == nil &&
				(existing.Type != req.Type || existing.Content != req.Content || existing.Url != req.Url || existing.FileName != req.FileName || existing.FileSize != req.FileSize || existing.FileType != req.FileType) {
				return fmt.Errorf("相同消息请求编号不能发送不同内容")
			}
			result.Message, result.Duplicate = existing, true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询重复私信失败: %w", err)
		}
		// 已成功请求可查询原结果；新发送仍需在锁内复核禁用与拉黑。
		for _, profile := range profiles {
			if profile.Status == userstatusenum.DISABLE {
				return fmt.Errorf("私信用户已禁用")
			}
		}
		if err := checkUserBlock(ctx, tx, userId, req.ReceiveId); err != nil {
			return err
		}
		sessions := userdao.NewSessionDAO(tx)
		owned, err := sessions.GetOwnedSessionForUpdate(ctx, userId, req.SessionId)
		if err != nil {
			return fmt.Errorf("查询本人私聊会话失败: %w", err)
		}
		if owned.ReceiveId != req.ReceiveId {
			return fmt.Errorf("无权向该会话发送私信")
		}
		receiver, err := ensurePrivateSession(ctx, sessions, req.ReceiveId, userId)
		if err != nil {
			return fmt.Errorf("创建或恢复接收方会话失败: %w", err)
		}
		sender := profiles[userId]
		message := models.Message{
			Uuid: uuid, SessionId: owned.Uuid, Type: req.Type, Content: req.Content, Url: req.Url,
			SendId: userId, SendName: sender.Nickname, SendAvatar: sender.Avatar, ReceiveId: req.ReceiveId,
			FileName: req.FileName, FileSize: req.FileSize, FileType: req.FileType, Status: 1, CreatedAt: time.Now(),
		}
		if err := messages.CreateMessage(ctx, &message); err != nil {
			return fmt.Errorf("保存私信失败: %w", err)
		}
		if err := sessions.RefreshPrivateMessagePreview(ctx, userId, req.ReceiveId); err != nil {
			return fmt.Errorf("更新私聊摘要失败: %w", err)
		}
		result.Message, result.ReceiverSessionId = message, receiver.Uuid
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ensurePrivateSession 在双方用户已锁定的事务内创建或恢复会话，避免并发打开与发送重复建会话。
func ensurePrivateSession(ctx context.Context, dao userdao.SessionDAO, userId, peerId string) (models.Session, error) {
	session, err := dao.GetSessionBySendAndReceiveId(ctx, userId, peerId)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return session, err
	}
	session, err = dao.GetDeletedSessionBySendAndReceiveId(ctx, userId, peerId)
	if err == nil {
		return session, dao.RestoreSession(ctx, session.Uuid)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return session, err
	}
	session = models.Session{Uuid: "S" + util.GetNowAndLenRandomString(11), SendId: userId, ReceiveId: peerId, CreatedAt: time.Now()}
	return session, dao.CreateSession(ctx, &session)
}

type SessionServiceImpl struct {
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
	blockDAO    userdao.BlockDAO
}

// NewSessionService 构造无需好友关系的用户私聊服务。
func NewSessionService(
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
	blockDAO userdao.BlockDAO,
) SessionService {
	return &SessionServiceImpl{
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
		blockDAO:    blockDAO,
	}
}

// OpenSession 锁定双方用户后创建或恢复私聊会话，与发送和拉黑保持同一锁顺序。
func (s *SessionServiceImpl) OpenSession(ctx context.Context, sendId string, receiveId string) (string, error) {
	if !strings.HasPrefix(sendId, "U") || !strings.HasPrefix(receiveId, "U") || sendId == receiveId {
		return "", fmt.Errorf("私聊目标无效")
	}
	var session models.Session
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		users := userdao.NewUserInfoDAO(tx)
		ids := []string{sendId, receiveId}
		sort.Strings(ids)
		for _, id := range ids {
			user, err := users.FindUserByUuidForUpdate(ctx, id)
			if err != nil {
				return fmt.Errorf("查询私聊用户失败: %w", err)
			}
			if user == nil || user.Status == userstatusenum.DISABLE {
				return fmt.Errorf("私聊用户不存在或已禁用")
			}
		}
		if err := checkUserBlock(ctx, tx, sendId, receiveId); err != nil {
			return err
		}
		var err error
		session, err = ensurePrivateSession(ctx, userdao.NewSessionDAO(tx), sendId, receiveId)
		return err
	})
	return session.Uuid, err
}

// OpenSessionWithInfo 打开用户私聊并返回会话编号。
func (s *SessionServiceImpl) OpenSessionWithInfo(ctx context.Context, sendId string, receiveId string) (*OpenSessionResult, error) {
	sessionId, err := s.OpenSession(ctx, sendId, receiveId)
	if err != nil {
		return nil, err
	}

	return &OpenSessionResult{
		SessionId: sessionId,
	}, nil
}

// CreateSession 复用加锁后的打开流程，避免为同一用户对重复创建会话。
func (s *SessionServiceImpl) CreateSession(ctx context.Context, sendId string, receiveId string) (string, error) {
	return s.OpenSession(ctx, sendId, receiveId)
}

// SetPinned 锁定本人有效私聊后幂等更新置顶状态，重复置顶保留原时间。
func (s *SessionServiceImpl) SetPinned(ctx context.Context, userId, sessionId string, pinned bool) (*time.Time, error) {
	var pinnedAt *time.Time
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dao := userdao.NewSessionDAO(tx)
		session, err := dao.GetOwnedSessionForUpdate(ctx, userId, sessionId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("会话不存在或已删除")
		}
		if err != nil {
			return fmt.Errorf("查询置顶会话失败: %w", err)
		}
		if !strings.HasPrefix(session.ReceiveId, "U") {
			return fmt.Errorf("仅支持置顶私聊会话")
		}
		if err := checkUserBlock(ctx, tx, userId, session.ReceiveId); err != nil {
			return err
		}
		if pinned {
			pinnedAt = session.PinnedAt
			if pinnedAt == nil {
				now := time.Now().Truncate(time.Millisecond)
				pinnedAt = &now
			}
		}
		if err := dao.SetPinnedAt(ctx, userId, sessionId, pinnedAt); err != nil {
			return fmt.Errorf("设置会话置顶失败: %w", err)
		}
		return nil
	})
	return pinnedAt, err
}

// GetSessionList 读取当前数据库会话状态，避免旧缓存恢复已取消置顶或拉黑移除的会话。
func (s *SessionServiceImpl) GetSessionList(ctx context.Context, userId string) ([]userresp.GetSessionListRespond, error) {
	sessions, err := s.sessionDAO.GetSessionListBySendId(ctx, userId)
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
		list, err := s.userInfoDAO.FindUsersByUuids(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("批量查询私聊用户失败: %w", err)
		}
		for _, user := range list {
			users[user.Uuid] = user
		}
	}
	result := make([]userresp.GetSessionListRespond, 0, len(ids))
	for _, session := range sessions {
		if !strings.HasPrefix(session.ReceiveId, "U") {
			continue
		}
		user, ok := users[session.ReceiveId]
		if !ok {
			continue
		}
		var lastMessageAt string
		if session.LastMessageAt.Valid {
			lastMessageAt = session.LastMessageAt.Time.Format("2006-01-02 15:04:05")
		}
		pinnedAt := ""
		if session.PinnedAt != nil {
			pinnedAt = session.PinnedAt.Format(time.RFC3339Nano)
		}
		result = append(result, userresp.GetSessionListRespond{
			SessionId: session.Uuid, Id: session.ReceiveId, Name: user.Nickname,
			Avatar: user.Avatar, LastMessage: session.LastMessage, LastMessageAt: lastMessageAt,
			IsPinned: session.PinnedAt != nil, PinnedAt: pinnedAt,
		})
	}
	return result, nil
}

// DeleteSession 删除会话
func (s *SessionServiceImpl) DeleteSession(ctx context.Context, userId string, sessionId string) error {
	// 仅软删除本人私聊会话，保留对方会话和历史消息。
	if err := s.sessionDAO.DeleteSession(ctx, userId, sessionId); err != nil {
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
func (s *SessionServiceImpl) CheckOpenSessionAllowed(ctx context.Context, sendId string, receiveId string) (bool, error) {
	if !strings.HasPrefix(sendId, "U") || !strings.HasPrefix(receiveId, "U") || sendId == receiveId {
		return false, fmt.Errorf("只能与其他用户发起私聊")
	}
	for _, id := range []string{sendId, receiveId} {
		user, err := s.userInfoDAO.FindUserByUuid(ctx, id)
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
	blocked, blockedBy, err := s.blockDAO.Status(ctx, sendId, receiveId)
	if err != nil {
		return false, fmt.Errorf("查询私聊拉黑状态失败: %w", err)
	}
	if blocked || blockedBy {
		return false, fmt.Errorf("存在拉黑关系，无法发送私信")
	}
	return true, nil
}
