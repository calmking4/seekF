<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部导航栏 -->
    <header class="sticky top-0 z-20 bg-white shadow-sm px-4 py-3">
      <div class="flex items-center gap-3">
        <Icon name="mdi:arrow-left" class="text-xl cursor-pointer text-gray-600" @click="navigateTo('/discover')" />
        <span class="text-base font-medium text-gray-800">{{ profile.nickname || '用户主页' }}</span>
      </div>
    </header>

    <!-- 用户信息卡片 -->
    <div class="bg-white px-5 py-6">
      <!-- 头像和基本信息 -->
      <div class="flex items-start gap-4">
        <img
          v-if="profile.avatar"
          :src="profile.avatar"
          class="w-16 h-16 rounded-full object-cover flex-shrink-0"
        />
        <div
          v-else
          class="w-16 h-16 rounded-full flex items-center justify-center text-white text-xl flex-shrink-0"
          :style="{ backgroundColor: getAvatarColor(profile.uuid) }"
        >
          {{ (profile.nickname || '?').slice(0, 1) }}
        </div>

        <div class="flex-1 min-w-0">
          <h2 class="text-lg font-semibold text-gray-900">{{ profile.nickname }}</h2>
          <p class="text-sm text-gray-500 mt-1 line-clamp-2">
            {{ profile.signature || '这个人很懒，什么都没写~' }}
          </p>
        </div>
      </div>

      <!-- 统计数据 -->
      <div class="flex items-center gap-6 mt-4">
        <div class="text-center">
          <div class="text-lg font-bold text-gray-900">{{ profile.post_count || 0 }}</div>
          <div class="text-xs text-gray-500">笔记</div>
        </div>
        <div class="text-center cursor-pointer" @click="goToFollowing">
          <div class="text-lg font-bold text-gray-900">{{ profile.following_count || 0 }}</div>
          <div class="text-xs text-gray-500">关注</div>
        </div>
        <div class="text-center cursor-pointer" @click="goToFollowers">
          <div class="text-lg font-bold text-gray-900">{{ profile.follower_count || 0 }}</div>
          <div class="text-xs text-gray-500">粉丝</div>
        </div>
        <div class="text-center">
          <div class="text-lg font-bold text-gray-900">{{ profile.total_likes || 0 }}</div>
          <div class="text-xs text-gray-500">获赞</div>
        </div>
      </div>

      <!-- 操作按钮（非自己时显示） -->
      <div v-if="!isSelf" class="flex items-center gap-3 mt-4">
        <button
          class="flex-1 py-2 rounded-full text-sm font-medium transition-colors"
          :class="isFollowing
            ? 'bg-gray-100 text-gray-600'
            : 'bg-[#60a5fa] text-white hover:bg-[#4b91e8]'"
          @click="handleToggleFollow"
        >
          {{ isFollowing ? '已关注' : '关注' }}
        </button>
        <button
          class="flex-1 py-2 rounded-full text-sm font-medium border border-gray-300 text-gray-600 hover:bg-gray-50 transition-colors"
          @click="goToChat"
        >
          发消息
        </button>
      </div>
    </div>

    <!-- Tab 切换 -->
    <div class="bg-white mt-2 border-b border-gray-100">
      <div class="flex">
        <div
          class="flex-1 text-center py-3 text-sm font-medium cursor-pointer transition-colors"
          :class="activeTab === 'posts' ? 'text-[#60a5fa] border-b-2 border-[#60a5fa]' : 'text-gray-500'"
          @click="activeTab = 'posts'"
        >
          笔记
        </div>
        <div
          class="flex-1 text-center py-3 text-sm font-medium cursor-pointer transition-colors"
          :class="activeTab === 'liked' ? 'text-[#60a5fa] border-b-2 border-[#60a5fa]' : 'text-gray-500'"
          @click="activeTab = 'liked'"
        >
          获赞
        </div>
      </div>
    </div>

    <!-- 帖子瀑布流 -->
    <div class="p-4">
      <div v-if="loading && posts.length === 0" class="text-center py-10 text-gray-400">
        加载中...
      </div>

      <div v-else-if="posts.length === 0" class="text-center py-10 text-gray-400">
        暂无笔记
      </div>

      <div v-else class="flex gap-4">
        <div
          v-for="(column, columnIndex) in columns"
          :key="columnIndex"
          class="flex-1 flex flex-col space-y-4"
        >
          <div
            v-for="item in column.items"
            :key="item.uuid"
            class="bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer"
            @click="navigateTo(`/discover/detail?uuid=${item.uuid}`)"
          >
            <!-- 图片 -->
            <div class="w-full relative">
              <img
                v-if="item.media_type !== 1"
                :src="item.first_url || item.cover_url"
                :alt="item.title"
                class="w-full object-cover"
                :style="{ height: `${item.height || 200}px` }"
                loading="lazy"
              />
              <div v-else class="relative" :style="{ height: `${item.height || 200}px` }">
                <img
                  v-if="item.cover_url"
                  :src="item.cover_url"
                  :alt="item.title"
                  class="w-full h-full object-cover"
                />
                <div v-else class="w-full h-full bg-gray-200 flex items-center justify-center">
                  <Icon name="mdi:play-circle" class="text-4xl text-white" />
                </div>
              </div>
            </div>

            <!-- 卡片内容 -->
            <div class="p-3">
              <h3 class="text-sm font-medium line-clamp-2 mb-2">{{ item.title }}</h3>
              <div class="flex items-center justify-between text-xs text-gray-500">
                <div class="flex items-center gap-1">
                  <Icon name="mdi:eye-outline" class="text-sm" />
                  <span>{{ item.view_count || 0 }}</span>
                </div>
                <div class="flex items-center gap-1" :class="{ 'text-red-500': item.is_liked }">
                  <Icon :name="item.is_liked ? 'solar:heart-angle-bold' : 'mdi:heart-outline'" class="text-sm" />
                  <span>{{ item.like_count || 0 }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 加载更多 -->
      <div v-if="loading && posts.length > 0" class="text-center py-4 text-gray-400 text-sm">
        加载中...
      </div>
      <div v-else-if="noMore && posts.length > 0" class="text-center py-4 text-gray-400 text-sm">
        没有更多了
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'

const route = useRoute()
const userId = route.params.id

// 页面级 SEO
useSeoMeta({
  title: () => profile.value.nickname ? `${profile.value.nickname} 的主页` : '用户主页',
  description: () => profile.value.signature || '查看用户主页',
})

// 响应式数据
const profile = ref({
  uuid: '',
  nickname: '',
  avatar: '',
  signature: '',
  post_count: 0,
  total_likes: 0,
  is_followed: false,
  is_friend: false,
  following_count: 0,
  follower_count: 0,
})
const posts = ref([])
const columns = ref([])
const loading = ref(false)
const noMore = ref(false)
const page = ref(1)
const pageSize = 12
const activeTab = ref('posts')
const isSelf = ref(false)
const isFollowing = ref(false)

// 头像颜色
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

// 瀑布流列数
const getColumnCount = () => {
  if (typeof window === 'undefined') return 2
  const width = window.innerWidth
  if (width >= 1280) return 4
  if (width >= 768) return 3
  return 2
}

// 分配帖子到各列
const distributeToColumns = (items) => {
  const colCount = getColumnCount()
  const cols = Array.from({ length: colCount }, () => ({ items: [], height: 0 }))

  items.forEach((item) => {
    // 找最短列
    let minIdx = 0
    for (let i = 1; i < cols.length; i++) {
      if (cols[i].height < cols[minIdx].height) minIdx = i
    }
    cols[minIdx].items.push(item)
    cols[minIdx].height += (item.height || 200) + 60
  })

  return cols
}

// 获取用户主页数据
const fetchProfile = async () => {
  loading.value = true
  try {
    const { data } = await useApi('/user/discover/profile', {
      body: { user_id: userId, page: page.value, page_size: pageSize },
    })

    if (data.value?.code === 200) {
      const d = data.value.data
      profile.value = {
        uuid: d.uuid,
        nickname: d.nickname,
        avatar: d.avatar,
        signature: d.signature,
        post_count: d.post_count,
        total_likes: d.total_likes,
        is_followed: d.is_followed,
        is_friend: d.is_friend,
        following_count: d.following_count || 0,
        follower_count: d.follower_count || 0,
      }

      // 检查是否是自己
      const { data: myData } = await useApi('/user/userinfo/getMyInfo')
      if (myData.value?.code === 200) {
        isSelf.value = myData.value.data.uuid === userId
      }

      // 获取关注状态
      if (!isSelf.value) {
        const { data: followData } = await useApi('/user/follow/counts', {
          body: { user_id: userId },
        })
        if (followData.value?.code === 200) {
          isFollowing.value = followData.value.data.is_following || false
        }
      }

      // 处理帖子列表
      const newPosts = (d.posts || []).map((p) => ({
        ...p,
        height: 180 + Math.floor(Math.random() * 120), // 模拟高度，实际应从图片获取
      }))

      if (page.value === 1) {
        posts.value = newPosts
      } else {
        posts.value = [...posts.value, ...newPosts]
      }

      if (newPosts.length < pageSize) {
        noMore.value = true
      }

      columns.value = distributeToColumns(posts.value)
    }
  } catch (e) {
    console.error('获取用户主页失败:', e)
  } finally {
    loading.value = false
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
    fetchProfile()
  }
}

// 操作
const handleToggleFollow = async () => {
  try {
    const { data } = await useApi('/user/follow/toggle', {
      body: { follow_user_id: userId },
    })
    if (data.value?.code === 200) {
      isFollowing.value = !isFollowing.value
      // 更新粉丝数（当前用户的关注操作影响对方的粉丝数）
      if (isFollowing.value) {
        profile.value.follower_count = (profile.value.follower_count || 0) + 1
      } else {
        profile.value.follower_count = Math.max(0, (profile.value.follower_count || 0) - 1)
      }
    }
  } catch (e) {
    console.error('关注操作失败:', e)
  }
}

const goToChat = () => {
  navigateTo(`/chat?userId=${userId}`)
}

const goToFollowing = () => {
  navigateTo(`/user/${userId}/following`)
}

const goToFollowers = () => {
  navigateTo(`/user/${userId}/followers`)
}

// Tab 切换时重新加载
watch(activeTab, () => {
  page.value = 1
  noMore.value = false
  posts.value = []
  columns.value = []
  fetchProfile()
})

onMounted(() => {
  fetchProfile()
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style scoped>
.fade-in {
  animation: fadeIn 0.3s ease-in;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
