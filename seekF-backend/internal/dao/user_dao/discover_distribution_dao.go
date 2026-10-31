package userdao

import (
	"context"
	"strings"
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

// InsertInbox 幂等写入候选，作品重新公开发布时同步更新收件箱排序时间。
func (d *DiscoverDistributionDAO) InsertInbox(rows []models.DiscoverFeedInbox) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	r := d.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "post_uuid"}},
		DoUpdates: clause.AssignmentColumns([]string{"post_created_at"}),
	}).Create(&rows)
	return r.RowsAffected, r.Error
}

// SetPostFeedMode 在调用方已锁定作品的事务内调整分发方式，避免异常粉丝数量导致无界分发。
func (d *DiscoverDistributionDAO) SetPostFeedMode(id int64, mode int8) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id=?", id).UpdateColumn("feed_mode", mode).Error
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
	if err := d.db.Exec(`DELETE FROM discover_feed_inbox WHERE NOT EXISTS (SELECT 1 FROM discover_post p JOIN user_follow f ON f.follow_user_id=p.user_id WHERE p.uuid COLLATE utf8mb4_unicode_ci=discover_feed_inbox.post_uuid AND f.user_id=discover_feed_inbox.user_id AND p.feed_mode=0 AND p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL AND p.created_at=discover_feed_inbox.post_created_at) LIMIT ?`, limit).Error; err != nil {
		return err
	}
	return nil
}

// ListHybridFeed 有界合并收件箱、拉取作品和新关注作者的推送作品，并补查待分发任务及窗口外历史。
func (d *DiscoverDAOImpl) ListHybridFeed(ctx context.Context, userId string, mediaType *int8, from, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error) {
	rows := make([]models.DiscoverPost, 0)
	if limit < 1 {
		return rows, nil
	}
	// 每路先完成权限和游标过滤，再各取一页候选，避免无权限数据占满候选名额。
	filters := ` WHERE f.user_id=? AND p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM user_block b WHERE
(b.user_id=? AND b.blocked_user_id=p.user_id COLLATE utf8mb4_unicode_ci)
OR (b.user_id=p.user_id COLLATE utf8mb4_unicode_ci AND b.blocked_user_id=?))`
	filterArgs := []any{userId, userId, userId}
	if mediaType != nil {
		filters += " AND p.media_type=?"
		filterArgs = append(filterArgs, *mediaType)
	}
	if !before.IsZero() {
		filters += " AND (p.created_at<? OR (p.created_at=? AND p.id<?))"
		filterArgs = append(filterArgs, before, before, beforeId)
	}
	branches := make([]string, 0, 4)
	args := make([]any, 0)
	// 各路放在同一条SQL中读取同一快照，分发完成与收件箱写入切换时不会漏读。
	addBranch := func(name, source, condition string, values []any, distinct bool) {
		query := "SELECT p.id,p.created_at " + source + filters + condition
		if distinct {
			query += " GROUP BY p.id,p.created_at"
		}
		query += " ORDER BY p.created_at DESC,p.id DESC LIMIT ?"
		branches = append(branches, "SELECT * FROM ("+query+") "+name)
		args = append(args, filterArgs...)
		args = append(args, values...)
		args = append(args, limit)
	}
	addBranch("pushed", `FROM discover_feed_inbox i JOIN discover_post p ON p.uuid=CAST(i.post_uuid AS BINARY)
JOIN user_follow f ON f.follow_user_id=p.user_id`,
		" AND i.user_id=? AND p.feed_mode=? AND i.post_created_at>=? AND i.post_created_at=p.created_at", []any{userId, constants.DiscoverFeedPush, from}, false)
	addBranch("pulled", "FROM discover_post p JOIN user_follow f ON f.follow_user_id=p.user_id",
		" AND p.feed_mode=? AND p.created_at>=?", []any{constants.DiscoverFeedPull, from}, false)
	// 最近24小时关注的普通作者允许读取关注前作品，不创建历史回填任务或写入收件箱。
	addBranch("recent_followed", "FROM user_follow f JOIN discover_post p ON p.user_id=f.follow_user_id",
		" AND f.created_at>=? AND p.feed_mode=? AND p.created_at>=?",
		[]any{time.Now().Add(-constants.DiscoverRecentFollowWindow), constants.DiscoverFeedPush, from}, false)
	addBranch("pending", `FROM discover_outbox o JOIN discover_post p ON p.uuid=CAST(o.post_uuid AS BINARY)
JOIN user_follow f ON f.follow_user_id=p.user_id`,
		" AND o.target=? AND o.status IN (0,1) AND o.post_version=p.version AND p.feed_mode=? AND p.created_at>=?",
		[]any{constants.DiscoverTargetFeed, constants.DiscoverFeedPush, from}, true)
	// UNION先去重再统一取pageSize+1条；额外一条用于判断下一页，不会提前消费它。
	// 近期候选已够一页及下一页判断时，不读取窗口外历史；历史页仍沿同一游标继续。
	recentSQL := "SELECT * FROM (" + strings.Join(branches, " UNION ") + ") merged ORDER BY created_at DESC,id DESC LIMIT ?"
	args = append(args, limit)
	branches = branches[:0]
	addBranch("history", "FROM discover_post p JOIN user_follow f ON f.follow_user_id=p.user_id",
		" AND p.created_at<? AND (SELECT COUNT(*) FROM recent)<?", []any{from, limit}, false)
	query := "WITH recent AS (" + recentSQL + ") SELECT p.* FROM discover_post p JOIN (SELECT * FROM recent UNION ALL " + branches[0] + ") candidates ON candidates.id=p.id ORDER BY candidates.created_at DESC,candidates.id DESC LIMIT ?"
	args = append(args, limit)
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}
