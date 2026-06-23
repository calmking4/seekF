package userservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	userdao "seekF-backend/internal/dao/user_dao"
	userresp "seekF-backend/internal/dto/user/user_resp"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"
	contactapplystatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_apply_status_enum"
	contactstatusenum "seekF-backend/internal/pkg/enum/contact_enum/contact_status_enum"
	contacttypeenum "seekF-backend/internal/pkg/enum/contact_enum/contact_type_enum"
	groupstatusenum "seekF-backend/internal/pkg/enum/group_enum/group_status_enum"
	userstatusenum "seekF-backend/internal/pkg/enum/user_enum/user_status_enum"
	myredis "seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/util"
	"seekF-backend/internal/pkg/zlog"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ContactService interface {
	GetUserList(userId string) ([]userresp.MyUserListRespond, error)
	GetContactInfo(contactId string) (userresp.GetContactInfoRespond, error)
	DeleteContact(userId string, contactId string) error
	ApplyContact(userId string, contactId string, message string) error
	GetNewContactList(userId string) ([]userresp.NewContactListRespond, error)
	PassContactApply(id string, contactId string, currentUserId string) error
	RefuseContactApply(id string, contactId string, currentUserId string) error
	BlackApply(id string, contactId string) error
	BlackContact(userId string, contactId string) error
	CancelBlackContact(userId string, contactId string) error
	GetApplyGroupList(groupId string, currentUserId string) ([]userresp.AddGroupListRespond, error)
	SearchUsers(keyword string, userId string) ([]userresp.SearchUsersRespond, error)
	GetMyApplyList(userId string) ([]userresp.MyApplyListRespond, error)
}

type ContactServiceImpl struct {
	contactDAO      userdao.ContactDAO
	sessionDAO      userdao.SessionDAO
	userInfoDAO     userdao.UserInfoDAO
	groupDAO        userdao.GroupDAO
	contactApplyDAO userdao.ContactApplyDAO
}

func NewContactService(
	contactDAO userdao.ContactDAO,
	sessionDAO userdao.SessionDAO,
	userInfoDAO userdao.UserInfoDAO,
	groupDAO userdao.GroupDAO,
	contactApplyDAO userdao.ContactApplyDAO,
) ContactService {
	return &ContactServiceImpl{
		contactDAO:      contactDAO,
		sessionDAO:      sessionDAO,
		userInfoDAO:     userInfoDAO,
		groupDAO:        groupDAO,
		contactApplyDAO: contactApplyDAO,
	}
}

// GetUserList 获取联系人列表
func (s *ContactServiceImpl) GetUserList(userId string) ([]userresp.MyUserListRespond, error) {
	rspString, err := myredis.GetKeyNilIsErr("contact_user_list_" + userId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// dao 层获取联系人列表
			contactList, err := s.contactDAO.GetUserContactList(userId)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}

			// 收集所有用户类型的联系人ID，避免 N+1 查询
			var userContactIds []string
			for _, contact := range contactList {
				if contact.ContactType == contacttypeenum.USER {
					userContactIds = append(userContactIds, contact.ContactId)
				}
			}

			// 批量查询用户信息
			userMap := make(map[string]*models.UserInfo)
			if len(userContactIds) > 0 {
				users, err := s.userInfoDAO.FindUsersByUuids(userContactIds)
				if err != nil {
					zlog.Error("批量获取用户信息失败: " + err.Error())
					return nil, err
				}
				for i := range users {
					userMap[users[i].Uuid] = &users[i]
				}
			}

			// dto 转换
			var userListRsp []userresp.MyUserListRespond
			for _, contact := range contactList {
				// 联系人中是用户的
				if contact.ContactType == contacttypeenum.USER {
					// 从 map 中获取用户信息
					user, ok := userMap[contact.ContactId]
					if !ok || user == nil {
						continue
					}
					userListRsp = append(userListRsp, userresp.MyUserListRespond{
						UserId:   user.Uuid,
						UserName: user.Nickname,
						Avatar:   user.Avatar,
					})
				}
			}
			rspString, err := json.Marshal(userListRsp)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}
			if err := myredis.SetKeyEx("contact_user_list_"+userId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zlog.Error(err.Error())
			}
			return userListRsp, nil
		} else {
			zlog.Error(err.Error())
			return nil, err
		}
	}
	var rsp []userresp.MyUserListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
		return nil, err
	}
	return rsp, nil
}

