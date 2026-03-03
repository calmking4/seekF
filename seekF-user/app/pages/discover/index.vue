<template>
  <div class="min-h-screen bg-gray-50">
    <!-- 顶部搜索栏 -->
    <header class="sticky top-0 z-10 bg-white shadow-sm px-4 py-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="text-xl font-bold text-red-500">seekF</span>
          <span class="text-gray-500">发现</span>
        </div>
        <div class="flex gap-4 text-gray-500">
          <Icon name="uil:search" />
          <Icon name="uil:bell" />
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
            class="waterfall-item bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow"
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
                <div class="flex items-center gap-1">
                  <div 
                    class="w-5 h-5 rounded-full flex items-center justify-center text-white text-xs"
                    :style="{ backgroundColor: getAvatarColor(item.id) }"
                  >
                    {{ item.title.slice(0, 1) }}
                  </div>
                  <span>用户{{ item.id.slice(-2) }}</span>
                </div>
                <div class="flex items-center gap-1">
                  <Icon name="uil:heart" />
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
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

// 响应式数据
const items = ref([])
const columns = ref([]) // [{ items: [], height: number }]
const loading = ref(false)
const noMore = ref(false)
const page = ref(1)
const pageSize = 12
const waterfallContainer = ref(null)
let sr = null // ScrollReveal 实例
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

// 加载数据（分页加载原始数组）
const loadMore = async () => {
  if (loading.value || noMore.value) return
  loading.value = true

  // 模拟接口请求延迟
  await new Promise(resolve => setTimeout(resolve, 800))

  // 分页截取原始数据
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
  loading.value = false

  // 判断是否加载完所有数据
  if (end >= originalItems.length) {
    noMore.value = true
  }

  // 数据加载后重新初始化动画
  if (process.client) {
    await nextTick()
    await initScrollReveal()
  }
}

// 滚动到底部监听
const handleScroll = () => {
  const scrollTop = document.documentElement.scrollTop || document.body.scrollTop
  const scrollHeight = document.documentElement.scrollHeight || document.body.scrollHeight
  const clientHeight = document.documentElement.clientHeight || window.innerHeight

  if (scrollTop + clientHeight >= scrollHeight - 200) {
    loadMore()
  }
}

// 监听窗口尺寸变化，做响应式列数调整
const handleResize = () => {
  rebuildColumns()
}

// 初始化 ScrollReveal 动画（仅客户端）
const initScrollReveal = async () => {
  if (!process.client || !waterfallContainer.value) return

  const ScrollReveal = (await import('scrollreveal')).default

  // 复用单例，避免重复创建
  if (!sr) {
    sr = ScrollReveal({
      origin: 'bottom',
      distance: '20px',
      duration: 600,
      delay: 80,
      interval: 60,
      opacity: 0,
      scale: 0.95,
      reset: false,
      disable: !process.client
    })

    // 只注册一次 reveal；后续新增节点用 sync()，避免旧节点重复动画
    sr.reveal('.waterfall-item')
    return
  }

  sr.sync?.()
}

// 卡片点击
const handleItemClick = (item) => {
  console.log('点击了笔记:', item)
  // 可跳转到详情页：navigateTo(`/note/${item.id}`)
}

// 仅在客户端执行的生命周期
onMounted(() => {
  if (process.client) {
    initColumns()
    loadMore()
    window.addEventListener('scroll', handleScroll)
    window.addEventListener('resize', handleResize)
  }
})

onUnmounted(() => {
  if (process.client) {
    window.removeEventListener('scroll', handleScroll)
    window.removeEventListener('resize', handleResize)
    sr?.destroy?.()
    sr = null
  }
})
</script>

<style scoped>
/* 瀑布流卡片动画 */
.waterfall-item {
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.waterfall-item:not(.opacity-0) {
  opacity: 1;
  transform: translateY(0);
}

/* 图片加载失败兜底样式 */
img {
  object-fit: cover;
}
img::after {
  content: '图片加载失败';
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background-color: #f5f5f5;
  color: #999;
}
</style>