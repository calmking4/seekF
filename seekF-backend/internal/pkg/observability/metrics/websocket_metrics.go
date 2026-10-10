package metrics

import (
	"seekF-backend/internal/pkg/websocket"

	"github.com/prometheus/client_golang/prometheus"
)

// NewOnlineUsersCollector 创建在线用户采集器，每次采集直接读取WebSocket当前在线人数。
func NewOnlineUsersCollector() prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "seekf_websocket_online_users",
		Help: "当前WebSocket在线用户数。",
	}, func() float64 {
		if websocket.ChatServer == nil {
			return 0
		}
		return float64(websocket.ChatServer.OnlineCount())
	})
}
