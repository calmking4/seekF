<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航栏 -->
    <header class="sticky top-0 z-20 bg-white shadow-sm px-4 py-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <Icon name="mdi:arrow-left" class="text-xl cursor-pointer text-gray-600" @click="navigateTo('/discover')" />
          <span class="text-base font-medium text-gray-800">通知</span>
        </div>
        <button
          v-if="unreadCount > 0"
          class="text-sm text-[#60a5fa] hover:text-[#4b91e8] transition-colors"
          @click="handleMarkAllAsRead"
        >
          全部已读
        </button>
      </div>
    </header>

    <!-- 通知列表 -->
    <div class="p-4">
      <div v-if="loading && list.length === 0" class="text-center py-10 text-gray-400">
        加载中...
      </div>

      <div v-else-if="list.length === 0" class="text-center py-10 text-gray-400">
        暂无通知
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="item in list"
          :key="item.id"
          class="bg-white rounded-xl p-4 cursor-pointer hover:bg-gray-50 transition-colors"
          :class="{ 'bg-blue-50': !item.is_read }"
          @click="handleNotificationClick(item)"
        >
          <div class="flex items-start gap-3">
            <!-- 头像 -->
            <img
              v-if="item.actor_avatar"
              :src="item.actor_avatar"
              class="w-10 h-10 rounded-full object-cover flex-shrink-0"
            />
            <div
              v-else
              class="w-10 h-10 rounded-full flex items-center justify-center text-white text-sm flex-shrink-0"
              :style="{ backgroundColor: getAvatarColor(item.actor_id) }"
            >
              {{ (item.actor_name || '?').slice(0, 1) }}
            </div>

            <!-- 内容 -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <span class="font-medium text-gray-900 text-sm">{{ item.actor_name }}</span>
                <span class="text-gray-500 text-sm">{{ getNotificationText(item.type) }}</span>
              </div>
              <div class="text-sm text-gray-600 line-clamp-2">{{ item.content }}</div>
              <div class="text-xs text-gray-400 mt-2">{{ formatTime(item.created_at) }}</div>
            </div>

            <!-- 未读标记 -->
            <div v-if="!item.is_read" class="w-2 h-2 rounded-full bg-red-500 flex-shrink-0 mt-2"></div>
          </div>
        </div>
      </div>

      <!-- 加载更多 -->
      <div v-if="loading && list.length > 0" class="text-center py-4 text-gray-400 text-sm">
        加载中...
      </div>
      <div v-else-if="noMore && list.length > 0" class="text-center py-4 text-gray-400 text-sm">
        没有更多了
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

useSeoMeta({
  title: '通知',
})

const list = ref([])
const loading = ref(false)
const noMore = ref(false)
const page = ref(1)
const pageSize = 20
const unreadCount = ref(0)

const avatarColors = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA'
]

const getAvatarColor = (id) => {
  const str = String(id ?? '')
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = (hash * 31 + str.charCodeAt(i)) >>> 0
  }
  return avatarColors[hash % avatarColors.length]
}

// 通知类型文本
const getNotificationText = (type) => {
  const typeMap = {
    1: '赞了你的帖子',
    2: '评论了你的帖子',
    3: '回复了你的评论',
    4: '关注了你',
  }
  return typeMap[type] || '发来一条通知'
}

// 格式化时间
const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now - date

  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`

  return timeStr.split(' ')[0]
}

// 获取通知列表
const fetchList = async () => {
  loading.value = true
  try {
    const { data } = await useApi('/user/notification/list', {
      body: { page: page.value, page_size: pageSize },
    })

    if (data.value?.code === 200) {
      const newList = data.value.data.list || []
      if (page.value === 1) {
        list.value = newList
      } else {
        list.value = [...list.value, ...newList]
      }
      unreadCount.value = data.value.data.total || 0
      if (newList.length < pageSize) {
        noMore.value = true
      }
    }
  } catch (e) {
    console.error('获取通知列表失败:', e)
  } finally {
    loading.value = false
  }
}

// 获取未读数
const fetchUnreadCount = async () => {
  try {
    const { data } = await useApi('/user/notification/unread-count')
    if (data.value?.code === 200) {
      unreadCount.value = data.value.data.unread_count || 0
    }
  } catch (e) {
    console.error('获取未读数失败:', e)
  }
}

// 标记单条已读
const markAsRead = async (ids) => {
  try {
    await useApi('/user/notification/read', {
      body: { ids },
    })
    // 更新本地状态
    list.value = list.value.map(item => {
      if (ids.includes(item.id)) {
        return { ...item, is_read: true }
      }
      return item
    })
    unreadCount.value = Math.max(0, unreadCount.value - ids.length)
  } catch (e) {
    console.error('标记已读失败:', e)
  }
}

// 全部标记已读
const handleMarkAllAsRead = async () => {
  try {
    await useApi('/user/notification/read-all')
    list.value = list.value.map(item => ({ ...item, is_read: true }))
    unreadCount.value = 0
  } catch (e) {
    console.error('全部标记已读失败:', e)
  }
}

// 点击通知
const handleNotificationClick = (item) => {
  // 标记已读
  if (!item.is_read) {
    markAsRead([item.id])
  }

  // 根据类型跳转
  switch (item.type) {
    case 1: // 点赞帖子
    case 2: // 评论帖子
    case 3: // 回复评论
      if (item.target_uuid) {
        navigateTo(`/discover/detail?uuid=${item.target_uuid}`)
      }
      break
    case 4: // 关注
      if (item.actor_id) {
        navigateTo(`/user/${item.actor_id}`)
      }
      break
  }
}

// 滚动加载更多
const handleScroll = () => {
  if (loading.value || noMore.value) return
  const scrollHeight = document.documentElement.scrollHeight
  const scrollTop = window.scrollY
  const clientHeight = window.innerHeight
  if (scrollTop + clientHeight >= scrollHeight - 200) {
    page.value++
    fetchList()
  }
}

onMounted(() => {
  fetchList()
  fetchUnreadCount()
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>
