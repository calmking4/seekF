<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部搜索栏 -->
    <header class="sticky top-0 z-20 bg-white shadow-sm px-4 py-3">
      <div class="flex items-center justify-between">
        <!-- 搜索框 -->
        <div class="w-[30rem] mx-auto relative">
          <div class="flex items-center bg-gray-100 rounded-full px-3 py-2 gap-2">
            <Icon name="mdi:magnify" class="text-gray-400 text-lg flex-shrink-0" />
            <input
              ref="searchInputRef"
              v-model="searchKeyword"
              type="text"
              placeholder="搜索"
              class="bg-transparent outline-none text-sm w-full text-gray-700 placeholder-gray-400"
              @focus="showSearchPanel = true"
              @keyup.enter="executeSearch(searchKeyword)"
            />
          </div>
          <!-- 搜索联想面板 -->
          <DiscoverSearchPanel
            v-if="showSearchPanel"
            ref="searchPanelRef"
            :keyword="searchKeyword"
            @select="executeSearch"
            @clear-history="handleClearHistory"
          />
        </div>
        <!-- 右侧图标 -->
        <div class="flex items-center gap-4 text-gray-600 text-xl absolute right-4">
          <Icon name="mdi:message-outline" class="text-[#60a5fa]"/>
          <Icon name="mdi:plus-circle-outline" class="text-[#60a5fa] cursor-pointer" @click="navigateTo('/discover/create')"/>
        </div>
      </div>
    </header>

    <!-- 搜索结果内容区 -->
    <div class="p-4">
      <!-- 搜索加载中 -->
      <div v-if="searchLoading" class="py-16 flex flex-col items-center justify-center text-gray-500">
        <Icon name="uil:spinner" class="animate-spin text-3xl mb-3" />
        <span class="text-sm">搜索中...</span>
      </div>

      <!-- 搜索结果瀑布流 -->
      <template v-else-if="searchResults.length > 0">
        <div class="flex gap-4">
          <div
            v-for="(column, colIndex) in searchColumns"
            :key="colIndex"
            class="flex-1 flex flex-col space-y-4"
          >
            <div
              v-for="item in column.items"
              :key="item.uuid"
              class="waterfall-item fade-in visible bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer"
              @click="handleItemClick(item)"
            >
              <!-- 图片/视频显示区域 -->
              <div class="w-full relative">
                <img
                  v-if="item.media_type !== 1"
                  :src="getOptimizedSrc(item.first_url)"
                  :alt="item.title"
                  class="w-full h-full object-cover"
                  :style="{ height: `${item.height}px` }"
                  loading="lazy"
                  decoding="async"
                />
                <div
                  v-else
                  class="video-cover"
                  :style="{ height: `${item.height}px` }"
                >
                  <img
                    v-if="item.cover_url"
                    :src="item.cover_url"
                    :alt="item.title"
                    class="w-full h-full object-cover"
                  />
                  <video
                    v-else
                    :src="item.first_url"
                    class="w-full h-full object-cover"
                    preload="metadata"
                    muted
                    playsinline
                  />
                </div>
              </div>

              <!-- 卡片内容 -->
              <div class="p-3">
                <h3 class="text-sm font-medium line-clamp-2 mb-2">{{ item.title }}</h3>
                <div class="flex items-center justify-between text-xs text-gray-500">
                  <div class="flex items-center gap-2">
                    <img
                      v-if="item.avatar"
                      :src="item.avatar"
                      class="w-7 h-7 rounded-full object-cover"
                    />
                    <div
                      v-else
                      class="w-7 h-7 rounded-full flex items-center justify-center text-white text-xs"
                      :style="{ backgroundColor: getAvatarColor(item.user_id) }"
                    >
                      {{ (item.nickname || '').slice(0, 1) }}
                    </div>
                    <span>{{ item.nickname || '匿名用户' }}</span>
                  </div>
                  <div
                    class="flex items-center gap-1 cursor-pointer"
                    :class="{ 'text-red-500': item.is_liked }"
                    @click.stop="toggleLike(item)"
                  >
                    <Icon :name="item.is_liked ? 'solar:heart-angle-bold' : 'mdi:heart-outline'" class="text-base" />
                    <span>{{ item.like_count || 0 }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 加载更多提示 -->
        <div v-if="searchLoadingMore" class="py-8 flex justify-center items-center text-gray-500">
          <Icon name="uil:spinner" class="animate-spin text-xl mr-2" />
          <span>加载中...</span>
        </div>
        <div v-if="!searchLoadingMore && !searchHasMore" class="py-8 text-center text-gray-400 text-sm">
          没有更多内容了
        </div>
      </template>

      <!-- 无搜索结果 -->
      <div v-else class="py-16 flex flex-col items-center justify-center text-gray-400">
        <Icon name="uil:search-alt" class="text-5xl mb-3" />
        <p class="text-sm">未找到相关帖子</p>
        <p class="text-xs mt-1">换个关键词试试吧</p>
      </div>
    </div>

    <!-- 笔记详情弹窗 -->
    <DiscoverDetail
      v-if="selectedItem"
      :item="selectedItem"
      @close="selectedItem = null"
      @like-updated="handleLikeUpdated"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'

// 页面级 SEO
useSeoMeta({
  title: '搜索发现',
  description: '搜索用户、动态和内容。',
})

const route = useRoute()

// 搜索相关状态
const searchKeyword = ref('')
const searchResults = ref([])
const searchColumns = ref([])
const searchLoading = ref(false)
const searchLoadingMore = ref(false)
const searchPage = ref(1)
const searchTotal = ref(0)
const searchHasMore = ref(true)
const showSearchPanel = ref(false)
const searchInputRef = ref(null)
const searchPanelRef = ref(null)
const selectedItem = ref(null)

// 点击外部关闭搜索面板
const handleClickOutside = (e) => {
  const searchContainer = searchInputRef.value?.closest('.relative')
  if (searchContainer && !searchContainer.contains(e.target)) {
    showSearchPanel.value = false
  }
}

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

const getOptimizedSrc = (src) => {
  if (!src) return src
  try {
    const url = new URL(src)
    if (url.hostname.includes('images.unsplash.com')) {
      url.searchParams.set('w', url.searchParams.get('w') || '800')
      url.searchParams.set('auto', url.searchParams.get('auto') || 'format')
      url.searchParams.set('fit', url.searchParams.get('fit') || 'crop')
      url.searchParams.set('q', url.searchParams.get('q') || '60')
      return url.toString()
    }
    if (url.hostname.includes('images.pexels.com')) {
      url.searchParams.set('auto', url.searchParams.get('auto') || 'compress')
      url.searchParams.set('cs', url.searchParams.get('cs') || 'tinysrgb')
      url.searchParams.set('w', url.searchParams.get('w') || '800')
      return url.toString()
    }
    return src
  } catch {
    return src
  }
}

// 列数计算
const getColumnCount = () => {
  if (!process.client) return 1
  const width = window.innerWidth
  if (width >= 1024) return 4
  if (width >= 768) return 3
  if (width >= 640) return 2
  return 1
}

// 将搜索结果分配到瀑布流列
const distributeSearchToColumns = (newItems) => {
  const count = getColumnCount()
  if (searchColumns.value.length !== count) {
    searchColumns.value = Array.from({ length: count }, () => ({ items: [], height: 0 }))
  }
  newItems.forEach((item) => {
    const height = 250 + Math.floor(Math.random() * 150)
    item.height = height
    let targetIndex = 0
    let minHeight = searchColumns.value[0].height
    for (let i = 1; i < searchColumns.value.length; i++) {
      if (searchColumns.value[i].height < minHeight) {
        minHeight = searchColumns.value[i].height
        targetIndex = i
      }
    }
    searchColumns.value[targetIndex].items.push(item)
    searchColumns.value[targetIndex].height += height
  })
}

// 执行搜索
const executeSearch = async (keyword) => {
  if (!keyword?.trim()) return
  searchKeyword.value = keyword.trim()
  showSearchPanel.value = false
  searchLoading.value = true
  searchPage.value = 1

  // 更新 URL
  navigateTo(`/discover/search?q=${encodeURIComponent(keyword.trim())}`, { replace: true })

  try {
    const res = await useApi$('/user/discover/search', {
      body: { keyword: searchKeyword.value, page: 1, page_size: 20 }
    })
    if (res.code === 200) {
      const list = res.data?.list || []
      searchResults.value = list
      searchTotal.value = res.data?.total || 0
      searchHasMore.value = searchResults.value.length < searchTotal.value
      searchPage.value = 2
      searchColumns.value = []
      distributeSearchToColumns(list)
    }
  } catch (e) {
    console.error('搜索失败:', e)
    searchResults.value = []
    searchColumns.value = []
  }
  searchLoading.value = false
}

// 加载更多搜索结果
const loadMoreSearch = async () => {
  if (searchLoadingMore.value || !searchHasMore.value) return
  searchLoadingMore.value = true
  try {
    const res = await useApi$('/user/discover/search', {
      body: { keyword: searchKeyword.value, page: searchPage.value, page_size: 20 }
    })
    if (res.code === 200 && res.data?.list) {
      const list = res.data.list
      searchResults.value.push(...list)
      searchPage.value++
      searchHasMore.value = searchResults.value.length < searchTotal.value
      distributeSearchToColumns(list)
    }
  } catch (e) {
    console.error('加载更多失败:', e)
  }
  searchLoadingMore.value = false
}

// 点击帖子
const handleItemClick = (item) => {
  selectedItem.value = {
    id: item.uuid,
    uid: item.uuid,
    src: item.first_url,
    coverUrl: item.cover_url || '',
    type: item.media_type === 1 ? 'video' : 'image',
    title: item.title,
    height: item.height || 300,
    likeCount: item.like_count || 0,
    isLiked: item.is_liked || false,
    nickname: item.nickname,
    avatar: item.avatar,
    order: 0
  }
}

// 点赞
const toggleLike = async (item) => {
  try {
    const res = await useApi$('/user/discover/like', { body: { target_uuid: item.uuid } })
    if (res.code === 200) {
      item.is_liked = res.data.is_liked
      item.like_count = res.data.like_count
    }
  } catch (e) { console.error('点赞失败:', e) }
}

// 点赞更新（从详情弹窗返回）
const handleLikeUpdated = ({ postId, likeCount, isLiked }) => {
  for (const column of searchColumns.value) {
    const item = column.items.find(i => i.uuid === postId)
    if (item) { item.like_count = likeCount; item.is_liked = isLiked; break }
  }
}

// 清除历史回调
const handleClearHistory = () => {}

// 滚动监听
const handleScroll = (e) => {
  const target = e?.target
  if (!target) return
  const { scrollTop, scrollHeight, clientHeight } = target
  if (scrollTop + clientHeight >= scrollHeight - 200) loadMoreSearch()
}

let scrollEl = null
const getScrollParent = () => {
  if (!process.client) return null
  let el = searchInputRef.value?.closest('.min-h-screen')
  while (el) {
    const style = window.getComputedStyle(el)
    if (style.overflowY === 'auto' || style.overflowY === 'scroll') return el
    el = el.parentElement
  }
  return window
}

onMounted(() => {
  if (process.client) {
    // 从 URL 获取搜索关键词
    const q = route.query.q
    if (q) {
      searchKeyword.value = q
      executeSearch(q)
    }

    nextTick(() => {
      scrollEl = getScrollParent()
      scrollEl?.addEventListener('scroll', handleScroll)
    })

    document.addEventListener('click', handleClickOutside)
  }
})

onUnmounted(() => {
  if (process.client) {
    scrollEl?.removeEventListener('scroll', handleScroll)
    document.removeEventListener('click', handleClickOutside)
  }
})
</script>

<style scoped>
.fade-in {
  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.fade-in.visible {
  opacity: 1;
  transform: translateY(0);
}
.video-cover {
  position: relative;
  overflow: hidden;
  background: #000;
}
.video-cover video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
