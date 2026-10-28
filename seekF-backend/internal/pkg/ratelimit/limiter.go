package ratelimit

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/redis"
	"seekF-backend/internal/pkg/zlog"

	"github.com/google/uuid"
)

//go:embed send.lua
var sendScript string

//go:embed login.lua
var loginScript string

//go:embed failure.lua
var failureScript string

//go:embed token.lua
var tokenScript string

var evaluate = redis.EvalRateLimit

// Error 表示限流拒绝或限流依赖不可用，RetryAfter为建议等待时长。
type Error struct {
	Message    string
	Status     int
	RetryAfter time.Duration
}

// Error 返回适合前端展示的中文提示。
func (e *Error) Error() string { return e.Message }

// key 对账号或IP散列，避免在Redis键中保存明文个人信息。
func key(scope, identity string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(identity))))
	return fmt.Sprintf("limit:{requests}:%s:%x", scope, sum)
}

// check 在截止时间内执行原子脚本，不可用时拒绝请求，不绕过限流。
func check(ctx context.Context, script, message string, keys []string, args ...any) error {
	task, cancel := context.WithTimeout(ctx, constants.RateLimitTimeout)
	defer cancel()
	wait, err := evaluate(task, script, keys, args...)
	if err != nil {
		zlog.Error("执行请求限流失败: " + err.Error())
		return &Error{Message: "限流服务暂时不可用，请稍后重试", Status: http.StatusServiceUnavailable}
	}
	if wait > 0 {
		return &Error{Message: message, Status: http.StatusTooManyRequests, RetryAfter: time.Duration(wait) * time.Millisecond}
	}
	return nil
}

// AllowSend 原子检查接收账号冷却与账号、IP固定窗口次数，全部通过后才占用额度。
func AllowSend(ctx context.Context, identity, ip string) error {
	return check(ctx, sendScript, "验证码发送过于频繁，请稍后重试", []string{key("send:cooldown", identity), key("send:account", identity), key("send:ip:minute", ip), key("send:ip:day", ip)}, constants.VerifyCooldownSeconds*1000, constants.VerifyAccountDaily, constants.VerifyIPMinute, constants.VerifyIPDaily)
}

// BeginAuth 检查账号及IP滑动窗口，并复核该凭据类型的账号与IP组合失败数。
func BeginAuth(ctx context.Context, identity, ip, kind string) error {
	return check(ctx, loginScript, "登录或验证码校验过于频繁，请稍后重试", []string{key("login:ip", ip), key("login:account", identity), key("failure:"+kind, identity+"|"+ip)}, constants.LoginIPMinute, constants.LoginAccountMinute, constants.LoginFailureSeconds*1000, constants.LoginFailures, uuid.NewString())
}

// FinishAuth 只记录真实凭据失败，成功时清理组合失败数，数据库及网络故障不累计。
func FinishAuth(ctx context.Context, identity, ip, kind string, success, failed bool) error {
	if !success && !failed {
		return nil
	}
	action := "1"
	if success {
		action = "0"
	}
	return check(ctx, failureScript, "", []string{key("failure:"+kind, identity+"|"+ip)}, action, constants.LoginFailureSeconds*1000, uuid.NewString())
}

// AllowAI 通过用户共享令牌桶控制AI请求频率
func AllowAI(ctx context.Context, userID string) error {
	return check(ctx, tokenScript, "AI请求过于频繁，请稍后重试", []string{key("ai:tokens", userID)}, constants.AIRequestsPerMinute, constants.AIBurst)
}

var aiConcurrency = struct {
	sync.Mutex
	total int
	users map[string]int
}{users: make(map[string]int)}

// AcquireAI 获取单实例共用AI名额，返回可重复安全调用的释放函数；取消请求不会提前释放仍在生成的任务。
func AcquireAI(ctx context.Context, userID string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	aiConcurrency.Lock()
	if aiConcurrency.total >= constants.AITotalConcurrent || aiConcurrency.users[userID] >= constants.AIUserConcurrent {
		aiConcurrency.Unlock()
		return nil, &Error{Message: "AI服务繁忙，请稍后重试", Status: http.StatusTooManyRequests, RetryAfter: time.Second}
	}
	aiConcurrency.total++
	aiConcurrency.users[userID]++
	aiConcurrency.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			aiConcurrency.Lock()
			defer aiConcurrency.Unlock()
			aiConcurrency.total--
			aiConcurrency.users[userID]--
			if aiConcurrency.users[userID] == 0 {
				delete(aiConcurrency.users, userID)
			}
		})
	}, nil
}
