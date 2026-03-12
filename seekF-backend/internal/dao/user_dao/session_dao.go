package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

// UpdateSessionByReceiveId 根据接收者ID更新会话信息
func UpdateSessionByReceiveId(receiveId string, receiveName string, avatar string) error {
	result := db.GormDB.Model(&models.Session{}).Where("receive_id = ?", receiveId).Updates(map[string]interface{}{
		"receive_name": receiveName,
		"avatar":       avatar,
	})
	return result.Error
}

// RemoveSessionBySendAndReceiveId 根据发送者ID和接收者ID删除会话
func RemoveSessionBySendAndReceiveId(sendId string, receiveId string) error {
	result := db.GormDB.Where("send_id = ? AND receive_id = ?", sendId, receiveId).Delete(&models.Session{})
	return result.Error
}

// RemoveSessionsByReceiveId 批量删除指定接收ID的会话
func RemoveSessionsByReceiveId(receiveId string) error {
	result := db.GormDB.Where("receive_id = ?", receiveId).Delete(&models.Session{})
	return result.Error
}
