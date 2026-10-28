package constants

// 通道缓冲配置。
const (
	CHANNEL_SIZE = 100 // 通道大小
)

// 通用错误提示。
const (
	SYSTEM_ERROR = "系统错误，请联系工作人员" // 系统错误
)

// 文件上传限制。
const (
	FILE_MAX_SIZE = 50000 // 文件最大大小
)

// Redis缓存时长。
const (
	REDIS_TIMEOUT = 5 // redis 缓存时间（分钟）
)

// AI助手身份与展示信息。
const (
	// AI助手
	AIAssistantUserId = "AI_ASSISTANT"
	AIAssistantName   = "AI助手"
	AIAssistantAvatar = "https://seekf.oss-cn-shenzhen.aliyuncs.com/common/ai_avatar/AI%E5%8A%A9%E6%89%8B%E5%A4%B4%E5%83%8F.png"
)

// 发现页后台事件状态。
const (
	// 表示事件尚待投递Kafka。
	DiscoverEventPending int8 = 0
	// 表示事件已投递Kafka，尚待搜索同步完成。
	DiscoverEventPublished int8 = 1
	// 表示搜索同步已完成，重复消费直接跳过。
	DiscoverEventCompleted int8 = 2
)
