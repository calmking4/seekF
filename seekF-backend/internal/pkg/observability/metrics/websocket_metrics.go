package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NewOnlineUsersCollector 通过注入的读取函数采集实时在线人数，避免指标包依赖WebSocket业务。
func NewOnlineUsersCollector(onlineCount func() int) prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "seekf_websocket_online_users",
		Help: "当前WebSocket在线用户数。",
	}, func() float64 {
		return float64(onlineCount())
	})
}
