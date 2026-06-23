<template>
  <div class="absolute top-full left-0 right-0 bg-white shadow-lg rounded-b-xl z-20 max-h-[70vh] overflow-y-auto">
    <!-- 联想结果 -->
    <div v-if="suggestions.length > 0" class="p-3">
      <div class="text-xs text-gray-400 mb-2">搜索联想</div>
      <div
        v-for="item in suggestions"
        :key="item.uuid"
        class="flex items-center gap-3 px-2 py-2.5 hover:bg-gray-50 rounded-lg cursor-pointer transition-colors"
        @click="$emit('select', item.title)"
      >
        <Icon name="uil:search" class="text-gray-400 text-sm flex-shrink-0" />
        <div class="flex-1 min-w-0">
          <p class="text-sm text-gray-700 truncate" v-html="highlightKeyword(item.title)"></p>
        </div>
      </div>
    </div>

    <!-- 无联想时显示热门搜索和历史 -->
    <template v-else>
      <!-- 热门搜索 -->
      <div class="p-3 border-b border-gray-100">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400">热门搜索</span>
        </div>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="tag in hotSearches"
            :key="tag"
            class="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-sm text-gray-600 rounded-full cursor-pointer transition-colors"
            @click="$emit('select', tag)"
          >
            {{ tag }}
          </span>
        </div>
      </div>

      <!-- 搜索历史 -->
      <div v-if="history.length > 0" class="p-3">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400">搜索历史</span>
          <button
            class="text-xs text-gray-400 hover:text-gray-600 transition-colors"
            @click.stop="handleClearHistory"
          >
            清空
          </button>
        </div>
        <div
          v-for="(item, index) in history"
          :key="index"
          class="flex items-center justify-between px-2 py-2 hover:bg-gray-50 rounded-lg cursor-pointer transition-colors group"
          @click="$emit('select', item)"
        >
          <div class="flex items-center gap-3">
            <Icon name="uil:clock" class="text-gray-400 text-sm" />
            <span class="text-sm text-gray-700">{{ item }}</span>
          </div>
          <button
            class="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-gray-600 transition-all"
            @click.stop="handleDeleteHistory(index)"
          >
            <Icon name="uil:times" class="text-sm" />
          </button>
        </div>
      </div>
    </template>

    <!-- 加载状态 -->
    <div v-if="loading" class="p-4 flex justify-center">
      <Icon name="uil:spinner" class="animate-spin text-xl text-gray-400" />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'

const props = defineProps({
  keyword: { type: String, default: '' },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['select', 'clear-history'])

// 热门搜索（硬编码，后续可从后端获取）
const hotSearches = ref([
  '旅行', '美食', '摄影', '穿搭', '健身',
  '音乐', '电影', '读书', '宠物', '手工'
])

// 搜索历史
const history = ref([])

// 联想结果
const suggestions = ref([])

// 从 localStorage 加载搜索历史
const loadHistory = () => {
  try {
    const saved = localStorage.getItem('discover_search_history')
    if (saved) {
      history.value = JSON.parse(saved)
    }
  } catch {
    history.value = []
  }
}

// 保存搜索历史
const saveHistory = (keyword) => {
  if (!keyword?.trim()) return
  // 去重并移到最前面
  const filtered = history.value.filter(k => k !== keyword)
  filtered.unshift(keyword.trim())
  // 最多保存 10 条
  history.value = filtered.slice(0, 10)
  localStorage.setItem('discover_search_history', JSON.stringify(history.value))
}

// 删除单条历史
const handleDeleteHistory = (index) => {
  history.value.splice(index, 1)
  localStorage.setItem('discover_search_history', JSON.stringify(history.value))
}

// 清空历史
const handleClearHistory = () => {
  history.value = []
  localStorage.removeItem('discover_search_history')
  emit('clear-history')
}

// 高亮关键词
const highlightKeyword = (text) => {
  if (!props.keyword || !text) return text
  const regex = new RegExp(`(${props.keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  return text.replace(regex, '<span class="text-blue-500 font-medium">$1</span>')
}

// 防抖定时器
let debounceTimer = null

// 监听关键词变化，触发联想
watch(() => props.keyword, (newVal) => {
  if (debounceTimer) clearTimeout(debounceTimer)

  if (!newVal?.trim()) {
    suggestions.value = []
    return
  }

  debounceTimer = setTimeout(async () => {
    try {
      const res = await useApi$('/user/discover/search', {
        body: { keyword: newVal.trim(), page: 1, page_size: 5 }
      })
      if (res.code === 200 && res.data?.list) {
        suggestions.value = res.data.list
      } else {
        suggestions.value = []
      }
    } catch {
      suggestions.value = []
    }
  }, 300)
})

// 暴露保存历史方法供父组件调用
defineExpose({
  saveHistory
})

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
/* 自定义滚动条 */
::-webkit-scrollbar {
  width: 4px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: #e5e7eb;
  border-radius: 2px;
}

::-webkit-scrollbar-thumb:hover {
  background: #d1d5db;
}
</style>
