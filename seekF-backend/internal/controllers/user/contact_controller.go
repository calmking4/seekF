package user

import (
	"net/http"

	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type ContactController struct {
	contactService userservice.ContactService
}

func NewContactController(contactService userservice.ContactService) *ContactController {
	return &ContactController{
		contactService: contactService,
	}
}

// GetUserList 获取联系人列表
func (c *ContactController) GetUserList(ctx *gin.Context) {
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	userList, err := c.contactService.GetUserList(userUuid.(string))
	if err != nil {
		zlog.Info("GetUserList service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取用户列表成功", userList)
}

// GetContactInfo 获取联系人信息
func (c *ContactController) GetContactInfo(ctx *gin.Context) {
	var getContactInfoReq userreq.GetContactInfoRequest
	if err := ctx.ShouldBindJSON(&getContactInfoReq); err != nil {
		zlog.Info("GetContactInfo err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	contactInfo, err := c.contactService.GetContactInfo(getContactInfoReq.ContactId)
	if err != nil {
		zlog.Info("GetContactInfo service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取联系人信息成功", contactInfo)
}

// DeleteContact 删除联系人（仅包含用户）
func (c *ContactController) DeleteContact(ctx *gin.Context) {
	var deleteContactReq userreq.DeleteContactRequest
	if err := ctx.ShouldBindJSON(&deleteContactReq); err != nil {
		zlog.Info("DeleteContact err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.contactService.DeleteContact(userUuid.(string), deleteContactReq.ContactId)
	if err != nil {
		zlog.Info("DeleteContact service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "删除联系人成功", nil)
}

// ApplyContact 申请添加联系人
func (c *ContactController) ApplyContact(ctx *gin.Context) {
	var applyContactReq userreq.ApplyContactRequest
	if err := ctx.ShouldBindJSON(&applyContactReq); err != nil {
		zlog.Info("ApplyContact err: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.contactService.ApplyContact(userUuid.(string), applyContactReq.ContactId, applyContactReq.Message)
	if err != nil {
		zlog.Info("ApplyContact service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "申请成功", nil)
}

// GetNewContactList 获取新的联系人申请列表
func (c *ContactController) GetNewContactList(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	contactList, err := c.contactService.GetNewContactList(userUuid.(string))
	if err != nil {
		zlog.Info("GetNewContactList service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", contactList)
}

// PassContactApply 通过联系人申请（用户和群聊）
func (c *ContactController) PassContactApply(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var passContactApplyReq userreq.PassContactApplyRequest
	if err := ctx.BindJSON(&passContactApplyReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	var err error
	if passContactApplyReq.GroupId != "" {
		// 群聊申请
		err = c.contactService.PassContactApply(passContactApplyReq.GroupId, passContactApplyReq.ContactId, userUuid.(string))
	} else {
		// 用户申请
		err = c.contactService.PassContactApply(userUuid.(string), passContactApplyReq.ContactId, "")
	}

	if err != nil {
		zlog.Info("PassContactApply service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "通过申请成功", nil)
}

// BlackContact 拉黑联系人
func (c *ContactController) BlackContact(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var blackContactReq userreq.BlackContactRequest
	if err := ctx.BindJSON(&blackContactReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	err := c.contactService.BlackContact(userUuid.(string), blackContactReq.ContactId)
	if err != nil {
		zlog.Info("BlackContact service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "已拉黑该联系人", nil)
}

// CancelBlackContact 解除拉黑联系人
func (c *ContactController) CancelBlackContact(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var blackContactReq userreq.BlackContactRequest
	if err := ctx.BindJSON(&blackContactReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	err := c.contactService.CancelBlackContact(userUuid.(string), blackContactReq.ContactId)
	if err != nil {
		zlog.Info("CancelBlackContact service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "已解除拉黑该联系人", nil)
}

// GetApplyGroupList 获取群聊申请列表
func (c *ContactController) GetApplyGroupList(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	// 绑定请求参数
	var getApplyGroupListReq userreq.GetApplyGroupListRequest
	if err := ctx.BindJSON(&getApplyGroupListReq); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "系统错误", http.StatusBadRequest)
		return
	}

	// 调用服务层方法
	data, err := c.contactService.GetApplyGroupList(getApplyGroupListReq.GroupId, userUuid.(string))
	if err != nil {
		zlog.Info("GetApplyGroupList service err: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取成功", data)
}
