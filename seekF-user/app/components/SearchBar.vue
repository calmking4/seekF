<template>
  <div class="p-3 border-b border-gray-100 flex items-center gap-3">
    <div class="flex-1 relative" ref="searchContainerRef">
      <input
        type="text"
        v-model="searchKeyword"
        placeholder="搜索聊天记录"
        class="w-full pl-4 pr-4 py-2 rounded-lg bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm"
        @focus="showSearchPanel = true"
      >
      <!-- 搜索联想面板 -->
      <ChatSearchPanel
        v-if="showSearchPanel"
        ref="chatSearchPanelRef"
        :keyword="searchKeyword"
        @select="handleSearchSelect"
        @clear-history="handleClearHistory"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['search-select'])

// 搜索相关
const searchKeyword = ref('')
const showSearchPanel = ref(false)
const searchContainerRef = ref(null)
const chatSearchPanelRef = ref(null)

// 处理搜索联想选中
const handleSearchSelect = (item) => {
  // 保存搜索历史
  if (chatSearchPanelRef.value && typeof item === 'string') {
    chatSearchPanelRef.value.saveHistory(item)
  } else if (chatSearchPanelRef.value && typeof item === 'object') {
    chatSearchPanelRef.value.saveHistory(searchKeyword.value)
  }
  showSearchPanel.value = false
  searchKeyword.value = ''
  emit('search-select', item)
}

// 清空搜索历史
const handleClearHistory = () => {
  // 历史已在组件内清空
}

// 点击外部关闭搜索面板
const handleClickOutside = (event) => {
  if (searchContainerRef.value && !searchContainerRef.value.contains(event.target)) {
    showSearchPanel.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

</script>