// GetContactInfo 获取联系人信息
// 调用这个接口的前提是该联系人没有处在删除或被删除，或者该用户还在群聊中
func (s *ContactServiceImpl) GetContactInfo(contactId string) (userresp.GetContactInfoRespond, error) {
	if contactId[0] == 'G' {
		// 获取群聊信息
		group, err := s.groupDAO.GetGroupInfoByUuid(contactId)
		if err != nil {
			zlog.Error(err.Error())
			return userresp.GetContactInfoRespond{}, err
		}
		// 没被禁用
		if group.Status != groupstatusenum.DISABLE {
			return userresp.GetContactInfoRespond{
				ContactId:        group.Uuid,
				ContactName:      group.Name,
				ContactAvatar:    group.Avatar,
				ContactNotice:    group.Notice,
				ContactAddMode:   group.AddMode,
				ContactMembers:   group.Members,
				ContactMemberCnt: group.MemberCnt,
				ContactOwnerId:   group.OwnerId,
			}, nil
		} else {
			zlog.Error("该群聊处于禁用状态")
			return userresp.GetContactInfoRespond{}, fmt.Errorf("该群聊处于禁用状态")
		}
	} else {
		// 获取用户信息
		user, err := s.userInfoDAO.FindUserByUuid(contactId)
		if err != nil {
			zlog.Error(err.Error())
			return userresp.GetContactInfoRespond{}, err
		}
		if user.Status != userstatusenum.DISABLE {
			birthday := ""
			if user.Birthday.Valid {
				birthday = user.Birthday.String
			}
			return userresp.GetContactInfoRespond{
				ContactId:        user.Uuid,
				ContactName:      user.Nickname,
				ContactAvatar:    user.Avatar,
				ContactBirthday:  birthday,
				ContactEmail:     user.Email,
				ContactPhone:     user.Telephone,
				ContactGender:    user.Gender,
				ContactSignature: user.Signature,
			}, nil
		} else {
			zlog.Info("该用户处于禁用状态")
			return userresp.GetContactInfoRespond{}, fmt.Errorf("该用户处于禁用状态")
		}
	}
}

