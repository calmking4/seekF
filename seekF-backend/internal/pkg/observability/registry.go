package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 持有独立注册表，统一注册和导出运行时及外部传入的指标。
type Metrics struct {
	registry *prometheus.Registry
}

// NewMetrics 注册Go运行时、进程和外部采集器，不创建或更新业务指标。
func NewMetrics(businessCollectors ...prometheus.Collector) *Metrics {
	m := &Metrics{registry: prometheus.NewRegistry()}
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	m.registry.MustRegister(businessCollectors...)
	return m
}

// Handler 返回仅导出当前实例指标的HTTP处理器，不查询外部依赖。
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
