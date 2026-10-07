export function useNotification() {
  const unreadCount = useState('notification-unread', () => 0)
  const unreadCategories = useState('notification-categories', () => ({ comments: 0, likes: 0, follows: 0 }))
  const revision = useState('notification-revision', () => 0)
  const readBeforeIds = ref({ comments: 0, likes: 0, follows: 0 })
  const pendingCategories = new Map()
  const fetchUnreadCount = async () => {
    const current = ++revision.value
    try {
      const res = await useApi$('/user/notification/unread-count')
      if (res.code !== 200) throw new Error(res.message || '获取未读通知数失败')
      if (current !== revision.value) return
      unreadCount.value = res.data?.unread_count || 0
      unreadCategories.value = { comments: 0, likes: 0, follows: 0, ...res.data?.categories }
    } catch (error) { console.error('获取未读通知数失败:', error) }
  }
  const markCategoryAsRead = (category) => {
    if (pendingCategories.has(category)) return pendingCategories.get(category)
    const pending = (async () => {
      const res = await useApi$('/user/notification/read-category', { body: { category } })
      if (res.code !== 200) throw new Error(res.message || '标记分类通知已读失败')
      revision.value++
      readBeforeIds.value[category] = Math.max(readBeforeIds.value[category], res.data?.read_before_id || 0)
      unreadCount.value = Math.max(0, unreadCount.value - unreadCategories.value[category])
      unreadCategories.value[category] = 0
      // 重新获取统计，保留标记期间新到的通知红点。
      await fetchUnreadCount()
    })().finally(() => pendingCategories.delete(category))
    pendingCategories.set(category, pending)
    return pending
  }
  const decrementUnread = (count = 1, category) => {
    revision.value++
    unreadCount.value = Math.max(0, unreadCount.value - count)
    if (category) unreadCategories.value[category] = Math.max(0, unreadCategories.value[category] - count)
  }
  return { unreadCount, unreadCategories, readBeforeIds, fetchUnreadCount, markCategoryAsRead, decrementUnread }
}
