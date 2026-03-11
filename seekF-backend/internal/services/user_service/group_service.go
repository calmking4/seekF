package userservice

import (
	"encoding/json"
	"errors"
	"fmt"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	contactstatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_status_enum"
	contacttypeenum "seekF-backend/internal/pkg/enum/contact_enum/contact_type_enum"
	groupstatusenum "seekF-backend/internal/pkg/enum/group_enum/group_status_enum"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"
	"time"

	"github.com/redis/go-redis/v9"
)

// CreateGroup 创建群聊
func CreateGroup(req *userreq.CreateGroupRequest) error {
	// 生成群组UUID
	groupUUID := fmt.Sprintf("G%s", util.GetNowAndLenRandomString(11))

	// 初始化群成员列表，只包含群主
	members := []string{req.OwnerId}
	membersBytes, err := json.Marshal(members)
	if err != nil {
		zlog.Info("Marshal members err: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 构建群组信息对象
	group := &models.GroupInfo{
		Uuid:      groupUUID,
		Name:      req.Name,
		Notice:    req.Notice,
		OwnerId:   req.OwnerId,
		MemberCnt: 1,
		AddMode:   req.AddMode,
		Avatar:    req.Avatar,
		Status:    groupstatusenum.NORMAL,
		Members:   membersBytes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 创建群组
	if err := userdao.CreateGroup(group); err != nil {
		zlog.Info("CreateGroup dao err: " + err.Error())
		return fmt.Errorf("创建群聊失败")
	}

	// 添加群主到联系人列表
	contact := &models.UserContact{
		UserId:      req.OwnerId,
		ContactId:   groupUUID,
		ContactType: contacttypeenum.GROUP,
		Status:      contactstatusenum.NORMAL,
		CreatedAt:   time.Now(),
		UpdateAt:    time.Now(),
	}

	if err := userdao.CreateUserContact(contact); err != nil {
		zlog.Info("CreateUserContact err: " + err.Error())
		return fmt.Errorf("添加联系人失败")
	}

	return nil
}

// LoadMyGroup 获取我创建的群聊
func LoadMyGroup(ownerId string) ([]userresp.LoadMyGroupRespond, error) {
	rspString, err := myredis.GetKeyNilIsErr("contact_mygroup_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 使用 DAO 层方法获取群组列表
			groupList, err := userdao.GetGroupInfoByOwnerId(ownerId)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}

			var groupListRsp []userresp.LoadMyGroupRespond
			for _, group := range groupList {
				groupListRsp = append(groupListRsp, userresp.LoadMyGroupRespond{
					GroupId:   group.Uuid,
					GroupName: group.Name,
					Avatar:    group.Avatar,
				})
			}
			rspString, err := json.Marshal(groupListRsp)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}
			// 缓存群组列表
			if err := myredis.SetKeyEx("contact_mygroup_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
				return nil, err
			}
			return groupListRsp, nil
		} else {
			zlog.Error(err.Error())
			return nil, err
		}
	}
	var groupListRsp []userresp.LoadMyGroupRespond
	if err := json.Unmarshal([]byte(rspString), &groupListRsp); err != nil {
		zlog.Error(err.Error())
		return nil, err
	}
	return groupListRsp, nil
}

// CheckGroupAddMode 检查群聊加群方式
func CheckGroupAddMode(groupId string) (int8, error) {
	group, err := userdao.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return -1, err
	}
	return group.AddMode, nil
}
