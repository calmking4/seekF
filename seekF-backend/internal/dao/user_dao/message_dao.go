package userdao

import (
	"strings"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MessageDAO 消息DAO接口
type MessageDAO interface {
	// 查询新增消息及指定已加载消息的当前可见状态，供网页恢复漏推。
	GetPrivateMessageUpdates(userId, peerId string, afterId int64, knownIds []string) ([]models.Message, []models.Message, error)
	// 查询指定编号的消息，调用方在发送事务中已锁定所属用户。
	GetMessageByUuid(uuid string) (models.Message, error)
	// 记录当前参与者的个人删除时间，不影响对方。
	DeleteMessageForUser(userId, messageId string, isSender bool) error
	// 锁定指定消息，供撤回事务校验。
	GetMessageForUpdate(uuid string) (models.Message, error)
	// 清除消息内容并保留撤回占位。
	RecallMessage(uuid string) error
	// 按本人有效私聊的双方用户搜索文字，返回本人会话编号及真实分页总数。
	SearchPrivateMessages(userId, sessionId, keyword string, page, pageSize int) ([]MessageSearchRow, int64, error)
	GetMessagesBetweenUsers(userOneId string, userTwoId string, limit int, offset int, beforeMessageId ...string) ([]models.Message, error)
	CountMessagesBetweenUsers(userOneId string, userTwoId string) (int64, error)
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

// MessageSearchRow 消息与查询用户会话编号的关联结果，不对应额外数据库字段。
type MessageSearchRow struct {
	models.Message
	ViewerSessionId string `gorm:"column:viewer_session_id"`
}

// SearchPrivateMessages 同时查询本人发送和收到的消息，避免双方会话编号不同导致漏搜。
func (d *MessageDAOImpl) SearchPrivateMessages(userId, sessionId, keyword string, page, pageSize int) ([]MessageSearchRow, int64, error) {
	pattern := "%" + strings.NewReplacer("=", "==", "%", "=%", "_", "=_").Replace(keyword) + "%"
	query := d.db.Table("message AS m").
		Joins("JOIN session AS s ON s.send_id = ? AND s.deleted_at IS NULL AND ((m.send_id = s.send_id AND m.receive_id = s.receive_id) OR (m.send_id = s.receive_id AND m.receive_id = s.send_id))", userId).
		Where("((m.send_id = ? AND m.send_deleted_at IS NULL) OR (m.receive_id = ? AND m.receive_deleted_at IS NULL))", userId, userId).
		Where("s.receive_id LIKE ? AND m.type = 0 AND m.content LIKE ? ESCAPE '='", "U%", pattern).
		Where("NOT EXISTS (SELECT 1 FROM user_block b WHERE b.user_id = ? AND b.blocked_user_id = s.receive_id COLLATE utf8mb4_unicode_ci)", userId)
	if sessionId != "" {
		query = query.Where("s.uuid = ?", sessionId)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]MessageSearchRow, 0)
	err := query.Select("m.*, s.uuid AS viewer_session_id").Order("m.created_at DESC, m.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error
	return rows, total, err
}

// MessageDAOImpl 消息DAO实现
type MessageDAOImpl struct {
	db *gorm.DB
}

// GetMessageByUuid 查询已有消息，避免对不存在的随机编号加间隙锁。
func (d *MessageDAOImpl) GetMessageByUuid(uuid string) (models.Message, error) {
	var message models.Message
	err := d.db.Where("uuid = ?", uuid).First(&message).Error
	return message, err
}

// GetPrivateMessageUpdates 从同一业务连接读取可见消息，个人删除后的编号不会返回。
func (d *MessageDAOImpl) GetPrivateMessageUpdates(userId, peerId string, afterId int64, knownIds []string) ([]models.Message, []models.Message, error) {
	query := d.db.Model(&models.Message{}).
		Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userId, peerId, peerId, userId).
		Where("(send_id = ? AND send_deleted_at IS NULL) OR (receive_id = ? AND receive_deleted_at IS NULL)", userId, userId)
	var added, updated []models.Message
	if err := query.Session(&gorm.Session{}).Where("id > ?", afterId).Order("id ASC").Limit(101).Find(&added).Error; err != nil {
		return nil, nil, err
	}
	if len(knownIds) > 0 {
		if err := query.Session(&gorm.Session{}).Where("uuid IN ?", knownIds).Find(&updated).Error; err != nil {
			return nil, nil, err
		}
	}
	return added, updated, nil
}

// NewMessageDAO 创建消息DAO实例
func NewMessageDAO(db *gorm.DB) MessageDAO {
	return &MessageDAOImpl{db: db}
}

// GetMessagesBetweenUsers 返回本人可见记录，使用消息锚点避免删除导致翻页遗漏。
func (d *MessageDAOImpl) GetMessagesBetweenUsers(userOneId string, userTwoId string, limit int, offset int, beforeMessageId ...string) ([]models.Message, error) {
	var messages []models.Message
	query := d.db.Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).
		Where("(send_id = ? AND send_deleted_at IS NULL) OR (receive_id = ? AND receive_deleted_at IS NULL)", userOneId, userOneId)
	if len(beforeMessageId) > 0 && beforeMessageId[0] != "" {
		var cursor models.Message
		if err := d.db.Where("uuid = ? AND ((send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?))", beforeMessageId[0], userOneId, userTwoId, userTwoId, userOneId).First(&cursor).Error; err != nil {
			return nil, err
		}
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.Id)
		offset = 0
	}
	err := query.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

// CountMessagesBetweenUsers 统计两个用户之间的消息总数
func (d *MessageDAOImpl) CountMessagesBetweenUsers(userOneId string, userTwoId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.Message{}).Where("(send_id = ? AND receive_id = ?) OR (send_id = ? AND receive_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Where("(send_id = ? AND send_deleted_at IS NULL) OR (receive_id = ? AND receive_deleted_at IS NULL)", userOneId, userOneId).Count(&count)
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

// GetMessageForUpdate 锁定消息以串行处理重复撤回。
func (d *MessageDAOImpl) GetMessageForUpdate(uuid string) (models.Message, error) {
	var message models.Message
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("uuid = ?", uuid).First(&message).Error
	return message, err
}

// RecallMessage 清空正文与文件信息并保留撤回占位。
func (d *MessageDAOImpl) RecallMessage(uuid string) error {
	return d.db.Model(&models.Message{}).Where("uuid = ?", uuid).Updates(map[string]interface{}{
		"type": constants.MessageRecalled, "content": "", "url": "", "file_type": "", "file_name": "", "file_size": "", "sources": "", "posts": "",
	}).Error
}

// DeleteMessageForUser 根据已校验的参与者身份写入个人删除标记，重复删除保持原时间。
func (d *MessageDAOImpl) DeleteMessageForUser(userId, messageId string, isSender bool) error {
	column, owner := "receive_deleted_at", "receive_id"
	if isSender {
		column, owner = "send_deleted_at", "send_id"
	}
	return d.db.Model(&models.Message{}).Where("uuid = ? AND "+owner+" = ? AND "+column+" IS NULL", messageId, userId).Update(column, gorm.Expr("CURRENT_TIMESTAMP(3)")).Error
}
