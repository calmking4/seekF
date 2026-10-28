package constants

// 发现页后台事件处理目标。
const (
	// 搜索同步目标，兼容历史事件默认值。
	DiscoverTargetSearch int8 = 0
	// 关注收件箱分发目标。
	DiscoverTargetFeed int8 = 1
)

// 关注流默认分发策略。
const (
	// 未配置时的大粉丝作者阈值，单位人。
	DiscoverDistributionThreshold = 500
	// 未配置时收件箱候选保留窗口，单位天。
	DiscoverDistributionDays = 30
)

// 关注收件箱维护批量。
const (
	// 每轮存储维护最多清理的候选数量，单位行，与发帖分发无关。
	DiscoverInboxCleanupLimit = 200
)
