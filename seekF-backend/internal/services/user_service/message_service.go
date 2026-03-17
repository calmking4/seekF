package userservice

import (
	"fmt"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/pkg/zlog"
)

// MessageService 消息服务接口
type MessageService interface {
	GetMessageList(userOneId string, userTwoId string) ([]userresp.GetMessageListRespond, error)
}

// MessageServiceImpl 消息服务实现
type MessageServiceImpl struct {
	messageDAO userdao.MessageDAO
}

// NewMessageService 创建消息服务实例
func NewMessageService(messageDAO userdao.MessageDAO) MessageService {
	return &MessageServiceImpl{
		messageDAO: messageDAO,
	}
}

// GetMessageList 获取聊天记录
func (s *MessageServiceImpl) GetMessageList(userOneId string, userTwoId string) ([]userresp.GetMessageListRespond, error) {
	// 从数据库获取消息列表
	messageList, err := s.messageDAO.GetMessageList(userOneId, userTwoId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	// 构建响应
	var rspList []userresp.GetMessageListRespond
	for _, message := range messageList {
		rspList = append(rspList, userresp.GetMessageListRespond{
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

	return rspList, nil
}
