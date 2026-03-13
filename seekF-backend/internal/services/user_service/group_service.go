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

type GroupService interface {
	CreateGroup(req *userreq.CreateGroupRequest) error
	LoadMyGroup(ownerId string) ([]userresp.LoadMyGroupRespond, error)
	LoadMyJoinedGroup(userId string) ([]userresp.LoadMyJoinedGroupRespond, error)
	CheckGroupAddMode(groupId string) (int8, error)
	GetGroupInfo(groupId string) (userresp.GetGroupInfoRespond, error)
	UpdateGroupInfo(req userreq.UpdateGroupInfoRequest, userId string) error
	GetGroupMemberList(groupId string) ([]userresp.GetGroupMemberListRespond, error)
	RemoveGroupMembers(req userreq.RemoveGroupMembersRequest, userId string) error
	EnterGroupDirectly(groupId string, userId string) error
	LeaveGroup(groupId string, userId string) error
	DismissGroup(groupId string, userId string) error
}

type GroupServiceImpl struct {
	groupDAO    userdao.GroupDAO
	contactDAO  userdao.ContactDAO
	sessionDAO  userdao.SessionDAO
	userInfoDAO userdao.UserInfoDAO
}

func NewGroupService(
	groupDAO userdao.GroupDAO,
	contactDAO userdao.ContactDAO,
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
) GroupService {
	return &GroupServiceImpl{
		groupDAO:    groupDAO,
		contactDAO:  contactDAO,
		sessionDAO:  sessionDAO,
		userInfoDAO: userInfoDAO,
	}
}

// CreateGroup 创建群聊
func (s *GroupServiceImpl) CreateGroup(req *userreq.CreateGroupRequest) error {
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
	if err := s.groupDAO.CreateGroup(group); err != nil {
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

	if err := s.contactDAO.CreateUserContact(contact); err != nil {
		zlog.Info("CreateUserContact err: " + err.Error())
		return fmt.Errorf("添加联系人失败")
	}

	return nil
}

// LoadMyGroup 获取我创建的群聊
func (s *GroupServiceImpl) LoadMyGroup(ownerId string) ([]userresp.LoadMyGroupRespond, error) {
	rspString, err := myredis.GetKeyNilIsErr("contact_mygroup_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 使用 DAO 层方法获取群组列表
			groupList, err := s.groupDAO.GetGroupInfoByOwnerId(ownerId)
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
			// 缓存我创建的群组列表
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

// LoadMyJoinedGroup 获取我加入的群聊
func (s *GroupServiceImpl) LoadMyJoinedGroup(userId string) ([]userresp.LoadMyJoinedGroupRespond, error) {
	rspString, err := myredis.GetKeyNilIsErr("my_joined_group_list_" + userId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 使用 DAO 层方法获取用户加入的群聊联系列表
			contactList, err := s.contactDAO.GetUserJoinedGroupContactsByUserId(userId)
			if err != nil {
				zlog.Error("获取用户加入的群聊联系列表失败: " + err.Error())
				return nil, err
			}

			var groupListRsp []userresp.LoadMyJoinedGroupRespond
			for _, contact := range contactList {
				// 检查是否为群聊且不是用户自己创建的群
				if contact.ContactId[0] == 'G' {
					// 获取群聊信息
					group, err := s.groupDAO.GetGroupInfoByUuid(contact.ContactId)
					if err != nil {
						zlog.Error("获取群聊信息失败: " + err.Error())
						continue // 跳过获取不到的群聊
					}

					// 群没被删除，同时群主不是自己
					if group.OwnerId != userId && !group.DeletedAt.Valid {
						groupListRsp = append(groupListRsp, userresp.LoadMyJoinedGroupRespond{
							GroupId:   group.Uuid,
							GroupName: group.Name,
							Avatar:    group.Avatar,
						})
					}
				}
			}

			rspString, err := json.Marshal(groupListRsp)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}

			if err := myredis.SetKeyEx("my_joined_group_list_"+userId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
			}

			return groupListRsp, nil
		} else {
			zlog.Error(err.Error())
			return nil, err
		}
	}

	var rsp []userresp.LoadMyJoinedGroupRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
		return nil, err
	}

	return rsp, nil
}

// CheckGroupAddMode 检查群聊加群方式
func (s *GroupServiceImpl) CheckGroupAddMode(groupId string) (int8, error) {
	group, err := s.groupDAO.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return -1, err
	}
	return group.AddMode, nil
}

// GetGroupInfo 获取群聊详情
func (s *GroupServiceImpl) GetGroupInfo(groupId string) (userresp.GetGroupInfoRespond, error) {
	group, err := s.groupDAO.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return userresp.GetGroupInfoRespond{}, err
	}

	rsp := userresp.GetGroupInfoRespond{
		Uuid:      group.Uuid,
		Name:      group.Name,
		Notice:    group.Notice,
		Avatar:    group.Avatar,
		MemberCnt: group.MemberCnt,
		OwnerId:   group.OwnerId,
		AddMode:   group.AddMode,
		Status:    group.Status,
		IsDeleted: group.DeletedAt.Valid,
	}

	return rsp, nil
}

