package userdao

import (
	"context"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DiscoverDistributionDAO 关注事件及收件箱候选存储，业务事务由Service编排。
type DiscoverDistributionDAO struct{ db *gorm.DB }

// NewDiscoverDistributionDAO 构造同连接的关注分发DAO。
func NewDiscoverDistributionDAO(database *gorm.DB) *DiscoverDistributionDAO {
	return &DiscoverDistributionDAO{db: database}
}

// ListDue 查询关注目标的到期待投递事件。
func (d *DiscoverDistributionDAO) ListDue(limit int) ([]models.DiscoverOutbox, error) {
	var rows []models.DiscoverOutbox
	err := d.db.Where("target = ? AND status IN (0,1) AND next_attempt_at <= ?", constants.DiscoverTargetFeed, time.Now()).Order("next_attempt_at,id").Limit(limit).Find(&rows).Error
	return rows, err
}

// ListFans 一次读取发帖时已有的粉丝，限制到拉模式阈值，避免计数异常导致无界写扩散。
func (d *DiscoverDistributionDAO) ListFans(author string, publishedAt time.Time, limit int) ([]models.UserFollow, error) {
	var rows []models.UserFollow
	err := d.db.Select("user_id").Where("follow_user_id = ? AND created_at <= ?", author, publishedAt).Limit(limit).Find(&rows).Error
	return rows, err
}

// InsertInbox 幂等写入候选并返回实际新增数量。
func (d *DiscoverDistributionDAO) InsertInbox(rows []models.DiscoverFeedInbox) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	r := d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return r.RowsAffected, r.Error
}

// DeletePostInbox 清理已不可见或窗口外作品的全部候选。
func (d *DiscoverDistributionDAO) DeletePostInbox(uuid string) error {
	return d.db.Where("post_uuid = ?", uuid).Delete(&models.DiscoverFeedInbox{}).Error
}

// CleanupFeedInbox 复用收件箱存储维护，每轮有界删除以避免长期占用数据库。
func (d *DiscoverDAOImpl) CleanupFeedInbox(ctx context.Context, from time.Time, limit int) error {
	return NewDiscoverDistributionDAO(d.db.WithContext(ctx)).CleanupInbox(from, limit)
}

// CleanupInbox 有界清理窗口外、失去权限或取消关注的候选，不参与新帖分发事务。
func (d *DiscoverDistributionDAO) CleanupInbox(from time.Time, limit int) error {
	if err := d.db.Exec(`DELETE FROM discover_feed_inbox WHERE post_created_at < ? LIMIT ?`, from, limit).Error; err != nil {
		return err
	}
	if err := d.db.Exec(`DELETE FROM discover_feed_inbox WHERE NOT EXISTS (SELECT 1 FROM discover_post p JOIN user_follow f ON f.follow_user_id=p.user_id WHERE p.uuid COLLATE utf8mb4_unicode_ci=discover_feed_inbox.post_uuid AND f.user_id=discover_feed_inbox.user_id AND p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL) LIMIT ?`, limit).Error; err != nil {
		return err
	}
	return nil
}

// ListHybridFeed 合并收件箱与全部缺失候选，覆盖大作者、分发延迟、阈值变化及窗口外历史。
func (d *DiscoverDAOImpl) ListHybridFeed(ctx context.Context, userId string, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error) {
	// UNION先去重再统一排序截断；最终关联数据库当前关注与权限，不信任收件箱状态。
	// 收件箱使用unicode_ci，跨表UUID比较转换原帖子列，兼容现有不同排序规则。
	query := `SELECT p.* FROM discover_post p JOIN (
SELECT i.post_uuid FROM discover_feed_inbox i WHERE i.user_id=?
UNION
SELECT p.uuid COLLATE utf8mb4_unicode_ci FROM discover_post p WHERE p.user_id IN (SELECT follow_user_id FROM user_follow WHERE user_id=?)
 AND NOT EXISTS (SELECT 1 FROM discover_feed_inbox i WHERE i.user_id=? AND i.post_uuid=p.uuid COLLATE utf8mb4_unicode_ci)
) candidates ON candidates.post_uuid=p.uuid COLLATE utf8mb4_unicode_ci
WHERE p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL
AND EXISTS (SELECT 1 FROM user_follow f WHERE f.user_id=? AND f.follow_user_id=p.user_id)`
	query += " AND NOT EXISTS (SELECT 1 FROM user_block b WHERE b.user_id=? AND b.blocked_user_id=p.user_id COLLATE utf8mb4_unicode_ci)"
	args := []any{userId, userId, userId, userId, userId}
	if mediaType != nil {
		query += " AND p.media_type=?"
		args = append(args, *mediaType)
	}
	if !before.IsZero() {
		query += " AND (p.created_at < ? OR (p.created_at=? AND p.id<?))"
		args = append(args, before, before, beforeId)
	}
	query += " ORDER BY p.created_at DESC,p.id DESC LIMIT ?"
	args = append(args, limit)
	var rows []models.DiscoverPost
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}
