// sortChatSessions 置顶按置顶时间排序，普通会话按最新活动排序，同值保留原顺序。
export function sortChatSessions(sessions) {
  const timestamp = value => Date.parse(value || '') || 0
  return [...sessions].sort((left, right) => Number(right.isPinned) - Number(left.isPinned) ||
    (left.isPinned ? timestamp(right.pinnedAt) - timestamp(left.pinnedAt) : right.lastActivity - left.lastActivity))
}
