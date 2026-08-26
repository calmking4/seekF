<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航栏 -->
    <header class="sticky top-0 z-20 bg-white shadow-sm px-4 py-3">
      <div class="flex items-center gap-3">
        <Icon name="mdi:arrow-left" class="text-xl cursor-pointer text-gray-600" @click="navigateTo(`/user/${userId}`)" />
        <span class="text-base font-medium text-gray-800">关注列表</span>
      </div>
    </header>

    <!-- 列表 -->
    <div class="p-4">
      <div v-if="loading && list.length === 0" class="text-center py-10 text-gray-400">
        加载中...
      </div>

      <div v-else-if="list.length === 0" class="text-center py-10 text-gray-400">
        暂无关注
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="item in list"
          :key="item.uuid"
          class="bg-white rounded-xl p-4 flex items-center gap-3 cursor-pointer hover:bg-gray-50 transition-colors"
          @click="navigateTo(`/user/${item.uuid}`)"
        >
          <!-- 头像 -->
          <img
            v-if="item.avatar"
            :src="item.avatar"
            class="w-12 h-12 rounded-full object-cover flex-shrink-0"
          />
          <div
            v-else
            class="w-12 h-12 rounded-full flex items-center justify-center text-white text-lg flex-shrink-0"
            :style="{ backgroundColor: getAvatarColor(item.uuid) }"
          >
            {{ (item.nickname || '?').slice(0, 1) }}
          </div>

          <!-- 信息 -->
          <div class="flex-1 min-w-0">
            <div class="font-medium text-gray-900">{{ item.nickname }}</div>
            <div class="text-sm text-gray-500 truncate">{{ item.signature || '这个人很懒，什么都没写~' }}</div>
          </div>

          <!-- 好友标签 -->
          <span
            v-if="item.is_friend"
            class="text-xs px-2 py-1 rounded-full bg-blue-50 text-blue-500"
          >
            好友
          </span>
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
import { ref, onMounted } from 'vue'

const route = useRoute()
const userId = route.params.id

useSeoMeta({
  title: '关注列表',
})

const list = ref([])
const loading = ref(false)
const noMore = ref(false)
const page = ref(1)
const pageSize = 20

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

const fetchList = async () => {
  loading.value = true
  try {
    const { data } = await useApi('/user/follow/list-following', {
      body: { user_id: userId, page: page.value, page_size: pageSize },
    })

    if (data.value?.code === 200) {
      const newList = data.value.data.list || []
      if (page.value === 1) {
        list.value = newList
      } else {
        list.value = [...list.value, ...newList]
      }
      if (newList.length < pageSize) {
        noMore.value = true
      }
    }
  } catch (e) {
    console.error('获取关注列表失败:', e)
  } finally {
    loading.value = false
  }
}

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
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>
