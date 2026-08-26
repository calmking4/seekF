import { ref, onMounted } from 'vue'

const unreadCount = ref(0)

export function useNotification() {
  // 获取未读通知数
  const fetchUnreadCount = async () => {
    try {
      const { data } = await useApi('/user/notification/unread-count')
      if (data.value?.code === 200) {
        unreadCount.value = data.value.data.unread_count || 0
      }
    } catch (e) {
      console.error('获取未读通知数失败:', e)
    }
  }

  // 标记所有已读
  const markAllAsRead = async () => {
    try {
      await useApi('/user/notification/read-all')
      unreadCount.value = 0
    } catch (e) {
      console.error('全部标记已读失败:', e)
    }
  }

  // 减少未读数（标记单条已读时调用）
  const decrementUnread = (count = 1) => {
    unreadCount.value = Math.max(0, unreadCount.value - count)
  }

  // 初始化时获取未读数
  onMounted(() => {
    fetchUnreadCount()
  })

  return {
    unreadCount,
    fetchUnreadCount,
    markAllAsRead,
    decrementUnread,
  }
}
