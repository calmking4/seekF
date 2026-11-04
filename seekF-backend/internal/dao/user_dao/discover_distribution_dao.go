package userdao

import (
	"context"
	"strings"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"gorm.io/gorm"
)

// DiscoverDistributionDAO 查询关注分发事件与粉丝，业务事务由Service管理。
type DiscoverDistributionDAO struct{ db *gorm.DB }

// NewDiscoverDistributionDAO 构造绑定调用方连接的关注分发DAO。
func NewDiscoverDistributionDAO(database *gorm.DB) *DiscoverDistributionDAO {
	return &DiscoverDistributionDAO{db: database}
}

// ListDue 查询到期待投递的关注分发事件。
func (d *DiscoverDistributionDAO) ListDue(limit int) ([]models.DiscoverOutbox, error) {
	var rows []models.DiscoverOutbox
	err := d.db.Where("target = ? AND status IN (0,1) AND next_attempt_at <= ?", constants.DiscoverTargetFeed, time.Now()).Order("next_attempt_at,id").Limit(limit).Find(&rows).Error
	return rows, err
}

// ListFans 只读取发布时已有的粉丝，并限制人数以避免无界写扩散。
func (d *DiscoverDistributionDAO) ListFans(author string, publishedAt time.Time, limit int) ([]models.UserFollow, error) {
	var rows []models.UserFollow
	err := d.db.Select("user_id").Where("follow_user_id = ? AND created_at <= ?", author, publishedAt).Limit(limit).Find(&rows).Error
	return rows, err
}

// SetPostFeedMode 在调用方已锁定作品的事务内调整分发方式。
func (d *DiscoverDistributionDAO) SetPostFeedMode(id int64, mode int8) error {
	return d.db.Model(&models.DiscoverPost{}).Where("id=?", id).UpdateColumn("feed_mode", mode).Error
}

// ListInboxPosts 仅查询当前关注作者的近期推模式索引，不复制正文或缓存访问权限。
func (d *DiscoverDAOImpl) ListInboxPosts(ctx context.Context, userId string, from time.Time, limit int) ([]models.DiscoverPost, error) {
	var rows []models.DiscoverPost
	err := d.db.WithContext(ctx).Model(&models.DiscoverPost{}).Select("id,uuid,created_at").
		Where("feed_mode=? AND status=? AND visibility=0 AND created_at>=?", constants.DiscoverFeedPush, constants.DiscoverPostPublished, from).
		Where("EXISTS (SELECT 1 FROM user_follow f WHERE f.user_id=? AND f.follow_user_id=discover_post.user_id)", userId).
		Order("created_at DESC,id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// ListHybridFeed 将有界Redis候选与其他召回放入同一SQL快照，避免分发状态切换时漏读。
func (d *DiscoverDAOImpl) ListHybridFeed(ctx context.Context, userId string, inboxIDs []int64, mediaType *int8, from, floor, before time.Time, floorID, beforeId int64, limit int) ([]models.DiscoverPost, error) {
	rows := make([]models.DiscoverPost, 0)
	if limit < 1 {
		return rows, nil
	}
	filters := ` WHERE f.user_id=? AND p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM user_block b WHERE
(b.user_id=? AND b.blocked_user_id=p.user_id COLLATE utf8mb4_unicode_ci)
OR (b.user_id=p.user_id COLLATE utf8mb4_unicode_ci AND b.blocked_user_id=?))
AND p.created_at>=? AND (p.created_at>? OR (p.created_at=? AND p.id>=?))`
	filterArgs := []any{userId, userId, userId, from, floor, floor, floorID}
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
	// 每路在权限与游标过滤之后再截断，不让失效候选占用最终页名额。
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
	if len(inboxIDs) > 0 {
		addBranch("pushed", "FROM discover_post p JOIN user_follow f ON f.follow_user_id=p.user_id",
			" AND p.feed_mode=? AND p.id IN ?", []any{constants.DiscoverFeedPush, inboxIDs}, false)
	}
	addBranch("pulled", "FROM discover_post p JOIN user_follow f ON f.follow_user_id=p.user_id",
		" AND p.feed_mode=?", []any{constants.DiscoverFeedPull}, false)
	addBranch("recent_followed", "FROM user_follow f JOIN discover_post p ON p.user_id=f.follow_user_id",
		" AND f.created_at>=? AND p.feed_mode=?", []any{time.Now().Add(-constants.DiscoverRecentFollowWindow), constants.DiscoverFeedPush}, false)
	addBranch("pending", `FROM discover_outbox o JOIN discover_post p ON p.uuid=CAST(o.post_uuid AS BINARY)
JOIN user_follow f ON f.follow_user_id=p.user_id`,
		" AND o.target=? AND o.status IN (0,1) AND o.post_version=p.version AND p.feed_mode=?",
		[]any{constants.DiscoverTargetFeed, constants.DiscoverFeedPush}, true)
	// 缓存窗口不足一页时由Service全量拉取，不把缓存末尾误当作历史结束。
	query := "SELECT p.* FROM discover_post p JOIN (SELECT * FROM (" + strings.Join(branches, " UNION ") + ") merged ORDER BY created_at DESC,id DESC LIMIT ?) candidates ON candidates.id=p.id ORDER BY candidates.created_at DESC,candidates.id DESC"
	args = append(args, limit)
	err := d.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}
