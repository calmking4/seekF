<template>
  <div class="absolute top-full left-0 right-0 bg-white shadow-lg rounded-b-xl z-20 max-h-[70vh] overflow-y-auto">
    <div v-if="keyword.trim()" class="p-3">
      <div class="text-xs text-gray-400 mb-2">搜索联想</div>
      <button v-for="item in suggestions" :key="item.uuid" type="button" class="w-full text-left flex items-center gap-3 px-2 py-2.5 hover:bg-gray-50 rounded-lg cursor-pointer transition-colors focus-visible:outline-blue-500" @click="$emit('select', item.title)">
        <Icon name="uil:search" class="text-gray-400 text-sm flex-shrink-0" />
        <span class="flex-1 min-w-0 text-sm text-gray-700 truncate"><span v-for="(part, index) in highlightParts(item.title)" :key="index" :class="{ 'text-blue-500 font-medium': part.matched }">{{ part.text }}</span></span>
      </button>
      <p v-if="error" class="py-3 text-xs text-gray-500">{{ error }}</p>
      <p v-else-if="!searching && !suggestions.length" class="py-3 text-xs text-gray-400">暂无联想结果，可按回车搜索</p>
    </div>
    <template v-else>
      <div class="p-3 border-b border-gray-100">
        <div class="flex items-center justify-between mb-2"><span class="text-xs text-gray-400">探索话题</span></div>
        <div class="flex flex-wrap gap-2"><button v-for="tag in topics" :key="tag" type="button" class="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-sm text-gray-600 rounded-full cursor-pointer transition-colors focus-visible:outline-blue-500" @click="$emit('select', tag)">{{ tag }}</button></div>
      </div>
      <div v-if="history.length" class="p-3">
        <div class="flex items-center justify-between mb-2"><span class="text-xs text-gray-400">搜索历史</span><button type="button" class="text-xs text-gray-400 hover:text-gray-600 transition-colors" @click="clearHistory">清空</button></div>
        <div v-for="(item, index) in history" :key="item" class="flex items-center justify-between px-2 py-2 hover:bg-gray-50 rounded-lg cursor-pointer transition-colors group">
          <button type="button" class="flex min-w-0 flex-1 items-center gap-3 text-left focus-visible:outline-blue-500" @click="$emit('select', item)"><Icon name="uil:clock" class="text-gray-400 text-sm flex-shrink-0" /><span class="text-sm text-gray-700 truncate">{{ item }}</span></button>
          <button type="button" :aria-label="'删除搜索历史：' + item" class="opacity-0 group-hover:opacity-100 focus-visible:opacity-100 text-gray-400 hover:text-gray-600 transition-all" @click="deleteHistory(index)"><Icon name="uil:times" class="text-sm" /></button>
        </div>
      </div>
    </template>
    <div v-if="searching" class="p-4 flex justify-center" role="status"><Icon name="uil:spinner" class="animate-spin text-xl text-gray-400" /></div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'
const props = defineProps({ keyword: { type: String, default: '' } })
const emit = defineEmits(['select', 'clear-history'])
const topics = ['旅行', '美食', '摄影', '读书', '科技', '音乐']
const history = ref([])
const suggestions = ref([])
const searching = ref(false)
const error = ref('')
// 只渲染文本片段，恢复原关键词高亮且不使用 v-html。
const highlightParts = text => {
  const source = String(text || '')
  const keyword = props.keyword.trim()
  if (!keyword) return [{ text: source, matched: false }]
  const parts = []
  const query = keyword.toLocaleLowerCase()
  const lowered = source.toLocaleLowerCase()
  let cursor = 0
  let index = lowered.indexOf(query, cursor)
  while (index >= 0) {
    if (index > cursor) parts.push({ text: source.slice(cursor, index), matched: false })
    parts.push({ text: source.slice(index, index + keyword.length), matched: true })
    cursor = index + keyword.length
    index = lowered.indexOf(query, cursor)
  }
  if (cursor < source.length) parts.push({ text: source.slice(cursor), matched: false })
  return parts
}
const historyKey = 'discover_search_history'
let timer
let version = 0
const persistHistory = () => { try { localStorage.setItem(historyKey, JSON.stringify(history.value)) } catch { /* 存储不可用时保留当前会话历史。 */ } }
const saveHistory = value => {
  const keyword = value?.trim()
  if (!keyword) return
  history.value = [keyword, ...history.value.filter(item => item !== keyword)].slice(0, 10)
  persistHistory()
}
const deleteHistory = index => { history.value.splice(index, 1); persistHistory() }
const clearHistory = () => { history.value = []; persistHistory(); emit('clear-history') }
watch(() => props.keyword, value => {
  clearTimeout(timer)
  const current = ++version
  suggestions.value = []
  error.value = ''
  searching.value = !!value.trim()
  if (!value.trim()) return
  timer = setTimeout(async () => {
    try {
      const response = await useApi$('/user/discover/search', { body: { keyword: value.trim(), page: 1, page_size: 5 }, retry: 0 })
      if (current !== version) return
      if (response.code !== 200) throw new Error('获取搜索联想失败')
      suggestions.value = response.data?.list || []
    } catch {
      if (current === version) error.value = '联想暂不可用，可按回车搜索'
    } finally { if (current === version) searching.value = false }
  }, 300)
}, { immediate: true })
onMounted(() => {
  try {
    const saved = JSON.parse(localStorage.getItem(historyKey) || '[]')
    if (Array.isArray(saved)) history.value = saved.filter(item => typeof item === 'string').slice(0, 10)
  } catch { history.value = [] }
})
onUnmounted(() => { version++; clearTimeout(timer) })
defineExpose({ saveHistory })
</script>
