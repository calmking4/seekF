# GitHub OAuth 登录实现

## 概述

本文档介绍 seekF 项目中 GitHub OAuth 第三方登录的完整实现流程，包括后端接口、OAuth 认证流程、前端回调处理等。

## 目录结构

```
seekF-backend/
├── config/config.toml                    # OAuth 配置
├── internal/
│   ├── configs/configs.go                # 配置结构体定义
│   ├── controllers/user/auth_controller.go  # OAuth 控制器
│   ├── services/user_service/auth_service.go # OAuth 业务逻辑
│   ├── pkg/oauth/github.go               # GitHub OAuth 工具包
│   └── router/router.go                  # 路由定义

seekF-user/
├── app/pages/login.vue                   # 登录页（触发 GitHub 登录）
└── app/pages/oauth/github/callback.vue   # OAuth 回调页面
```

## OAuth 2.0 认证流程

### 流程图

```
用户浏览器                seekF后端                 GitHub服务器
    │                       │                         │
    │ ①点击GitHub登录       │                         │
    │──────────────────────>│                         │
    │                       │                         │
    │ ②返回授权URL（带state）│                         │
    │<──────────────────────│                         │
    │                       │                         │
    │ ③跳转到GitHub授权页   │                         │
    │────────────────────────────────────────────────>│
    │                       │                         │
    │ ④用户点击"Authorize"同意授权                     │
    │────────────────────────────────────────────────>│
    │                       │                         │
    │ ⑤GitHub回调，重定向到seekF（带code和state）      │
    │<────────────────────────────────────────────────│
    │                       │                         │
    │ ⑥前端把code+state发给后端                        │
    │──────────────────────>│                         │
    │                       │                         │
    │                       │ ⑦用code换取access_token │
    │                       │────────────────────────>│
    │                       │                         │
    │                       │ ⑧返回access_token       │
    │                       │<────────────────────────│
    │                       │                         │
    │                       │ ⑨用access_token获取用户信息
    │                       │────────────────────────>│
    │                       │                         │
    │                       │ ⑩返回用户信息            │
    │                       │<────────────────────────│
    │                       │                         │
    │ ⑪返回登录结果（seekF token + 用户信息）          │
    │<──────────────────────│                         │
```

### 三个凭证的严格区分

| 凭证 | 颁发者 | 持有者 | 生成时机 | 用途 |
|------|--------|--------|----------|------|
| `code` | GitHub | seekF后端（从URL参数获取） | 用户在GitHub点击授权后 | 一次性凭证，用于换取`access_token` |
| `access_token` | GitHub | seekF后端 | seekF用`code`向GitHub交换时 | 调用GitHub API获取用户信息 |
| `token` | seekF | 用户浏览器（Cookie） | seekF验证完用户身份后 | 用户访问seekF时的身份凭证 |

### 详细流程说明

**① 用户点击 GitHub 登录按钮**
- 前端跳转到 `GET /user/github/login`

**② 后端生成授权地址（生成 state）**
- 生成随机 `state` 参数（使用 `crypto/rand`，防 CSRF 攻击）
- 将 `state` 存储到 Redis（10 分钟有效期）
- 返回 GitHub OAuth 授权页面 URL（URL 中携带 `state`）

**③ 浏览器跳转到 GitHub 授权页面**
- URL 格式：`https://github.com/login/oauth/authorize?client_id=xxx&state=xxx&scope=xxx`
- 此时 `state` 由 seekF 后端生成，存在于 URL 参数和 Redis 中

**④ 用户在 GitHub 页面点击 "Authorize" 同意授权**
- 用户确认授权 seekF 访问其 GitHub 基本信息

**⑤ GitHub 回调 seekF（生成 code）**
- GitHub 生成临时授权码 `code`（有效期约10分钟，只能使用一次）
- GitHub 重定向到 seekF 的回调地址：`http://localhost:8080/user/github/callback?code=AUTH_CODE&state=xxx`
- 此时 `code` 由 GitHub 颁发，通过 URL 参数传递给 seekF

**⑥ 前端把 code 和 state 发给后端**
- 浏览器访问回调 URL，seekF 后端从 URL 参数中获取 `code` 和 `state`

**⑦ seekF 后端用 code 向 GitHub 换取 access_token**
- seekF 后端向 GitHub 发送请求：`POST https://github.com/login/oauth/access_token`
- 请求体包含：`client_id`、`client_secret`、`code`
- 此时 `code` 被消耗，换取到 `access_token`

