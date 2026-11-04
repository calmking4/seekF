package userservice

import (
	"context"
	"fmt"
	"sort"

	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/zlog"

	"gorm.io/gorm"
)

// BlockService 编排用户拉黑和黑名单查询。
type BlockService interface {
	// 幂等设置本人对目标用户的拉黑状态。
	Set(ctx context.Context, userId, target string, blocked bool) error
	// 查询当前用户与目标的双向拉黑状态。
	Status(ctx context.Context, userId, target string) (bool, bool, error)
	// 分页查询本人的黑名单。
	List(ctx context.Context, userId string, page, pageSize int) ([]userdao.BlockUserInfo, int64, error)
}

// BlockServiceImpl 实现拉黑关系的事务编排与列表查询。
type BlockServiceImpl struct{ blocks userdao.BlockDAO }

// NewBlockService 构造拉黑服务。
func NewBlockService(blocks userdao.BlockDAO) BlockService { return &BlockServiceImpl{blocks: blocks} }

// Set 按固定顺序锁定双方用户后幂等设置本人拉黑状态。
func (s *BlockServiceImpl) Set(ctx context.Context, userId, target string, blocked bool) error {
	if userId == target || target == "" {
		return fmt.Errorf("拉黑目标无效，不能拉黑自己")
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ids := []string{userId, target}
		sort.Strings(ids)
		users := userdao.NewUserInfoDAO(tx)
		for _, id := range ids {
			user, err := users.FindUserByUuidForUpdate(ctx, id)
			if err != nil {
				return fmt.Errorf("查询拉黑用户失败: %w", err)
			}
			// 已注销目标仍允许从黑名单移除。
			if user == nil && (blocked || id == userId) {
				return fmt.Errorf("用户不存在")
			}
		}
		if err := userdao.NewBlockDAO(tx).Set(ctx, userId, target, blocked); err != nil {
			return fmt.Errorf("设置拉黑状态失败: %w", err)
		}
		if blocked {
			if err := userdao.NewSessionDAO(tx).DeletePeerSessions(ctx, userId, target); err != nil {
				return fmt.Errorf("移除拉黑用户会话失败: %w", err)
			}
		}
		return nil
	})
}

// Status 查询当前用户与目标的双向拉黑状态。
func (s *BlockServiceImpl) Status(ctx context.Context, userId, target string) (bool, bool, error) {
	return s.blocks.Status(ctx, userId, target)
}

// List 返回本人黑名单，页大小限制为1至50条。
func (s *BlockServiceImpl) List(ctx context.Context, userId string, page, pageSize int) ([]userdao.BlockUserInfo, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	return s.blocks.List(ctx, userId, page, pageSize)
}

// checkUserBlock 使用当前业务连接检查拉黑隔离，查询失败时拒绝访问。
func checkUserBlock(ctx context.Context, database *gorm.DB, userId, target string) error {
	if userId == target {
		return nil
	}
	blocked, blockedBy, err := userdao.NewBlockDAO(database).Status(ctx, userId, target)
	if err != nil {
		return fmt.Errorf("查询拉黑状态失败: %w", err)
	}
	if blocked || blockedBy {
		return fmt.Errorf("存在拉黑关系，无法访问帖子或发送私信")
	}
	return nil
}

// canViewUnblockedDiscoverPost 使用当前业务连接复核帖子可见性及双方拉黑关系，查询失败时拒绝访问。
func canViewUnblockedDiscoverPost(ctx context.Context, database *gorm.DB, post *models.DiscoverPost, userId string) bool {
	if !CanViewDiscoverPost(post, userId) {
		return false
	}
	blocked, blockedBy, err := userdao.NewBlockDAO(database).Status(ctx, userId, post.UserId)
	if err != nil {
		zlog.Error("查询帖子拉黑权限失败，用户=" + userId + "，作者=" + post.UserId + ": " + err.Error())
		return false
	}
	return !blocked && !blockedBy
}
