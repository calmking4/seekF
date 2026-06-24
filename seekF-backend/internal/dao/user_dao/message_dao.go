package userdao

import (
	"seekF-backend/internal/models"

	"gorm.io/gorm"
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
	GetMessagesBySessionIdDesc(sessionId string, limit int, offset int) ([]models.Message, error)
	CountMessagesBySessionId(sessionId string) (int64, error)
	DeleteMessagesBySessionId(sessionId string) error
	// GetMessagesBySessionIdWithCursor 游标分页，基于 created_at 时间戳
	GetMessagesBySessionIdWithCursor(sessionId string, cursor string, limit int, direction string) ([]models.Message, error)
	// SearchMessagesBySessionIds 在指定会话列表中搜索消息（MySQL LIKE 降级）
	SearchMessagesBySessionIds(sessionIds []string, keyword string, limit int) ([]models.Message, error)
}

// MessageDAOImpl 消息DAO实现
type MessageDAOImpl struct {
	db *gorm.DB
}

// NewMessageDAO 创建消息DAO实例
func NewMessageDAO(db *gorm.DB) MessageDAO {
	return &MessageDAOImpl{db: db}
}

// GetMessagesBetweenUsers 获取两个用户之间的消息记录（分页，按时间倒序）
func (d *MessageDAOImpl) GetMessagesBetweenUsers(userOneId string, userTwoId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := d.db.Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Order("created_at DESC").Limit(limit).Offset(offset).Find(&messageList)
	return messageList, result.Error
}

// GetMessagesByReceiverId 根据接收者ID获取消息记录（分页，按时间倒序）
func (d *MessageDAOImpl) GetMessagesByReceiverId(receiverId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := d.db.Where("receive_id = ?", receiverId).Order("created_at DESC").Limit(limit).Offset(offset).Find(&messageList)
	return messageList, result.Error
}

// CountMessagesBetweenUsers 统计两个用户之间的消息总数
func (d *MessageDAOImpl) CountMessagesBetweenUsers(userOneId string, userTwoId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.Message{}).Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Count(&count)
	return count, result.Error
}

// CountMessagesByReceiverId 统计群聊消息总数
func (d *MessageDAOImpl) CountMessagesByReceiverId(receiverId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.Message{}).Where("receive_id = ?", receiverId).Count(&count)
	return count, result.Error
}

// CreateMessage 创建消息
func (d *MessageDAOImpl) CreateMessage(message *models.Message) error {
	return d.db.Create(message).Error
}

// UpdateMessageStatus 更新消息状态
func (d *MessageDAOImpl) UpdateMessageStatus(uuid string, status int8) error {
	return d.db.Model(&models.Message{}).Where("uuid = ?", uuid).Update("status", status).Error
}

// GetMessagesBySessionId 根据会话ID获取消息记录（分页，按时间正序，用于构建上下文）
func (d *MessageDAOImpl) GetMessagesBySessionId(sessionId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := d.db.Where("session_id = ?", sessionId).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messageList)
	return messageList, result.Error
}

// GetMessagesBySessionIdDesc 根据会话ID获取消息记录（分页，按时间倒序，用于获取历史消息）
func (d *MessageDAOImpl) GetMessagesBySessionIdDesc(sessionId string, limit int, offset int) ([]models.Message, error) {
	var messageList []models.Message
	result := d.db.Where("session_id = ?", sessionId).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messageList)
	return messageList, result.Error
}

// CountMessagesBySessionId 统计指定会话的消息总数
func (d *MessageDAOImpl) CountMessagesBySessionId(sessionId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.Message{}).Where("session_id = ?", sessionId).Count(&count)
	return count, result.Error
}

// DeleteMessagesBySessionId 删除指定会话的所有消息
func (d *MessageDAOImpl) DeleteMessagesBySessionId(sessionId string) error {
	result := d.db.Where("session_id = ?", sessionId).Delete(&models.Message{})
	return result.Error
}

// GetMessagesBySessionIdWithCursor 游标分页，基于 created_at 时间戳
// direction: "prev" 向前翻页（更旧的消息），"next" 向后翻页（更新的消息）
func (d *MessageDAOImpl) GetMessagesBySessionIdWithCursor(sessionId string, cursor string, limit int, direction string) ([]models.Message, error) {
	var messageList []models.Message
	query := d.db.Where("session_id = ?", sessionId)

	if cursor != "" {
		if direction == "prev" {
			// 向前翻页：获取比 cursor 更旧的消息
			query = query.Where("created_at < ?", cursor).Order("created_at DESC")
		} else {
			// 向后翻页：获取比 cursor 更新的消息
			query = query.Where("created_at > ?", cursor).Order("created_at ASC")
		}
	} else {
		// 无游标时默认按时间倒序
		query = query.Order("created_at DESC")
	}

	result := query.Limit(limit).Find(&messageList)

	// 如果是向后翻页，需要反转顺序以保持时间倒序
	if direction == "next" && cursor != "" {
		for i, j := 0, len(messageList)-1; i < j; i, j = i+1, j-1 {
			messageList[i], messageList[j] = messageList[j], messageList[i]
		}
	}

	return messageList, result.Error
}

// SearchMessagesBySessionIds 在指定会话列表中搜索消息（MySQL LIKE 降级）
func (d *MessageDAOImpl) SearchMessagesBySessionIds(sessionIds []string, keyword string, limit int) ([]models.Message, error) {
	if len(sessionIds) == 0 || keyword == "" {
		return nil, nil
	}
	var messageList []models.Message
	likePattern := "%" + keyword + "%"
	result := d.db.Where("session_id IN ? AND content LIKE ?", sessionIds, likePattern).
		Order("created_at DESC").
		Limit(limit).
		Find(&messageList)
	return messageList, result.Error
}
