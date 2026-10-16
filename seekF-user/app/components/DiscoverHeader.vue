<template>
  <header ref="header" class="sticky top-0 z-20 px-4" :class="immersive ? 'bg-gray-50 pt-6 pb-6 sm:pt-8' : 'bg-white shadow-sm py-3'">
    <div class="flex items-center justify-between">
      <div ref="searchContainer" class="mx-auto relative" :class="immersive ? 'w-full max-w-4xl' : 'w-[30rem] max-w-full'">
        <form class="flex items-center rounded-full px-3 gap-2" :class="immersive ? 'border border-gray-200 bg-white py-3 shadow-[0_8px_32px_rgba(0,0,0,0.04)] sm:py-4' : 'bg-gray-100 py-2'" @submit.prevent="search(keyword)">
          <Icon name="mdi:magnify" class="text-gray-400 text-lg flex-shrink-0" />
          <input v-model="keyword" type="text" aria-label="搜索发现内容" placeholder="搜索" class="bg-transparent outline-none text-sm w-full text-gray-700 placeholder-gray-400" @focus="showPanel = true" @keydown.esc="showPanel = false" />
        </form>
        <Transition name="search-panel" @before-enter="element => { element.inert = false }" @before-leave="element => { element.inert = true }">
          <DiscoverSearchPanel v-if="showPanel" ref="panel" :keyword="keyword" @select="search" />
        </Transition>
      </div>
    </div>
    <div v-if="immersive" class="grid transition-[grid-template-rows,opacity,margin] duration-300 ease-out motion-reduce:transition-none" :class="tabsCollapsed ? 'mt-0 grid-rows-[0fr] opacity-0' : 'mt-5 grid-rows-[1fr] opacity-100'" :inert="tabsCollapsed">
      <div class="min-h-0 overflow-hidden">
        <nav class="relative mx-auto grid w-full max-w-md grid-cols-5" aria-label="发现内容分类">
          <button v-for="tab in tabs" :key="tab.key" type="button" class="min-w-0 px-2 py-3 text-sm transition-colors duration-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500 sm:text-base" :class="channel === tab.key ? 'font-medium text-gray-900' : 'text-gray-500 hover:text-gray-800'" :aria-pressed="channel === tab.key" :aria-label="tab.key === 'following' && unreadAuthors ? '关注，' + unreadAuthors + '位作者有更新' : tab.label" @click="$emit('update:channel', tab.key)"><span class="relative inline-block">{{ tab.label }}<Transition enter-active-class="transition duration-200 ease-out motion-reduce:transition-none" enter-from-class="scale-50 opacity-0" leave-active-class="transition duration-150 motion-reduce:transition-none" leave-to-class="scale-50 opacity-0"><span v-if="tab.key === 'following' && unreadAuthors > 0" class="absolute -right-4 -top-2 flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-semibold leading-none text-white ring-2 ring-gray-50" aria-hidden="true">{{ unreadAuthors > 99 ? '99+' : unreadAuthors }}</span></Transition></span></button>
          <span aria-hidden="true" class="pointer-events-none absolute bottom-0 left-0 flex w-1/5 justify-center transition-transform duration-300 ease-out motion-reduce:transition-none" :style="{ transform: 'translateX(' + activeTabIndex * 100 + '%)' }"><span class="h-0.5 w-8 rounded-full bg-[#60a5fa]" /></span>
        </nav>
      </div>
    </div>
  </header>
</template>

<script setup>
import { computed, ref, watch, onMounted, onUnmounted, onActivated, onDeactivated } from 'vue'
const props = defineProps({ initialKeyword: { type: String, default: '' }, immersive: { type: Boolean, default: false }, channel: { type: String, default: 'recommended' }, unreadAuthors: { type: Number, default: 0 } })
defineEmits(['update:channel'])
const header = ref(null)
const tabs = [{ key: 'recommended', label: '推荐' }, { key: 'image', label: '图文' }, { key: 'video', label: '视频' }, { key: 'hot', label: '热榜' }, { key: 'following', label: '关注' }]
const activeTabIndex = computed(() => Math.max(0, tabs.findIndex(tab => tab.key === props.channel)))
const tabsCollapsed = ref(false)
let scrollTarget
let previousOverflowAnchor
let lastScroll = 0
const scrollPosition = () => scrollTarget === window ? window.scrollY : scrollTarget?.scrollTop || 0
// 跟随实际内容滚动容器：下滑收起分类，上滑或回到顶部时展开。
const handleScroll = () => {
  const current = scrollPosition()
  if (current <= 24) { tabsCollapsed.value = false; lastScroll = current; return }
  if (Math.abs(current - lastScroll) < 8) return
  if (current > lastScroll && current > 72) tabsCollapsed.value = true
  else if (current < lastScroll) tabsCollapsed.value = false
  lastScroll = current
}
const stopScrollTracking = () => {
  scrollTarget?.removeEventListener('scroll', handleScroll)
  if (scrollTarget && scrollTarget !== window) scrollTarget.style.overflowAnchor = previousOverflowAnchor
  scrollTarget = null
}
const startScrollTracking = () => {
  stopScrollTracking()
  if (!props.immersive) return
  scrollTarget = header.value?.closest('main') || window
  // 标签高度变化时不让滚动锚点自动补偿，避免误判为反向滚动。
  if (scrollTarget !== window) {
    previousOverflowAnchor = scrollTarget.style.overflowAnchor
    scrollTarget.style.overflowAnchor = 'none'
  }
  lastScroll = scrollPosition()
  if (lastScroll <= 24) tabsCollapsed.value = false
  scrollTarget.addEventListener('scroll', handleScroll, { passive: true })
}
onMounted(startScrollTracking)
onActivated(startScrollTracking)
onDeactivated(stopScrollTracking)
onUnmounted(stopScrollTracking)
const keyword = ref(props.initialKeyword)
const showPanel = ref(false)
const panel = ref(null)
const searchContainer = ref(null)
watch(() => props.initialKeyword, value => { keyword.value = value })
const search = value => {
  const query = value?.trim()
  if (!query) return
  panel.value?.saveHistory(query)
  showPanel.value = false
  return navigateTo('/discover/search?q=' + encodeURIComponent(query))
}
const outside = event => { if (!searchContainer.value?.contains(event.target)) showPanel.value = false }
onMounted(() => document.addEventListener('click', outside))
onUnmounted(() => document.removeEventListener('click', outside))
</script>

<style scoped>
/* 从搜索栏下沿展开/收起，裁切面板而不压缩文字；负边距保留原阴影。 */
:deep(.search-panel-enter-active), :deep(.search-panel-leave-active) {
  transition: clip-path 280ms cubic-bezier(.22,.75,.25,1), opacity 180ms ease;
}
:deep(.search-panel-enter-active), :deep(.search-panel-leave-active) {
  clip-path: inset(-16px -16px -16px -16px);
}
:deep(.search-panel-enter-from), :deep(.search-panel-leave-to) {
  clip-path: inset(0 -16px 100% -16px);
  opacity: 0;
}
:deep(.search-panel-leave-active) { pointer-events: none; }
@media (prefers-reduced-motion: reduce) {
  :deep(.search-panel-enter-active), :deep(.search-panel-leave-active) { transition: none; }
}
</style>
