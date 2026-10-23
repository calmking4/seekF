<template>
  <div class="absolute top-full left-0 right-0 z-20 max-h-[70vh] overflow-y-auto rounded-b-xl bg-white shadow-lg" aria-label="聊天记录搜索结果">
    <div v-if="keyword.trim()" class="p-3">
      <div class="mb-2 text-xs text-gray-400">{{ mode === 'search' ? `搜索结果（${total}条）` : '搜索联想 · Enter查看全部' }}</div>
      <button v-for="item in results" :key="item.uuid" type="button" class="flex w-full cursor-pointer items-center gap-3 rounded-lg px-2 py-2.5 text-left transition-colors hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" @click="$emit('select', item)">
        <Icon name="uil:search" class="shrink-0 text-sm text-gray-400" />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm text-gray-700"><span v-for="(part, index) in splitChatSearchText(item.content, keyword)" :key="index" :class="part.match ? 'font-medium text-blue-500' : ''">{{ part.text }}</span></span>
          <span class="mt-0.5 block text-xs text-gray-400">{{ item.send_name }} · {{ formatTime(item.created_at) }}</span>
        </span>
      </button>
      <p v-if="!loading && !error && !results.length" class="py-6 text-center text-sm text-gray-400">没有找到相关聊天记录</p>
      <p v-if="error" class="py-3 text-center text-xs text-gray-500" role="alert">{{ error }}<button type="button" class="ml-2 text-blue-500" @click="retry">重试</button></p>
      <button v-if="mode === 'search' && results.length < total && !error" type="button" class="w-full py-3 text-sm text-blue-500 disabled:opacity-50" :disabled="loading" @click="fetchPage(page + 1, true)">{{ loading ? '加载中…' : '加载更多结果' }}</button>
    </div>
    <div v-if="loading && !results.length" class="flex justify-center p-4" role="status" aria-label="正在搜索"><Icon name="uil:spinner" class="animate-spin text-xl text-gray-400" /></div>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue'
import { splitChatSearchText } from '~/utils/chatSearch'
const props = defineProps({ keyword: { type: String, default: '' }, userId: { type: String, default: '' } })
defineEmits(['select'])
const results = ref([])
const mode = ref('suggestions')
const loading = ref(false)
const error = ref('')
const page = ref(0)
const total = ref(0)
let timer
let requestId = 0
let controller
let disposed = false
let failedPage = 1

// cancelRequest 取消防抖及在途查询，使旧关键词或旧账号的结果失效。
const cancelRequest = () => { clearTimeout(timer); controller?.abort(); requestId++; loading.value = false }
// fetchPage 查询联想或完整结果分页，失败保留已加载结果及原页码供重试。
const fetchPage = async (targetPage = 1, append = false) => {
  clearTimeout(timer)
  controller?.abort()
  const id = ++requestId
  const keyword = props.keyword.trim()
  if (!keyword) return
  controller = new AbortController()
  loading.value = true; error.value = ''; failedPage = targetPage
  try {
    const response = await useApi$(mode.value === 'search' ? '/user/message/search' : '/user/message/searchSuggestions', {
      body: { keyword, page: targetPage, page_size: mode.value === 'search' ? 20 : 5 }, signal: controller.signal
    })
    if (disposed || id !== requestId) return
    if (response.code !== 200) throw new Error(response.message || '搜索失败，请重试')
    const list = response.data?.list || []
    const existing = new Set(append ? results.value.map(item => item.uuid) : [])
    results.value = append ? [...results.value, ...list.filter(item => !existing.has(item.uuid))] : list
    total.value = mode.value === 'search' ? Number(response.data?.total) || 0 : list.length
    page.value = targetPage
  } catch (cause) {
    if (!disposed && id === requestId) error.value = cause?.data?.message || cause?.message || '搜索失败，请重试'
  } finally { if (!disposed && id === requestId) loading.value = false }
}

// searchNow Enter触发完整搜索，取消尚未执行的联想。
const searchNow = () => {
  cancelRequest(); mode.value = 'search'; results.value = []; page.value = 0; total.value = 0; error.value = ''
  if (!props.keyword.trim()) return
  return fetchPage(1)
}
const retry = () => fetchPage(failedPage, page.value > 0)
const formatTime = value => value ? new Date(value).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : ''

watch(() => props.keyword, keyword => {
  cancelRequest(); mode.value = 'suggestions'; results.value = []; page.value = 0; total.value = 0; error.value = ''
  if (keyword.trim()) { loading.value = true; timer = setTimeout(() => { void fetchPage(1) }, 300) }
}, { immediate: true })
watch(() => props.userId, () => { cancelRequest(); results.value = []; page.value = 0; total.value = 0; mode.value = 'suggestions'; error.value = '' })
onUnmounted(() => { disposed = true; cancelRequest() })
defineExpose({ searchNow })
</script>
