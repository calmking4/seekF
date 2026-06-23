package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"seekF-backend/internal/configs"

	"golang.org/x/oauth2"
)

// GiteeUser Gitee 用户信息
type GiteeUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// giteeEmail Gitee 邮箱信息
type giteeEmail struct {
	Email   string `json:"email"`
	Primary bool   `json:"primary"`
}

// GiteeEndpoint Gitee OAuth2 端点
var GiteeEndpoint = oauth2.Endpoint{
	AuthURL:  "https://gitee.com/oauth/authorize",
	TokenURL: "https://gitee.com/oauth/token",
}

// newGiteeOAuthConfig 创建 Gitee OAuth2 配置
func newGiteeOAuthConfig() *oauth2.Config {
	cfg := configs.GetConfig()
	return &oauth2.Config{
		ClientID:     cfg.GiteeOAuthConfig.ClientID,
		ClientSecret: cfg.GiteeOAuthConfig.ClientSecret,
		RedirectURL:  cfg.GiteeOAuthConfig.RedirectURL,
		Scopes:       []string{"user_info"},
		Endpoint:     GiteeEndpoint,
	}
}

// GetGiteeAuthCodeURL 生成 Gitee 授权跳转地址
func GetGiteeAuthCodeURL(state string) string {
	return newGiteeOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeAndGetGiteeUser 用授权码换取令牌并获取 Gitee 用户信息
func ExchangeAndGetGiteeUser(ctx context.Context, code string) (*GiteeUser, error) {
	oauthCfg := newGiteeOAuthConfig()

	// Gitee 国内访问，无需代理
	httpClient := &http.Client{}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("换取 Gitee 令牌失败: %w", err)
	}

	// 创建带 token 的客户端用于 API 请求
	tokenSource := oauth2.StaticTokenSource(token)
	apiClient := oauth2.NewClient(ctx, tokenSource)

	user, err := fetchGiteeUser(apiClient)
	if err != nil {
		return nil, err
	}

	if user.Email == "" {
		email, err := fetchGiteePrimaryEmail(apiClient)
		if err == nil && email != "" {
			user.Email = email
		}
		// 邮箱获取失败不阻断登录，用户可能未公开邮箱
	}

	return user, nil
}

func fetchGiteeUser(client *http.Client) (*GiteeUser, error) {
	resp, err := client.Get("https://gitee.com/api/v5/user")
	if err != nil {
		return nil, fmt.Errorf("请求 Gitee 用户信息失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Gitee 用户信息失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 Gitee 用户信息失败: %s", string(body))
	}

	var user GiteeUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("解析 Gitee 用户信息失败: %w", err)
	}
	if user.ID == 0 {
		return nil, fmt.Errorf("Gitee 用户 ID 无效")
	}

	return &user, nil
}

func fetchGiteePrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://gitee.com/api/v5/emails")
	if err != nil {
		return "", fmt.Errorf("请求 Gitee 邮箱列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Gitee 邮箱列表失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取 Gitee 邮箱列表失败: %s", string(body))
	}

	var emails []giteeEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("解析 Gitee 邮箱列表失败: %w", err)
	}

	for _, item := range emails {
		if item.Primary {
			return item.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", nil
}
