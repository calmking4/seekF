package userservice

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"seekF-backend/internal/configs"
	userdao "seekF-backend/internal/dao/user_dao"
	"seekF-backend/internal/pkg/db"

	"gorm.io/gorm"
)

// FollowingAuthorInfo 关注栏作者及确认更新所需的服务端快照。
type FollowingAuthorInfo struct {
	userdao.FollowingAuthorUpdate
	ReadToken string `json:"read_token"`
}

// FollowingUpdatesInfo 未查看更新作者数及关注作者列表。
type FollowingUpdatesInfo struct {
	UnreadAuthors int                   `json:"unread_authors"`
	Authors       []FollowingAuthorInfo `json:"authors"`
}

// ErrFollowingReadExpired 表示更新快照失效，需要重新获取作者状态。
var ErrFollowingReadExpired = errors.New("作者更新快照已失效，请刷新后重试")

// followingReadSnapshot 绑定账号、作者及本次关注关系的已发布作品边界。
type followingReadSnapshot struct {
	User   string `json:"u"`
	Author string `json:"a"`
	Follow int64  `json:"f"`
	Post   int64  `json:"p"`
}

// followingReadKey 优先使用服务端密钥，未配置JWT密钥时生成进程内随机密钥。
var followingReadKey = sync.OnceValue(func() []byte {
	if secret := configs.GetConfig().JWTConfig.Secret; secret != "" {
		return []byte(secret)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("生成关注更新签名密钥失败: " + err.Error())
	}
	return key
})

// signFollowingSnapshot 对阅读快照签名，防止客户端自行确认未来更新。
func signFollowingSnapshot(raw []byte) []byte {
	mac := hmac.New(sha256.New, followingReadKey())
	mac.Write(raw)
	return mac.Sum(nil)
}

// FollowingUpdates 按作者汇总未查看状态，不依赖异步收件箱是否已写入。
func (s *DiscoverServiceImpl) FollowingUpdates(ctx context.Context, userId string) (*FollowingUpdatesInfo, error) {
	if userId == "" {
		return nil, fmt.Errorf("请先登录")
	}
	rows, err := s.discoverDAO.FollowingUpdates(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("查询关注作者更新失败: %w", err)
	}
	result := &FollowingUpdatesInfo{Authors: make([]FollowingAuthorInfo, 0, len(rows))}
	for _, row := range rows {
		item := FollowingAuthorInfo{FollowingAuthorUpdate: row}
		if row.HasUpdate {
			result.UnreadAuthors++
			raw, err := json.Marshal(followingReadSnapshot{userId, row.AuthorUuid, row.FollowId, row.LatestPostId})
			if err != nil {
				return nil, fmt.Errorf("生成作者更新快照失败: %w", err)
			}
			item.ReadToken = base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(signFollowingSnapshot(raw))
		}
		result.Authors = append(result.Authors, item)
	}
	return result, nil
}

// ReadFollowingUpdates 校验签名并锁定关系后前移进度，重复、乱序或重新关注不误清提醒。
func (s *DiscoverServiceImpl) ReadFollowingUpdates(ctx context.Context, userId, token string) error {
	invalid := ErrFollowingReadExpired
	parts := strings.Split(token, ".")
	if len(token) > 2048 || len(parts) != 2 {
		return invalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return invalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, signFollowingSnapshot(raw)) {
		return invalid
	}
	var value followingReadSnapshot
	if json.Unmarshal(raw, &value) != nil || userId == "" || value.User != userId || value.Author == "" || value.Post <= 0 {
		return invalid
	}
	return db.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		follow, err := userdao.NewFollowDAO(tx).FindFollowForUpdate(userId, value.Author)
		if err != nil {
			return err
		}
		if follow == nil || follow.Id != value.Follow {
			return ErrFollowingReadExpired
		}
		return userdao.NewFollowingReadDAO(tx).Advance(follow.Id, value.Post)
	})
}
