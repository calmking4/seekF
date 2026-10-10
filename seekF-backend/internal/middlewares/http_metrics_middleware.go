package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const upgradedKey = "seekf_observability_websocket_upgraded"

// HTTPMetricsRecorder 定义请求统计接口，避免中间件依赖具体业务指标包。
type HTTPMetricsRecorder interface {
	Record(method, route string, status int, durationSeconds float64)
}

// HTTPMetricsMiddleware 记录最终响应状态和完整耗时，应安装在Recovery与认证中间件之前。
func HTTPMetricsMiddleware(recorder HTTPMetricsRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/metrics" || path == "/debug/pprof" || strings.HasPrefix(path, "/debug/pprof/") {
			c.Next()
			return
		}
		startedAt := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := c.Writer.Status()
		// Gorilla直接向劫持的连接写入101，Gin的状态字段不会同步更新。
		if c.GetBool(upgradedKey) {
			status = http.StatusSwitchingProtocols
		}
		recorder.Record(normalizedMethod(c.Request.Method), route, status, time.Since(startedAt).Seconds())
	}
}

// MarkWebSocketUpgrade 在升级成功后标记真实握手状态，不包装或改写连接。
func MarkWebSocketUpgrade(c *gin.Context) {
	c.Set(upgradedKey, true)
}

// normalizedMethod 收敛自定义HTTP方法，防止未知请求造成标签无限增长。
func normalizedMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
		http.MethodHead, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}
