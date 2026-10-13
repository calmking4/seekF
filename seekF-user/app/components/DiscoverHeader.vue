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
      <div class="absolute right-4 flex items-center">
        <NuxtLink to="/discover/create" aria-label="发布动态" title="发布动态" class="flex h-8 w-8 items-center justify-center rounded-full bg-[#60a5fa] text-white shadow-sm hover:bg-blue-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"><Icon name="mdi:plus" class="text-2xl" /></NuxtLink>
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
