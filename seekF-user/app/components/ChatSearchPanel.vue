<template>
  <div class="absolute top-full left-0 right-0 bg-white shadow-lg rounded-b-xl z-20 max-h-[70vh] overflow-y-auto">
    <!-- 联想结果 -->
    <div v-if="suggestions.length > 0" class="p-3">
      <div class="text-xs text-gray-400 mb-2">搜索联想</div>
      <div
        v-for="(item, index) in suggestions"
        :key="index"
        class="flex items-center gap-3 px-2 py-2.5 hover:bg-gray-50 rounded-lg cursor-pointer transition-colors"
        @click="handleSelect(item)"
      >
        <Icon name="uil:search" class="text-gray-400 text-sm flex-shrink-0" />
        <div class="flex-1 min-w-0">
          <p class="text-sm text-gray-700 truncate" v-html="highlightKeyword(item.content)"></p>
          <p class="text-xs text-gray-400 mt-0.5">{{ item.send_name }} · {{ formatTime(item.created_at) }}</p>
        </div>
      </div>
    </div>

    <!-- 无联想时显示搜索历史 -->
    <template v-else>
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

      <!-- 无历史时的提示 -->
      <div v-else class="p-6 text-center text-gray-400 text-sm">
        输入关键词搜索聊天记录
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

// 搜索历史
const history = ref([])

// 联想结果
const suggestions = ref([])

// 从 localStorage 加载搜索历史
const loadHistory = () => {
  try {
    const saved = localStorage.getItem('chat_search_history')
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
  const filtered = history.value.filter(k => k !== keyword)
  filtered.unshift(keyword.trim())
  history.value = filtered.slice(0, 10)
  localStorage.setItem('chat_search_history', JSON.stringify(history.value))
}

// 删除单条历史
const handleDeleteHistory = (index) => {
  history.value.splice(index, 1)
  localStorage.setItem('chat_search_history', JSON.stringify(history.value))
}

// 清空历史
const handleClearHistory = () => {
  history.value = []
  localStorage.removeItem('chat_search_history')
  emit('clear-history')
}

// 高亮关键词
const highlightKeyword = (text) => {
  if (!props.keyword || !text) return text
  const regex = new RegExp(`(${props.keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  return text.replace(regex, '<span class="text-blue-500 font-medium">$1</span>')
}

// 格式化时间
const formatTime = (time) => {
  if (!time) return ''
  const date = new Date(time)
  const now = new Date()
  const diffMs = now - date
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays === 0) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (diffDays === 1) {
    return '昨天'
  } else if (diffDays < 7) {
    return `${diffDays}天前`
  } else {
    return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
  }
}

// 处理选中联想结果
const handleSelect = (item) => {
  emit('select', item)
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
      const res = await useApi$('/user/message/searchSuggestions', {
        body: { keyword: newVal.trim(), page_size: 5 }
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

// 暴露方法供父组件调用
defineExpose({
  saveHistory
})

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
</style>
