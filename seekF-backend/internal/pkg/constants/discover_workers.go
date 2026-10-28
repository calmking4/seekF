package constants

import "time"

// 帖子浏览量更新协程池。
const (
	// 浏览量更新的共用工作协程数量。
	DiscoverViewWorkers = 4
	// 浏览量更新队列的最大等待任务数。
	DiscoverViewQueueSize = 256
	// 单次浏览量更新超时。
	DiscoverViewTimeout = 5 * time.Second
)

// AI评论回复协程池。
const (
	// AI评论工作协程数量及批次最大任务数。
	AICommentWorkers = 4
	// AI评论收集批次的等待时间。
	AICommentBatchWait = 100 * time.Millisecond
)
