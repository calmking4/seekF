package constants

import "time"

// 私聊消息撤回状态与时限。
const (
	// 已撤回的私聊占位消息。
	MessageRecalled int8 = 9
	// 私聊消息允许撤回的最长时间。
	MessageRecallLimit = 2 * time.Minute
)
