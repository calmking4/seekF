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

// RankingCandidate 召回结果附带近7天互动分，仅用于查询，不对应数据库表。
type RankingCandidate struct {
	models.DiscoverPost
	ActivityScore float64
}

// InterestHistory 用户互动的帖子标签及权重，不对应数据库表。
type InterestHistory struct {
	Tags   string
	Weight float64
}

// SaveClick 使用数据库唯一约束消除并发或重试产生的重复点击。
func (d *DiscoverDAOImpl) SaveClick(click *models.DiscoverClick) (bool, error) {
	result := d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(click)
	return result.RowsAffected == 1, result.Error
}

// SetFeedback 写入或删除同一用户的目标记录，重复设置不改变结果。
func (d *DiscoverDAOImpl) SetFeedback(feedback *models.DiscoverFeedback, active bool) error {
	if active {
		return d.db.Clauses(clause.OnConflict{DoNothing: true}).Create(feedback).Error
	}
	return d.db.Where("user_id = ? AND target_type = ? AND target_uuid = ?", feedback.UserId, feedback.TargetType, feedback.TargetUuid).Delete(&models.DiscoverFeedback{}).Error
}

// withoutFeedback 将帖子及作者负反馈作为当前数据库查询条件。
func withoutFeedback(query *gorm.DB, userId, alias string) *gorm.DB {
	return query.Where("NOT EXISTS (SELECT 1 FROM discover_feedback f WHERE f.user_id = ? AND ((f.target_type = 'post' AND f.target_uuid = "+alias+".uuid) OR (f.target_type = 'author' AND f.target_uuid = "+alias+".user_id)))", userId)
}

// ListRecommendationFeed 依赖数据库过滤负反馈，Redis不可用时仍保留用户选择。
func (d *DiscoverDAOImpl) ListRecommendationFeed(ctx context.Context, userId string, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error) {
	posts := []models.DiscoverPost{}
	query := withoutFeedback(d.db.WithContext(ctx).Model(&models.DiscoverPost{}).Where("status = 0 AND visibility = 0"), userId, "discover_post")
	if mediaType != nil {
		query = query.Where("media_type = ?", *mediaType)
	}
	if !before.IsZero() {
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", before, before, beforeId)
	}
	err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&posts).Error
	return posts, err
}

// FindRankingPosts 批量读取候选的当前内容，软删除由GORM过滤。
func (d *DiscoverDAOImpl) FindRankingPosts(ctx context.Context, userId string, ids []string, mediaType *int8, recommended bool) ([]models.DiscoverPost, error) {
	posts := []models.DiscoverPost{}
	if len(ids) == 0 {
		return posts, nil
	}
	query := d.db.WithContext(ctx).Model(&models.DiscoverPost{}).Where("uuid IN ? AND status = 0 AND visibility = 0", ids)
	if mediaType != nil {
		query = query.Where("media_type = ?", *mediaType)
	}
	if recommended {
		query = withoutFeedback(query, userId, "discover_post")
	}
	err := query.Find(&posts).Error
	return posts, err
}

