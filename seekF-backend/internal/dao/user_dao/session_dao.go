package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/util"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SessionDAO 会话DAO接口
type SessionDAO interface {
	// 查询本人未删除私聊会话的未读数及消息边界。
	GetUnreadSessions(userId string) ([]SessionUnreadRow, error)
	// 锁定当前用户拥有的未删除会话。
	GetOwnedSessionForUpdate(userId string, sessionId string) (models.Session, error)
	// 验证阅读边界是该会话中的入站私信。
	HasIncomingMessage(userId string, peerId string, messageId int64) (bool, error)
	// 单调推进本人会话的阅读边界。
	AdvanceRead(userId string, sessionId string, messageId int64) error
	// 更新本人已锁定会话的置顶时间，nil表示取消置顶。
	SetPinnedAt(userId, sessionId string, pinnedAt *time.Time) error
	// 软删除本人对指定用户的会话并清除置顶，不影响历史消息。
	DeletePeerSessions(userId, peerId string) error
	GetSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error)
	GetDeletedSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error)
	RestoreSession(sessionId string) error
	CreateSession(session *models.Session) error
	// 软删除当前用户拥有的私聊会话，保留消息记录。
	DeleteSession(userId string, sessionId string) error
	GetSessionListBySendId(userId string) ([]models.Session, error)
	GetSessionByUuid(uuid string) (models.Session, error)
	UpdateSessionLastMessage(sessionId string, lastMessage string, lastMessageAt time.Time) error
	// AI会话相关方法
	CreateAISession(userId string, modelType string) (*models.Session, error)
	GetAISessionList(userId string) ([]models.Session, error)
	GetAISessionByUuid(uuid string) (*models.Session, error)
	DeleteAISession(uuid string) error
	UpdateSessionFirstMessage(sessionId string, firstMessage string) error
}

// SessionUnreadRow 私聊会话未读聚合结果，不作为数据库表模型。
type SessionUnreadRow struct {
	SessionId         string
	PeerId            string
	UnreadCount       int64
	LatestMessageId   int64
	LastReadMessageId int64
}

// GetUnreadSessions 在同一查询快照内统计本人私聊会话的入站消息和未读数。
func (d *SessionDAOImpl) GetUnreadSessions(userId string) ([]SessionUnreadRow, error) {
	rows := make([]SessionUnreadRow, 0)
	err := d.db.Table("session AS s").Select(`s.uuid AS session_id, s.receive_id AS peer_id,
		s.last_read_message_id, COALESCE(MAX(m.id), 0) AS latest_message_id,
		COALESCE(SUM(CASE WHEN m.id > s.last_read_message_id THEN 1 ELSE 0 END), 0) AS unread_count`).
		Joins("LEFT JOIN message AS m ON m.send_id = s.receive_id AND m.receive_id = s.send_id AND m.type IN (0, 2)").
		Where("s.send_id = ? AND s.receive_id LIKE ? AND s.deleted_at IS NULL", userId, "U%").
		Where("NOT EXISTS (SELECT 1 FROM user_block b WHERE b.user_id = ? AND b.blocked_user_id = s.receive_id COLLATE utf8mb4_unicode_ci)", userId).
		Group("s.uuid, s.receive_id, s.last_read_message_id").Scan(&rows).Error
	return rows, err
}

// GetOwnedSessionForUpdate 锁定本人会话，串行处理已读确认和删除。
func (d *SessionDAOImpl) GetOwnedSessionForUpdate(userId string, sessionId string) (models.Session, error) {
	var session models.Session
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("uuid = ? AND send_id = ?", sessionId, userId).First(&session).Error
	return session, err
}

// HasIncomingMessage 校验指定主键属于当前私聊的文字或文件入站消息。
func (d *SessionDAOImpl) HasIncomingMessage(userId string, peerId string, messageId int64) (bool, error) {
	var count int64
	err := d.db.Model(&models.Message{}).Where("id = ? AND send_id = ? AND receive_id = ? AND type IN (0, 2)", messageId, peerId, userId).Count(&count).Error
	return count > 0, err
}

// AdvanceRead 通过条件更新保证重复或乱序的确认不会使阅读进度倒退。
func (d *SessionDAOImpl) AdvanceRead(userId string, sessionId string, messageId int64) error {
	return d.db.Model(&models.Session{}).Where("uuid = ? AND send_id = ? AND last_read_message_id < ?", sessionId, userId, messageId).Update("last_read_message_id", messageId).Error
}

type SessionDAOImpl struct {
	db *gorm.DB
}

func NewSessionDAO(db *gorm.DB) SessionDAO {
	return &SessionDAOImpl{db: db}
}

// GetSessionBySendAndReceiveId 根据发送者ID和接收者ID获取会话
func (d *SessionDAOImpl) GetSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error) {
	var session models.Session
	result := d.db.Where("send_id = ? and receive_id = ?", sendId, receiveId).First(&session)
	return session, result.Error
}

// GetDeletedSessionBySendAndReceiveId 获取已删除的会话（软删除）
func (d *SessionDAOImpl) GetDeletedSessionBySendAndReceiveId(sendId string, receiveId string) (models.Session, error) {
	var session models.Session
	result := d.db.Unscoped().Where("send_id = ? AND receive_id = ? AND deleted_at IS NOT NULL", sendId, receiveId).First(&session)
	return session, result.Error
}

