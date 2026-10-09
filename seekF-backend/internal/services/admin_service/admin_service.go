package adminservice

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	admindao "seekF-backend/internal/dao/admin_dao"
	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/websocket"
)

// AdminService 聚合只读管理数据及轻量依赖探测。
type AdminService struct {
	dao       *admindao.AdminDAO
	startedAt time.Time
	mutex     sync.Mutex
	cache     *adminresp.Overview
	client    *http.Client
}

// NewAdminService 创建管理服务，限制探测超时为两秒。
func NewAdminService(dao *admindao.AdminDAO) *AdminService {
	return &AdminService{dao: dao, startedAt: time.Now(), client: &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Identity 读取用户实时权限与账号状态，供控制器校验访问资格。
func (s *AdminService) Identity(ctx context.Context, uuid string) (*models.UserInfo, error) {
	return s.dao.FindUser(ctx, uuid)
}

// Overview 返回最多三十秒的共享业务概览，减少低内存单机的重复数据库扫描。
func (s *AdminService) Overview(ctx context.Context) (*adminresp.Overview, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.cache != nil && time.Since(s.cache.GeneratedAt) < 30*time.Second {
		return s.cache, nil
	}
	result, err := s.dao.Overview(ctx)
	if err != nil {
		return nil, err
	}
	s.cache = result
	return result, nil
}

// System 返回当前进程状态、MySQL及Prometheus的即时探测结果。
func (s *AdminService) System(ctx context.Context) *adminresp.System {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	result := &adminresp.System{UptimeSeconds: int64(time.Since(s.startedAt).Seconds()), HeapBytes: mem.HeapAlloc, Goroutines: runtime.NumGoroutine(), GrafanaURL: publicURL("ADMIN_GRAFANA_URL", "http://localhost:3001"), KibanaURL: publicURL("ADMIN_KIBANA_URL", "http://localhost:5601"), GeneratedAt: time.Now()}
	if websocket.ChatServer != nil {
		result.OnlineUsers = websocket.ChatServer.OnlineCount()
	}
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	err := s.dao.Ping(dbCtx)
	cancel()
	dbHealth := adminresp.Health{Name: "MySQL", Status: "up", Detail: "数据库连接正常"}
	if err != nil {
		dbHealth.Status = "down"
		dbHealth.Detail = "数据库连接失败"
	}
	result.Services = append(result.Services, dbHealth)
	prom := adminresp.Health{Name: "Prometheus", Status: "down", Detail: "采集服务无法访问，请检查后端配置"}
	base := publicURL("ADMIN_PROMETHEUS_URL", "http://localhost:9090")
	if base != "" {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/-/ready", nil)
		if reqErr == nil {
			response, requestErr := s.client.Do(req)
			if requestErr == nil {
				response.Body.Close()
				if response.StatusCode == http.StatusOK {
					prom.Status = "up"
					prom.Detail = "采集服务就绪；业务指标请在 Grafana 查看"
				}
			}
		}
	}
	result.Services = append(result.Services, prom)
	return result
}

// publicURL 读取固定配置中的HTTP工具入口，不接受浏览器传入地址。
func publicURL(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = fallback
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return value
}
