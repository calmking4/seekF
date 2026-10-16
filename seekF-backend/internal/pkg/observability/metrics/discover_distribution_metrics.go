package metrics

import (
	"context"
	"sync"
	"time"

	"seekF-backend/internal/pkg/db"

	"github.com/prometheus/client_golang/prometheus"
)

var distributionOnce sync.Once
var distributionEvent *prometheus.HistogramVec
var distributionWrites prometheus.Counter
var followingQuery *prometheus.HistogramVec

// initDistributionMetrics 创建进程内共享的分发与关注读取指标。
func initDistributionMetrics() {
	distributionOnce.Do(func() {
		distributionEvent = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "seekf_discover_distribution_event_seconds", Help: "关注分发事件处理耗时，单位秒。", Buckets: prometheus.DefBuckets}, []string{"result"})
		distributionWrites = prometheus.NewCounter(prometheus.CounterOpts{Name: "seekf_discover_distribution_inbox_writes_total", Help: "消费事务已提交的收件箱新增行数，单位行。"})
		followingQuery = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "seekf_discover_following_query_seconds", Help: "关注流候选查询耗时，单位秒。", Buckets: prometheus.DefBuckets}, []string{"mode", "result"})
	})
}

// distributionCollector 合并分发积压、收件箱规模及共享运行指标。
type distributionCollector struct{ pending, inbox, success *prometheus.Desc }

// NewDiscoverDistributionCollector 创建关注分发采集器，统一由路由注册。
func NewDiscoverDistributionCollector() prometheus.Collector {
	initDistributionMetrics()
	return &distributionCollector{
		pending: prometheus.NewDesc("seekf_discover_distribution_pending_events", "关注分发未完成事件数。", nil, nil),
		inbox:   prometheus.NewDesc("seekf_discover_distribution_inbox_rows", "关注收件箱当前候选行数。", nil, nil),
		success: prometheus.NewDesc("seekf_discover_distribution_collect_success", "关注分发存储采集是否成功。", nil, nil),
	}
}

// Describe 声明分发存储与进程运行指标。
func (c *distributionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.pending
	ch <- c.inbox
	ch <- c.success
	distributionEvent.Describe(ch)
	distributionWrites.Describe(ch)
	followingQuery.Describe(ch)
}

// Collect 查询分发积压和收件箱规模，失败时显式报告采集失败。
func (c *distributionCollector) Collect(ch chan<- prometheus.Metric) {
	distributionEvent.Collect(ch)
	distributionWrites.Collect(ch)
	followingQuery.Collect(ch)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var row struct {
		Pending int64
		Inbox   int64
	}
	err := db.GormDB.WithContext(ctx).Raw(`SELECT (SELECT COUNT(*) FROM discover_outbox WHERE target=1 AND status IN (0,1)) AS pending,(SELECT COUNT(*) FROM discover_feed_inbox) AS inbox`).Scan(&row).Error
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(c.pending, prometheus.GaugeValue, float64(row.Pending))
	ch <- prometheus.MustNewConstMetric(c.inbox, prometheus.GaugeValue, float64(row.Inbox))
}

// ObserveDiscoverDistributionEvent 记录完整分发事件提交结果，失败事务不累计写入数量。
func ObserveDiscoverDistributionEvent(elapsed time.Duration, inserted int64, err error) {
	initDistributionMetrics()
	result := "success"
	if err != nil {
		result = "error"
	} else {
		distributionWrites.Add(float64(inserted))
	}
	distributionEvent.WithLabelValues(result).Observe(elapsed.Seconds())
}

// ObserveDiscoverFollowingQuery 记录候选查询方式、结果与耗时。
func ObserveDiscoverFollowingQuery(mode string, elapsed time.Duration, err error) {
	initDistributionMetrics()
	result := "success"
	if err != nil {
		result = "error"
	}
	followingQuery.WithLabelValues(mode, result).Observe(elapsed.Seconds())
}
