package userdao

import (
	"context"
	"time"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BlockDAO 保存和查询用户拉黑关系，事务由服务层管理。
type BlockDAO struct{ db *gorm.DB }

// BlockUserInfo 黑名单分页查询结果，不对应数据库表。
type BlockUserInfo struct {
	Uuid      string    `json:"uuid"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
}

// NewBlockDAO 构造使用指定连接的拉黑数据访问器。
func NewBlockDAO(database *gorm.DB) *BlockDAO { return &BlockDAO{db: database} }

// WithContext 派生保留当前数据库连接及请求取消信号的数据访问器。
func (d *BlockDAO) WithContext(ctx context.Context) *BlockDAO {
	return NewBlockDAO(d.db.WithContext(ctx))
}

// BlockedByAuthors 批量查询哪些候选作者拉黑了当前用户，供列表输出占位内容。
func (d *BlockDAO) BlockedByAuthors(viewer string, authors []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if len(authors) == 0 {
		return result, nil
	}
	var ids []string
	if err := d.db.Model(&models.UserBlock{}).Where("blocked_user_id = ? AND user_id IN ?", viewer, authors).Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		if id != viewer {
			result[id] = true
		}
	}
	return result, nil
}

// WithoutHiddenAuthors 仅过滤当前用户主动拉黑的作者，反向拉黑由服务层输出占位卡片。
func WithoutHiddenAuthors(query *gorm.DB, viewer, alias string) *gorm.DB {
	return query.Where("NOT EXISTS (SELECT 1 FROM user_block b WHERE b.user_id = ? AND b.blocked_user_id = "+alias+".user_id COLLATE utf8mb4_unicode_ci)", viewer)
}

// Status 查询双方各自设置的拉黑状态。
func (d *BlockDAO) Status(userId, target string) (bool, bool, error) {
	var rows []models.UserBlock
	err := d.db.Where("(user_id = ? AND blocked_user_id = ?) OR (user_id = ? AND blocked_user_id = ?)", userId, target, target, userId).Find(&rows).Error
	var blocked, blockedBy bool
	for _, row := range rows {
		if row.UserId == userId {
			blocked = true
		} else {
			blockedBy = true
		}
	}
	return blocked, blockedBy, err
}

// Set 幂等写入或删除本人设置的拉黑关系。
func (d *BlockDAO) Set(userId, target string, blocked bool) error {
	if blocked {
		return d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.UserBlock{UserId: userId, BlockedUserId: target, CreatedAt: time.Now()}).Error
	}
	return d.db.Where("user_id = ? AND blocked_user_id = ?", userId, target).Delete(&models.UserBlock{}).Error
}

// List 按拉黑时间倒序读取本人黑名单及真实总数。
func (d *BlockDAO) List(userId string, page, pageSize int) ([]BlockUserInfo, int64, error) {
	query := d.db.Table("user_block b").Joins("LEFT JOIN user_info u ON u.uuid COLLATE utf8mb4_unicode_ci = b.blocked_user_id AND u.deleted_at IS NULL").Where("b.user_id = ?", userId)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []BlockUserInfo{}
	err := query.Select("b.blocked_user_id AS uuid, COALESCE(u.nickname,'已注销用户') AS nickname, COALESCE(u.avatar,'') AS avatar, b.created_at").Order("b.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error
	return rows, total, err
}

// WithoutBlockedAuthors 在分页前过滤双向拉黑，固定别名及显式排序规则兼容已有表的不同字符排序。
func WithoutBlockedAuthors(query *gorm.DB, viewer, alias string) *gorm.DB {
	return query.Where("NOT EXISTS (SELECT 1 FROM user_block b WHERE (b.user_id = ? AND b.blocked_user_id = "+alias+".user_id COLLATE utf8mb4_unicode_ci) OR (b.user_id = "+alias+".user_id COLLATE utf8mb4_unicode_ci AND b.blocked_user_id = ?))", viewer, viewer)
}
