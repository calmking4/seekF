package admindao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/models"
)

// AdminDAO 提供管理端只读数据访问。
type AdminDAO struct{ db *gorm.DB }

// NewAdminDAO 创建管理端数据访问对象。
func NewAdminDAO(db *gorm.DB) *AdminDAO { return &AdminDAO{db: db} }

// FindUser 按唯一标识读取用户的实时权限。
func (d *AdminDAO) FindUser(ctx context.Context, uuid string) (*models.UserInfo, error) {
	var user models.UserInfo
	err := d.db.WithContext(ctx).Select("uuid", "nickname", "avatar", "is_admin", "status").Where("uuid = ?", uuid).First(&user).Error
	return &user, err
}

// Ping 检查数据库连接可用性。
func (d *AdminDAO) Ping(ctx context.Context) error {
	db, err := d.db.DB()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}

// Overview 聚合当前业务数量及最近七个自然日的活动。
func (d *AdminDAO) Overview(ctx context.Context) (*adminresp.Overview, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -6)
	end := today.AddDate(0, 0, 1)
	result := &adminresp.Overview{GeneratedAt: now, Activity: make([]adminresp.DayActivity, 7)}
	counts := []struct {
		model  any
		target *int64
		daily  bool
	}{
		{&models.UserInfo{}, &result.Users, false},
		{&models.GroupInfo{}, &result.Groups, false},
		{&models.DiscoverPost{}, &result.Posts, false},
		{&models.Knowledge{}, &result.Knowledge, false},
		{&models.Message{}, &result.TodayMessages, true},
		{&models.UserInfo{}, &result.TodayUsers, true},
	}
	for _, item := range counts {
		query := d.db.WithContext(ctx).Model(item.model)
		if item.daily {
			query = query.Where("created_at >= ? AND created_at < ?", today, end)
		}
		if err := query.Count(item.target).Error; err != nil {
			return nil, fmt.Errorf("统计管理端业务数量失败: %w", err)
		}
	}
	for i := range result.Activity {
		result.Activity[i].Date = start.AddDate(0, 0, i).Format("2006-01-02")
	}
	for _, kind := range []string{"users", "messages", "posts"} {
		var model any
		switch kind {
		case "users":
			model = &models.UserInfo{}
		case "messages":
			model = &models.Message{}
		case "posts":
			model = &models.DiscoverPost{}
		}
		var rows []struct {
			Date  string
			Count int64
		}
		// 分组表达式与投影保持一致，兼容MySQL的ONLY_FULL_GROUP_BY模式。
		err := d.db.WithContext(ctx).Model(model).Select("DATE_FORMAT(created_at, '%Y-%m-%d') AS date, COUNT(*) AS count").Where("created_at >= ? AND created_at < ?", start, end).Group("DATE_FORMAT(created_at, '%Y-%m-%d')").Scan(&rows).Error
		if err != nil {
			return nil, fmt.Errorf("统计管理端活动趋势失败: %w", err)
		}
		for _, row := range rows {
			for i := range result.Activity {
				if result.Activity[i].Date == row.Date {
					switch kind {
					case "users":
						result.Activity[i].Users = row.Count
					case "messages":
						result.Activity[i].Messages = row.Count
					case "posts":
						result.Activity[i].Posts = row.Count
					}
				}
			}
		}
	}
	return result, nil
}
