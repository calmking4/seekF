package userdao

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"seekF-backend/internal/models"
	"time"
)

// FileDAO 上传资源、归属及引用数据访问接口。
type FileDAO interface {
	// CountOwnedAssetReferences 统计作者原作品中的历史资源引用。
	CountOwnedAssetReferences(fileURL, userId string) (int64, error)
	// FindAsset 查询资源状态，不持有行锁。
	FindAsset(fileURL string) (*models.FileAsset, error)
	// CountReadableAssetReferences 统计查看者可访问作品中的资源引用。
	CountReadableAssetReferences(fileURL, userId, environment string, now time.Time) (int64, error)
	CreateAsset(asset *models.FileAsset) error
	FindAssetForUpdate(fileURL string) (*models.FileAsset, error)
	SaveAsset(asset *models.FileAsset) error
	CountAssetReferences(asset *models.FileAsset) (int64, error)
	ListExpiredAssets(before time.Time, afterId int64, limit int) ([]models.FileAsset, error)
}

// FileDAOImpl 文件资源数据访问实现。
type FileDAOImpl struct{ db *gorm.DB }

// NewFileDAO 创建绑定指定数据库或事务连接的资源 DAO。
func NewFileDAO(db *gorm.DB) FileDAO { return &FileDAOImpl{db: db} }

// FileURLHash 返回文件地址的稳定哈希。
func FileURLHash(fileURL string) string {
	sum := sha256.Sum256([]byte(fileURL))
	return hex.EncodeToString(sum[:])
}

// CreateAsset 保存已验证的上传资源归属。
func (d *FileDAOImpl) CreateAsset(asset *models.FileAsset) error {
	asset.URLHash = FileURLHash(asset.URL)
	return d.db.Create(asset).Error
}

// FindAssetForUpdate 查询并锁定资源，必须在事务中调用。
func (d *FileDAOImpl) FindAssetForUpdate(fileURL string) (*models.FileAsset, error) {
	var asset models.FileAsset
	err := d.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("url_hash = ?", FileURLHash(fileURL)).First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &asset, err
}

// SaveAsset 更新上传完成、保活或删除状态。
func (d *FileDAOImpl) SaveAsset(asset *models.FileAsset) error { return d.db.Save(asset).Error }

// CountAssetReferences 检查未删除作品的媒体和封面关联，保护仍被使用的资源。
func (d *FileDAOImpl) CountAssetReferences(asset *models.FileAsset) (int64, error) {
	var media, cover int64
	err := d.db.Table("discover_media AS m").Joins("JOIN discover_post AS p ON p.id = m.post_id AND p.deleted_at IS NULL").Where("m.asset_id = ?", asset.Id).Count(&media).Error
	if err != nil {
		return 0, err
	}
	err = d.db.Model(&models.DiscoverPost{}).Where("cover_asset_id = ?", asset.Id).Count(&cover).Error
	return media + cover, err
}

// ListExpiredAssets 查询七天未活动的发现页未删除资源，引用检查由 Service 再次完成。
func (d *FileDAOImpl) ListExpiredAssets(before time.Time, afterId int64, limit int) ([]models.FileAsset, error) {
	var assets []models.FileAsset
	err := d.db.Where("id > ? AND object_key <> '' AND (state = 3 OR (state IN (0, 1) AND updated_at < ?)) AND category IN ?", afterId, before, []string{"discover_image", "discover_video"}).Order("id ASC").Limit(limit).Find(&assets).Error
	return assets, err
}

// FindAsset 查询资源状态，不持有行锁。
func (d *FileDAOImpl) FindAsset(fileURL string) (*models.FileAsset, error) {
	var asset models.FileAsset
	err := d.db.Where("url_hash = ?", FileURLHash(fileURL)).First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &asset, err
}

// CountReadableAssetReferences 校验图文和视频的作品状态及实时权益，封面仅允许图片资源引用。
func (d *FileDAOImpl) CountReadableAssetReferences(fileURL, userId, environment string, now time.Time) (int64, error) {
	var media, cover int64
	err := WithoutBlockedAuthors(d.db.Table("discover_media AS m"), userId, "p").Joins("JOIN discover_post AS p ON p.id = m.post_id").Joins("JOIN file_asset a ON a.id = m.asset_id").
		Where("a.url_hash = ? AND a.state = 1 AND p.status = 0 AND p.deleted_at IS NULL AND (p.visibility = 0 OR p.user_id = ?)", FileURLHash(fileURL), userId).
		Where(`((m.type = 0 AND a.category = 'discover_image') OR
		(m.type = 1 AND a.category = 'discover_video')) AND
		(p.support_only = 0 OR p.user_id = ? OR EXISTS
		(SELECT 1 FROM support_entitlement e WHERE e.user_id = ? AND e.creator_id = p.user_id AND e.environment = ? AND e.expires_at > ?))`, userId, userId, environment, now).Count(&media).Error
	if err != nil {
		return 0, err
	}
	err = WithoutBlockedAuthors(d.db.Model(&models.DiscoverPost{}), userId, "discover_post").Where("cover_asset_id IN (SELECT id FROM file_asset WHERE url_hash = ? AND category = 'discover_image' AND state = 1) AND status = 0 AND (visibility = 0 OR user_id = ?)", FileURLHash(fileURL), userId).Count(&cover).Error
	return media + cover, err
}

// CountOwnedAssetReferences 仅统计作者的历史媒体，用于编辑草稿恢复。
func (d *FileDAOImpl) CountOwnedAssetReferences(fileURL, userId string) (int64, error) {
	var media, cover int64
	err := d.db.Table("discover_media AS m").Joins("JOIN discover_post AS p ON p.id = m.post_id").Joins("JOIN file_asset a ON a.id = m.asset_id").Where("a.url_hash = ? AND p.user_id = ? AND p.deleted_at IS NULL", FileURLHash(fileURL), userId).Count(&media).Error
	if err != nil {
		return 0, err
	}
	err = d.db.Model(&models.DiscoverPost{}).Where("cover_asset_id IN (SELECT id FROM file_asset WHERE url_hash = ?) AND user_id = ?", FileURLHash(fileURL), userId).Count(&cover).Error
	return media + cover, err
}
