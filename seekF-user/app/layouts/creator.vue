<template>
  <div class="flex flex-col bg-gray-50 text-gray-700" :class="route.path === '/discover/manage' ? 'h-screen overflow-hidden' : 'min-h-screen'">
    <header class="flex h-[72px] shrink-0 items-center justify-between gap-4 border-b border-gray-100 bg-white px-5 sm:px-8">
      <div class="flex items-center gap-3"><span class="text-2xl font-bold text-[#60a5fa]">seekF</span><span class="hidden text-lg font-medium sm:inline">创作中心</span></div>
      <div class="flex min-w-0 items-center gap-2 text-sm"><img v-if="user?.avatar" :src="user.avatar" alt="" class="h-7 w-7 rounded-full object-cover" /><span class="max-w-[160px] truncate">{{ user?.nickname || '创作者' }}</span></div>
    </header>
    <div class="flex min-h-0 flex-1 flex-col md:flex-row">
      <aside class="relative isolate flex shrink-0 items-center gap-1 border-b border-gray-100 bg-white px-2 py-3 md:w-[200px] md:flex-col md:items-stretch md:border-b-0 md:border-r md:py-6">
        <LiquidSelectionIndicator :selected="isSupportRoute ? '/discover/support' : route.path" fill-class="bg-blue-50" />
        <NuxtLink v-for="item in creatorNav" :key="item.path" :to="item.path" :data-liquid-key="item.path" class="relative z-10 flex items-center gap-3 rounded-lg px-3 py-3 font-medium transition-colors duration-300 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 md:mb-1" :class="route.path === item.path ? 'bg-blue-50 text-[#0073ff] hover:!bg-blue-50' : ''"><Icon :name="item.icon" class="text-lg" /><span>{{ item.label }}</span></NuxtLink>
        <div class="relative z-10 min-w-0">
          <button type="button" data-liquid-key="/discover/support" class="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left font-medium transition-colors duration-300 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 md:mb-1" :class="isSupportRoute ? 'bg-blue-50 text-[#0073ff] hover:!bg-blue-50' : ''" :aria-expanded="supportOpen" aria-controls="creator-support-nav" @click="supportOpen = !supportOpen"><Icon name="mdi:hand-heart-outline" class="shrink-0 text-lg" /><span class="whitespace-nowrap">创作助力</span><Icon name="mdi:chevron-down" class="ml-auto shrink-0 text-base transition-transform duration-300 motion-reduce:transition-none" :class="{ 'rotate-180': supportOpen }" /></button>
          <div id="creator-support-nav" :inert="!supportOpen" :aria-hidden="!supportOpen" class="grid transition-[grid-template-rows,opacity] duration-300 ease-[cubic-bezier(.22,.75,.25,1)] motion-reduce:transition-none" :class="supportOpen ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'">
            <div class="min-h-0 overflow-hidden">
              <div class="relative isolate mt-1 flex w-full flex-col gap-1">
            <LiquidSelectionIndicator v-if="isSupportRoute" :selected="`support-child:${route.path}`" fill-class="bg-blue-50" />
            <NuxtLink v-for="item in supportNav" :key="item.path" :to="item.path" :data-liquid-key="`support-child:${item.path}`" :aria-current="route.path === item.path ? 'page' : undefined" class="relative z-10 flex w-full items-center justify-start rounded-lg py-2.5 pl-[42px] pr-3 text-left text-sm transition-colors duration-300 hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" :class="route.path === item.path ? 'font-medium text-[#0073ff] hover:!bg-transparent' : 'text-gray-500'">{{ item.label }}</NuxtLink>
              </div>
            </div>
          </div>
        </div>
      </aside>
      <main class="min-h-0 min-w-0 flex-1 p-3 sm:p-5" :class="route.path === '/discover/manage' ? 'overflow-hidden' : 'overflow-y-auto'"><slot /></main>
    </div>
  </div>
</template>
<script setup>
const route = useRoute()
const creatorNav = [{ path: '/discover/create', label: '发布', icon: 'uil:plus-square' }, { path: '/discover/manage', label: '作品管理', icon: 'uil:clipboard-notes' }]
const supportNav = [{ path: '/discover/support', label: '助力设置' }, { path: '/discover/support-records', label: '助力记录' }]
const isSupportRoute = computed(() => supportNav.some(item => item.path === route.path))
const supportOpen = ref(isSupportRoute.value)
watch(() => route.path, () => { if (isSupportRoute.value) supportOpen.value = true })
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
