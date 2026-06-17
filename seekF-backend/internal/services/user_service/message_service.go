package userservice

import (
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"
)

// MessageService 消息服务接口
type MessageService interface {
	GetUserMessageList(userOneId string, userTwoId string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error)
	GetGroupMessageList(groupId string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error)
	// SearchMessages 搜索聊天消息（ES全文搜索）
	SearchMessages(sessionId string, keyword string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error)
}

// MessageServiceImpl 消息服务实现
type MessageServiceImpl struct {
	messageDAO    userdao.MessageDAO
}

// NewMessageService 创建消息服务实例
func NewMessageService(messageDAO userdao.MessageDAO) MessageService {
	return &MessageServiceImpl{
		messageDAO: messageDAO,
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

// GetGroupMessageList 获取群聊消息记录（分页）
func (s *MessageServiceImpl) GetGroupMessageList(groupId string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 获取消息总数
	total, err := s.messageDAO.CountMessagesByReceiverId(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, 0, fmt.Errorf("系统错误")
	}

	// 从数据库获取消息列表（倒序）
	messageList, err := s.messageDAO.GetMessagesByReceiverId(groupId, pageSize, offset)
	if err != nil {
		zlog.Error(err.Error())
		return nil, 0, fmt.Errorf("系统错误")
	}

	// 构建响应
	var rspList []userresp.GetMessageListRespond
	for _, message := range messageList {
		rspList = append(rspList, userresp.GetMessageListRespond{
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

// SearchMessages 搜索聊天消息（ES全文搜索）
func (s *MessageServiceImpl) SearchMessages(sessionId string, keyword string, page int, pageSize int) ([]userresp.GetMessageListRespond, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	// 检查ES是否可用
	if db.ESClient == nil {
		return nil, 0, fmt.Errorf("搜索服务暂不可用")
	}

	esMessageDAO := userdao.NewESMessageDAO()
	messages, total, err := esMessageDAO.SearchMessages(sessionId, keyword, page, pageSize)
	if err != nil {
		zlog.Error("ES搜索消息失败: " + err.Error())
		return nil, 0, fmt.Errorf("搜索失败")
	}

	var rspList []userresp.GetMessageListRespond
	for _, message := range messages {
		rspList = append(rspList, userresp.GetMessageListRespond{
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
