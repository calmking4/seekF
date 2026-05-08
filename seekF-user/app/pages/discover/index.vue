<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部搜索栏 -->
    <header class="sticky top-0 z-10 bg-white shadow-sm px-4 py-3">
      <div class="flex items-center justify-between">
        <!-- 搜索框 -->
        <div class="w-[30rem] mx-auto flex items-center bg-gray-100 rounded-full px-3 py-2 gap-2">
          <Icon name="mdi:magnify" class="text-gray-400 text-lg flex-shrink-0" />
          <input
            type="text"
            placeholder="搜索"
            class="bg-transparent outline-none text-sm w-full text-gray-700 placeholder-gray-400"
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
      <!-- 多列瀑布流容器（自定义列，避免加载更多时已有内容重新排布） -->
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
            <!-- 图片显示区域 -->
            <div class="w-full relative">
              <img
                :src="getOptimizedSrc(item.src)"
                :alt="item.title"
                class="w-full h-full object-cover"
                :style="{ height: `${item.height}px` }"
                loading="lazy"
                decoding="async"
                :fetchpriority="item.order <= 4 ? 'high' : 'low'"
              />
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
    <DiscoverCard
      v-if="selectedItem"
      :item="selectedItem"
      @close="selectedItem = null"
      @like-updated="handleLikeUpdated"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

// 响应式数据
const items = ref([])
const columns = ref([]) // [{ items: [], height: number }]
const loading = ref(false)
const noMore = ref(false)
const selectedItem = ref(null)
const page = ref(1)
const pageSize = 12
const waterfallContainer = ref(null)
let observer = null
let orderCounter = 0

