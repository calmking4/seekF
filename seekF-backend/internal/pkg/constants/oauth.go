package constants

import "time"

// 第三方登录请求与浏览器绑定。
const (
	// GitHub授权平台标识，用于隔离Redis记录及临时Cookie。
	OAuthProviderGithub = "github"
	// Gitee授权平台标识，用于隔离Redis记录及临时Cookie。
	OAuthProviderGitee = "gitee"
	// 授权请求及浏览器绑定最长保留10分钟。
	OAuthStateTTL = 10 * time.Minute
	// OAuth随机状态由16字节随机数编码为32个十六进制字符。
	OAuthStateLength = 32
	// 浏览器临时授权Cookie名称前缀。
	OAuthStateCookiePrefix = "oauth_state_"
	// Redis一次性授权请求记录的键前缀。
	OAuthStateKeyPrefix = "oauth_state:"
)

// 第三方登录昵称生成限制。
const (
	// 用户昵称最大字符数，与 user_info.nickname 字段长度一致。
	OAuthNicknameMaxLength = 20
	// 第三方首次登录最多尝试的昵称数量，包含原昵称。
	OAuthNicknameMaxAttempts = 1000
)
