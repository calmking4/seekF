package userservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/zlog"
)

// MessageService 消息服务接口
type MessageService interface {
	// 查询新增私信及已加载消息的当前状态，不推进阅读进度。
	RefreshPrivateMessages(ctx context.Context, userId, peerId, afterId string, knownIds []string) ([]userresp.GetMessageListRespond, []userresp.GetMessageListRespond, bool, error)
	// 仅从本人私聊记录中删除指定消息。
	DeleteMessage(ctx context.Context, userId, messageId string) error
	// 撤回本人在时限内发送的私聊消息。
	RecallMessage(ctx context.Context, userId, messageId string) (models.Message, error)
	GetUserMessageList(userOneId string, userTwoId string, page int, pageSize int, beforeMessageId ...string) ([]userresp.GetMessageListRespond, int64, error)
	// 搜索当前用户的私聊消息（双向文字匹配）。
	SearchMessages(userId string, sessionId string, keyword string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error)
	// 查询当前用户的跨私聊会话消息联想。
	SearchMessageSuggestions(userId string, keyword string, pageSize int) ([]userresp.SearchMessageSuggestionResp, error)
}

// MessageServiceImpl 消息服务实现
type MessageServiceImpl struct {
	messageDAO userdao.MessageDAO
	sessionDAO userdao.SessionDAO
}

// NewMessageService 创建消息服务实例
func NewMessageService(messageDAO userdao.MessageDAO, sessionDAO userdao.SessionDAO) MessageService {
	return &MessageServiceImpl{
		messageDAO: messageDAO,
		sessionDAO: sessionDAO,
	}
}