**⑧ GitHub 返回 access_token**
- GitHub 验证 `code` 有效后，返回 `access_token`（对应授权用户的 GitHub 账号）
- `access_token` 由 GitHub 颁发，由 seekF 后端持有

**⑨ seekF 后端用 access_token 获取用户信息**
- seekF 后端向 GitHub API 发送请求：`GET https://api.github.com/user`
- 请求头携带：`Authorization: token {access_token}`
- GitHub 返回用户信息（头像、邮箱、昵称等）

**⑩ GitHub 返回用户信息**
- seekF 后端获取到用户的 GitHub ID、头像、邮箱等信息

**⑪ seekF 后端处理登录，生成 seekF token**
- 根据 GitHub ID 查找或创建 seekF 用户
- 生成 seekF 自己的 `token`（JWT 或 session）
- 将 `token` 设置到浏览器 Cookie
- 重定向到前端回调地址，携带用户信息

### CSRF 攻击与 state 防护

#### 什么是 CSRF 攻击

CSRF（Cross-Site Request Forgery，跨站请求伪造）是一种攻击方式：攻击者诱导用户在已登录的状态下访问恶意链接，利用用户的身份执行非预期操作。

#### 没有 state 防护的攻击场景

```
① 攻击者自己走 GitHub 登录流程，拿到 code="abc"（对应攻击者的 GitHub 账号）

② 攻击者构造链接发给受害者：
   https://your-seekF.com/callback?code=abc

③ 受害者（已登录 seekF）点击链接
   → 受害者的浏览器向 seekF 后端发请求，带上 code="abc"

④ seekF 后端收到 code="abc"
   → 用 code="abc" 向 GitHub 换取 access_token="xyz"

⑤ GitHub 验证 code="abc" 有效
   → 返回 access_token="xyz"（对应攻击者的 GitHub 账号）

⑥ seekF 后端用 access_token="xyz" 向 GitHub API 获取用户信息
   → GitHub 返回攻击者的头像、邮箱、昵称等

⑦ seekF 后端找到/创建攻击者的 seekF 账号
   → 生成 token="def"（攻击者的 seekF token）
   → 设置到受害者的浏览器 Cookie 中

⑧ 结果：受害者的浏览器持有攻击者的 seekF token
   → 受害者以为自己登录的是自己的账号
   → 实际上是以攻击者的身份在操作
```

#### state 如何防护

```
正常流程：
① seekF 后端生成随机 state="random123"，存入 Redis
② 用户跳转 GitHub 授权页，URL 带 state="random123"
③ GitHub 回调 seekF，也带回 state="random123"
④ seekF 后端比对：回调的 state == Redis 中存的？✅ 通过

攻击流程：
① 攻击者构造链接：https://your-seekF.com/callback?code=abc
   （没有 state 参数，或者 state 不正确）
② 受害者点击链接
③ seekF 后端比对：state 为空或 Redis 中找不到 ❌ 拒绝请求
```

**关键点**：`state` 是一个只有 seekF 后端才知道的随机数，攻击者无法猜测，因此伪造的请求无法通过验证。

## 代码实现

### 1. 配置文件

**config.toml**

```toml
[githubOAuthConfig]
clientID = ""                                    # GitHub OAuth App Client ID
clientSecret = ""                                # GitHub OAuth App Client Secret
redirectURL = "http://localhost:8080/user/github/callback"  # 后端回调地址
frontendRedirectURL = "http://localhost:3000/oauth/github/callback"  # 前端回调地址
proxyURL = "http://127.0.0.1:7890"               # 代理地址
```

### 2. 配置结构体

**internal/configs/configs.go**

```go
// GithubOAuthConfig GitHub OAuth配置
type GithubOAuthConfig struct {
    ClientID            string `toml:"clientID"`
    ClientSecret        string `toml:"clientSecret"`
    RedirectURL         string `toml:"redirectURL"`
    FrontendRedirectURL string `toml:"frontendRedirectURL"`
    ProxyURL            string `toml:"proxyURL"` // HTTP 代理地址
}
```

### 3. 路由定义

**internal/router/router.go**

```go
// 无需认证的接口
userGroup.GET("/github/login", authController.GithubLogin)
userGroup.GET("/github/callback", authController.GithubCallback)
```

### 4. 控制器层

**internal/controllers/user/auth_controller.go**

