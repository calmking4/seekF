package health

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"seekF-backend/internal/pkg/db"
	"seekF-backend/internal/pkg/kafka"
	"seekF-backend/internal/pkg/redis"
)

// probeHTTPClient 复用探测连接并禁止重定向，单次请求最多等待两秒。
var probeHTTPClient = &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}}

// Check 并行检查六项依赖，MySQL连接探测由调用方提供以保持原有DAO连接。
func Check(ctx context.Context, pingMySQL func(context.Context) error) []Result {
	return collectDependencyHealth(ctx, []healthProbe{
		{name: "MySQL", unavailable: "数据库连接失败或探测超时", check: func(ctx context.Context) (string, error) {
			return "数据库连接正常", pingMySQL(ctx)
		}},
		{name: "Prometheus", unavailable: "采集服务无法访问，请检查后端配置", check: prometheusHealth},
		{name: "Redis", unavailable: "缓存服务连接失败或探测超时", check: func(ctx context.Context) (string, error) {
			return "缓存服务命令响应正常", redis.Ping(ctx)
		}},
		{name: "Kafka", unavailable: "消息服务连接失败、元数据不可读或探测超时", check: func(ctx context.Context) (string, error) {
			return "消息服务节点元数据读取正常", kafka.KafkaServiceInstance.Ping(ctx)
		}},
		{name: "Elasticsearch", unavailable: "搜索服务不可用、集群异常或探测超时", check: elasticsearchHealth},
		{name: "Qdrant", unavailable: "向量服务未初始化、连接失败或探测超时", check: func(ctx context.Context) (string, error) {
			return "向量服务健康检查通过", db.GetQdrant().Ping(ctx)
		}},
	})
}

// prometheusHealth 检查采集服务就绪状态，不将其等同于后端采集成功。
func prometheusHealth(ctx context.Context) (string, error) {
	base := strings.TrimSpace(os.Getenv("ADMIN_PROMETHEUS_URL"))
	if base == "" {
		base = "http://localhost:9090"
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("Prometheus地址无效")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/-/ready", nil)
	if err != nil {
		return "", fmt.Errorf("创建Prometheus探测请求失败: %w", err)
	}
	response, err := probeHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("探测Prometheus失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Prometheus服务未就绪")
	}
	return "采集服务已就绪；请在 Prometheus Targets 确认后端采集状态", nil
}

// elasticsearchHealth 读取集群状态，允许副本未分配的黄色状态并说明其限制。
func elasticsearchHealth(ctx context.Context) (string, error) {
	if db.ESClient == nil {
		return "", fmt.Errorf("Elasticsearch客户端未初始化")
	}
	response, err := db.ESClient.Cluster.Health(
		db.ESClient.Cluster.Health.WithContext(ctx),
		db.ESClient.Cluster.Health.WithTimeout(time.Second),
		db.ESClient.Cluster.Health.WithMasterTimeout(time.Second),
	)
	if err != nil {
		return "", fmt.Errorf("探测Elasticsearch失败: %w", err)
	}
	defer response.Body.Close()
	if response.IsError() {
		return "", fmt.Errorf("Elasticsearch健康检查返回异常")
	}
	var health struct {
		Status   string `json:"status"`
		TimedOut bool   `json:"timed_out"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&health); err != nil {
		return "", fmt.Errorf("解析Elasticsearch健康状态失败: %w", err)
	}
	if health.TimedOut {
		return "", fmt.Errorf("Elasticsearch集群健康查询超时")
	}
	switch health.Status {
	case "green":
		return "搜索服务连接正常，主分片与副本均已分配", nil
	case "yellow":
		return "搜索服务可用，部分副本未分配（黄色状态）", nil
	default:
		return "", fmt.Errorf("Elasticsearch集群状态异常")
	}
}