// DeleteContact 删除联系人（只包含用户）
func (s *ContactServiceImpl) DeleteContact(userId string, contactId string) error {

	err := db.GormDB.Transaction(func(tx *gorm.DB) error {

		txContactDAO := userdao.NewContactDAO(tx)
		txSessionDAO := userdao.NewSessionDAO(tx)
		txContactApplyDAO := userdao.NewContactApplyDAO(tx)

		// 将自己的联系人状态更新为删除状态
		if err := txContactDAO.UpdateUserContactStatusAndDelete(userId, contactId, contactstatusenum.DELETE); err != nil {
			zlog.Error(err.Error())
			return err
		}

		// 将对方对自己的联系人状态更新为被删除状态
		if err := txContactDAO.UpdateUserContactStatusAndDelete(contactId, userId, contactstatusenum.BE_DELETE); err != nil {
			zlog.Error(err.Error())
			return err
		}

		// 删除从自己到对方的会话记录
		if err := txSessionDAO.RemoveSessionBySendAndReceiveId(userId, contactId); err != nil {
			zlog.Error(err.Error())
			return err
		}

		// 删除从对方到自己的会话记录
		if err := txSessionDAO.RemoveSessionBySendAndReceiveId(contactId, userId); err != nil {
			zlog.Error(err.Error())
			return err
		}

		// 删除自己向对方发送的联系人申请记录
		if err := txContactApplyDAO.RemoveContactApply(userId, contactId); err != nil {
			zlog.Error(err.Error())
			return err
		}

		// 删除对方向自己发送的联系人申请记录
		if err := txContactApplyDAO.RemoveContactApply(contactId, userId); err != nil {
			zlog.Error(err.Error())
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 删除缓存中的联系人列表，以便下次获取时重新加载
	if err := myredis.DelKeyIfExists("contact_user_list_" + userId); err != nil {
		zlog.Error(err.Error())
	}

	return nil
}

// ApplyContact 申请添加联系人
func (s *ContactServiceImpl) ApplyContact(userId string, contactId string, message string) error {
	if contactId[0] == 'U' {
		// 检查用户是否存在
		user, err := s.userInfoDAO.FindUserByUuid(contactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				zlog.Error("用户不存在")
				return fmt.Errorf("用户不存在")
			} else {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}
		}

		// 检查用户状态
		if user.Status == userstatusenum.DISABLE {
			zlog.Info("用户已被禁用")
			return fmt.Errorf("用户已被禁用")
		}

		// 检查是否已存在申请记录
		contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(userId, contactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新的申请记录
				contactApply = models.ContactApply{
					Uuid:        fmt.Sprintf("A%s", util.GetNowAndLenRandomString(11)),
					UserId:      userId,
					ContactId:   contactId,
					ContactType: contacttypeenum.USER,
					Status:      contactapplystatusenum.PENDING,
					Message:     message,
					LastApplyAt: time.Now(),
				}
				if err := s.contactApplyDAO.CreateContactApply(&contactApply); err != nil {
					zlog.Error(err.Error())
					return fmt.Errorf("系统错误")
				}
			} else {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}
		}

		// 检查是否被拉黑
		if contactApply.Status == contactapplystatusenum.BLACK {
			return fmt.Errorf("对方已将你拉黑")
		}

		// 更新申请记录
		contactApply.LastApplyAt = time.Now()
		contactApply.Status = contactapplystatusenum.PENDING
		if err := s.contactApplyDAO.UpdateContactApply(&contactApply); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		return nil
	} else if contactId[0] == 'G' {
		// 检查群聊是否存在
		group, err := s.groupDAO.GetGroupInfoByUuid(contactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				zlog.Error("群聊不存在")
				return fmt.Errorf("群聊不存在")
			} else {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}
		}

		// 检查群聊状态
		if group.Status == groupstatusenum.DISABLE {
			zlog.Info("群聊已被禁用")
			return fmt.Errorf("群聊已被禁用")
		}

		// 检查是否已存在申请记录
		contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(userId, contactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新的申请记录
				contactApply = models.ContactApply{
					Uuid:        fmt.Sprintf("A%s", util.GetNowAndLenRandomString(11)),
					UserId:      userId,
					ContactId:   contactId,
					ContactType: contacttypeenum.GROUP,
					Status:      contactapplystatusenum.PENDING,
					Message:     message,
					LastApplyAt: time.Now(),
				}
				if err := s.contactApplyDAO.CreateContactApply(&contactApply); err != nil {
					zlog.Error(err.Error())
					return fmt.Errorf("系统错误")
				}
			} else {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}
		}

		// 更新申请记录
		contactApply.LastApplyAt = time.Now()
		if err := s.contactApplyDAO.UpdateContactApply(&contactApply); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		return nil
	} else {
		return fmt.Errorf("用户/群聊不存在")
	}
}

