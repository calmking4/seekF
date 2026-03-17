package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

// MessageDAO 消息DAO接口
type MessageDAO interface {
	GetMessagesBetweenUsers(userOneId string, userTwoId string) ([]models.Message, error)
	GetMessagesByReceiverId(receiverId string) ([]models.Message, error)
}

// MessageDAOImpl 消息DAO实现
type MessageDAOImpl struct{}

// NewMessageDAO 创建消息DAO实例
func NewMessageDAO() MessageDAO {
	return &MessageDAOImpl{}
}

// GetMessagesBetweenUsers 获取两个用户之间的消息记录
func (d *MessageDAOImpl) GetMessagesBetweenUsers(userOneId string, userTwoId string) ([]models.Message, error) {
	var messageList []models.Message
	result := db.GormDB.Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Order("created_at ASC").Find(&messageList)
	return messageList, result.Error
}

// GetMessagesByReceiverId 根据接收者ID获取消息记录
func (d *MessageDAOImpl) GetMessagesByReceiverId(receiverId string) ([]models.Message, error) {
	var messageList []models.Message
	result := db.GormDB.Where("receive_id = ?", receiverId).Order("created_at ASC").Find(&messageList)
	return messageList, result.Error
}
