package constants

import "time"

const (
	// 每种召回最多200条候选，合并后不超过600条。
	DiscoverRankingPoolLimit = 200
	// 一轮推荐或热榜翻页顺序保留30分钟。
	DiscoverRankingSessionTTL = 30 * time.Minute
	// 热榜按北京时间每两小时更新一个排名版本。
	DiscoverHotRefresh = 2 * time.Hour
	// 热榜统计近7天真实互动。
	DiscoverActivityWindow = 7 * 24 * time.Hour
	// 用户兴趣使用最近30天互动。
	DiscoverInterestWindow = 30 * 24 * time.Hour
	// 点击权重，同一用户同一帖子在窗口内计一次。
	DiscoverClickWeight = 1.0
	// 当前有效点赞权重。
	DiscoverLikeWeight = 3.0
	// 当前有效收藏权重，同一用户多收藏夹计一次。
	DiscoverCollectWeight = 5.0
	// 有效评论用户权重，同一用户多条评论计一次。
	DiscoverCommentWeight = 4.0
	// 表示对单条帖子不感兴趣。
	DiscoverFeedbackPost = "post"
	// 表示对作者全部帖子不感兴趣，仅影响推荐。
	DiscoverFeedbackAuthor = "author"
)
