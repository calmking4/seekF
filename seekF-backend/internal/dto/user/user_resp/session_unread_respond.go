package userresp

// SessionUnreadItem 本人私聊会话未读统计，消息主键使用字符串避免前端整数精度损失。
type SessionUnreadItem struct {
	SessionId         string `json:"session_id"`
	PeerId            string `json:"peer_id"`
	UnreadCount       int64  `json:"unread_count"`
	LatestMessageId   int64  `json:"latest_message_id,string"`
	LastReadMessageId int64  `json:"last_read_message_id,string"`
}

// SessionUnreadRespond 当前用户全部有效私聊会话的未读汇总。
type SessionUnreadRespond struct {
	List  []SessionUnreadItem `json:"list"`
	Total int64               `json:"total"`
}
