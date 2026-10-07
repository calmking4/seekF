package userdao

import (
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

type NotificationDAO interface {
	// CreateNotification 创建通知
	CreateNotification(notification *models.Notification) error
	// ListByUserId 查询用户通知列表（分页）
	ListByUserId(userId string, page, pageSize int, types []int8, unreadOnly bool) ([]models.Notification, error)
	CountByUserId(userId string, types []int8, unreadOnly bool) (int64, error)
	CountUnreadByType(userId string) (map[int8]int64, error)
	// CountUnread 统计未读通知数
	CountUnread(userId string) (int64, error)
	// MarkAsRead 批量标记已读
	MarkAsRead(userId string, notificationIds []int64) error
	// MarkAllAsRead 全部标记已读
	MarkAllAsRead(userId string) error
	MarkCategoryAsRead(userId string, types []int8) (int64, error)
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

func (d *NotificationDAOImpl) query(userId string, types []int8, unreadOnly bool) *gorm.DB {
	query := d.db.Model(&models.Notification{}).Where("user_id = ?", userId)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if unreadOnly {
		query = query.Where("is_read = 0")
	}
	return query
}

func (d *NotificationDAOImpl) CountByUserId(userId string, types []int8, unreadOnly bool) (int64, error) {
	var count int64
	err := d.query(userId, types, unreadOnly).Count(&count).Error
	return count, err
}

func (d *NotificationDAOImpl) CountUnreadByType(userId string) (map[int8]int64, error) {
	var rows []struct {
		Type  int8
		Count int64
	}
	err := d.query(userId, nil, true).Select("type, COUNT(*) AS count").Group("type").Scan(&rows).Error
	counts := make(map[int8]int64)
	for _, row := range rows {
		counts[row.Type] = row.Count
	}
	return counts, err
}

func (d *NotificationDAOImpl) ListByUserId(userId string, page, pageSize int, types []int8, unreadOnly bool) ([]models.Notification, error) {
	var notifications []models.Notification
	offset := (page - 1) * pageSize
	result := d.query(userId, types, unreadOnly).
		Order("created_at DESC, id DESC").
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

// MarkCategoryAsRead 返回本次已读范围，避免把请求期间新到的消息误标为已读。
func (d *NotificationDAOImpl) MarkCategoryAsRead(userId string, types []int8) (int64, error) {
	var lastId int64
	if err := d.query(userId, types, false).Select("COALESCE(MAX(id), 0)").Scan(&lastId).Error; err != nil {
		return 0, err
	}
	err := d.query(userId, types, true).Where("id <= ?", lastId).Update("is_read", 1).Error
	return lastId, err
}
