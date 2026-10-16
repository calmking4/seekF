package constants

const (
	// 用户联系人列表缓存前缀，后接用户ID。
	ContactUserListKeyPrefix = "contact:user:list:"
	// 用户创建的群组列表缓存前缀，后接用户ID。
	ContactOwnedGroupListKeyPrefix = "contact:group:owned:"
	// 用户加入的群组列表缓存前缀，后接用户ID。
	ContactJoinedGroupListKeyPrefix = "contact:group:joined:"
	// 用户会话列表缓存前缀，后接用户ID。
	SessionListKeyPrefix = "chat:sessions:"
)
