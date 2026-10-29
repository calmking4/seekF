package constants

import "time"

// 私聊消息撤回状态与时限。
const (
	// 已撤回的私聊占位消息。
	MessageRecalled int8 = 9
	// 私聊消息允许撤回的最长时间。
	MessageRecallLimit = 2 * time.Minute
)

// 网页私聊输入与连接限制。
const (
	// 文字消息最多字符数。
	PrivateChatContentLimit = 10000
	// 请求编号最多字节数。
	PrivateChatRequestIDLimit = 64
	// 单个WebSocket请求最多字节数。
	PrivateChatReadLimit = 128 * 1024
)

// 网页连接发送队列。
const (
	// 每条网页连接最多等待发送的消息数量，单位条。
	CHANNEL_SIZE = 100
)
