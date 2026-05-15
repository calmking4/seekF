package user

import (
	"net/http"
	userreq "seekF-backend/internal/dto/user/user_req"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
	userservice "seekF-backend/internal/services/user_service"

	"github.com/gin-gonic/gin"
)

type GroupController struct {
	groupService userservice.GroupService
}

func NewGroupController(groupService userservice.GroupService) *GroupController {
	return &GroupController{
		groupService: groupService,
	}
}

// CreateGroup 创建群聊
func (c *GroupController) CreateGroup(ctx *gin.Context) {
	var createGroupReq userreq.CreateGroupRequest
	if err := ctx.ShouldBindJSON(&createGroupReq); err != nil {
		zlog.Info("创建群组参数错误: " + err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}
	// 从上下文获取当前用户UUID作为群主ID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}
	// 设置群主ID为当前用户UUID
	createGroupReq.OwnerId = userUuid.(string)

	err := c.groupService.CreateGroup(&createGroupReq)
	if err != nil {
		zlog.Info("创建群组服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "创建群聊成功", nil)
}

// LoadMyGroup 获取我创建的群聊
func (c *GroupController) LoadMyGroup(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	groupList, err := c.groupService.LoadMyGroup(userUuid.(string))
	if err != nil {
		zlog.Info("获取我的群组服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取群聊成功", groupList)
}

// LoadMyJoinedGroup 获取我加入的群聊
func (c *GroupController) LoadMyJoinedGroup(ctx *gin.Context) {
	// 从上下文获取当前用户UUID
	userUuid, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	groupList, err := c.groupService.LoadMyJoinedGroup(userUuid.(string))
	if err != nil {
		zlog.Info("获取我加入的群组服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取加入群聊成功", groupList)
}

// CheckGroupAddMode 检查群聊加群方式
func (c *GroupController) CheckGroupAddMode(ctx *gin.Context) {
	var req userreq.CheckGroupAddModeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	addMode, err := c.groupService.CheckGroupAddMode(req.GroupId)
	if err != nil {
		zlog.Info("检查群聊加群方式服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "加群方式获取成功", addMode)
}

// GetGroupInfo 获取群聊详情
func (c *GroupController) GetGroupInfo(ctx *gin.Context) {
	var req userreq.GetGroupInfoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	groupInfo, err := c.groupService.GetGroupInfo(req.GroupId)
	if err != nil {
		zlog.Info("获取群组信息服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取群聊详情成功", groupInfo)
}

// UpdateGroupInfo 更新群组详情
func (c *GroupController) UpdateGroupInfo(ctx *gin.Context) {
	var req userreq.UpdateGroupInfoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户UUID
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.groupService.UpdateGroupInfo(req, userId.(string))
	if err != nil {
		zlog.Info("更新群组信息服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "更新群聊信息成功", nil)
}

// GetGroupMemberList 获取群聊成员列表
func (c *GroupController) GetGroupMemberList(ctx *gin.Context) {
	var req userreq.GetGroupMemberListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	groupMemberList, err := c.groupService.GetGroupMemberList(req.GroupId)
	if err != nil {
		zlog.Info("获取群组成员列表服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "获取群聊成员列表成功", groupMemberList)
}

// RemoveGroupMembers 移除群聊成员
func (c *GroupController) RemoveGroupMembers(ctx *gin.Context) {
	var req userreq.RemoveGroupMembersRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户UUID
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.groupService.RemoveGroupMembers(req, userId.(string))
	if err != nil {
		zlog.Info("移除群组成员服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "移除群聊成员成功", nil)
}

// EnterGroupDirectly 直接加入群聊
func (c *GroupController) EnterGroupDirectly(ctx *gin.Context) {
	var req userreq.EnterGroupDirectlyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户UUID
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.groupService.EnterGroupDirectly(req.GroupId, userId.(string))
	if err != nil {
		zlog.Info("加入群聊服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "进群成功", nil)
}

// LeaveGroup 退群
func (c *GroupController) LeaveGroup(ctx *gin.Context) {
	var req userreq.LeaveGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户UUID
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.groupService.LeaveGroup(req.GroupId, userId.(string))
	if err != nil {
		zlog.Info("退出群聊服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "退群成功", nil)
}

// DismissGroup 解散群聊
func (c *GroupController) DismissGroup(ctx *gin.Context) {
	var req userreq.DismissGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户UUID
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	err := c.groupService.DismissGroup(req.GroupId, userId.(string))
	if err != nil {
		zlog.Info("解散群聊服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "解散群聊成功", nil)
}

// SearchGroups 根据关键词搜索群组
func (c *GroupController) SearchGroups(ctx *gin.Context) {
	var req struct {
		Keyword string `json:"keyword"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		zlog.Error(err.Error())
		resp.Error(ctx, "参数绑定失败", http.StatusBadRequest)
		return
	}

	// 从上下文获取当前用户UUID
	userId, exists := ctx.Get("Uuid")
	if !exists {
		resp.Error(ctx, "无法获取用户信息", http.StatusUnauthorized)
		return
	}

	groupList, err := c.groupService.SearchGroups(req.Keyword, userId.(string))
	if err != nil {
		zlog.Info("搜索群组服务错误: " + err.Error())
		resp.Error(ctx, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Success(ctx, "搜索群组成功", groupList)
}
