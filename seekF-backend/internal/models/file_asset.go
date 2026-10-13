package models

import "time"

// FileAsset 记录上传资源所有者和状态，0 上传中、1 可用、2 已删除、3 删除中。
type FileAsset struct {
	Id          int64     `gorm:"primaryKey;comment:自增主键"`
	UserId      string    `gorm:"type:char(20);not null;index;comment:上传资源所有者UUID"`
	URL         string    `gorm:"column:url;type:varchar(500);not null;comment:资源原始地址，不包含鉴权代理地址"`
	URLHash     string    `gorm:"column:url_hash;type:char(64);not null;uniqueIndex;comment:资源地址SHA256摘要，用于唯一约束"`
	ObjectKey   string    `gorm:"type:varchar(500);not null;comment:OSS对象键"`
	Category    string    `gorm:"type:varchar(32);not null;index:idx_asset_expiry,priority:1;comment:上传用途分类，如discover_image或discover_video"`
	UploadId    string    `gorm:"type:varchar(256);comment:OSS分片上传任务ID，普通上传为空"`
	Size        int64     `gorm:"not null;default:0;comment:文件大小，单位字节"`
	ContentType string    `gorm:"type:varchar(100);comment:文件MIME类型"`
	Width       int       `gorm:"type:int;not null;default:0;comment:文件宽度，单位像素，0表示未知"`
	Height      int       `gorm:"type:int;not null;default:0;comment:文件高度，单位像素，0表示未知"`
	Verified    bool      `gorm:"not null;default:0;comment:是否通过上传格式验证"`
	State       int8      `gorm:"not null;default:0;index:idx_asset_expiry,priority:2;comment:资源状态：0上传中，1可用，2已删除，3删除中"`
	CreatedAt   time.Time `gorm:"not null;comment:资源登记时间"`
	UpdatedAt   time.Time `gorm:"not null;index:idx_asset_expiry,priority:3;comment:最后更新时间，用于未引用资源过期判断"`
}

// TableName 返回上传资源表名。
func (FileAsset) TableName() string { return "file_asset" }
