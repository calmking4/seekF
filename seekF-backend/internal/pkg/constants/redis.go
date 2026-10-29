package constants

// 私聊会话缓存键。
const (
	// 用户会话列表缓存前缀，后接用户ID。
	SessionListKeyPrefix = "chat:sessions:"
)

// Redis通用缓存时长。
const (
	// 缓存保留时长，单位分钟。
	REDIS_TIMEOUT = 5
)
