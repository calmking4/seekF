package models

import "time"

// DiscoverOutbox 保存与作品同事务提交的搜索及关注分发任务，不存储帖子正文。
type DiscoverOutbox struct {
	Target        int8       `gorm:"not null;default:0;index:idx_outbox_target_due,priority:1;index:idx_outbox_post_target,priority:2;comment:处理目标，0搜索同步、1关注分发"`
	Id            int64      `gorm:"column:id;index:idx_outbox_target_due,priority:4;index:idx_outbox_due,priority:3;primaryKey;autoIncrement;comment:事件自增ID"`
	PostUuid      string     `gorm:"column:post_uuid;index:idx_outbox_post_target,priority:1;type:char(20);not null;index:idx_outbox_post,priority:1;comment:待同步帖子UUID"`
	PostVersion   uint64     `gorm:"column:post_version;index:idx_outbox_post,priority:2;not null;comment:事件产生时的作品版本"`
	Status        int8       `gorm:"column:status;not null;default:0;index:idx_outbox_post_target,priority:3;index:idx_outbox_target_due,priority:2;index:idx_outbox_due,priority:1;comment:状态，0待投递、1已投递待处理、2已完成"`
	RetryCount    int        `gorm:"column:retry_count;not null;default:0;comment:投递或消费累计失败次数，单位次"`
	NextAttemptAt time.Time  `gorm:"column:next_attempt_at;type:datetime(3);not null;index:idx_outbox_target_due,priority:3;index:idx_outbox_due,priority:2;comment:下次投递或补偿时间"`
	LastError     string     `gorm:"column:last_error;type:varchar(1000);not null;default:;comment:最近一次投递或处理错误，最多1000字"`
	PublishedAt   *time.Time `gorm:"column:published_at;type:datetime(3);comment:最近一次成功投递Kafka的时间"`
	ProcessedAt   *time.Time `gorm:"column:processed_at;type:datetime(3);comment:对应目标处理完成时间"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:datetime(3);not null;comment:事件创建时间"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:datetime(3);not null;comment:事件最后更新时间"`
}

// TableName 返回发现页事件表名。
func (DiscoverOutbox) TableName() string { return "discover_outbox" }
