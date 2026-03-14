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
	GetUserList(userUuid string) ([]userresp.MyUserListRespond, error)
	GetContactInfo(contactId string) (userresp.GetContactInfoRespond, error)
	DeleteContact(ownerId string, contactId string) error
	ApplyContact(ownerId string, contactId string, message string) error
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
func (s *ContactServiceImpl) GetUserList(userUuid string) ([]userresp.MyUserListRespond, error) {
	rspString, err := myredis.GetKeyNilIsErr("contact_user_list_" + userUuid)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// dao 层获取联系人列表
			contactList, err := s.contactDAO.GetUserContactList(userUuid)
			if err != nil {
				zlog.Error(err.Error())
				return nil, err
			}

			// dto 转换
			var userListRsp []userresp.MyUserListRespond
			for _, contact := range contactList {
				// 联系人中是用户的
				if contact.ContactType == contacttypeenum.USER {
					// 获取用户信息
					user, err := s.userInfoDAO.FindUserByUuid(contact.ContactId)
					if err != nil {
						zlog.Error(err.Error())
						return nil, err
					}
					if user == nil {
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
			if err := myredis.SetKeyEx("contact_user_list_"+userUuid, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
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
			return userresp.GetContactInfoRespond{
				ContactId:        user.Uuid,
				ContactName:      user.Nickname,
				ContactAvatar:    user.Avatar,
				ContactBirthday:  user.Birthday,
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
func (s *ContactServiceImpl) DeleteContact(userUuid string, contactId string) error {
	// 将自己的联系人状态更新为删除状态
	if err := s.contactDAO.UpdateUserContactStatusAndDelete(userUuid, contactId, contactstatusenum.DELETE); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 将对方对自己的联系人状态更新为被删除状态
	if err := s.contactDAO.UpdateUserContactStatusAndDelete(contactId, userUuid, contactstatusenum.BE_DELETE); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 删除从自己到对方的会话记录
	if err := s.sessionDAO.RemoveSessionBySendAndReceiveId(userUuid, contactId); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 删除从对方到自己的会话记录
	if err := s.sessionDAO.RemoveSessionBySendAndReceiveId(contactId, userUuid); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 删除自己向对方发送的联系人申请记录
	if err := s.contactDAO.RemoveContactApply(userUuid, contactId); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 删除对方向自己发送的联系人申请记录
	if err := s.contactDAO.RemoveContactApply(contactId, userUuid); err != nil {
		zlog.Error(err.Error())
		return err
	}

	// 删除缓存中的联系人列表，以便下次获取时重新加载
	if err := myredis.DelKeyIfExists("contact_user_list_" + userUuid); err != nil {
		zlog.Error(err.Error())
	}

	return nil
}

// ApplyContact 申请添加联系人
func (s *ContactServiceImpl) ApplyContact(ownerId string, contactId string, message string) error {
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
		contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(ownerId, contactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新的申请记录
				contactApply = models.ContactApply{
					Uuid:        fmt.Sprintf("A%s", util.GetNowAndLenRandomString(11)),
					UserId:      ownerId,
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
		contactApply, err := s.contactApplyDAO.GetContactApplyByUserIdAndContactId(ownerId, contactId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新的申请记录
				contactApply = models.ContactApply{
					Uuid:        fmt.Sprintf("A%s", util.GetNowAndLenRandomString(11)),
					UserId:      ownerId,
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
