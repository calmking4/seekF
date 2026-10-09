package adminresp

import "time"

// Identity 管理员身份，不包含登录令牌和私人联系方式。
type Identity struct {
	UUID     string `json:"uuid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// DayActivity 每日业务数量，按后端本地时区统计。
type DayActivity struct {
	Date     string `json:"date"`
	Users    int64  `json:"users"`
	Messages int64  `json:"messages"`
	Posts    int64  `json:"posts"`
}

// Overview 管理端业务概览，最多缓存三十秒。
type Overview struct {
	Users         int64         `json:"users"`
	Groups        int64         `json:"groups"`
	Posts         int64         `json:"posts"`
	TodayMessages int64         `json:"todayMessages"`
	TodayUsers    int64         `json:"todayUsers"`
	Knowledge     int64         `json:"knowledge"`
	Activity      []DayActivity `json:"activity"`
	GeneratedAt   time.Time     `json:"generatedAt"`
}

// Health 描述单项依赖的即时探测结果。
type Health struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// System 展示当前 Go 进程状态及运维工具入口，不代表宿主机或容器内存。
type System struct {
	UptimeSeconds int64     `json:"uptimeSeconds"`
	HeapBytes     uint64    `json:"heapBytes"`
	Goroutines    int       `json:"goroutines"`
	OnlineUsers   int       `json:"onlineUsers"`
	Services      []Health  `json:"services"`
	GrafanaURL    string    `json:"grafanaUrl"`
	KibanaURL     string    `json:"kibanaUrl"`
	GeneratedAt   time.Time `json:"generatedAt"`
}
