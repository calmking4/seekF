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
              @keyup.enter="goSearch(searchKeyword)"
            />
          </div>
          <!-- 搜索联想面板 -->
          <DiscoverSearchPanel
            v-if="showSearchPanel"
            ref="searchPanelRef"
            :keyword="searchKeyword"
            @select="goSearch"
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

    <!-- 瀑布流内容区 -->
    <div class="p-4">
      <!-- 多列瀑布流容器 -->
      <div
        ref="waterfallContainer"
        class="flex gap-4"
      >
        <div
          v-for="(column, columnIndex) in columns"
          :key="columnIndex"
          class="flex-1 flex flex-col space-y-4"
        >
          <div
            v-for="item in column.items"
            :key="item.uid"
            class="waterfall-item fade-in bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow"
            @click="handleItemClick(item)"
            :data-id="item.uid"
          >
            <!-- 图片/视频显示区域 -->
            <div class="w-full relative">
              <img
                v-if="item.type !== 'video'"
                :src="getOptimizedSrc(item.src)"
                :alt="item.title"
                class="w-full h-full object-cover"
                :style="{ height: `${item.height}px` }"
                loading="lazy"
                decoding="async"
                :fetchpriority="item.order <= 4 ? 'high' : 'low'"
              />
              <div
                v-else
                class="video-cover"
                :style="{ height: `${item.height}px` }"
              >
                <img
                  v-if="item.coverUrl"
                  :src="item.coverUrl"
                  :alt="item.title"
                  class="w-full h-full object-cover"
                />
                <video
                  v-else
                  :src="item.src"
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
                    :style="{ backgroundColor: getAvatarColor(item.id) }"
                  >
                    {{ item.title.slice(0, 1) }}
                  </div>
                  <span>{{ item.nickname || '用户' + item.id.slice(-2) }}</span>
                </div>
                <div
                  class="flex items-center gap-1 cursor-pointer"
                  :class="{ 'text-red-500': item.isLiked }"
                  @click.stop="toggleLike(item)"
                >
                  <Icon :name="item.isLiked ? 'solar:heart-angle-bold' : 'mdi:heart-outline'" class="text-base" />
                  <span>{{ item.likeCount }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 加载更多提示 -->
      <div v-if="loading" class="py-8 flex justify-center items-center text-gray-500">
        <Icon name="uil:spinner" class="animate-spin text-xl mr-2" />
        <span>加载中...</span>
      </div>
      <div v-if="!loading && noMore" class="py-8 text-center text-gray-400 text-sm">
        没有更多内容了
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
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

// 页面级 SEO
useSeoMeta({
  title: '发现',
  description: '发现有趣的人和内容，探索 seekF 社区的精彩动态。',
})

// 响应式数据
const items = ref([])
const columns = ref([])
const loading = ref(false)
const noMore = ref(false)
const selectedItem = ref(null)
const page = ref(1)
const pageSize = 12
const waterfallContainer = ref(null)
let observer = null
let orderCounter = 0

// 搜索框相关
const searchKeyword = ref('')
const showSearchPanel = ref(false)
const searchInputRef = ref(null)
const searchPanelRef = ref(null)

// 跳转到搜索结果页
const goSearch = (keyword) => {
  if (!keyword?.trim()) return
  // 保存搜索历史
  if (searchPanelRef.value) {
    searchPanelRef.value.saveHistory(keyword.trim())
  }
  showSearchPanel.value = false
  navigateTo(`/discover/search?q=${encodeURIComponent(keyword.trim())}`)
}

// 清除历史回调
const handleClearHistory = () => {}

// 点击外部关闭搜索面板
const handleClickOutside = (e) => {
  const searchContainer = searchInputRef.value?.closest('.relative')
  if (searchContainer && !searchContainer.contains(e.target)) {
    showSearchPanel.value = false
  }
}

// 模拟数据
const originalItems = [
  { id: '1', src: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e', type: 'image', title: 'Tropical Beach', height: 300 },
  { id: '2', src: 'https://images.unsplash.com/photo-1519681393784-d120267933ba', type: 'image', title: 'Mountain Peak', height: 400 },
  { id: '3', src: 'https://images.pexels.com/photos/417074/pexels-photo-417074.jpeg', type: 'image', title: 'Lake Reflection', height: 250 },
  { id: '4', src: 'https://images.unsplash.com/photo-1472214103451-9374bd1c798e', type: 'image', title: 'Forest Path', height: 350 },
  { id: '5', src: 'https://images.pexels.com/photos/443446/pexels-photo-443446.jpeg', type: 'image', title: 'Snowy Mountains', height: 280 },
  { id: '6', src: 'https://images.pexels.com/photos/462118/pexels-photo-462118.jpeg', type: 'image', title: 'Flower Close-up', height: 260 },
  { id: '7', src: 'https://images.pexels.com/photos/735911/pexels-photo-735911.jpeg', type: 'image', title: 'Laptop Workspace', height: 290 },
  { id: '8', src: 'https://images.unsplash.com/photo-1493246507139-91e8fad9978e', type: 'image', title: 'River Valley', height: 340 },
  { id: '9', src: 'https://images.unsplash.com/photo-1519046904884-53103b34b206', type: 'image', title: 'Ocean Waves', height: 310 },
  { id: '10', src: 'https://images.pexels.com/photos/1323550/pexels-photo-1323550.jpeg', type: 'image', title: 'Sunset Horizon', height: 330 },
  { id: '11', src: 'https://images.unsplash.com/photo-1506748686214-e9df14d4d9d0', type: 'image', title: 'Urban Street', height: 360 },
  { id: '12', src: 'https://images.pexels.com/photos/34950/pexels-photo.jpg', type: 'image', title: 'Waterfall Scene', height: 300 },
]

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

const initColumns = () => {
  const count = getColumnCount()
  columns.value = Array.from({ length: count }, () => ({ items: [], height: 0 }))
}

const rebuildColumns = () => {
  if (!process.client) return
  const count = getColumnCount()
  if (count === columns.value.length) return
  columns.value = Array.from({ length: count }, () => ({ items: [], height: 0 }))
  distributeToColumns(items.value)
}

const distributeToColumns = (newItems) => {
  if (!columns.value.length) initColumns()
  newItems.forEach((item) => {
    let targetIndex = 0
    let minHeight = columns.value[0].height
    for (let i = 1; i < columns.value.length; i++) {
      if (columns.value[i].height < minHeight) {
        minHeight = columns.value[i].height
        targetIndex = i
      }
    }
    columns.value[targetIndex].items.push(item)
    columns.value[targetIndex].height += item.height || 0
  })
}

// 加载数据
const loadMore = async () => {
  if (loading.value || noMore.value) return
  loading.value = true
  try {
    const res = await useApi$('/user/discover/list', {
      body: { page: page.value, page_size: pageSize },
    })
    if (res.code === 200 && res.data?.list?.length > 0) {
      const apiData = res.data.list.map((item, idx) => {
        orderCounter++
        return {
          id: item.uuid,
          uid: `${page.value}-${idx}-${item.uuid}-${orderCounter}`,
          src: item.first_url,
          coverUrl: item.cover_url || '',
          type: item.media_type === 1 ? 'video' : 'image',
          title: item.title,
          height: 250 + Math.floor(Math.random() * 150),
          likeCount: item.like_count,
          isLiked: item.is_liked || false,
          nickname: item.nickname,
          avatar: item.avatar,
          order: orderCounter,
        }
      })
      items.value.push(...apiData)
      distributeToColumns(apiData)
      page.value++
      if (items.value.length >= res.data.total) noMore.value = true
    } else {
      await loadMockData()
    }
  } catch {
    await loadMockData()
  }
  loading.value = false
  if (process.client) {
    await nextTick()
    observeNewItems()
  }
}

const loadMockData = async () => {
  await new Promise(resolve => setTimeout(resolve, 500))
  const start = (page.value - 1) * pageSize
  const end = start + pageSize
  const newData = originalItems.slice(start, end).map((item, idx) => {
    orderCounter++
    return { ...item, uid: `${page.value}-${start + idx}-${item.id}-${orderCounter}`, likeCount: Math.floor(Math.random() * 10000), order: orderCounter }
  })
  items.value.push(...newData)
  distributeToColumns(newData)
  page.value++
  if (end >= originalItems.length) noMore.value = true
}

// 滚动监听
const handleScroll = (e) => {
  const target = e?.target
  if (!target) return
  const { scrollTop, scrollHeight, clientHeight } = target
  if (scrollTop + clientHeight >= scrollHeight - 200) loadMore()
}

const handleResize = () => rebuildColumns()

const observeNewItems = () => {
  if (!waterfallContainer.value) return
  const items = waterfallContainer.value.querySelectorAll('.fade-in:not(.visible)')
  items.forEach((item, index) => {
    item.style.transitionDelay = `${index * 50}ms`
    observer?.observe(item)
  })
}

let scrollEl = null
const getScrollParent = () => {
  if (!process.client) return null
  let el = waterfallContainer.value?.parentElement
  while (el) {
    const style = window.getComputedStyle(el)
    if (style.overflowY === 'auto' || style.overflowY === 'scroll') return el
    el = el.parentElement
  }
  return window
}

onMounted(() => {
  if (process.client) {
    initColumns()
    observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('visible')
          observer?.unobserve(entry.target)
        }
      })
    }, { threshold: 0.1 })
    loadMore()
    nextTick(() => {
      scrollEl = getScrollParent()
      scrollEl?.addEventListener('scroll', handleScroll)
    })
    window.addEventListener('resize', handleResize)
    document.addEventListener('click', handleClickOutside)
  }
})

onUnmounted(() => {
  if (process.client) {
    scrollEl?.removeEventListener('scroll', handleScroll)
    window.removeEventListener('resize', handleResize)
    document.removeEventListener('click', handleClickOutside)
    observer?.disconnect()
    observer = null
  }
})

// 卡片点击
const handleItemClick = (item) => selectedItem.value = item

// 点赞更新
const handleLikeUpdated = ({ postId, likeCount, isLiked }) => {
  const item = items.value.find(i => i.id === postId)
  if (item) { item.likeCount = likeCount; item.isLiked = isLiked }
  for (const column of columns.value) {
    const columnItem = column.items.find(i => i.id === postId)
    if (columnItem) { columnItem.likeCount = likeCount; columnItem.isLiked = isLiked; break }
  }
}

const toggleLike = async (item) => {
  try {
    const res = await useApi$('/user/discover/like', { body: { target_uuid: item.id } })
    if (res.code === 200) { item.isLiked = res.data.is_liked; item.likeCount = res.data.like_count }
  } catch (e) { console.error('点赞失败:', e) }
}
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
