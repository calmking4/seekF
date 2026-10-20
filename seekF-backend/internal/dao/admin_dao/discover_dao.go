package admindao

import (
	"context"
	"gorm.io/gorm"
	"seekF-backend/internal/models"
)

// DiscoverReviewRow 帖子与作者名称查询结果，不对应额外数据库字段。
type DiscoverReviewRow struct {
	models.DiscoverPost
	AuthorName     string
	ReviewCoverURL string
}

// ListDiscoverReview 按标题、作者及状态分页，计数不受选中状态影响。
func (d *AdminDAO) ListDiscoverReview(ctx context.Context, keyword string, status *int8, page int) ([]DiscoverReviewRow, int64, map[int8]int64, error) {
	query := d.db.WithContext(ctx).Table("discover_post AS p").Joins("LEFT JOIN user_info AS u ON u.uuid = p.user_id").Where("p.deleted_at IS NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(p.title LIKE ? OR u.nickname LIKE ? OR p.uuid = ? OR p.user_id = ?)", like, like, keyword, keyword)
	}
	var groups []struct {
		Status int8
		Total  int64
	}
	if err := query.Session(&gorm.Session{}).Select("p.status, COUNT(*) AS total").Group("p.status").Scan(&groups).Error; err != nil {
		return nil, 0, nil, err
	}
	counts := map[int8]int64{0: 0, 1: 0, 2: 0, 3: 0}
	var total int64
	for _, g := range groups {
		counts[g.Status] = g.Total
		if status == nil || *status == g.Status {
			total += g.Total
		}
	}
	if status != nil {
		query = query.Where("p.status = ?", *status)
	}
	rows := make([]DiscoverReviewRow, 0)
	// 历史图文未保存封面时取排序最前的图片，不能将视频文件用作图片封面。
	err := query.Select("p.*, COALESCE(u.nickname,'已删除用户') AS author_name, COALESCE(NULLIF(p.cover_url,''), (SELECT m.url FROM discover_media m WHERE m.post_id=p.id AND m.type=0 ORDER BY m.sort_order,m.id LIMIT 1),'') AS review_cover_url").Order("p.updated_at DESC,p.id DESC").Offset((page - 1) * 20).Limit(20).Scan(&rows).Error
	return rows, total, counts, err
}
