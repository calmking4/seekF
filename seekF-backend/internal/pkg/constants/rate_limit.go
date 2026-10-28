package constants

import "time"

// 认证限流类别与结果标记。
const (
	// 密码认证限流类别，用户端与管理端共用。
	AuthPassword = "password"
	// 注册及登录的验证码校验限流类别。
	AuthCode = "code"
	// 验证码发送限流类别。
	AuthSend = "send"
	// Gin上下文中的凭据失败标记。
	AuthCredentialFailed = "auth_credential_failed"
)

// 验证码发送：冷却时间与固定窗口次数上限。
const (
	// 验证码最小发送间隔，单位秒。
	VerifyCooldownSeconds = 60
	// 每个接收账号每日最多发送次数。
	VerifyAccountDaily = 10
	// 每个IP每分钟最多发送次数。
	VerifyIPMinute = 5
	// 每个IP每日最多发送次数。
	VerifyIPDaily = 50
)

// 登录与验证码校验：滑动窗口请求上限与凭据失败保护。
const (
	// 每个IP每分钟最多认证请求数。
	LoginIPMinute = 30
	// 每个账号每分钟最多认证请求数。
	LoginAccountMinute = 10
	// 账号与IP组合在窗口内最多凭据失败次数。
	LoginFailures = 5
	// 认证失败统计窗口，单位秒。
	LoginFailureSeconds = 900
)

// AI调用：令牌桶速率、容量及单实例并发上限。
const (
	// 每用户AI令牌每分钟补充数量。
	AIRequestsPerMinute = 12
	// 每用户AI令牌桶容量。
	AIBurst = 4
	// 单用户在当前进程内最多并发AI请求数。
	AIUserConcurrent = 2
	// 当前进程最多并发AI请求数。
	AITotalConcurrent = 8
	// 旧版AI评论任务缺少发起人时使用的并发隔离标识。
	LegacyAICommentUser = "legacy-ai-comment"
)

// 限流公共参数。
const (
	// 限流Redis操作最大耗时。
	RateLimitTimeout = 2 * time.Second
)