// ListRankingCandidates 分别召回近期、热门和兴趣关注作品，作者自互动及AI评论不贡献热度。
func (d *DiscoverDAOImpl) ListRankingCandidates(ctx context.Context, userId, kind string, tags []string, mediaType *int8, now time.Time) ([]RankingCandidate, error) {
	from := now.Add(-constants.DiscoverActivityWindow)
	query := d.db.WithContext(ctx).Table("discover_post p").Where("p.status = 0 AND p.visibility = 0 AND p.deleted_at IS NULL AND p.created_at <= ?", now)
	// 各互动按用户去重，收藏跨收藏夹去重，取消后原记录不存在即不再贡献。
	query = query.Joins("LEFT JOIN (SELECT c.post_uuid, COUNT(DISTINCT c.user_id) AS n FROM discover_click c JOIN discover_post t ON t.uuid = c.post_uuid WHERE c.created_at >= ? AND c.created_at <= ? AND c.user_id <> t.user_id GROUP BY c.post_uuid) clicks ON clicks.post_uuid = p.uuid", from, now)
	query = query.Joins("LEFT JOIN (SELECT l.target_uuid, COUNT(DISTINCT l.user_id) AS n FROM discover_like l JOIN discover_post t ON t.uuid = l.target_uuid WHERE l.created_at >= ? AND l.created_at <= ? AND l.user_id <> t.user_id GROUP BY l.target_uuid) likes ON likes.target_uuid = p.uuid", from, now)
	query = query.Joins("LEFT JOIN (SELECT c.target_uuid, COUNT(DISTINCT c.user_id) AS n FROM discover_collection c JOIN discover_post t ON t.uuid = c.target_uuid WHERE c.created_at >= ? AND c.created_at <= ? AND c.user_id <> t.user_id GROUP BY c.target_uuid) collects ON collects.target_uuid = p.uuid", from, now)
	query = query.Joins("LEFT JOIN (SELECT c.post_id, COUNT(DISTINCT c.user_id) AS n FROM discover_comment c JOIN discover_post t ON t.id = c.post_id WHERE c.created_at >= ? AND c.created_at <= ? AND c.deleted_at IS NULL AND c.user_id <> t.user_id AND c.user_id <> ? GROUP BY c.post_id) comments ON comments.post_id = p.id", from, now, constants.AIAssistantUserId)
	score := "COALESCE(clicks.n,0)*? + COALESCE(likes.n,0)*? + COALESCE(collects.n,0)*? + COALESCE(comments.n,0)*?"
	query = query.Select("p.*, ("+score+") AS activity_score", constants.DiscoverClickWeight, constants.DiscoverLikeWeight, constants.DiscoverCollectWeight, constants.DiscoverCommentWeight)
	if mediaType != nil {
		query = query.Where("p.media_type = ?", *mediaType)
	}
	switch kind {
	case "hot":
		query = query.Where("COALESCE(clicks.n,0)+COALESCE(likes.n,0)+COALESCE(collects.n,0)+COALESCE(comments.n,0) > 0").Order("activity_score DESC, p.created_at DESC, p.id DESC")
	case "affinity":
		conditions := []string{"EXISTS (SELECT 1 FROM user_follow f WHERE f.user_id = ? AND f.follow_user_id = p.user_id)"}
		args := []interface{}{userId}
		for _, tag := range tags {
			conditions = append(conditions, "JSON_CONTAINS(p.tags, JSON_QUOTE(?))")
			args = append(args, tag)
		}
		query = query.Where("("+strings.Join(conditions, " OR ")+")", args...).Order("p.created_at DESC, p.id DESC")
	default:
		query = query.Order("p.created_at DESC, p.id DESC")
	}
	var candidates []RankingCandidate
	err := query.Limit(constants.DiscoverRankingPoolLimit).Scan(&candidates).Error
	return candidates, err
}

// ListInterestHistory 合并当前有效互动，已取消的赞和收藏自然退出兴趣计算。
func (d *DiscoverDAOImpl) ListInterestHistory(ctx context.Context, userId string, now time.Time) ([]InterestHistory, error) {
	from := now.Add(-constants.DiscoverInterestWindow)
	var rows []InterestHistory
	err := d.db.WithContext(ctx).Raw(`SELECT p.tags, SUM(a.weight) AS weight FROM (
SELECT post_uuid, ? AS weight FROM discover_click WHERE user_id = ? AND created_at >= ? AND created_at <= ? GROUP BY post_uuid
UNION ALL SELECT target_uuid, ? FROM discover_like WHERE user_id = ? AND created_at >= ? AND created_at <= ? GROUP BY target_uuid
UNION ALL SELECT target_uuid, ? FROM discover_collection WHERE user_id = ? AND created_at >= ? AND created_at <= ? GROUP BY target_uuid
UNION ALL SELECT p.uuid, ? FROM discover_comment c JOIN discover_post p ON p.id = c.post_id WHERE c.user_id = ? AND c.created_at >= ? AND c.created_at <= ? AND c.deleted_at IS NULL GROUP BY p.uuid
) a JOIN discover_post p ON p.uuid = a.post_uuid
WHERE p.status = 0 AND p.visibility = 0 AND p.deleted_at IS NULL AND p.user_id <> ?
AND NOT EXISTS (SELECT 1 FROM discover_feedback f WHERE f.user_id = ? AND ((f.target_type = 'post' AND f.target_uuid = p.uuid) OR (f.target_type = 'author' AND f.target_uuid = p.user_id)))
GROUP BY p.id, p.tags ORDER BY weight DESC, p.id DESC LIMIT 200`,
		constants.DiscoverClickWeight, userId, from, now, constants.DiscoverLikeWeight, userId, from, now,
		constants.DiscoverCollectWeight, userId, from, now, constants.DiscoverCommentWeight, userId, from, now, userId, userId).Scan(&rows).Error
	return rows, err
}

// ListFollowedAuthorIds 查询唯一关注作者，用于轻量推荐加分。
func (d *DiscoverDAOImpl) ListFollowedAuthorIds(ctx context.Context, userId string) ([]string, error) {
	var ids []string
	err := d.db.WithContext(ctx).Model(&models.UserFollow{}).Where("user_id = ?", userId).Distinct("follow_user_id").Pluck("follow_user_id", &ids).Error
	return ids, err
}
