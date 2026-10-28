package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/ratelimit"
	"seekF-backend/internal/pkg/resp"
	"seekF-backend/internal/pkg/zlog"
)

var limitEmailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// rejectRateLimit 返回明确的429/503响应与建议等待时间，并停止后续业务。
func rejectRateLimit(c *gin.Context, err error) {
	var limitErr *ratelimit.Error
	if errors.As(err, &limitErr) {
		if limitErr.RetryAfter > 0 {
			seconds := int64((limitErr.RetryAfter + time.Second - 1) / time.Second)
			c.Header("Retry-After", strconv.FormatInt(seconds, 10))
			limitErr = &ratelimit.Error{Message: fmt.Sprintf("%s，%d秒后重试", strings.TrimSuffix(limitErr.Message, "，请稍后重试"), seconds), Status: limitErr.Status}
		}
		resp.Error(c, limitErr.Message, limitErr.Status)
	} else {
		resp.Error(c, "请求已取消或服务暂时不可用", http.StatusServiceUnavailable)
	}
	c.Abort()
}

// AuthRequest 对发送验证码或凭据校验实施限流，恢复请求体供控制器正常绑定。
func AuthRequest(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			resp.Error(c, "认证请求过大或无法读取", http.StatusBadRequest)
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var input struct {
			Telephone string `json:"telephone"`
			Email     string `json:"email"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			resp.Error(c, "认证请求格式错误", http.StatusBadRequest)
			c.Abort()
			return
		}
		// 与认证服务选择邮箱或手机号的规则保持一致，散列前不改变服务实际收到的账号。
		identity := "phone:" + strings.TrimSpace(input.Telephone)
		if limitEmailPattern.MatchString(input.Email) {
			identity = "email:" + input.Email
		} else if strings.TrimSpace(input.Telephone) == "" {
			resp.Error(c, "请填写有效手机号或邮箱", http.StatusBadRequest)
			c.Abort()
			return
		}
		// 直接使用连接对端IP，避免未配置可信代理时伪造转发头绕过限制。
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.Request.RemoteAddr
		}
		if kind == constants.AuthSend {
			if err := ratelimit.AllowSend(c.Request.Context(), identity, ip); err != nil {
				rejectRateLimit(c, err)
				return
			}
			c.Next()
			return
		}
		if err := ratelimit.BeginAuth(c.Request.Context(), identity, ip, kind); err != nil {
			rejectRateLimit(c, err)
			return
		}
		c.Next()
		// 客户端断开不应免除已发生的凭据失败统计；独立截止时间仍限制Redis耗时。
		task, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), constants.RateLimitTimeout)
		defer cancel()
		failed, _ := c.Get(constants.AuthCredentialFailed)
		if err := ratelimit.FinishAuth(task, identity, ip, kind, c.Writer.Status() == http.StatusOK, failed == true); err != nil {
			zlog.Error("保存认证限流结果失败: " + err.Error())
		}
	}
}

// AIRequest 在响应流启动前检查令牌桶，并按需持有并发名额直至整个处理器结束。
func AIRequest(acquireSlot bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("Uuid")
		if userID == "" {
			resp.Error(c, "请先登录", http.StatusUnauthorized)
			c.Abort()
			return
		}
		if acquireSlot {
			release, err := ratelimit.AcquireAI(c.Request.Context(), userID)
			if err != nil {
				rejectRateLimit(c, err)
				return
			}
			defer release()
		}
		if err := ratelimit.AllowAI(c.Request.Context(), userID); err != nil {
			rejectRateLimit(c, err)
			return
		}
		c.Next()
	}
}
