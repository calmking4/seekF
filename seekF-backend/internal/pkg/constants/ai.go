package constants

import "time"

// AI回答生成与回放时限。
const (
	// 单次后台生成最多运行15分钟。
	AIStreamTimeout = 15 * time.Minute
	// 回答事件在最近写入后保留30分钟。
	AIStreamTTL = 30 * time.Minute
	// 后台任务每5秒写入一次心跳。
	AIStreamHeartbeat = 5 * time.Second
	// 超过30秒未更新且未落库的任务视为中断。
	AIStreamStaleTimeout = 30 * time.Second
	// 单次Redis读写最多等待3秒。
	AIStreamRedisTimeout = 3 * time.Second
	// 每轮最多读取100条事件。
	AIStreamReadCount = 100
	// 回答事件流的Redis键前缀。
	AIStreamKeyPrefix = "ai:stream:"
)

// AI助手身份与展示信息。
const (
	// AI评论使用的助手用户编号。
	AIAssistantUserId = "AI_ASSISTANT"
	// AI助手的展示名称。
	AIAssistantName = "AI助手"
	// AI助手的默认头像地址。
	AIAssistantAvatar = "https://seekf.oss-cn-shenzhen.aliyuncs.com/common/ai_avatar/AI%E5%8A%A9%E6%89%8B%E5%A4%B4%E5%83%8F.png"
)

// AI评论回复协程池。
const (
	// AI评论工作协程数量及批次最大任务数。
	AICommentWorkers = 4
	// AI评论收集批次的等待时间。
	AICommentBatchWait = 100 * time.Millisecond
)
