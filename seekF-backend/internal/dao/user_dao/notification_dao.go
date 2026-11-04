package userdao

import (
	"context"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

// NotificationDAO 提供站内通知、未读统计和已读状态的数据访问能力。
type NotificationDAO interface {
	// 创建通知
	CreateNotification(ctx context.Context, notification *models.Notification) error
	// 查询用户通知列表（分页）
	ListByUserId(ctx context.Context, userId string, page, pageSize int, types []int8, unreadOnly bool) ([]models.Notification, error)
	// 统计指定筛选范围内的本人通知数量。
	CountByUserId(ctx context.Context, userId string, types []int8, unreadOnly bool) (int64, error)
	// 按通知类型汇总本人的未读数量。
	CountUnreadByType(ctx context.Context, userId string) (map[int8]int64, error)
	// 统计未读通知数
	CountUnread(ctx context.Context, userId string) (int64, error)
	// 批量标记已读
	MarkAsRead(ctx context.Context, userId string, notificationIds []int64) error
	// 全部标记已读
	MarkAllAsRead(ctx context.Context, userId string) error
	// 返回本次已读范围，避免把请求期间新到的消息误标为已读。
	MarkCategoryAsRead(ctx context.Context, userId string, types []int8) (int64, error)
}

// NotificationDAOImpl 使用指定数据库或事务连接访问站内通知。
type NotificationDAOImpl struct {
	db *gorm.DB
}

// NewNotificationDAO 构造绑定指定连接的站内通知数据访问器。
func NewNotificationDAO(database *gorm.DB) NotificationDAO {
	return &NotificationDAOImpl{db: database}
}

// CreateNotification 保存站内通知，事务由调用方管理。
func (d *NotificationDAOImpl) CreateNotification(ctx context.Context, notification *models.Notification) error {
	return d.db.WithContext(ctx).Create(notification).Error
}

// query 统一限定通知归属、类型和未读状态。
func (d *NotificationDAOImpl) query(ctx context.Context, userId string, types []int8, unreadOnly bool) *gorm.DB {
	query := d.db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ?", userId)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if unreadOnly {
		query = query.Where("is_read = 0")
	}
	return query
}

// CountByUserId 统计指定筛选范围内的本人通知数量。
func (d *NotificationDAOImpl) CountByUserId(ctx context.Context, userId string, types []int8, unreadOnly bool) (int64, error) {
	var count int64
	err := d.query(ctx, userId, types, unreadOnly).Count(&count).Error
	return count, err
}

// CountUnreadByType 按通知类型汇总本人的未读数量。
func (d *NotificationDAOImpl) CountUnreadByType(ctx context.Context, userId string) (map[int8]int64, error) {
	var rows []struct {
		Type  int8
		Count int64
	}
	err := d.query(ctx, userId, nil, true).Select("type, COUNT(*) AS count").Group("type").Scan(&rows).Error
	counts := make(map[int8]int64)
	for _, row := range rows {
		counts[row.Type] = row.Count
	}
	return counts, err
}

// ListByUserId 按创建时间和主键倒序分页查询本人通知。
func (d *NotificationDAOImpl) ListByUserId(ctx context.Context, userId string, page, pageSize int, types []int8, unreadOnly bool) ([]models.Notification, error) {
	var notifications []models.Notification
	offset := (page - 1) * pageSize
	result := d.query(ctx, userId, types, unreadOnly).
		Order("created_at DESC, id DESC").
		Offset(offset).Limit(pageSize).
		Find(&notifications)
	return notifications, result.Error
}

// CountUnread 统计本人的全部未读通知数量。
func (d *NotificationDAOImpl) CountUnread(ctx context.Context, userId string) (int64, error) {
	var count int64
	result := d.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = 0", userId).
		Count(&count)
	return count, result.Error
}

// MarkAsRead 仅将本人指定的通知标记为已读。
func (d *NotificationDAOImpl) MarkAsRead(ctx context.Context, userId string, notificationIds []int64) error {
	return d.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND id IN ?", userId, notificationIds).
		Update("is_read", 1).Error
}

// MarkAllAsRead 将本人的全部未读通知标记为已读。
func (d *NotificationDAOImpl) MarkAllAsRead(ctx context.Context, userId string) error {
	return d.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = 0", userId).
		Update("is_read", 1).Error
}

// MarkCategoryAsRead 返回本次已读范围，避免把请求期间新到的消息误标为已读。
func (d *NotificationDAOImpl) MarkCategoryAsRead(ctx context.Context, userId string, types []int8) (int64, error) {
	var lastId int64
	if err := d.query(ctx, userId, types, false).Select("COALESCE(MAX(id), 0)").Scan(&lastId).Error; err != nil {
		return 0, err
	}
	err := d.query(ctx, userId, types, true).Where("id <= ?", lastId).Update("is_read", 1).Error
	return lastId, err
}