```go
// GithubLogin 跳转 GitHub OAuth 授权页
func (c *AuthController) GithubLogin(ctx *gin.Context) {
    authURL, err := c.authService.GithubAuthURL()
    if err != nil {
        zlog.Error("生成 GitHub 授权地址失败: " + err.Error())
        resp.Error(ctx, err.Error(), http.StatusBadRequest)
        return
    }
    ctx.Redirect(http.StatusFound, authURL)
}

// GithubCallback GitHub OAuth 回调
func (c *AuthController) GithubCallback(ctx *gin.Context) {
    cfg := configs.GetConfig()
    frontendURL := strings.TrimSpace(cfg.GithubOAuthConfig.FrontendRedirectURL)
    if frontendURL == "" {
        frontendURL = "http://localhost:3000/oauth/github/callback"
    }

    redirectWithError := func(message string) {
        target := fmt.Sprintf("%s?error=%s", frontendURL, url.QueryEscape(message))
        ctx.Redirect(http.StatusFound, target)
    }

    // 处理错误
    if errMsg := strings.TrimSpace(ctx.Query("error")); errMsg != "" {
        redirectWithError("GitHub 授权已取消")
        return
    }

    code := strings.TrimSpace(ctx.Query("code"))
    state := strings.TrimSpace(ctx.Query("state"))

    // 调用 Service 层处理登录
    result, err := c.authService.LoginByGithub(code, state)
    if err != nil {
        redirectWithError(err.Error())
        return
    }

    // 设置 Cookie
    expireSeconds := cfg.SessionExpireMinutes * 60
    ctx.SetCookie("token", result.Token, int(expireSeconds), "/", "localhost", false, true)

    // 重定向到前端，携带用户信息
    userBytes, _ := json.Marshal(result.User)
    ctx.Redirect(http.StatusFound, fmt.Sprintf("%s?user=%s", frontendURL, url.QueryEscape(string(userBytes))))
}
```

### 5. Service 层

**internal/services/user_service/auth_service.go**

```go
// GithubAuthURL 生成 GitHub OAuth 授权地址
func (s *AuthServiceImpl) GithubAuthURL() (string, error) {
    cfg := configs.GetConfig()
    if strings.TrimSpace(cfg.GithubOAuthConfig.ClientID) == "" {
        return "", fmt.Errorf("GitHub OAuth 未配置")
    }

    // 生成随机 state
    state, err := generateOAuthState()
    if err != nil {
        return "", fmt.Errorf("生成 OAuth 状态码失败：%v", err)
    }

    // 存储 state 到 Redis（10分钟有效）
    key := fmt.Sprintf("oauth_state:github:%s", state)
    if err := redis.SetKeyEx(key, "1", 10*time.Minute); err != nil {
        return "", fmt.Errorf("保存 OAuth 状态码失败：%v", err)
    }

    return oauth.GetAuthCodeURL(state), nil
}

// LoginByGithub GitHub OAuth 登录
func (s *AuthServiceImpl) LoginByGithub(code, state string) (*LoginRespond, error) {
    // 验证 state
    key := fmt.Sprintf("oauth_state:github:%s", state)
    stored, err := redis.GetKey(key)
    if err != nil || stored == "" {
        return nil, fmt.Errorf("OAuth 状态码无效或已过期")
    }
    redis.DelKeyIfExists(key)

    // 用 code 换取用户信息
    githubUser, err := oauth.ExchangeAndGetUser(context.Background(), code)
    if err != nil {
        return nil, fmt.Errorf("GitHub 登录失败：%v", err)
    }

    // 查找或创建用户
    user, err := s.userInfoDAO.FindUserByGithubId(githubUser.ID)
    if err != nil {
        return nil, fmt.Errorf("查询用户失败：%v", err)
    }

    if user == nil {
        // 创建新用户
        userUUID := "U" + util.GetNowAndLenRandomString(11)
        user = &models.UserInfo{
            Uuid:     userUUID,
            Nickname: githubUser.Name,
            Email:    githubUser.Email,
            Avatar:   githubUser.AvatarURL,
            GithubId: githubUser.ID,
        }
        s.userInfoDAO.CreateUser(user)
    } else {
        // 更新用户信息（头像、邮箱）
        // ...
    }

    // 生成 Token
    token, _ := s.createLoginToken(user, mode)

    return s.buildLoginRespond(user, token), nil
}
```

### 6. OAuth 工具包

**internal/pkg/oauth/github.go**

