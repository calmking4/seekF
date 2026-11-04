package constants

import "time"

// AI消息搜索索引后台同步时限。
const (
	// 单次索引写入或会话索引删除最多等待5秒。
	AIMessageIndexTimeout = 5 * time.Second
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
