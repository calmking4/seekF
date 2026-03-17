package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

type SessionDAO interface {
	UpdateSessionByReceiveId(receiveId string, receiveName string, avatar string) error
	RemoveSessionBySendAndReceiveId(sendId string, receiveId string) error
	RemoveSessionsByReceiveId(receiveId string) error
	GetSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error)
	CreateSession(session *models.Session) error
	GetSessionListBySendId(userId string) ([]models.Session, error)
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

// CreateSession 创建会话
func (d *SessionDAOImpl) CreateSession(session *models.Session) error {
	result := db.GormDB.Create(session)
	return result.Error
}

// GetSessionListBySendId 根据发送者ID获取用户会话列表
func (d *SessionDAOImpl) GetSessionListBySendId(userId string) ([]models.Session, error) {
	var sessionList []models.Session
	result := db.GormDB.Order("created_at DESC").Where("send_id = ?", userId).Find(&sessionList)
	return sessionList, result.Error
}
