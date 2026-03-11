package userservice

import (
	"encoding/json"
	"fmt"
	userdao "seekF-backend/internal/dao/user_dao"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/models"
	contactstatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_status_enum"
	contacttypeenum "seekF-backend/internal/pkg/enum/contact_enum/contact_type_enum"
	groupstatusenum "seekF-backend/internal/pkg/enum/group_enum/group_status_enum"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"
	"time"
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