// GetNewContactList 获取新的联系人申请列表
func (s *ContactServiceImpl) GetNewContactList(userId string) ([]userresp.NewContactListRespond, error) {
	// 查询状态为 PENDING 的联系人申请
	contactApplyList, err := s.contactApplyDAO.GetPendingContactAppliesByContactId(userId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	var rsp []userresp.NewContactListRespond
	for _, contactApply := range contactApplyList {
		// 构建消息
		var message string
		if contactApply.Message == "" {
			message = "申请理由：无"
		} else {
			message = "申请理由：" + contactApply.Message
		}

		// 获取申请人信息
		user, err := s.userInfoDAO.FindUserByUuid(contactApply.UserId)
		if err != nil {
			zlog.Error(err.Error())
			return nil, fmt.Errorf("系统错误")
		}
		if user == nil {
			continue // 跳过不存在的用户
		}

		// 构建响应
		newContact := userresp.NewContactListRespond{
			ContactId:     user.Uuid,
			ContactName:   user.Nickname,
			ContactAvatar: user.Avatar,
			Message:       message,
		}
		rsp = append(rsp, newContact)
	}

	return rsp, nil
}

// PassContactApply 通过联系人申请（用户和群聊）
func (s *ContactServiceImpl) PassContactApply(id string, contactId string, currentUserId string) error {
	// 查询申请记录
	contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(contactId, id)
	if err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	if id[0] == 'U' {
		// 检查申请人状态
		user, err := s.userInfoDAO.FindUserByUuid(contactId)
		if err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}
		if user == nil {
			return fmt.Errorf("用户不存在")
		}
		if user.Status == userstatusenum.DISABLE {
			zlog.Error("用户已被禁用")
			return fmt.Errorf("用户已被禁用")
		}

		err = db.GormDB.Transaction(func(tx *gorm.DB) error {

			txContactApplyDAO := userdao.NewContactApplyDAO(tx)
			txContactDAO := userdao.NewContactDAO(tx)

			// 更新申请状态为同意
			contactApply.Status = contactapplystatusenum.AGREE
			if err := txContactApplyDAO.UpdateContactApply(&contactApply); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}

			// 创建双方的联系人关系
			newContact := models.UserContact{
				UserId:      id,
				ContactId:   contactId,
				ContactType: contacttypeenum.USER,
				Status:      contactstatusenum.NORMAL,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := txContactDAO.CreateUserContact(&newContact); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}

			anotherContact := models.UserContact{
				UserId:      contactId,
				ContactId:   id,
				ContactType: contacttypeenum.USER,
				Status:      contactstatusenum.NORMAL,
				CreatedAt:   newContact.CreatedAt,
				UpdatedAt:   newContact.UpdatedAt,
			}
			if err := txContactDAO.CreateUserContact(&anotherContact); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}

			return nil
		})
		if err != nil {
			return err
		}

		// 删除缓存
		if err := myredis.DelKeyIfExists("contact_user_list_" + id); err != nil {
			zlog.Error(err.Error())
		}
		if err := myredis.DelKeyIfExists("contact_user_list_" + contactId); err != nil {
			zlog.Error(err.Error())
		}

		return nil
	} else {
		// 群聊申请，只有群主才能通过
		group, err := s.groupDAO.GetGroupInfoByUuid(id)
		if err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		// 检查是否是群主
		if group.OwnerId != currentUserId {
			return fmt.Errorf("只有群主才能通过加群申请")
		}

		if group.Status == groupstatusenum.DISABLE {
			zlog.Error("群聊已被禁用")
			return fmt.Errorf("群聊已被禁用")
		}

		err = db.GormDB.Transaction(func(tx *gorm.DB) error {

			txContactApplyDAO := userdao.NewContactApplyDAO(tx)
			txContactDAO := userdao.NewContactDAO(tx)
			txGroupDAO := userdao.NewGroupDAO(tx)

			// 更新申请状态为同意
			contactApply.Status = contactapplystatusenum.AGREE
			if err := txContactApplyDAO.UpdateContactApply(&contactApply); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}

			// 创建用户与群聊的联系人关系
			newContact := models.UserContact{
				UserId:      contactId,
				ContactId:   id,
				ContactType: contacttypeenum.GROUP,
				Status:      contactstatusenum.NORMAL,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := txContactDAO.CreateUserContact(&newContact); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}

			// 更新群聊成员
			var members []string
			if err := json.Unmarshal([]byte(group.Members), &members); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}
			members = append(members, contactId)
			group.MemberCnt = len(members)
			membersJson, err := json.Marshal(members)
			if err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}
			group.Members = membersJson

			if err := txGroupDAO.UpdateGroupInfo(&group); err != nil {
				zlog.Error(err.Error())
				return fmt.Errorf("系统错误")
			}

			return nil
		})
		if err != nil {
			return err
		}

		// 删除缓存
		if err := myredis.DelKeyIfExists("my_joined_group_list_" + contactId); err != nil {
			zlog.Error(err.Error())
		}

		return nil
	}
}

