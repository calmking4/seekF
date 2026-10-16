package userdao

import (
	"errors"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DiscoverOutboxDAO 负责事件及当前帖子查询，事务由Service管理。
type DiscoverOutboxDAO struct {
	db *gorm.DB
}

// NewDiscoverOutboxDAO 构造绑定指定数据库或业务事务的事件DAO。
func NewDiscoverOutboxDAO(database *gorm.DB) *DiscoverOutboxDAO {
	return &DiscoverOutboxDAO{db: database}
}

// Create 保存搜索同步及可选的新帖分发事件，目标由Service决定并使用同一事务。
func (d *DiscoverOutboxDAO) Create(post *models.DiscoverPost, distribute bool) error {
	now := time.Now()
	event := models.DiscoverOutbox{
		PostUuid:      post.Uuid,
		PostVersion:   post.Version,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := d.db.Create(&event).Error; err != nil {
		return err
	}
	if !distribute {
		return nil
	}
	// 仅Service确定的新公开帖子创建关注分发任务，编辑与历史内容由读取补查。
	event.Id = 0
	event.Target = constants.DiscoverTargetFeed
	return d.db.Create(&event).Error
}

// ListDue 返回到期的待投递或长时间未处理事件，后续仍须加锁复核。
func (d *DiscoverOutboxDAO) ListDue(limit int) ([]models.DiscoverOutbox, error) {
	var events []models.DiscoverOutbox
	err := d.db.Where("target = ? AND status IN (0,1) AND next_attempt_at <= ?", constants.DiscoverTargetSearch, time.Now()).Order("next_attempt_at ASC,id ASC").Limit(limit).Find(&events).Error
	return events, err
}

// LockEvent 在事务内锁定事件，投递器跳过其他实例正在处理的行。
func (d *DiscoverOutboxDAO) LockEvent(id int64, skipLocked bool) (*models.DiscoverOutbox, error) {
	var event models.DiscoverOutbox
	lock := clause.Locking{Strength: "UPDATE"}
	if skipLocked {
		lock.Options = "SKIP LOCKED"
	}
	err := d.db.Clauses(lock).First(&event, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}

// Save 保存事件投递、重试或处理状态。
func (d *DiscoverOutboxDAO) Save(event *models.DiscoverOutbox) error {
	return d.db.Save(event).Error
}

// CurrentPost 查询包含软删除状态的最新作品，消费者不得使用事件携带的旧正文。
func (d *DiscoverOutboxDAO) CurrentPost(uuid string) (*models.DiscoverPost, error) {
	var post models.DiscoverPost
	err := d.db.Unscoped().Where("uuid = ?", uuid).First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &post, err
}

// HasIncomplete 检查索引是否还有未完成同步，搜索期间有积压则使用当前数据库。
func (d *DiscoverOutboxDAO) HasIncomplete() (bool, error) {
	var events []models.DiscoverOutbox
	err := d.db.Select("id").Where("target = ? AND status IN (0,1)", constants.DiscoverTargetSearch).Limit(1).Find(&events).Error
	return len(events) > 0, err
}
