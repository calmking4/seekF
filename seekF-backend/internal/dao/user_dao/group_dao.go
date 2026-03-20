package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

type GroupDAO interface {
	CreateGroup(group *models.GroupInfo) error
	GetGroupInfoByOwnerId(ownerId string) ([]models.GroupInfo, error)
	GetGroupInfoByUuid(uuid string) (models.GroupInfo, error)
	UpdateGroupInfo(group *models.GroupInfo) error
	GetGroupMembersByUuid(uuid string) (models.GroupInfo, error)
	DeleteGroupByUUid(groupId string) error
	SearchGroups(keyword string) ([]models.GroupInfo, error)
}

type GroupDAOImpl struct{}

func NewGroupDAO() GroupDAO {
	return &GroupDAOImpl{}
}

// CreateGroup 创建群组
func (d *GroupDAOImpl) CreateGroup(group *models.GroupInfo) error {
	result := db.GormDB.Create(group)
	return result.Error
}

// GetGroupInfoByOwnerId 根据群组拥有者id获取群组信息
func (d *GroupDAOImpl) GetGroupInfoByOwnerId(ownerId string) ([]models.GroupInfo, error) {
	var groupList []models.GroupInfo
	result := db.GormDB.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList)
	return groupList, result.Error
}

// GetGroupInfoByUuid 根据群组uuid获取群组详情
func (d *GroupDAOImpl) GetGroupInfoByUuid(uuid string) (models.GroupInfo, error) {
	var group models.GroupInfo
	result := db.GormDB.Unscoped().Where("uuid = ?", uuid).First(&group)
	return group, result.Error
}

// UpdateGroupInfo 更新群组详情
func (d *GroupDAOImpl) UpdateGroupInfo(group *models.GroupInfo) error {
	result := db.GormDB.Save(group)
	return result.Error
}

// GetGroupMembersByUuid 根据群组uuid获取群组成员
func (d *GroupDAOImpl) GetGroupMembersByUuid(uuid string) (models.GroupInfo, error) {
	var group models.GroupInfo
	result := db.GormDB.Where("uuid = ?", uuid).First(&group)
	return group, result.Error
}

// DeleteGroupByUUid 根据群组uuid删除群组
func (d *GroupDAOImpl) DeleteGroupByUUid(groupId string) error {
	result := db.GormDB.Where("uuid = ?", groupId).Delete(&models.GroupInfo{})
	return result.Error
}

// SearchGroups 根据关键词搜索群组
func (d *GroupDAOImpl) SearchGroups(keyword string) ([]models.GroupInfo, error) {
	var groupList []models.GroupInfo
	result := db.GormDB.Where("name LIKE ?", "%"+keyword+"%").Find(&groupList)
	return groupList, result.Error
}
