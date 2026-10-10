package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics 持有HTTP请求计数器与耗时直方图，由注册表导出、中间件更新。
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewHTTPMetrics 创建按方法、路由及状态码分类的HTTP指标。
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "seekf_http_requests_total",
			Help: "HTTP请求总数，按方法、路由模板和状态码分类。",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "seekf_http_request_duration_seconds",
			Help: "HTTP请求完整处理耗时，单位为秒。",
			// 覆盖AI流式请求的15分钟超时，避免长请求分位数被压到10秒。
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 900},
		}, []string{"method", "route"}),
	}
}

// RequestsCollector 返回请求计数采集器，供统一注册表导出。
func (m *HTTPMetrics) RequestsCollector() prometheus.Collector {
	return m.requests
}

// DurationCollector 返回请求耗时采集器，供统一注册表导出。
func (m *HTTPMetrics) DurationCollector() prometheus.Collector {
	return m.duration
}

// Record 累计一次请求及其完整处理耗时，durationSeconds单位为秒。
func (m *HTTPMetrics) Record(method, route string, status int, durationSeconds float64) {
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route).Observe(durationSeconds)
}
