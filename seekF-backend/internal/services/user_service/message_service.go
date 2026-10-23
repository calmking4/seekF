package userservice

import (
	"fmt"
	"strings"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/zlog"
)

// MessageService 消息服务接口
type MessageService interface {
	GetUserMessageList(userOneId string, userTwoId string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error)
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
func (s *MessageServiceImpl) GetUserMessageList(userOneId string, userTwoId string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error) {
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
	messageList, err := s.messageDAO.GetMessagesBetweenUsers(userOneId, userTwoId, pageSize, offset)
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
