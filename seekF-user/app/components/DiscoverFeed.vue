<template>
  <section class="p-4" :aria-busy="loading">
    <p v-if="search" class="mb-4 text-sm text-gray-500">{{ keyword ? '“' + keyword + '” 的搜索结果' : '输入关键词开始搜索' }}<span v-if="!loading && !error && keyword"> · {{ total }} 条</span></p>
    <div v-if="search && (!mounted || initialLoading)" class="py-16 flex flex-col items-center justify-center text-gray-500" role="status">
      <Icon name="uil:spinner" class="animate-spin text-3xl mb-3" />
      <span class="text-sm">搜索中...</span>
    </div>
    <DiscoverPostGrid v-if="items.length" :posts="items" @open="selectedItem = $event" @like-updated="updateLike" />
    <div v-if="error" class="my-8 rounded-xl border border-red-100 bg-white p-6 text-center" role="alert">
      <p class="mb-3 text-sm text-gray-600">{{ error }}</p><button type="button" class="rounded-full bg-blue-500 px-5 py-2 text-sm text-white hover:bg-blue-600 focus-visible:outline-blue-500" :disabled="loading" @click="loadMore">重新加载</button>
    </div>
    <div v-else-if="mounted && !loading && !items.length" class="py-20 text-center text-gray-400">
      <Icon :name="search ? 'mdi:magnify' : 'mdi:compass-outline'" class="mb-3 text-4xl" />
      <p class="text-sm">{{ search ? (keyword ? '未找到相关动态，试试其他关键词' : '搜索你感兴趣的内容') : '社区暂无动态，分享第一条内容吧' }}</p>
      <NuxtLink v-if="!search" to="/discover/create" class="mt-4 inline-block rounded-full bg-blue-500 px-5 py-2 text-sm text-white hover:bg-blue-600">发布动态</NuxtLink>
    </div>
    <div v-if="(!mounted || loading) && (!search || items.length)" class="py-8 flex justify-center items-center text-gray-500" role="status">
      <Icon name="uil:spinner" class="animate-spin text-xl mr-2" />
      <span>加载中...</span>
    </div>
    <div ref="sentinel" class="h-1" />
    <div v-if="items.length && !loading && !error" class="py-8 text-center text-gray-400 text-sm">
      <span v-if="noMore">没有更多内容了</span>
      <button v-else type="button" class="rounded px-4 py-2 text-blue-500 hover:bg-blue-50 focus-visible:outline-blue-500" @click="loadMore">加载更多</button>
    </div>
    <DiscoverDetail v-if="selectedItem" :item="selectedItem" @close="selectedItem = null" @like-updated="updateLike" />
  </section>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'
const props = defineProps({ search: { type: Boolean, default: false }, keyword: { type: String, default: '' }, channel: { type: String, default: 'recommended' } })
const { items, loading, initialLoading, error, noMore, total, loadMore, reset, updateLike } = useDiscoverFeed(props.search ? () => props.keyword : undefined, () => props.channel)
const mounted = ref(false)
const selectedItem = ref(null)
const sentinel = ref(null)
let observer
onMounted(() => {
  mounted.value = true
  reset()
  observer = new IntersectionObserver(entries => {
    if (entries[0].isIntersecting && items.value.length && !error.value) loadMore()
  }, { rootMargin: '200px' })
  if (sentinel.value) observer.observe(sentinel.value)
})
watch(() => props.keyword, () => { selectedItem.value = null; reset() })
onUnmounted(() => observer?.disconnect())
</script>