// 原始模拟数据（你提供的数组）
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
  { id: '13', src: 'https://images.pexels.com/photos/1565982/pexels-photo-1565982.jpeg', type: 'image', title: 'Food Plate', height: 320 },
  { id: '14', src: 'https://images.pexels.com/photos/531880/pexels-photo-531880.jpeg', type: 'image', title: 'Wooden Texture', height: 290 },
  { id: '15', src: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e', type: 'image', title: 'Tropical Beach', height: 300 },
  { id: '16', src: 'https://images.unsplash.com/photo-1519681393784-d120267933ba', type: 'image', title: 'Mountain Peak', height: 400 },
  { id: '17', src: 'https://images.pexels.com/photos/417074/pexels-photo-417074.jpeg', type: 'image', title: 'Lake Reflection', height: 250 },
  { id: '18', src: 'https://images.unsplash.com/photo-1472214103451-9374bd1c798e', type: 'image', title: 'Forest Path', height: 350 },
  { id: '19', src: 'https://images.pexels.com/photos/443446/pexels-photo-443446.jpeg', type: 'image', title: 'Snowy Mountains', height: 280 },
  { id: '20', src: 'https://images.pexels.com/photos/462118/pexels-photo-462118.jpeg', type: 'image', title: 'Flower Close-up', height: 260 },
  { id: '21', src: 'https://images.pexels.com/photos/735911/pexels-photo-735911.jpeg', type: 'image', title: 'Laptop Workspace', height: 290 },
  { id: '22', src: 'https://images.unsplash.com/photo-1493246507139-91e8fad9978e', type: 'image', title: 'River Valley', height: 340 },
  { id: '23', src: 'https://images.unsplash.com/photo-1519046904884-53103b34b206', type: 'image', title: 'Ocean Waves', height: 310 },
  { id: '24', src: 'https://images.pexels.com/photos/1323550/pexels-photo-1323550.jpeg', type: 'image', title: 'Sunset Horizon', height: 330 },
  { id: '25', src: 'https://images.unsplash.com/photo-1506748686214-e9df14d4d9d0', type: 'image', title: 'Urban Street', height: 360 },
  { id: '26', src: 'https://images.pexels.com/photos/34950/pexels-photo.jpg', type: 'image', title: 'Waterfall Scene', height: 300 },
  { id: '27', src: 'https://images.pexels.com/photos/1565982/pexels-photo-1565982.jpeg', type: 'image', title: 'Food Plate', height: 320 },
  { id: '28', src: 'https://images.pexels.com/photos/531880/pexels-photo-531880.jpeg', type: 'image', title: 'Wooden Texture', height: 290 },
  { id: '29', src: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e', type: 'image', title: 'Tropical Beach', height: 300 },
  { id: '30', src: 'https://images.unsplash.com/photo-1519681393784-d120267933ba', type: 'image', title: 'Mountain Peak', height: 400 },
  { id: '31', src: 'https://images.pexels.com/photos/417074/pexels-photo-417074.jpeg', type: 'image', title: 'Lake Reflection', height: 250 },
  { id: '32', src: 'https://images.unsplash.com/photo-1472214103451-9374bd1c798e', type: 'image', title: 'Forest Path', height: 350 },
  { id: '33', src: 'https://images.pexels.com/photos/443446/pexels-photo-443446.jpeg', type: 'image', title: 'Snowy Mountains', height: 280 },
  { id: '34', src: 'https://images.pexels.com/photos/462118/pexels-photo-462118.jpeg', type: 'image', title: 'Flower Close-up', height: 260 },
  { id: '35', src: 'https://images.pexels.com/photos/735911/pexels-photo-735911.jpeg', type: 'image', title: 'Laptop Workspace', height: 290 },
  { id: '36', src: 'https://images.unsplash.com/photo-1493246507139-91e8fad9978e', type: 'image', title: 'River Valley', height: 340 },
  { id: '37', src: 'https://images.unsplash.com/photo-1519046904884-53103b34b206', type: 'image', title: 'Ocean Waves', height: 310 },
  { id: '38', src: 'https://images.pexels.com/photos/1323550/pexels-photo-1323550.jpeg', type: 'image', title: 'Sunset Horizon', height: 330 },
  { id: '39', src: 'https://images.unsplash.com/photo-1506748686214-e9df14d4d9d0', type: 'image', title: 'Urban Street', height: 360 },
  { id: '40', src: 'https://images.pexels.com/photos/34950/pexels-photo.jpg', type: 'image', title: 'Waterfall Scene', height: 300 },
  { id: '41', src: 'https://images.pexels.com/photos/1565982/pexels-photo-1565982.jpeg', type: 'image', title: 'Food Plate', height: 320 },
  { id: '42', src: 'https://images.pexels.com/photos/531880/pexels-photo-531880.jpeg', type: 'image', title: 'Wooden Texture', height: 290 },
  { id: '43', src: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e', type: 'image', title: 'Tropical Beach', height: 300 },
  { id: '44', src: 'https://images.unsplash.com/photo-1519681393784-d120267933ba', type: 'image', title: 'Mountain Peak', height: 400 },
  { id: '45', src: 'https://images.pexels.com/photos/417074/pexels-photo-417074.jpeg', type: 'image', title: 'Lake Reflection', height: 250 },
  { id: '46', src: 'https://images.unsplash.com/photo-1472214103451-9374bd1c798e', type: 'image', title: 'Forest Path', height: 350 },
  { id: '47', src: 'https://images.pexels.com/photos/443446/pexels-photo-443446.jpeg', type: 'image', title: 'Snowy Mountains', height: 280 },
  { id: '48', src: 'https://images.pexels.com/photos/462118/pexels-photo-462118.jpeg', type: 'image', title: 'Flower Close-up', height: 260 },
  { id: '49', src: 'https://images.pexels.com/photos/735911/pexels-photo-735911.jpeg', type: 'image', title: 'Laptop Workspace', height: 290 },
  { id: '50', src: 'https://images.unsplash.com/photo-1493246507139-91e8fad9978e', type: 'image', title: 'River Valley', height: 340 },
  { id: '51', src: 'https://images.unsplash.com/photo-1519046904884-53103b34b206', type: 'image', title: 'Ocean Waves', height: 310 },
  { id: '52', src: 'https://images.pexels.com/photos/1323550/pexels-photo-1323550.jpeg', type: 'image', title: 'Sunset Horizon', height: 330 },
  { id: '53', src: 'https://images.unsplash.com/photo-1506748686214-e9df14d4d9d0', type: 'image', title: 'Urban Street', height: 360 },
  { id: '54', src: 'https://images.pexels.com/photos/34950/pexels-photo.jpg', type: 'image', title: 'Waterfall Scene', height: 300 },
  { id: '55', src: 'https://images.pexels.com/photos/1565982/pexels-photo-1565982.jpeg', type: 'image', title: 'Food Plate', height: 320 },
  { id: '56', src: 'https://images.pexels.com/photos/531880/pexels-photo-531880.jpeg', type: 'image', title: 'Wooden Texture', height: 290 },
]

// 头像颜色数组
const avatarColors = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA'
]

// 根据ID生成固定的头像颜色（避免每次刷新颜色变化）
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
    // Unsplash: reduce bytes to improve scroll smoothness
    if (url.hostname.includes('images.unsplash.com')) {
      url.searchParams.set('w', url.searchParams.get('w') || '800')
      url.searchParams.set('auto', url.searchParams.get('auto') || 'format')
      url.searchParams.set('fit', url.searchParams.get('fit') || 'crop')
      url.searchParams.set('q', url.searchParams.get('q') || '60')
      return url.toString()
    }
    // Pexels: compress + width
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

// 计算当前应使用的列数（与 Tailwind 断点保持一致）
const getColumnCount = () => {
  if (!process.client) return 1
  const width = window.innerWidth
  if (width >= 1024) return 4
  if (width >= 768) return 3
  if (width >= 640) return 2
  return 1
}

// 初始化列数据结构
const initColumns = () => {
  const count = getColumnCount()
  columns.value = Array.from({ length: count }, () => ({
    items: [],
    height: 0
  }))
}

// 按当前屏幕列数重新构建布局（用于窗口尺寸变化时）
const rebuildColumns = () => {
  if (!process.client) return
  const count = getColumnCount()
  // 列数没变就不动，避免无意义重排
  if (count === columns.value.length) return

  columns.value = Array.from({ length: count }, () => ({
    items: [],
    height: 0
  }))

  // 按新的列数重新分配已有 items，保持从上到下的顺序
  distributeToColumns(items.value)
}

// 将新数据分配到列中（按当前列高度最小的列追加，保证已存在内容不重新排布）
const distributeToColumns = (newItems) => {
  if (!columns.value.length) {
    initColumns()
  }

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

// 加载数据（分页加载，优先使用API，回退到模拟数据）
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
      if (items.value.length >= res.data.total) {
        noMore.value = true
      }
    } else {
      // API无数据时回退到模拟数据
      await loadMockData()
    }
  } catch {
    // API失败时回退到模拟数据
    await loadMockData()
  }

  loading.value = false

  // 数据加载后观察新元素
  if (process.client) {
    await nextTick()
    observeNewItems()
  }
}