// UpdateGroupInfo 更新群组详情
func (s *GroupServiceImpl) UpdateGroupInfo(req userreq.UpdateGroupInfoRequest, userId string) error {
	group, err := s.groupDAO.GetGroupInfoByUuid(req.Uuid)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 检查用户是否为群主
	if group.OwnerId != userId {
		return errors.New("只有群主才能更新群组信息")
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.AddMode != -1 {
		group.AddMode = req.AddMode
	}
	if req.Notice != "" {
		group.Notice = req.Notice
	}
	if req.Avatar != "" {
		group.Avatar = req.Avatar
	}

	if err := s.groupDAO.UpdateGroupInfo(&group); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 更新会话
	if err := s.sessionDAO.UpdateSessionByReceiveId(req.Uuid, group.Name, group.Avatar); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 清除我的群组列表缓存
	if err := myredis.DelKeyIfExists("contact_mygroup_list_" + group.OwnerId); err != nil {
		zlog.Error(err.Error())
	}

	// 让所有群成员的“我加入的群列表”立即刷新（群名/头像变更）
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("解析群组成员失败: " + err.Error())
	} else {
		for _, memberId := range members {
			if err := myredis.DelKeyIfExists("my_joined_group_list_" + memberId); err != nil {
				zlog.Error(err.Error())
			}
		}
	}

	return nil
}

