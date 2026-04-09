package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

// MessageDAO 消息DAO接口
type MessageDAO interface {
	GetMessagesBetweenUsers(userOneId string, userTwoId string, limit int, offset int) ([]models.Message, error)
	GetMessagesByReceiverId(receiverId string, limit int, offset int) ([]models.Message, error)
	CountMessagesBetweenUsers(userOneId string, userTwoId string) (int64, error)
	CountMessagesByReceiverId(receiverId string) (int64, error)
	CreateMessage(message *models.Message) error
	UpdateMessageStatus(uuid string, status int8) error
	// AI消息相关方法
	GetMessagesBySessionId(sessionId string, limit int, offset int) ([]models.Message, error)
	CountMessagesBySessionId(sessionId string) (int64, error)
	DeleteMessagesBySessionId(sessionId string) error
}

// MessageDAOImpl 消息DAO实现
type MessageDAOImpl struct{}

// NewMessageDAO 创建消息DAO实例
func NewMessageDAO() MessageDAO {
	return &MessageDAOImpl{}
}

// GetMessagesBetweenUsers 获取两个用户之间的消息记录（分页，按时间倒序）
func (d *MessageDAOImpl) GetMessagesBetweenUsers(userOneId string, userTwoId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := db.GormDB.Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Order("created_at DESC").Limit(limit).Offset(offset).Find(&messageList)
	return messageList, result.Error
}

// GetMessagesByReceiverId 根据接收者ID获取消息记录（分页，按时间倒序）
func (d *MessageDAOImpl) GetMessagesByReceiverId(receiverId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := db.GormDB.Where("receive_id = ?", receiverId).Order("created_at DESC").Limit(limit).Offset(offset).Find(&messageList)
	return messageList, result.Error
}

// CountMessagesBetweenUsers 统计两个用户之间的消息总数
func (d *MessageDAOImpl) CountMessagesBetweenUsers(userOneId string, userTwoId string) (int64, error) {
	var count int64
	result := db.GormDB.Model(&models.Message{}).Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Count(&count)
	return count, result.Error
}

// CountMessagesByReceiverId 统计群聊消息总数
func (d *MessageDAOImpl) CountMessagesByReceiverId(receiverId string) (int64, error) {
	var count int64
	result := db.GormDB.Model(&models.Message{}).Where("receive_id = ?", receiverId).Count(&count)
	return count, result.Error
}

// CreateMessage 创建消息
func (d *MessageDAOImpl) CreateMessage(message *models.Message) error {
	return db.GormDB.Create(message).Error
}

// UpdateMessageStatus 更新消息状态
func (d *MessageDAOImpl) UpdateMessageStatus(uuid string, status int8) error {
	return db.GormDB.Model(&models.Message{}).Where("uuid = ?", uuid).Update("status", status).Error
}

// GetMessagesBySessionId 根据会话ID获取消息记录（分页，按时间正序）
func (d *MessageDAOImpl) GetMessagesBySessionId(sessionId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := db.GormDB.Where("session_id = ?", sessionId).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messageList)
	return messageList, result.Error
}

// CountMessagesBySessionId 统计指定会话的消息总数
func (d *MessageDAOImpl) CountMessagesBySessionId(sessionId string) (int64, error) {
	var count int64
	result := db.GormDB.Model(&models.Message{}).Where("session_id = ?", sessionId).Count(&count)
	return count, result.Error
}

// DeleteMessagesBySessionId 删除指定会话的所有消息
func (d *MessageDAOImpl) DeleteMessagesBySessionId(sessionId string) error {
	result := db.GormDB.Where("session_id = ?", sessionId).Delete(&models.Message{})
	return result.Error
}
