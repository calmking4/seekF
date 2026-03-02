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
      <!-- 多列瀑布流容器 -->
      <div
        ref="waterfallContainer"
        class="columns-1 sm:columns-2 md:columns-3 lg:columns-4 gap-4 space-y-4"
      >
        <div
          v-for="(item, index) in items"
          :key="item.id"
          class="waterfall-item break-inside-avoid bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow"
          :class="{ 'opacity-0': !item.visible }"
          @click="handleItemClick(item)"
          :data-id="item.id"
        >
          <!-- 图片显示区域 -->
          <div class="w-full relative">
            <img
              :src="item.src"
              :alt="item.title"
              class="w-full h-full object-cover"
              :style="{ height: `${item.height}px` }"
              loading="lazy"
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
                <span>{{ Math.floor(Math.random() * 10000) }}</span>
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
import { ref, onMounted, onUnmounted, onUpdated } from 'vue'

// 响应式数据
const items = ref([])
const loading = ref(false)
const noMore = ref(false)
const page = ref(1)
const pageSize = 12
const waterfallContainer = ref(null)
let sr = null // 存储 ScrollReveal 实例

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
]

// 头像颜色数组
const avatarColors = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA'
]

// 根据ID生成固定的头像颜色（避免每次刷新颜色变化）
const getAvatarColor = (id) => {
  const index = (Number(id) - 1) % avatarColors.length
  return avatarColors[index]
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
  const newData = originalItems.slice(start, end).map(item => ({
    ...item,
    visible: false // 新增visible字段控制动画
  }))

  items.value.push(...newData)
  page.value++
  loading.value = false

  // 判断是否加载完所有数据
  if (end >= originalItems.length) {
    noMore.value = true
  }

  // 数据加载后重新初始化动画
  if (process.client) {
    initScrollReveal()
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

// 初始化 ScrollReveal 动画（仅客户端）
const initScrollReveal = async () => {
  // 确保在浏览器环境，且有DOM元素
  if (!process.client || !waterfallContainer.value) return

  // 动态导入 ScrollReveal，避免服务端打包时引入
  const ScrollReveal = (await import('scrollreveal')).default
  
  // 销毁旧实例，避免重复初始化
  if (sr) {
    sr.destroy()
  }

  // 初始化动画
  sr = ScrollReveal({
    origin: 'bottom',
    distance: '20px',
    duration: 600,
    delay: 100,
    interval: 100,
    opacity: 0,
    scale: 0.95,
    reset: false,
    // 强制只在客户端执行
    disable: !process.client
  })

  // 给新元素添加动画
  sr.reveal('.waterfall-item', {
    afterReveal: (el) => {
      const itemId = el.dataset.id
      const item = items.value.find(i => i.id === itemId)
      if (item) item.visible = true
    }
  })
}

// 卡片点击
const handleItemClick = (item) => {
  console.log('点击了笔记:', item)
  // 可跳转到详情页：navigateTo(`/note/${item.id}`)
}

// 仅在客户端执行的生命周期
onMounted(() => {
  if (process.client) {
    loadMore()
    window.addEventListener('scroll', handleScroll)
    initScrollReveal()
  }
})

onUpdated(() => {
  // 页面更新后重新初始化动画（解决加载更多后新元素无动画）
  if (process.client) {
    initScrollReveal()
  }
})

onUnmounted(() => {
  if (process.client) {
    window.removeEventListener('scroll', handleScroll)
    // 销毁 ScrollReveal 实例
    if (sr) sr.destroy()
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