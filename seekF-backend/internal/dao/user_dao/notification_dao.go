package userdao

import (
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationDAO interface {
	// CreateNotification 创建通知
	CreateNotification(notification *models.Notification) error
	// ListByUserId 查询用户通知列表（分页）
	ListByUserId(userId string, page, pageSize int) ([]models.Notification, error)
	// CountUnread 统计未读通知数
	CountUnread(userId string) (int64, error)
	// MarkAsRead 批量标记已读
	MarkAsRead(userId string, notificationIds []int64) error
	// MarkAllAsRead 全部标记已读
	MarkAllAsRead(userId string) error
}

type NotificationDAOImpl struct {
	db *gorm.DB
}

func NewNotificationDAO(db *gorm.DB) NotificationDAO {
	return &NotificationDAOImpl{db: db}
}

func (d *NotificationDAOImpl) CreateNotification(notification *models.Notification) error {
	return d.db.Create(notification).Error
}

func (d *NotificationDAOImpl) ListByUserId(userId string, page, pageSize int) ([]models.Notification, error) {
	var notifications []models.Notification
	offset := (page - 1) * pageSize
	result := d.db.Where("user_id = ?", userId).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&notifications)
	return notifications, result.Error
}

func (d *NotificationDAOImpl) CountUnread(userId string) (int64, error) {
	var count int64
	result := d.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = 0", userId).
		Count(&count)
	return count, result.Error
}

func (d *NotificationDAOImpl) MarkAsRead(userId string, notificationIds []int64) error {
	return d.db.Model(&models.Notification{}).
		Where("user_id = ? AND id IN ?", userId, notificationIds).
		Update("is_read", 1).Error
}

func (d *NotificationDAOImpl) MarkAllAsRead(userId string) error {
	return d.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = 0", userId).
		Update("is_read", 1).Error
}
