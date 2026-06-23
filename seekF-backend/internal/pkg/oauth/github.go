package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"seekF-backend/internal/configs"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type gitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func newGithubOAuthConfig() *oauth2.Config {
	cfg := configs.GetConfig()
	return &oauth2.Config{
		ClientID:     cfg.GithubOAuthConfig.ClientID,
		ClientSecret: cfg.GithubOAuthConfig.ClientSecret,
		RedirectURL:  cfg.GithubOAuthConfig.RedirectURL,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}
}

// newHTTPClient 创建带代理的 HTTP 客户端
func newHTTPClient(ctx context.Context) *http.Client {
	cfg := configs.GetConfig()
	proxyURL := cfg.GithubOAuthConfig.ProxyURL
	if proxyURL == "" {
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(nil))
	}

	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(nil))
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	return &http.Client{Transport: transport}
}

// GetAuthCodeURL 生成 GitHub 授权跳转地址
func GetAuthCodeURL(state string) string {
	return newGithubOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeAndGetUser 用授权码换取令牌并获取 GitHub 用户信息
func ExchangeAndGetUser(ctx context.Context, code string) (*GitHubUser, error) {
	oauthCfg := newGithubOAuthConfig()

	// 创建带代理的 HTTP 客户端
	proxyURL := configs.GetConfig().GithubOAuthConfig.ProxyURL
	var httpClient *http.Client
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
			httpClient = &http.Client{Transport: transport}
		}
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("换取 GitHub 令牌失败: %w", err)
	}

	// 创建带代理和 token 的客户端用于 API 请求
	tokenSource := oauth2.StaticTokenSource(token)
	apiClient := oauth2.NewClient(ctx, tokenSource)
	// 保留代理设置
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err == nil {
			apiClient.Transport = &oauth2.Transport{
				Source: tokenSource,
				Base: &http.Transport{
					Proxy: http.ProxyURL(proxy),
				},
			}
		}
	}

	user, err := fetchGitHubUser(apiClient)
	if err != nil {
		return nil, err
	}

	if user.Email == "" {
		email, err := fetchPrimaryEmail(apiClient)
		if err != nil {
			return nil, err
		}
		user.Email = email
	}

	return user, nil
}

func fetchGitHubUser(client *http.Client) (*GitHubUser, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("请求 GitHub 用户信息失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 GitHub 用户信息失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取 GitHub 用户信息失败: %s", string(body))
	}

	var user GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("解析 GitHub 用户信息失败: %w", err)
	}
	if user.ID == 0 {
		return nil, fmt.Errorf("GitHub 用户 ID 无效")
	}

	return &user, nil
}

func fetchPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", fmt.Errorf("请求 GitHub 邮箱列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 GitHub 邮箱列表失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取 GitHub 邮箱列表失败: %s", string(body))
	}

	var emails []gitHubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("解析 GitHub 邮箱列表失败: %w", err)
	}

	for _, item := range emails {
		if item.Primary && item.Verified {
			return item.Email, nil
		}
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
