package constants

import "time"

// 帖子搜索候选规模。
const (
	// 单路搜索最多合并10000条候选，超出后使用数据库完整分页。
	DiscoverSearchCandidateLimit = 10000
)

// 帖子发布与审核状态。
const (
	// 帖子状态：已发布、已下架、审核中、未通过。
	DiscoverPostPublished int8 = 0
	DiscoverPostOffline   int8 = 1
	DiscoverPostPending   int8 = 2
	DiscoverPostRejected  int8 = 3
)

// 推荐召回、翻页快照与用户兴趣窗口。
const (
	// 每种召回最多200条候选，合并后不超过600条。
	DiscoverRankingPoolLimit = 200
	// 一轮推荐或热榜翻页顺序保留30分钟。
	DiscoverRankingSessionTTL = 30 * time.Minute
	// 用户兴趣使用最近30天互动。
	DiscoverInterestWindow = 30 * 24 * time.Hour
)

// 热榜更新周期与互动统计窗口。
const (
	// 热榜按北京时间每两小时更新一个排名版本。
	DiscoverHotRefresh = 2 * time.Hour
	// 热榜统计近7天真实互动。
	DiscoverActivityWindow = 7 * 24 * time.Hour
)

// 互动排名权重。
const (
	// 点击权重，同一用户同一帖子在窗口内计一次。
	DiscoverClickWeight = 1.0
	// 当前有效点赞权重。
	DiscoverLikeWeight = 3.0
	// 当前有效收藏权重，同一用户多收藏夹计一次。
	DiscoverCollectWeight = 5.0
	// 有效评论用户权重，同一用户多条评论计一次。
	DiscoverCommentWeight = 4.0
)

// 不感兴趣反馈范围。
const (
	// 表示对单条帖子不感兴趣。
	DiscoverFeedbackPost = "post"
	// 表示对作者全部帖子不感兴趣，仅影响推荐。
	DiscoverFeedbackAuthor = "author"
)

// 创作者数据中心事件类型。
const (
	// 关注关系增减事件，增量为1或-1。
	CreatorAnalyticsFollower = "follower"
	// 成功分享事件，增量为1。
	CreatorAnalyticsShare = "share"
	// 数据中心埋点启用标记，不参与指标汇总。
	CreatorAnalyticsStarted = "started"
)

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

// 帖子浏览量更新协程池。
const (
	// 浏览量更新的共用工作协程数量。
	DiscoverViewWorkers = 4
	// 浏览量更新队列的最大等待任务数。
	DiscoverViewQueueSize = 256
	// 单次浏览量更新超时。
	DiscoverViewTimeout = 5 * time.Second
)

// 发现页后台事件状态。
const (
	// 事件尚待投递Kafka。
	DiscoverEventPending int8 = 0
	// 事件已投递Kafka，尚待搜索同步或关注分发完成。
	DiscoverEventPublished int8 = 1
	// 事件处理已完成，重复消费直接跳过。
	DiscoverEventCompleted int8 = 2
)
