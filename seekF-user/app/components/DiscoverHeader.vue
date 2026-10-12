<template>
  <header class="sticky top-0 z-20 bg-white shadow-sm px-4 py-3">
    <div class="flex items-center justify-between">
      <div ref="searchContainer" class="w-[30rem] max-w-[calc(100%-5rem)] mx-auto relative">
        <form class="flex items-center bg-gray-100 rounded-full px-3 py-2 gap-2" @submit.prevent="search(keyword)">
          <Icon name="mdi:magnify" class="text-gray-400 text-lg flex-shrink-0" />
          <input v-model="keyword" type="text" aria-label="搜索发现内容" placeholder="搜索" class="bg-transparent outline-none text-sm w-full text-gray-700 placeholder-gray-400" @focus="showPanel = true" @keydown.esc="showPanel = false" />
        </form>
        <DiscoverSearchPanel v-if="showPanel" ref="panel" :keyword="keyword" @select="search" />
      </div>
      <div class="flex items-center gap-4 text-gray-600 text-xl absolute right-4">
        <NuxtLink to="/notification" aria-label="查看通知" class="text-[#60a5fa] focus-visible:outline-blue-500"><Icon name="mdi:message-outline" /></NuxtLink>
        <NuxtLink to="/discover/create" aria-label="发布动态" class="text-[#60a5fa] cursor-pointer focus-visible:outline-blue-500"><Icon name="mdi:plus-circle-outline" /></NuxtLink>
      </div>
    </div>
  </header>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'
const props = defineProps({ initialKeyword: { type: String, default: '' } })
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
