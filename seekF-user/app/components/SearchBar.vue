<template>
  <div class="p-3 border-b border-gray-100 flex items-center gap-3">
    <div class="flex-1 relative" ref="searchContainerRef">
      <input
        type="text"
        v-model="searchKeyword"
        placeholder="搜索聊天记录"
        maxlength="200"
        class="w-full pl-4 pr-4 py-2 rounded-lg bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm"
        @focus="showSearchPanel = true"
        @keydown.enter="submitSearch"
        @keydown.esc="showSearchPanel = false"
      >
      <!-- 搜索联想面板 -->
      <ChatSearchPanel
        v-if="showSearchPanel && searchKeyword.trim()"
        ref="chatSearchPanelRef"
        :keyword="searchKeyword"
        :user-id="userId"
        @select="handleSearchSelect"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'

const props = defineProps({ userId: { type: String, default: '' } })

const emit = defineEmits(['search-select'])

// 搜索相关
const searchKeyword = ref('')
const showSearchPanel = ref(false)
const searchContainerRef = ref(null)
const chatSearchPanelRef = ref(null)
watch(() => props.userId, () => { searchKeyword.value = ''; showSearchPanel.value = false })

// 处理搜索联想选中
const handleSearchSelect = (item) => {
  showSearchPanel.value = false
  searchKeyword.value = ''
  emit('search-select', item)
}

// submitSearch Enter实际执行完整搜索，输入法组合输入期间不触发。
const submitSearch = async event => {
  if (event?.isComposing || event?.keyCode === 229) return
  event?.preventDefault()
  showSearchPanel.value = true
  await nextTick()
  void chatSearchPanelRef.value?.searchNow()
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
