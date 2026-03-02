<template>
  <div class="flex min-h-screen bg-white">
    <!-- 左侧侧边栏 -->
    <aside class="fixed top-0 left-0 bottom-0 w-[200px] bg-white border-r border-gray-100 p-0 flex flex-col">
      <!-- 侧边栏头部 Logo -->
      <div class="px-5 py-4 mb-4">
        <div class="text-xl font-bold text-[#60a5fa]">seekF</div>
      </div>

      <!-- 侧边栏导航 -->
      <nav class="flex-1 px-2">
        <NuxtLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-3 py-3 rounded-lg mb-1 transition-colors hover:bg-gray-50"
          :class="{ 'bg-blue-50 text-[#0073ff]': route.path === item.path }"
        >
          <Icon :name="item.icon" class="text-lg" />
          <span class="font-medium">{{ item.label }}</span>
        </NuxtLink>
      </nav>

      <!-- 侧边栏底部 -->
      <div class="px-2 pb-4">
        <button class="flex items-center gap-3 w-full px-3 py-3 rounded-lg border border-gray-200 hover:bg-gray-50 transition-colors">
          <Icon name="uil:bars" class="text-lg" />
          <span class="font-medium">更多</span>
        </button>
      </div>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="ml-[200px] flex-1 p-6">
      <slot />
    </main>
  </div>
</template>

<script setup>
// 引入路由对象，用于判断当前激活的导航项
const route = useRoute()

// 侧边栏导航数据
const navItems = [
  { path: '/chat', label: '消息', icon: 'uil:comment-alt' },
  { path: '/contact', label: '联系人', icon: 'uil:users-alt' },
  { path: '/discover', label: '发现', icon: 'uil:plus-square' },
  { path: '/aichat', label: 'AIChat', icon: 'uil:robot' },
  { path: '/my', label: '我', icon: 'uil:user' }
]
</script>