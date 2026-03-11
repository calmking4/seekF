package userdao

import (
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
)

// CreateGroup 创建群组
func CreateGroup(group *models.GroupInfo) error {
	result := db.GormDB.Create(group)
	return result.Error
}

// GetGroupInfoByOwnerId 根据群组拥有者id获取群组信息
func GetGroupInfoByOwnerId(ownerId string) ([]models.GroupInfo, error) {
	var groupList []models.GroupInfo
	result := db.GormDB.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList)
	return groupList, result.Error
}

// GetGroupInfoByUuid 根据群组uuid获取群组详情
func GetGroupInfoByUuid(uuid string) (models.GroupInfo, error) {
	var group models.GroupInfo
	result := db.GormDB.Unscoped().Where("uuid = ?", uuid).First(&group)
	return group, result.Error
}

// UpdateGroupInfo 更新群组详情
func UpdateGroupInfo(group *models.GroupInfo) error {
	result := db.GormDB.Save(group)
	return result.Error
}
