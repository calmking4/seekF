package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/util"
	"time"
)

// SessionDAO 会话DAO接口
type SessionDAO interface {
	UpdateSessionByReceiveId(receiveId string, receiveName string, avatar string) error
	RemoveSessionBySendAndReceiveId(sendId string, receiveId string) error
	RemoveSessionsByReceiveId(receiveId string) error
	GetSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error)
	GetDeletedSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error)
	RestoreSession(sessionId string) error
	CreateSession(session *models.Session) error
	DeleteSession(sessionId string) error
	GetSessionListBySendId(userId string) ([]models.Session, error)
	GetSessionByUuid(uuid string) (models.Session, error)
	UpdateSessionLastMessage(sessionId string, lastMessage string, lastMessageAt time.Time) error
	UpdateSessionLastMessageByReceiveId(receiveId string, lastMessage string, lastMessageAt time.Time) error
	// AI会话相关方法
	CreateAISession(userId string, modelType string) (*models.Session, error)
	GetAISessionList(userId string) ([]models.Session, error)
	GetAISessionByUuid(uuid string) (*models.Session, error)
	DeleteAISession(uuid string) error
}

type SessionDAOImpl struct{}

func NewSessionDAO() SessionDAO {
	return &SessionDAOImpl{}
}

// UpdateSessionByReceiveId 根据接收者ID更新会话信息
func (d *SessionDAOImpl) UpdateSessionByReceiveId(receiveId string, receiveName string, avatar string) error {
	result := db.GormDB.Model(&models.Session{}).Where("receive_id = ?", receiveId).Updates(map[string]interface{}{
		"receive_name": receiveName,
		"avatar":       avatar,
	})
	return result.Error
}

// RemoveSessionBySendAndReceiveId 根据发送者ID和接收者ID删除会话
func (d *SessionDAOImpl) RemoveSessionBySendAndReceiveId(sendId string, receiveId string) error {
	result := db.GormDB.Where("send_id = ? AND receive_id = ?", sendId, receiveId).Delete(&models.Session{})
	return result.Error
}

// RemoveSessionsByReceiveId 批量删除指定接收ID的会话
func (d *SessionDAOImpl) RemoveSessionsByReceiveId(receiveId string) error {
	result := db.GormDB.Where("receive_id = ?", receiveId).Delete(&models.Session{})
	return result.Error
}

// GetSessionBySendAndReceiveId 根据发送者ID和接收者ID获取会话
func (d *SessionDAOImpl) GetSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error) {
	var session models.Session
	result := db.GormDB.Where("send_id = ? and receive_id = ?", sendId, receiveId).First(&session)
	return session, result.Error
}

// GetDeletedSessionBySendAndReceiveId 获取已删除的会话（软删除）
func (d *SessionDAOImpl) GetDeletedSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error) {
	var session models.Session
	result := db.GormDB.Unscoped().Where("send_id = ? AND receive_id = ? AND deleted_at IS NOT NULL", sendId, receiveId).First(&session)
	return session, result.Error
}

// RestoreSession 恢复已删除的会话
func (d *SessionDAOImpl) RestoreSession(sessionId string) error {
	result := db.GormDB.Unscoped().Model(&models.Session{}).Where("uuid = ?", sessionId).Update("deleted_at", nil)
	return result.Error
}

// CreateSession 创建会话
func (d *SessionDAOImpl) CreateSession(session *models.Session) error {
	result := db.GormDB.Create(session)
	return result.Error
}

// DeleteSession 删除会话
func (d *SessionDAOImpl) DeleteSession(sessionId string) error {
	result := db.GormDB.Delete(&models.Session{}, "uuid = ?", sessionId)
	return result.Error
}

// GetSessionListBySendId 根据发送者ID获取用户会话列表
func (d *SessionDAOImpl) GetSessionListBySendId(userId string) ([]models.Session, error) {
	var sessionList []models.Session
	result := db.GormDB.Order("created_at DESC").Where("send_id = ?", userId).Find(&sessionList)
	return sessionList, result.Error
}

// GetSessionByUuid 根据UUID获取会话
func (d *SessionDAOImpl) GetSessionByUuid(uuid string) (models.Session, error) {
	var session models.Session
	result := db.GormDB.Where("uuid = ?", uuid).First(&session)
	return session, result.Error
}

// UpdateSessionLastMessage 根据会话ID更新最后消息
func (d *SessionDAOImpl) UpdateSessionLastMessage(sessionId string, lastMessage string, lastMessageAt time.Time) error {
	return db.GormDB.Model(&models.Session{}).Where("uuid = ?", sessionId).Updates(map[string]interface{}{
		"last_message":    lastMessage,
		"last_message_at": lastMessageAt,
	}).Error
}

// UpdateSessionLastMessageByReceiveId 根据接收者ID更新最后消息（用于群聊）
func (d *SessionDAOImpl) UpdateSessionLastMessageByReceiveId(receiveId string, lastMessage string, lastMessageAt time.Time) error {
	return db.GormDB.Model(&models.Session{}).Where("receive_id = ?", receiveId).Updates(map[string]interface{}{
		"last_message":    lastMessage,
		"last_message_at": lastMessageAt,
	}).Error
}

// CreateAISession 创建AI会话（receive_id以'A'开头标识AI）
func (d *SessionDAOImpl) CreateAISession(userId string, modelType string) (*models.Session, error) {
	sessionId := "S" + util.GetNowAndLenRandomString(11)
	aiId := "A" + util.GetNowAndLenRandomString(11)

	session := &models.Session{
		Uuid:        sessionId,
		SendId:      userId,
		ReceiveId:   aiId,
		LastMessage: modelType,
		CreatedAt:   time.Now(),
	}

	result := db.GormDB.Create(session)
	if result.Error != nil {
		return nil, result.Error
	}

	return session, nil
}

// GetAISessionList 获取用户的AI会话列表（按最后消息时间倒序）
func (d *SessionDAOImpl) GetAISessionList(userId string) ([]models.Session, error) {
	var sessions []models.Session
	result := db.GormDB.Where("send_id = ? AND receive_id LIKE ?", userId, "A%").
		Order("last_message_at DESC, created_at DESC").
		Find(&sessions)
	return sessions, result.Error
}

// GetAISessionByUuid 根据UUID获取AI会话（校验receive_id以'A'开头）
func (d *SessionDAOImpl) GetAISessionByUuid(uuid string) (*models.Session, error) {
	var session models.Session
	result := db.GormDB.Where("uuid = ? AND receive_id LIKE ?", uuid, "A%").First(&session)
	if result.Error != nil {
		return nil, result.Error
	}
	return &session, nil
}

// DeleteAISession 删除AI会话
func (d *SessionDAOImpl) DeleteAISession(uuid string) error {
	return db.GormDB.Where("uuid = ? AND receive_id LIKE ?", uuid, "A%").Delete(&models.Session{}).Error
}
