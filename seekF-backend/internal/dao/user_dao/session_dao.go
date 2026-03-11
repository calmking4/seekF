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
