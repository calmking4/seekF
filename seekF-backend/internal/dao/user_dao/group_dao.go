package userdao

import (
	"seekF-backend/internal/models"

	"gorm.io/gorm"
)

type GroupDAO interface {
	CreateGroup(group *models.GroupInfo) error
	GetGroupInfoByOwnerId(ownerId string) ([]models.GroupInfo, error)
	GetGroupInfoByUuid(uuid string) (models.GroupInfo, error)
	GetGroupsByUuids(uuids []string) ([]models.GroupInfo, error)
	UpdateGroupInfo(group *models.GroupInfo) error
	GetGroupMembersByUuid(uuid string) (models.GroupInfo, error)
	DeleteGroupByUUid(groupId string) error
	SearchGroups(keyword string) ([]models.GroupInfo, error)
}

type GroupDAOImpl struct {
	db *gorm.DB
}

func NewGroupDAO(db *gorm.DB) GroupDAO {
	return &GroupDAOImpl{db: db}
}

// CreateGroup 创建群组
func (d *GroupDAOImpl) CreateGroup(group *models.GroupInfo) error {
	result := d.db.Create(group)
	return result.Error
}

// GetGroupInfoByOwnerId 根据群组拥有者id获取群组信息
func (d *GroupDAOImpl) GetGroupInfoByOwnerId(ownerId string) ([]models.GroupInfo, error) {
	var groupList []models.GroupInfo
	result := d.db.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList)
	return groupList, result.Error
}

// GetGroupInfoByUuid 根据群组uuid获取群组详情
func (d *GroupDAOImpl) GetGroupInfoByUuid(uuid string) (models.GroupInfo, error) {
	var group models.GroupInfo
	result := d.db.Unscoped().Where("uuid = ?", uuid).Find(&group)
	return group, result.Error
}

// GetGroupsByUuids 根据 UUID 列表批量查找群组
func (d *GroupDAOImpl) GetGroupsByUuids(uuids []string) ([]models.GroupInfo, error) {
	if len(uuids) == 0 {
		return nil, nil
	}
	var groups []models.GroupInfo
	result := d.db.Where("uuid IN ?", uuids).Find(&groups)
	return groups, result.Error
}

// UpdateGroupInfo 更新群组详情
func (d *GroupDAOImpl) UpdateGroupInfo(group *models.GroupInfo) error {
	result := d.db.Save(group)
	return result.Error
}

// GetGroupMembersByUuid 根据群组uuid获取群组成员
func (d *GroupDAOImpl) GetGroupMembersByUuid(uuid string) (models.GroupInfo, error) {
	var group models.GroupInfo
	result := d.db.Where("uuid = ?", uuid).First(&group)
	return group, result.Error
}

// DeleteGroupByUUid 根据群组uuid删除群组
func (d *GroupDAOImpl) DeleteGroupByUUid(groupId string) error {
	result := d.db.Where("uuid = ?", groupId).Delete(&models.GroupInfo{})
	return result.Error
}

// SearchGroups 根据关键词搜索群组
func (d *GroupDAOImpl) SearchGroups(keyword string) ([]models.GroupInfo, error) {
	var groupList []models.GroupInfo
	result := d.db.Where("name LIKE ?", "%"+keyword+"%").Find(&groupList)
	return groupList, result.Error
}
