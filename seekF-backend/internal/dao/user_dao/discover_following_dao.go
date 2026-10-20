package userdao

import (
	"context"
	"time"

	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

// FollowingAuthorUpdate 关注作者更新查询结果，不属于数据库表模型。
type FollowingAuthorUpdate struct {
	AuthorUuid   string     `json:"author_uuid"`
	Nickname     string     `json:"nickname"`
	Avatar       string     `json:"avatar"`
	HasUpdate    bool       `json:"has_update"`
	LatestPostId int64      `json:"-"`
	FollowId     int64      `json:"-"`
	LatestAt     *time.Time `json:"latest_at"`
}

// FollowingUpdates 查询全部关注作者，公开作品存在未查看边界时标记更新。
func (d *DiscoverDAOImpl) FollowingUpdates(ctx context.Context, userId string) ([]FollowingAuthorUpdate, error) {
	rows := make([]FollowingAuthorUpdate, 0)
	err := d.db.WithContext(ctx).Raw(`SELECT f.id AS follow_id, f.follow_user_id AS author_uuid, u.nickname, u.avatar,
COALESCE(MAX(p.id),0) AS latest_post_id, MAX(p.created_at) AS latest_at,
COALESCE(MAX(p.id),0)>f.last_seen_post_id AS has_update
FROM user_follow f JOIN user_info u ON u.uuid=f.follow_user_id AND u.deleted_at IS NULL
LEFT JOIN discover_post p ON p.user_id=f.follow_user_id AND p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL
WHERE f.user_id=? AND NOT EXISTS (SELECT 1 FROM user_block b WHERE (b.user_id=? AND b.blocked_user_id=f.follow_user_id COLLATE utf8mb4_unicode_ci) OR (b.user_id=f.follow_user_id COLLATE utf8mb4_unicode_ci AND b.blocked_user_id=?)) GROUP BY f.id,f.follow_user_id,u.nickname,u.avatar,f.last_seen_post_id
ORDER BY has_update DESC,latest_at DESC,f.id DESC`, userId, userId, userId).Scan(&rows).Error
	return rows, err
}

// ListFollowingAuthorFeed 使用当前关注关系限定作者，复用原帖时间与主键分页。
func (d *DiscoverDAOImpl) ListFollowingAuthorFeed(ctx context.Context, userId, author string, mediaType *int8, before time.Time, beforeId int64, limit int) ([]models.DiscoverPost, error) {
	rows := make([]models.DiscoverPost, 0)
	q := WithoutHiddenAuthors(d.db.WithContext(ctx), userId, "discover_post").Where("user_id=? AND status=0 AND visibility=0", author).
		Where("EXISTS (SELECT 1 FROM user_follow f WHERE f.user_id=? AND f.follow_user_id=discover_post.user_id)", userId)
	if mediaType != nil {
		q = q.Where("media_type=?", *mediaType)
	}
	if !before.IsZero() {
		q = q.Where("created_at < ? OR (created_at=? AND id<?)", before, before, beforeId)
	}
	err := q.Order("created_at DESC,id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// FollowingReadDAO 关注更新进度访问器，事务由Service管理。
type FollowingReadDAO struct{ db *gorm.DB }

// NewFollowingReadDAO 构造绑定当前业务事务的更新进度访问器。
func NewFollowingReadDAO(database *gorm.DB) *FollowingReadDAO { return &FollowingReadDAO{db: database} }

// LatestPostID 读取关注时作者已有的最大作品主键，旧作品不产生新关注提醒。
func (d *FollowingReadDAO) LatestPostID(author string) (int64, error) {
	var id int64
	err := d.db.Model(&models.DiscoverPost{}).Select("COALESCE(MAX(id),0)").Where("user_id=?", author).Scan(&id).Error
	return id, err
}

// Advance 仅前移指定关注关系的已查看边界，旧快照不覆盖新进度。
func (d *FollowingReadDAO) Advance(followId, postId int64) error {
	return d.db.Model(&models.UserFollow{}).Where("id=?", followId).UpdateColumn("last_seen_post_id", gorm.Expr("GREATEST(last_seen_post_id,?)", postId)).Error
}
