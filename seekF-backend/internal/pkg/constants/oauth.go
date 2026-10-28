package constants

// 第三方登录昵称生成限制。
const (
	// 用户昵称最大字符数，与 user_info.nickname 字段长度一致。
	OAuthNicknameMaxLength = 20
	// 第三方首次登录最多尝试的昵称数量，包含原昵称。
	OAuthNicknameMaxAttempts = 1000
)