// 模拟数据加载（回退方案）
const loadMockData = async () => {
  await new Promise(resolve => setTimeout(resolve, 500))
  const start = (page.value - 1) * pageSize
  const end = start + pageSize
  const newData = originalItems.slice(start, end).map((item, idx) => {
    orderCounter++
    return {
      ...item,
      uid: `${page.value}-${start + idx}-${item.id}-${orderCounter}`,
      likeCount: Math.floor(Math.random() * 10000),
      order: orderCounter
    }
  })
  items.value.push(...newData)
  distributeToColumns(newData)
  page.value++
  if (end >= originalItems.length) {
    noMore.value = true
  }
}

// 滚动到底部监听
const handleScroll = (e) => {
  const target = e?.target
  if (!target) return
  
  const scrollTop = target.scrollTop
  const scrollHeight = target.scrollHeight
  const clientHeight = target.clientHeight

  if (scrollTop + clientHeight >= scrollHeight - 200) {
    loadMore()
  }
}

// 监听窗口尺寸变化，做响应式列数调整
const handleResize = () => {
  rebuildColumns()
}

// 使用 IntersectionObserver 观察新元素，添加可见类
const observeNewItems = () => {
  if (!waterfallContainer.value) return

  const items = waterfallContainer.value.querySelectorAll('.fade-in:not(.visible)')
  items.forEach((item, index) => {
    // 添加延迟，实现依次出现的效果
    item.style.transitionDelay = `${index * 50}ms`
    observer?.observe(item)
  })
}

// 滚动事件监听的元素
let scrollEl = null

// 获取最近的可滚动父元素
const getScrollParent = () => {
  if (!process.client) return null
  let el = waterfallContainer.value?.parentElement
  while (el) {
    const style = window.getComputedStyle(el)
    if (style.overflowY === 'auto' || style.overflowY === 'scroll') {
      return el
    }
    el = el.parentElement
  }
  return window
}

// 仅在客户端执行的生命周期
onMounted(() => {
  if (process.client) {
    initColumns()
    
    // 创建 IntersectionObserver
    observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('visible')
          observer?.unobserve(entry.target)
        }
      })
    }, {
      threshold: 0.1
    })

    loadMore()
    
    nextTick(() => {
      scrollEl = getScrollParent()
      scrollEl?.addEventListener('scroll', handleScroll)
    })
    
    window.addEventListener('resize', handleResize)
  }
})

onUnmounted(() => {
  if (process.client) {
    scrollEl?.removeEventListener('scroll', handleScroll)
    window.removeEventListener('resize', handleResize)
    observer?.disconnect()
    observer = null
  }
})

// 卡片点击
const handleItemClick = (item) => {
  selectedItem.value = item
}

// 点赞更新处理
const handleLikeUpdated = ({ postId, likeCount, isLiked }) => {
  // 更新 items 中对应帖子的赞数和点赞状态
  const item = items.value.find(i => i.id === postId)
  if (item) {
    item.likeCount = likeCount
    item.isLiked = isLiked
  }
  // 更新 columns 中的赞数和点赞状态
  for (const column of columns.value) {
    const columnItem = column.items.find(i => i.id === postId)
    if (columnItem) {
      columnItem.likeCount = likeCount
      columnItem.isLiked = isLiked
      break
    }
  }
}

// 首页卡片点赞
const toggleLike = async (item) => {
  try {
    const res = await useApi$('/user/discover/like', {
      body: { target_uuid: item.id },
    })
    if (res.code === 200) {
      item.isLiked = res.data.is_liked
      item.likeCount = res.data.like_count
    }
  } catch (e) {
    console.error('点赞失败:', e)
  }
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
</style>