```go
package oauth

import (
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

// newGithubOAuthConfig 创建 OAuth2 配置
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

// GetAuthCodeURL 生成授权地址
func GetAuthCodeURL(state string) string {
    return newGithubOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeAndGetUser 用 code 换取用户信息
func ExchangeAndGetUser(ctx context.Context, code string) (*GitHubUser, error) {
    oauthCfg := newGithubOAuthConfig()

    // 创建带代理的客户端
    httpClient := newHTTPClient(ctx)
    ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

    // 用 code 换取 token
    token, err := oauthCfg.Exchange(ctx, code)
    if err != nil {
        return nil, fmt.Errorf("换取 GitHub 令牌失败: %w", err)
    }

    // 获取用户信息
    apiClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
    user, err := fetchGitHubUser(apiClient)
    if err != nil {
        return nil, err
    }

    // 如果没有邮箱，单独获取
    if user.Email == "" {
        email, _ := fetchPrimaryEmail(apiClient)
        user.Email = email
    }

    return user, nil
}
```

### 7. 前端实现

**login.vue**（触发登录）

```vue
<script setup>
const loginWithGithub = () => {
  const config = useRuntimeConfig()
  window.location.href = `${config.public.apiBase}user/github/login`
}
</script>

<template>
  <button @click="loginWithGithub">
    使用 GitHub 登录
  </button>
</template>
```

**oauth/github/callback.vue**（回调处理）

```vue
<template>
  <div class="flex min-h-screen items-center justify-center">
    <div class="text-center">
      <!-- 加载/成功/失败状态展示 -->
    </div>
  </div>
</template>

<script setup>
const status = ref('loading')

onMounted(() => {
  const route = useRoute()
  const userParam = route.query.user

  if (userParam) {
    const user = JSON.parse(userParam)
    const authState = useAuthState()
    authState.setUser(user)
    status.value = 'success'
    setTimeout(() => navigateTo('/'), 1500)
  } else {
    status.value = 'error'
  }
})
</script>
```

## GitHub OAuth App 配置

### 1. 创建 OAuth App

1. 访问 [GitHub Developer Settings](https://github.com/settings/developers)
2. 点击 "New OAuth App"
3. 填写信息：
   - **Application name**: seekF
   - **Homepage URL**: `http://localhost:3000`
   - **Authorization callback URL**: `http://localhost:8080/user/github/callback`

### 2. 获取凭证

创建完成后获取：
- **Client ID**: 填入 `config.toml` 或 `.env`
- **Client Secret**: 填入 `.env`（敏感信息）

## 安全措施

### 1. State 参数防 CSRF

```go
// 生成随机 state
state, err := generateOAuthState()  // 使用 crypto/rand

// 存储到 Redis
redis.SetKeyEx(key, "1", 10*time.Minute)

// 回调时验证
stored, _ := redis.GetKey(key)
if stored == "" {
    return nil, fmt.Errorf("OAuth 状态码无效")
}
```

### 2. 代理支持

```go
// 支持通过代理访问 GitHub API
func newHTTPClient(ctx context.Context) *http.Client {
    proxyURL := configs.GetConfig().GithubOAuthConfig.ProxyURL
    if proxyURL != "" {
        proxy, _ := url.Parse(proxyURL)
        transport := &http.Transport{Proxy: http.ProxyURL(proxy)}
        return &http.Client{Transport: transport}
    }
    return &http.Client{}
}
```

## 错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| 用户取消授权 | GitHub 返回 `error` 参数，提示"授权已取消" |
| State 无效/过期 | Redis 中找不到对应 state，提示"状态码无效" |
| Code 换取 Token 失败 | 返回错误信息到前端 |
| 获取用户信息失败 | 返回错误信息到前端 |

## 调试日志

```go
zlog.Info("GitHub 授权被拒绝: " + errMsg)
zlog.Error("生成 GitHub 授权地址失败: " + err.Error())
zlog.Error("GitHub 登录失败: " + err.Error())
zlog.Error("序列化 GitHub 登录用户信息失败: " + err.Error())
```

## 常见问题

### 1. 回调地址不匹配

确保 `config.toml` 中的 `redirectURL` 与 GitHub OAuth App 配置一致。

### 2. 代理问题

国内访问 GitHub API 可能需要代理，配置 `proxyURL`：

```toml
proxyURL = "http://127.0.0.1:7890"
```

### 3. 邮箱为空

GitHub 用户可能设置了邮箱隐私，需要单独调用 `/user/emails` API 获取。

## 相关依赖

```go
// go.mod
require (
    golang.org/x/oauth2 v0.x.x
)
```

## 参考资料

- [GitHub OAuth Documentation](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)
- [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