// RestoreSession 恢复已删除的会话
func (d *SessionDAOImpl) RestoreSession(sessionId string) error {
	result := d.db.Unscoped().Model(&models.Session{}).Where("uuid = ?", sessionId).Update("deleted_at", nil)
	return result.Error
}

// CreateSession 创建会话
func (d *SessionDAOImpl) CreateSession(session *models.Session) error {
	result := d.db.Create(session)
	return result.Error
}

// DeleteSession 按归属软删除私聊会话，重复删除保持幂等。
func (d *SessionDAOImpl) DeleteSession(userId string, sessionId string) error {
	return d.db.Model(&models.Session{}).Where("send_id = ? AND receive_id LIKE ? AND uuid = ?", userId, "U%", sessionId).
		Updates(map[string]any{"deleted_at": time.Now(), "pinned_at": nil}).Error
}

// SetPinnedAt 更新事务内已锁定的本人会话置顶状态。
func (d *SessionDAOImpl) SetPinnedAt(userId, sessionId string, pinnedAt *time.Time) error {
	return d.db.Model(&models.Session{}).Where("send_id = ? AND uuid = ?", userId, sessionId).Update("pinned_at", pinnedAt).Error
}

// DeletePeerSessions 拉黑时软删除本人对应会话并清除置顶，重复执行保持幂等。
func (d *SessionDAOImpl) DeletePeerSessions(userId, peerId string) error {
	return d.db.Model(&models.Session{}).Where("send_id = ? AND receive_id = ?", userId, peerId).
		Updates(map[string]any{"deleted_at": time.Now(), "pinned_at": nil}).Error
}

// GetSessionListBySendId 查询本人会话并过滤主动拉黑目标，置顶优先后按最近活动排序。
func (d *SessionDAOImpl) GetSessionListBySendId(userId string) ([]models.Session, error) {
	var sessionList []models.Session
	result := d.db.Order("pinned_at DESC, COALESCE(last_message_at, created_at) DESC, id DESC").Where("send_id = ?", userId).
		Where("NOT EXISTS (SELECT 1 FROM user_block b WHERE b.user_id = ? AND b.blocked_user_id = session.receive_id COLLATE utf8mb4_unicode_ci)", userId).Find(&sessionList)
	return sessionList, result.Error
}

// GetSessionByUuid 根据UUID获取会话
func (d *SessionDAOImpl) GetSessionByUuid(uuid string) (models.Session, error) {
	var session models.Session
	result := d.db.Where("uuid = ?", uuid).First(&session)
	return session, result.Error
}

// UpdateSessionLastMessage 根据会话ID更新最后消息
func (d *SessionDAOImpl) UpdateSessionLastMessage(sessionId string, lastMessage string, lastMessageAt time.Time) error {
	return d.db.Model(&models.Session{}).Where("uuid = ?", sessionId).Updates(map[string]interface{}{
		"last_message":    lastMessage,
		"last_message_at": lastMessageAt,
	}).Error
}

// CreateAISession 创建AI会话（receive_id以'A'开头标识AI）
func (d *SessionDAOImpl) CreateAISession(userId string, modelType string) (*models.Session, error) {
	sessionId := "S" + util.GetNowAndLenRandomString(11)
	aiId := "A" + util.GetNowAndLenRandomString(11)

	session := &models.Session{
		Uuid:      sessionId,
		SendId:    userId,
		ReceiveId: aiId,
		CreatedAt: time.Now(),
	}

	result := d.db.Create(session)
	if result.Error != nil {
		return nil, result.Error
	}

	return session, nil
}

// GetAISessionList 获取用户的AI会话列表（按创建时间倒序）
func (d *SessionDAOImpl) GetAISessionList(userId string) ([]models.Session, error) {
	var sessions []models.Session
	result := d.db.Where("send_id = ? AND receive_id LIKE ?", userId, "A%").
		Order("created_at DESC").
		Find(&sessions)
	return sessions, result.Error
}

// GetAISessionByUuid 根据UUID获取AI会话（校验receive_id以'A'开头）
func (d *SessionDAOImpl) GetAISessionByUuid(uuid string) (*models.Session, error) {
	var session models.Session
	result := d.db.Where("uuid = ? AND receive_id LIKE ?", uuid, "A%").First(&session)
	if result.Error != nil {
		return nil, result.Error
	}
	return &session, nil
}

// DeleteAISession 删除AI会话
func (d *SessionDAOImpl) DeleteAISession(uuid string) error {
	return d.db.Where("uuid = ? AND receive_id LIKE ?", uuid, "A%").Delete(&models.Session{}).Error
}

// UpdateSessionFirstMessage 更新会话的第一条消息
func (d *SessionDAOImpl) UpdateSessionFirstMessage(sessionId string, firstMessage string) error {
	return d.db.Model(&models.Session{}).Where("uuid = ? AND first_message = ''", sessionId).Updates(map[string]interface{}{
		"first_message": firstMessage,
	}).Error
}