// GetUserMessageList 获取用户聊天记录（分页）
func (s *MessageServiceImpl) GetUserMessageList(userOneId string, userTwoId string, page int, pageSize int, beforeMessageId ...string) ([]userresp.GetMessageListRespond, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 获取消息总数
	total, err := s.messageDAO.CountMessagesBetweenUsers(userOneId, userTwoId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, 0, fmt.Errorf("系统错误")
	}

	// 从数据库获取消息列表（倒序）
	messageList, err := s.messageDAO.GetMessagesBetweenUsers(userOneId, userTwoId, pageSize, offset, beforeMessageId...)
	if err != nil {
		zlog.Error(err.Error())
		return nil, 0, fmt.Errorf("系统错误")
	}

	// 构建响应（倒序返回，前端需要反转显示）
	var rspList []userresp.GetMessageListRespond
	for _, message := range messageList {
		rspList = append(rspList, userresp.GetMessageListRespond{
			Id:         message.Id,
			Uuid:       message.Uuid,
			SessionId:  message.SessionId,
			SendId:     message.SendId,
			SendName:   message.SendName,
			SendAvatar: message.SendAvatar,
			ReceiveId:  message.ReceiveId,
			Content:    message.Content,
			Url:        message.Url,
			Type:       message.Type,
			FileType:   message.FileType,
			FileName:   message.FileName,
			FileSize:   message.FileSize,
			CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return rspList, total, nil
}

// privateMessageRespond 序列化数据库当前状态，不使用ES或历史通知中的旧正文。
func privateMessageRespond(message models.Message) userresp.GetMessageListRespond {
	return userresp.GetMessageListRespond{
		Id: message.Id, Uuid: message.Uuid, SessionId: message.SessionId, SendId: message.SendId,
		SendName: message.SendName, SendAvatar: message.SendAvatar, ReceiveId: message.ReceiveId,
		Type: message.Type, Content: message.Content, Url: message.Url, FileType: message.FileType,
		FileName: message.FileName, FileSize: message.FileSize, CreatedAt: message.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// RefreshPrivateMessages 使用数据库快照刷新当前会话，最多返回100条新增消息和100条已知消息。
func (s *MessageServiceImpl) RefreshPrivateMessages(ctx context.Context, userId, peerId, afterId string, knownIds []string) ([]userresp.GetMessageListRespond, []userresp.GetMessageListRespond, bool, error) {
	id, err := strconv.ParseInt(afterId, 10, 64)
	if err != nil || id < 0 || len(knownIds) > 100 || !strings.HasPrefix(userId, "U") || !strings.HasPrefix(peerId, "U") || userId == peerId {
		return nil, nil, false, fmt.Errorf("私聊刷新参数无效")
	}
	added := make([]userresp.GetMessageListRespond, 0)
	updated := make([]userresp.GetMessageListRespond, 0)
	var hasMore bool
	err = db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := userdao.NewSessionDAO(tx).GetSessionBySendAndReceiveId(userId, peerId)
		if err != nil {
			return fmt.Errorf("查询本人私聊会话失败: %w", err)
		}
		messages, current, err := userdao.NewMessageDAO(tx).GetPrivateMessageUpdates(userId, peerId, id, knownIds)
		if err != nil {
			return fmt.Errorf("刷新私聊消息失败: %w", err)
		}
		hasMore = len(messages) > 100
		if hasMore {
			messages = messages[:100]
		}
		for _, message := range messages {
			item := privateMessageRespond(message)
			item.SessionId = session.Uuid
			added = append(added, item)
		}
		for _, message := range current {
			item := privateMessageRespond(message)
			item.SessionId = session.Uuid
			updated = append(updated, item)
		}
		return nil
	})
	return added, updated, hasMore, err
}

// SearchMessages 搜索本人有效私聊中的双向文字消息，并返回本人会话编号。
func (s *MessageServiceImpl) SearchMessages(userId string, sessionId string, keyword string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if userId == "" {
		return nil, 0, fmt.Errorf("无法获取用户信息")
	}
	if keyword == "" {
		return []userresp.GetMessageListRespond{}, 0, nil
	}
	if len([]rune(keyword)) > 200 {
		return nil, 0, fmt.Errorf("搜索关键词不能超过200个字符")
	}
	if sessionId != "" {
		session, err := s.sessionDAO.GetSessionByUuid(sessionId)
		if err != nil {
			return nil, 0, fmt.Errorf("查询私聊会话失败: %w", err)
		}
		if session.SendId != userId || !strings.HasPrefix(session.ReceiveId, "U") {
			return nil, 0, fmt.Errorf("只能搜索自己的私聊记录")
		}
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	rows, total, err := s.messageDAO.SearchPrivateMessages(userId, sessionId, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("搜索聊天记录失败: %w", err)
	}
	result := make([]userresp.GetMessageListRespond, 0, len(rows))
	for _, row := range rows {
		result = append(result, userresp.GetMessageListRespond{
			Id: row.Id, Uuid: row.Uuid, SessionId: row.ViewerSessionId,
			SendId: row.SendId, SendName: row.SendName, SendAvatar: row.SendAvatar,
			ReceiveId: row.ReceiveId, Content: row.Content, Type: row.Type,
			CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, total, nil
}

// SearchMessageSuggestions 复用双向私聊搜索生成联想，结果携带定位消息所需的编号。
func (s *MessageServiceImpl) SearchMessageSuggestions(userId string, keyword string, pageSize int) ([]userresp.SearchMessageSuggestionResp, error) {
	if pageSize < 1 {
		pageSize = 5
	}
	if pageSize > 10 {
		pageSize = 10
	}
	messages, _, err := s.SearchMessages(userId, "", keyword, 1, pageSize)
	if err != nil {
		return nil, err
	}
	result := make([]userresp.SearchMessageSuggestionResp, 0, len(messages))
	for _, message := range messages {
		result = append(result, userresp.SearchMessageSuggestionResp{
			Uuid: message.Uuid, SessionId: message.SessionId, Content: message.Content,
			SendName: message.SendName, CreatedAt: message.CreatedAt,
		})
	}
	return result, nil
}

// RecallMessage 在事务内校验归属与期限，撤回并更新双方摘要。
func (s *MessageServiceImpl) RecallMessage(ctx context.Context, userId, messageId string) (models.Message, error) {
	var message models.Message
	if userId == "" || messageId == "" {
		return message, fmt.Errorf("消息编号和登录信息不能为空")
	}
	err := db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		messages := userdao.NewMessageDAO(tx)
		var err error
		message, err = messages.GetMessageForUpdate(messageId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("消息不存在")
		}
		if err != nil {
			return fmt.Errorf("查询撤回消息失败: %w", err)
		}
		if message.SendId != userId || !strings.HasPrefix(message.ReceiveId, "U") || !strings.HasPrefix(message.SendId, "U") {
			return fmt.Errorf("只能撤回自己发送的私聊消息")
		}
		if message.Type == constants.MessageRecalled {
			return nil
		}
		if message.Type != 0 && message.Type != 2 {
			return fmt.Errorf("该消息不支持撤回")
		}
		if time.Since(message.CreatedAt) > constants.MessageRecallLimit {
			return fmt.Errorf("只能撤回两分钟内的消息")
		}
		if err := messages.RecallMessage(messageId); err != nil {
			return fmt.Errorf("撤回消息失败: %w", err)
		}
		if err := userdao.NewSessionDAO(tx).RefreshPrivateMessagePreview(message.SendId, message.ReceiveId); err != nil {
			return fmt.Errorf("更新撤回消息摘要失败: %w", err)
		}
		message.Type = constants.MessageRecalled
		message.Content, message.Url, message.FileName, message.FileSize, message.FileType, message.Sources, message.Posts = "", "", "", "", "", "", ""
		return nil
	})
	return message, err
}

// DeleteMessage 锁定私聊消息并在同一事务内保存个人删除标记和更新摘要。
func (s *MessageServiceImpl) DeleteMessage(ctx context.Context, userId, messageId string) error {
	if userId == "" || messageId == "" {
		return fmt.Errorf("消息编号和登录信息不能为空")
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		messages := userdao.NewMessageDAO(tx)
		message, err := messages.GetMessageForUpdate(messageId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("消息不存在")
		}
		if err != nil {
			return fmt.Errorf("查询删除消息失败: %w", err)
		}
		if !strings.HasPrefix(message.SendId, "U") || !strings.HasPrefix(message.ReceiveId, "U") || (message.SendId != userId && message.ReceiveId != userId) {
			return fmt.Errorf("只能删除自己参与的私聊消息")
		}
		if err := messages.DeleteMessageForUser(userId, messageId, message.SendId == userId); err != nil {
			return fmt.Errorf("删除私聊消息失败: %w", err)
		}
		if err := userdao.NewSessionDAO(tx).RefreshPrivateMessagePreview(message.SendId, message.ReceiveId); err != nil {
			return fmt.Errorf("更新删除消息摘要失败: %w", err)
		}
		return nil
	})
}