// BlackContact 拉黑联系人
func (s *ContactServiceImpl) BlackContact(userId string, contactId string) error {

	err := db.GormDB.Transaction(func(tx *gorm.DB) error {

		txContactDAO := userdao.NewContactDAO(tx)
		txSessionDAO := userdao.NewSessionDAO(tx)

		// 将自己对联系人的状态更新为拉黑
		if err := txContactDAO.UpdateUserContactStatus(userId, contactId, contactstatusenum.BLACK); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		// 将联系人对自己的状态更新为被拉黑
		if err := txContactDAO.UpdateUserContactStatus(contactId, userId, contactstatusenum.BE_BLACK); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		// 删除从自己到对方的会话记录
		if err := txSessionDAO.RemoveSessionBySendAndReceiveId(userId, contactId); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 删除缓存
	if err := myredis.DelKeyIfExists("contact_user_list_" + userId); err != nil {
		zlog.Error(err.Error())
	}

	return nil
}

// CancelBlackContact 解除拉黑联系人
func (s *ContactServiceImpl) CancelBlackContact(userId string, contactId string) error {
	// 检查自己对联系人的状态是否为拉黑
	blackContact, err := s.contactDAO.GetUserContactByUserIdAndContactId(userId, contactId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("未拉黑该联系人，无需解除拉黑")
		}
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	if blackContact.Status != contactstatusenum.BLACK {
		return fmt.Errorf("未拉黑该联系人，无需解除拉黑")
	}

	// 检查联系人对自己的状态是否为被拉黑
	beBlackContact, err := s.contactDAO.GetUserContactByUserIdAndContactId(contactId, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("该联系人未被拉黑，无需解除拉黑")
		}
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	if beBlackContact.Status != contactstatusenum.BE_BLACK {
		return fmt.Errorf("该联系人未被拉黑，无需解除拉黑")
	}

	err = db.GormDB.Transaction(func(tx *gorm.DB) error {

		txContactDAO := userdao.NewContactDAO(tx)

		// 取消拉黑，将双方状态更新为正常
		if err := txContactDAO.UpdateUserContactStatus(userId, contactId, contactstatusenum.NORMAL); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		if err := txContactDAO.UpdateUserContactStatus(contactId, userId, contactstatusenum.NORMAL); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 删除缓存
	if err := myredis.DelKeyIfExists("contact_user_list_" + userId); err != nil {
		zlog.Error(err.Error())
	}
	if err := myredis.DelKeyIfExists("contact_user_list_" + contactId); err != nil {
		zlog.Error(err.Error())
	}

	return nil
}

// GetApplyGroupList 获取群聊申请列表
func (s *ContactServiceImpl) GetApplyGroupList(groupId string, currentUserId string) ([]userresp.AddGroupListRespond, error) {
	// 检查是否是群主
	group, err := s.groupDAO.GetGroupInfoByUuid(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	if group.OwnerId != currentUserId {
		return nil, fmt.Errorf("只有群主才能查看群聊申请列表")
	}

	// 查询状态为 PENDING 的群聊申请
	contactApplyList, err := s.contactApplyDAO.GetPendingContactAppliesByContactId(groupId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	var rsp []userresp.AddGroupListRespond
	for _, contactApply := range contactApplyList {
		// 构建消息
		var message string
		if contactApply.Message == "" {
			message = "申请理由：无"
		} else {
			message = "申请理由：" + contactApply.Message
		}

		// 获取申请人信息
		user, err := s.userInfoDAO.FindUserByUuid(contactApply.UserId)
		if err != nil {
			zlog.Error(err.Error())
			return nil, fmt.Errorf("系统错误")
		}
		if user == nil {
			continue // 跳过不存在的用户
		}

		// 构建响应
		newContact := userresp.AddGroupListRespond{
			ContactId:     user.Uuid,
			ContactName:   user.Nickname,
			ContactAvatar: user.Avatar,
			Message:       message,
		}
		rsp = append(rsp, newContact)
	}

	return rsp, nil
}

// RefuseContactApply 拒绝联系人申请(用户和群聊)
func (s *ContactServiceImpl) RefuseContactApply(id string, contactId string, currentUserId string) error {
	// 查询申请记录
	contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(contactId, id)
	if err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	if id[0] == 'U' {
		// 更新申请状态为拒绝
		contactApply.Status = contactapplystatusenum.REFUSE
		if err := s.contactApplyDAO.UpdateContactApply(&contactApply); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		return nil
	} else {
		// 群聊申请，只有群主才能拒绝
		group, err := s.groupDAO.GetGroupInfoByUuid(id)
		if err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		// 检查是否是群主
		if group.OwnerId != currentUserId {
			return fmt.Errorf("只有群主才能拒绝加群申请")
		}

		// 更新申请状态为拒绝
		contactApply.Status = contactapplystatusenum.REFUSE
		if err := s.contactApplyDAO.UpdateContactApply(&contactApply); err != nil {
			zlog.Error(err.Error())
			return fmt.Errorf("系统错误")
		}

		return nil
	}
}

// BlackApply 拉黑申请
func (s *ContactServiceImpl) BlackApply(id string, contactId string) error {
	// 查询申请记录
	contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(contactId, id)
	if err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	// 更新申请状态为拉黑
	contactApply.Status = contactapplystatusenum.BLACK
	if err := s.contactApplyDAO.UpdateContactApply(&contactApply); err != nil {
		zlog.Error(err.Error())
		return fmt.Errorf("系统错误")
	}

	return nil
}

// SearchUsers 根据关键词搜索用户
func (s *ContactServiceImpl) SearchUsers(keyword string, userId string) ([]userresp.SearchUsersRespond, error) {
	// 使用 DAO 层方法搜索用户
	userList, err := s.userInfoDAO.SearchUsers(keyword)
	if err != nil {
		zlog.Error(err.Error())
		return nil, err
	}

	var userListRsp []userresp.SearchUsersRespond
	for _, user := range userList {
		// 过滤掉当前用户自己
		if user.Uuid == userId {
			continue
		}

		userListRsp = append(userListRsp, userresp.SearchUsersRespond{
			UserId:   user.Uuid,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Phone:    user.Telephone,
			Email:    user.Email,
		})
	}

	return userListRsp, nil
}

// GetMyApplyList 获取用户自己的申请状态列表
func (s *ContactServiceImpl) GetMyApplyList(userId string) ([]userresp.MyApplyListRespond, error) {
	// 查询用户发送的所有申请记录
	sentApplyList, err := s.contactApplyDAO.GetContactAppliesByUserId(userId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	var rsp []userresp.MyApplyListRespond

	// 处理发送的申请记录
	for _, contactApply := range sentApplyList {
		var contactName, contactAvatar string
		var contactType string

		if contactApply.ContactId[0] == 'U' {
			// 获取用户信息
			user, err := s.userInfoDAO.FindUserByUuid(contactApply.ContactId)
			if err != nil {
				zlog.Error(err.Error())
				continue
			}
			if user == nil {
				continue
			}
			contactName = user.Nickname
			contactAvatar = user.Avatar
			contactType = "user"
		} else if contactApply.ContactId[0] == 'G' {
			// 获取群聊信息
			group, err := s.groupDAO.GetGroupInfoByUuid(contactApply.ContactId)
			if err != nil {
				zlog.Error(err.Error())
				continue
			}
			contactName = group.Name
			contactAvatar = group.Avatar
			contactType = "group"
		} else {
			continue
		}

		// 构建消息
		var message string
		if contactApply.Message == "" {
			message = "申请理由：无"
		} else {
			message = "申请理由：" + contactApply.Message
		}

		// 构建响应
		myApply := userresp.MyApplyListRespond{
			UserId:        contactApply.UserId,
			UserName:      "", // 发出的申请不需要额外用户名
			UserAvatar:    "", // 发出的申请不需要额外用户头像
			ContactId:     contactApply.ContactId,
			ContactName:   contactName,
			ContactAvatar: contactAvatar,
			ContactType:   contactType,
			Status:        int(contactApply.Status),
			Message:       message,
			ApplyTime:     contactApply.LastApplyAt.Format("2006-01-02 15:04:05"),
			IsReceived:    false,
		}
		rsp = append(rsp, myApply)
	}

	// 查询用户收到的好友申请（其他人向该用户发出的好友申请）
	receivedApplyList, err := s.contactApplyDAO.GetContactAppliesByContactId(userId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	// 处理收到的好友申请
	for _, contactApply := range receivedApplyList {
		var contactName, contactAvatar string
		var contactType string

		// 这些是用户申请，获取申请人信息
		user, err := s.userInfoDAO.FindUserByUuid(contactApply.UserId)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}
		if user == nil {
			continue
		}
		contactName = user.Nickname
		contactAvatar = user.Avatar
		contactType = "user"

		// 构建消息
		var message string
		if contactApply.Message == "" {
			message = "申请理由：无"
		} else {
			message = "申请理由：" + contactApply.Message
		}

		// 构建响应
		myApply := userresp.MyApplyListRespond{
			UserId:        user.Uuid,
			UserName:      user.Nickname,
			UserAvatar:    user.Avatar,
			ContactId:     contactApply.ContactId,
			ContactName:   contactName,
			ContactAvatar: contactAvatar,
			ContactType:   contactType,
			Status:        int(contactApply.Status),
			Message:       message,
			ApplyTime:     contactApply.LastApplyAt.Format("2006-01-02 15:04:05"),
			IsReceived:    true,
		}
		rsp = append(rsp, myApply)
	}

	// 查询用户创建的所有群聊
	createdGroups, err := s.groupDAO.GetGroupInfoByOwnerId(userId)
	if err != nil {
		zlog.Error(err.Error())
		return nil, fmt.Errorf("系统错误")
	}

	// 查询用户创建的群聊收到的申请
	for _, group := range createdGroups {
		groupApplyList, err := s.contactApplyDAO.GetContactAppliesByContactId(group.Uuid)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}

		for _, contactApply := range groupApplyList {
			// 获取申请人信息
			user, err := s.userInfoDAO.FindUserByUuid(contactApply.UserId)
			if err != nil {
				zlog.Error(err.Error())
				continue
			}
			if user == nil {
				continue
			}

			// 构建消息
			var message string
			if contactApply.Message == "" {
				message = "申请理由：无"
			} else {
				message = "申请理由：" + contactApply.Message
			}

			// 构建响应 - 这是用户创建的群收到的申请
			// 现在正确设置：申请人信息放在 UserId、UserName 和 UserAvatar 字段，群聊信息放在 Contact 相关字段
			myApply := userresp.MyApplyListRespond{
				UserId:        user.Uuid,     // 申请人ID
				UserName:      user.Nickname, // 申请人姓名
				UserAvatar:    user.Avatar,   // 申请人头像
				ContactId:     group.Uuid,    // 群聊ID
				ContactName:   group.Name,    // 群聊名称
				ContactAvatar: group.Avatar,  // 群聊头像
				ContactType:   "group",       // 申请类型
				Status:        int(contactApply.Status),
				Message:       message,
				ApplyTime:     contactApply.LastApplyAt.Format("2006-01-02 15:04:05"),
				IsReceived:    true, // 对于用户来说，这是收到的申请
			}
			rsp = append(rsp, myApply)
		}
	}

	return rsp, nil
}
