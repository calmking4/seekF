package adminservice

import (
	"context"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"seekF-backend/internal/configs"
	admindao "seekF-backend/internal/dao/admin_dao"
	adminresp "seekF-backend/internal/dto/admin"
	"seekF-backend/internal/models"
	dependencyhealth "seekF-backend/internal/pkg/observability/health"
	"seekF-backend/internal/pkg/websocket"
)

// AdminService 聚合只读管理数据及轻量依赖探测。
type AdminService struct {
	dao       *admindao.AdminDAO
	startedAt time.Time
	mutex     sync.Mutex
	cache     *adminresp.Overview
}

// NewAdminService 创建管理服务，依赖探测由观测模块统一处理。
func NewAdminService(dao *admindao.AdminDAO) *AdminService {
	return &AdminService{dao: dao, startedAt: time.Now()}
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
	result, err := s.dao.Overview(ctx, configs.GetConfig().AlipayConfig.Environment)
	if err != nil {
		return nil, err
	}
	s.cache = result
	return result, nil
}

// System 返回当前进程状态及六项依赖的并行即时探测结果。
func (s *AdminService) System(ctx context.Context) *adminresp.System {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	result := &adminresp.System{UptimeSeconds: int64(time.Since(s.startedAt).Seconds()), HeapBytes: mem.HeapAlloc, Goroutines: runtime.NumGoroutine(), GrafanaURL: publicURL("ADMIN_GRAFANA_URL", "http://localhost:3001"), KibanaURL: publicURL("ADMIN_KIBANA_URL", "http://localhost:5601"), GeneratedAt: time.Now()}
	if websocket.ChatServer != nil {
		result.OnlineUsers = websocket.ChatServer.OnlineCount()
	}
	for _, status := range dependencyhealth.Check(ctx, s.dao.Ping) {
		result.Services = append(result.Services, adminresp.Health{Name: status.Name, Status: status.Status, Detail: status.Detail})
	}
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
