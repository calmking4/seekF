package metrics

import (
	"context"
	"time"

	"seekF-backend/internal/models"
	"seekF-backend/internal/pkg/db"

	"github.com/prometheus/client_golang/prometheus"
)

// discoverOutboxCollector 自定义Prometheus采集器，一次数据库查询生成多个Outbox指标。
type discoverOutboxCollector struct {
	pending *prometheus.Desc
	oldest  *prometheus.Desc
	retries *prometheus.Desc
	success *prometheus.Desc
}

// NewDiscoverOutboxCollector 创建自定义采集器，导出事件积压、等待秒数、失败次数和采集状态。
func NewDiscoverOutboxCollector() prometheus.Collector {
	return &discoverOutboxCollector{
		pending: prometheus.NewDesc("seekf_discover_outbox_pending_events", "当前未完成发现页事件数。", []string{"state"}, nil),
		oldest:  prometheus.NewDesc("seekf_discover_outbox_oldest_age_seconds", "最长未完成事件等待时长，单位秒。", nil, nil),
		retries: prometheus.NewDesc("seekf_discover_outbox_retry_failures", "当前未完成事件累计失败次数，单位次。", nil, nil),
		success: prometheus.NewDesc("seekf_discover_outbox_collect_success", "事件积压采集是否成功，1成功、0失败。", nil, nil),
	}
}

// Describe 实现自定义Collector接口，向注册表声明指标名称、说明及标签，不采集数值。
func (c *discoverOutboxCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.pending
	ch <- c.oldest
	ch <- c.retries
	ch <- c.success
}

// Collect 实现自定义Collector接口，一次查询生成当前指标，数据库错误时仅报告采集失败。
func (c *discoverOutboxCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var rows []struct {
		Status  int8
		Total   int64
		Retries int64
		Oldest  *time.Time
	}
	err := db.GormDB.WithContext(ctx).Model(&models.DiscoverOutbox{}).Where("status IN (0,1)").Select("status,COUNT(*) AS total,SUM(retry_count) AS retries,MIN(created_at) AS oldest").Group("status").Scan(&rows).Error
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 0)
		return
	}
	counts := map[int8]int64{}
	age := float64(0)
	failures := int64(0)
	for _, row := range rows {
		counts[row.Status] = row.Total
		failures += row.Retries
		if row.Oldest != nil {
			wait := time.Since(*row.Oldest).Seconds()
			if wait > age {
				age = wait
			}
		}
	}
	ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(c.pending, prometheus.GaugeValue, float64(counts[0]), "pending")
	ch <- prometheus.MustNewConstMetric(c.pending, prometheus.GaugeValue, float64(counts[1]), "published")
	ch <- prometheus.MustNewConstMetric(c.oldest, prometheus.GaugeValue, age)
	ch <- prometheus.MustNewConstMetric(c.retries, prometheus.GaugeValue, float64(failures))
}