// GetGroupMemberList 获取群聊成员列表
func (s *GroupServiceImpl) GetGroupMemberList(groupId string) ([]userresp.GetGroupMemberListRespond, error) {
	// 获取群组信息
	group, err := s.groupDAO.GetGroupMembersByUuid(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("获取群组信息失败")
	}

	// 解析群组成员列表
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("解析群组成员失败: " + err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	var rspList []userresp.GetGroupMemberListRespond
	for _, memberId := range members {
		// 获取用户信息
		user, err := s.userInfoDAO.FindUserByUuid(memberId)
		if err != nil {
			zlog.Error("获取用户信息失败: " + err.Error())
			continue // 跳过获取不到的用户
		}

		if user != nil {
			rspList = append(rspList, userresp.GetGroupMemberListRespond{
				UserId:   user.Uuid,
				Nickname: user.Nickname,
				Avatar:   user.Avatar,
			})
		}
	}

	return rspList, nil
}

// RemoveGroupMembers 移除群聊成员
func (s *GroupServiceImpl) RemoveGroupMembers(req userreq.RemoveGroupMembersRequest, userId string) error {
	// 获取群组信息
	group, err := s.groupDAO.GetGroupInfoByUuid(req.GroupId)
	if err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("获取群组信息失败")
	}

	// 检查用户是否为群主
	if group.OwnerId != userId {
		return fmt.Errorf("只有群主才能移除群成员")
	}

	// 解析群组成员列表
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("解析群组成员失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 遍历要移除的成员
	for _, uuid := range req.UuidList {
		// 不能移除群主
		if group.OwnerId == uuid {
			return fmt.Errorf("不能移除群主")
		}

		// 从成员列表中移除指定用户
		for i, member := range members {
			if member == uuid {
				members = append(members[:i], members[i+1:]...)
				break
			}
		}

		// 更新群成员数量
		if group.MemberCnt > 0 {
			group.MemberCnt -= 1
		}

		// 删除对应的会话记录
		if err := s.sessionDAO.RemoveSessionBySendAndReceiveId(uuid, req.GroupId); err != nil {
			zlog.Error("删除会话记录失败: " + err.Error())
			return fmt.Errorf("系统错误")
		}

		// 删除对应的联系人
		if err := s.contactDAO.RemoveContact(uuid, req.GroupId); err != nil {
			zlog.Error("删除联系人记录失败: " + err.Error())
			return fmt.Errorf("系统错误")
		}

		// 删除对应的申请记录
		if err := s.contactDAO.RemoveContactApply(uuid, req.GroupId); err != nil {
			zlog.Error("删除申请记录失败: " + err.Error())
			return fmt.Errorf("系统错误")
		}
	}

	// 更新群组成员列表
	group.Members, err = json.Marshal(members)
	if err != nil {
		zlog.Error("序列化群组成员失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 保存群组信息
	if err := s.groupDAO.UpdateGroupInfo(&group); err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("更新群组信息失败")
	}

	return nil
}

// EnterGroupDirectly 直接加入群聊
func (s *GroupServiceImpl) EnterGroupDirectly(groupId string, userId string) error {
	// 获取群组信息
	group, err := s.groupDAO.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error("获取群组信息失败: " + err.Error())
		return fmt.Errorf("获取群组信息失败")
	}

	// 检查群组状态是否允许直接加入
	if group.Status != groupstatusenum.NORMAL {
		return fmt.Errorf("群组不可用")
	}

	// 检查加群方式
	if group.AddMode != 0 { // 0 表示直接加入
		return fmt.Errorf("该群不允许直接加入")
	}

	// 解析群组成员列表
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("解析群组成员失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 检查用户是否已经在群中
	for _, member := range members {
		if member == userId {
			return fmt.Errorf("已在群中")
		}
	}

	// 将用户添加到成员列表
	members = append(members, userId)
	memberBytes, err := json.Marshal(members)
	if err != nil {
		zlog.Error("序列化群组成员失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}
	group.Members = memberBytes

	// 更新群成员数量
	group.MemberCnt += 1

	// 保存群组信息
	if err := s.groupDAO.UpdateGroupInfo(&group); err != nil {
		zlog.Error("更新群组信息失败: " + err.Error())
		return fmt.Errorf("更新群组信息失败")
	}

	// 创建用户联系记录
	contact := &models.UserContact{
		UserId:      userId,
		ContactId:   groupId,
		ContactType: contacttypeenum.GROUP,
		Status:      contactstatusenum.NORMAL,
		CreatedAt:   time.Now(),
		UpdateAt:    time.Now(),
	}

	if err := s.contactDAO.CreateUserContact(contact); err != nil {
		zlog.Error("创建用户联系记录失败: " + err.Error())
		return fmt.Errorf("添加联系人失败")
	}

	return nil
}

// LeaveGroup 退群
func (s *GroupServiceImpl) LeaveGroup(groupId string, userId string) error {
	// 获取群组信息
	group, err := s.groupDAO.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error("获取群组信息失败: " + err.Error())
		return fmt.Errorf("获取群组信息失败")
	}

	// 检查用户是否在群中
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zlog.Error("解析群组成员失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	userInGroup := false
	for i, member := range members {
		if member == userId {
			members = append(members[:i], members[i+1:]...) // 从成员列表中移除用户
			userInGroup = true
			break
		}
	}

	if !userInGroup {
		return fmt.Errorf("用户不在群中")
	}

	// 如果用户是群主，则不能退群（群主只能解散群）
	if group.OwnerId == userId {
		return fmt.Errorf("群主不能退群，请先转让群主或解散群")
	}

	// 更新群组成员列表
	memberBytes, err := json.Marshal(members)
	if err != nil {
		zlog.Error("序列化群组成员失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}
	group.Members = memberBytes

	// 更新群成员数量
	if group.MemberCnt > 0 {
		group.MemberCnt -= 1
	}

	// 保存群组信息
	if err := s.groupDAO.UpdateGroupInfo(&group); err != nil {
		zlog.Error("更新群组信息失败: " + err.Error())
		return fmt.Errorf("更新群组信息失败")
	}

	// 删除会话记录
	if err := s.sessionDAO.RemoveSessionBySendAndReceiveId(userId, groupId); err != nil {
		zlog.Error("删除会话记录失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 更新用户联系记录状态并软删除
	if err := s.contactDAO.UpdateUserContactStatusAndDelete(userId, groupId, contactstatusenum.QUIT_GROUP); err != nil {
		zlog.Error("更新用户联系记录失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 删除对应的申请记录
	if err := s.contactDAO.RemoveContactApply(userId, groupId); err != nil {
		zlog.Error("删除申请记录失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	return nil
}

// DismissGroup 解散群聊
func (s *GroupServiceImpl) DismissGroup(groupId string, userId string) error {
	// 获取群组信息
	group, err := s.groupDAO.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error("获取群组信息失败: " + err.Error())
		return fmt.Errorf("获取群组信息失败")
	}

	// 检查用户是否为群主
	if group.OwnerId != userId {
		return fmt.Errorf("只有群主才能解散群聊")
	}

	// 删除群组信息
	if err := s.groupDAO.DeleteGroupByUUid(groupId); err != nil {
		zlog.Error("删除群组失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 删除所有与群组相关的会话
	if err := s.sessionDAO.RemoveSessionsByReceiveId(groupId); err != nil {
		zlog.Error("删除群组会话失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 删除所有与群组相关的用户联系
	if err := s.contactDAO.RemoveContactsByContactId(groupId); err != nil {
		zlog.Error("删除群组用户联系失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 删除群组相关的申请记录
	if err := s.contactDAO.RemoveContactAppliesByContactId(groupId); err != nil {
		zlog.Error("删除群组申请记录失败: " + err.Error())
		return fmt.Errorf("系统错误")
	}

	// 清除相关缓存
	if err := myredis.DelKeyIfExists("contact_mygroup_list_" + userId); err != nil {
		zlog.Error("删除缓存失败: " + err.Error())
	}

	return nil
}
