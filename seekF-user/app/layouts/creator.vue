<template>
  <div class="flex flex-col bg-gray-50 text-gray-700" :class="route.path === '/discover/manage' ? 'h-screen overflow-hidden' : 'min-h-screen'">
    <header class="flex h-[72px] shrink-0 items-center justify-between gap-4 border-b border-gray-100 bg-white px-5 sm:px-8">
      <div class="flex items-center gap-3"><span class="text-2xl font-bold text-[#60a5fa]">seekF</span><span class="hidden text-lg font-medium sm:inline">创作中心</span></div>
      <div class="flex min-w-0 items-center gap-2 text-sm"><img v-if="user?.avatar" :src="user.avatar" alt="" class="h-7 w-7 rounded-full object-cover" /><span class="max-w-[160px] truncate">{{ user?.nickname || '创作者' }}</span></div>
    </header>
    <div class="flex min-h-0 flex-1 flex-col md:flex-row">
      <aside class="relative isolate flex shrink-0 items-center gap-1 border-b border-gray-100 bg-white px-2 py-3 md:w-[200px] md:flex-col md:items-stretch md:border-b-0 md:border-r md:py-6">
        <LiquidSelectionIndicator :selected="route.path" fill-class="bg-blue-50" />
        <NuxtLink v-for="item in creatorNav" :key="item.path" :to="item.path" :data-liquid-key="item.path" class="relative z-10 flex items-center gap-3 rounded-lg px-3 py-3 font-medium transition-colors duration-300 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 md:mb-1" :class="route.path === item.path ? 'bg-blue-50 text-[#0073ff] hover:!bg-blue-50' : ''"><Icon :name="item.icon" class="text-lg" /><span>{{ item.label }}</span></NuxtLink>
      </aside>
      <main class="min-h-0 min-w-0 flex-1 p-3 sm:p-5" :class="route.path === '/discover/manage' ? 'overflow-hidden' : 'overflow-y-auto'"><slot /></main>
    </div>
  </div>
</template>
<script setup>
const route = useRoute()
const creatorNav = [{ path: '/discover/create', label: '发布', icon: 'uil:plus-square' }, { path: '/discover/manage', label: '作品管理', icon: 'uil:clipboard-notes' }, { path: '/discover/support', label: '创作助力', icon: 'mdi:hand-heart-outline' }]
const auth = useAuthState()
const user = computed(() => auth.getUser())
</script>

<style scoped>
/* 路由页淡入滑动，保留发布页和管理页各自的滚动容器。 */
:deep(.creator-page-enter-active), :deep(.creator-page-leave-active) {
  transition: opacity 220ms ease, transform 280ms cubic-bezier(.22,.75,.25,1);
}
:deep(.creator-page-enter-from) { opacity: 0; transform: translateX(28px); }
:deep(.creator-page-leave-to) { opacity: 0; transform: translateX(-28px); }
@media (prefers-reduced-motion: reduce) {
  :deep(.creator-page-enter-active), :deep(.creator-page-leave-active) { transition: none; }
  :deep(.creator-page-enter-from), :deep(.creator-page-leave-to) { transform: none; }
}
</style>
