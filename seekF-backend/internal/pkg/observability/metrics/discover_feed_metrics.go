package metrics

import (
	"context"
	"sync"
	"time"

	"seekF-backend/internal/pkg/constants"
	"seekF-backend/internal/pkg/db"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	discoverActionsOnce   sync.Once
	discoverFallbacksOnce sync.Once
	discoverActions       *prometheus.CounterVec
	discoverFeedFallbacks *prometheus.CounterVec
)

// NewDiscoverActionsCollector 创建并复用发现页成功行为计数器，供路由统一注册。
func NewDiscoverActionsCollector() *prometheus.CounterVec {
	discoverActionsOnce.Do(func() {
		discoverActions = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "seekf_discover_actions_total", Help: "成功新增的发现页行为数量，单位次。"}, []string{"action"})
	})
	return discoverActions
}

// NewDiscoverFeedFallbacksCollector 创建并复用推荐与热榜降级计数器。
func NewDiscoverFeedFallbacksCollector() *prometheus.CounterVec {
	discoverFallbacksOnce.Do(func() {
		discoverFeedFallbacks = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "seekf_discover_feed_fallbacks_total", Help: "推荐或热榜首屏降级时间流次数。"}, []string{"source"})
	})
	return discoverFeedFallbacks
}

// RecordDiscoverClick 累计一次数据库实际新增且已提交的有效点击。
func RecordDiscoverClick() { NewDiscoverActionsCollector().WithLabelValues("click").Inc() }

// RecordDiscoverFeedFallback 累计一次推荐或热榜首屏降级，来源限定为两个业务频道。
func RecordDiscoverFeedFallback(source string) {
	if source == "recommended" || source == "hot" {
		NewDiscoverFeedFallbacksCollector().WithLabelValues(source).Inc()
	}
}

// discoverActivityCollector 从真实业务记录导出近7天有效互动，避免以请求次数替代有效数据。
type discoverActivityCollector struct {
	counts  *prometheus.Desc
	success *prometheus.Desc
}

// NewDiscoverActivityCollector 创建发现页真实互动窗口采集器，按用户与帖子去重。
func NewDiscoverActivityCollector() prometheus.Collector {
	return &discoverActivityCollector{
		counts:  prometheus.NewDesc("seekf_discover_interactions_window", "近7天当前公开帖子有效互动数量，按用户和帖子去重，单位次。", []string{"action"}, nil),
		success: prometheus.NewDesc("seekf_discover_interactions_collect_success", "真实互动采集是否成功，1成功、0失败。", nil, nil),
	}
}

// Describe 声明真实互动窗口指标。
func (c *discoverActivityCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.counts
	ch <- c.success
}

// Collect 合并四类真实互动，一次查询并排除作者自身互动和AI自动评论。
func (c *discoverActivityCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	now := time.Now()
	from := now.Add(-constants.DiscoverActivityWindow)
	var rows []struct {
		Action string
		Total  int64
	}
	err := db.GormDB.WithContext(ctx).Raw(`SELECT a.action,COUNT(*) AS total FROM (
SELECT 'click' AS action,c.user_id,c.post_uuid FROM discover_click c WHERE c.created_at >= ? AND c.created_at <= ? GROUP BY c.user_id,c.post_uuid
UNION ALL SELECT 'like',user_id,target_uuid FROM discover_like WHERE created_at >= ? AND created_at <= ? GROUP BY user_id,target_uuid
UNION ALL SELECT 'collect',user_id,target_uuid FROM discover_collection WHERE created_at >= ? AND created_at <= ? GROUP BY user_id,target_uuid
UNION ALL SELECT 'comment',c.user_id,p.uuid FROM discover_comment c JOIN discover_post p ON p.id=c.post_id WHERE c.created_at >= ? AND c.created_at <= ? AND c.deleted_at IS NULL AND c.user_id <> ? GROUP BY c.user_id,p.uuid
) a JOIN discover_post p ON p.uuid=a.post_uuid WHERE p.status=0 AND p.visibility=0 AND p.deleted_at IS NULL AND a.user_id <> p.user_id GROUP BY a.action`, from, now, from, now, from, now, from, now, constants.AIAssistantUserId).Scan(&rows).Error
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 0)
		return
	}
	counts := map[string]int64{}
	for _, row := range rows {
		counts[row.Action] = row.Total
	}
	for _, action := range []string{"click", "like", "collect", "comment"} {
		ch <- prometheus.MustNewConstMetric(c.counts, prometheus.GaugeValue, float64(counts[action]), action)
	}
	ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 1)
}
