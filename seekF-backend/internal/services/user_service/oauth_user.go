package userservice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/constants"

	"github.com/go-sql-driver/mysql"
)

// oauthUserDAO 提供第三方首次登录所需的昵称查询和用户创建能力。
type oauthUserDAO interface {
	// 查询昵称对应的用户。
	FindUserByNickname(string) (*models.UserInfo, error)
	// 创建用户，由数据库唯一约束保证并发安全。
	CreateUser(*models.UserInfo) error
}

// createOAuthUser 创建第三方用户，重名时依次追加数字，并用第三方身份处理并发登录。
func createOAuthUser(dao oauthUserDAO, user *models.UserInfo, findIdentity func() (*models.UserInfo, error)) (*models.UserInfo, error) {
	base := strings.TrimSpace(user.Nickname)
	for attempt := 0; attempt < constants.OAuthNicknameMaxAttempts; attempt++ {
		user.Nickname = oauthNickname(base, attempt)
		existing, err := dao.FindUserByNickname(user.Nickname)
		if err != nil {
			return nil, fmt.Errorf("查询第三方用户昵称失败：%w", err)
		}
		if existing != nil {
			continue
		}
		// 预查询不能避免并发抢占，最终以数据库唯一约束为准。
		user.Id = 0
		err = dao.CreateUser(user)
		if err == nil {
			return user, nil
		}
		var sqlErr *mysql.MySQLError
		if !errors.As(err, &sqlErr) || sqlErr.Number != 1062 {
			return nil, err
		}
		// 只能复用同一个第三方身份创建的账号，不能因为昵称相同合并账号。
		existing, lookupErr := findIdentity()
		if lookupErr != nil {
			return nil, fmt.Errorf("查询并发创建的第三方用户失败：%w", lookupErr)
		}
		if existing != nil {
			return existing, nil
		}
		if !strings.Contains(sqlErr.Message, "idx_user_info_nickname") {
			return nil, err
		}
	}
	return nil, fmt.Errorf("自动分配第三方用户昵称失败，请稍后重试")
}

// oauthNickname 生成原昵称或纯数字后缀昵称，按字符截断以满足数据库长度限制。
func oauthNickname(base string, attempt int) string {
	suffix := ""
	if attempt > 0 {
		suffix = strconv.Itoa(attempt)
	}
	runes := []rune(base)
	limit := constants.OAuthNicknameMaxLength - len(suffix)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes) + suffix
}
