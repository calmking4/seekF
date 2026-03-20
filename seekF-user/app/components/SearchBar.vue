<template>
  <div class="p-3 border-b border-gray-100 flex items-center gap-3">
    <div class="flex-1 relative">
      <input
        type="text"
        placeholder="搜索"
        class="w-full pl-10 pr-4 py-2 rounded-lg bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm"
      >
      <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">🔍</span>
    </div>
    <button @click="showCreateMenu = !showCreateMenu" class="text-gray-500 hover:text-blue-500 transition-colors">
      <Icon name="uil:plus" class="w-5 h-5" />
    </button>
    <!-- 创建菜单 -->
    <div v-if="showCreateMenu" class="absolute top-14 right-3 bg-white rounded-lg shadow-lg border border-gray-200 py-2 z-10">
      <div class="px-4 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2">
        <Icon name="uil:users-alt" class="w-4 h-4" />
        <span>创建群聊</span>
      </div>
      <div class="px-4 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2">
        <Icon name="uil:user-plus" class="w-4 h-4" />
        <span>添加好友</span>
      </div>
      <div class="px-4 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2">
        <Icon name="uil:comment-alt" class="w-4 h-4" />
        <span>加入群聊</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const showCreateMenu = ref(false)
</script>

<style scoped>
/* 定位样式 */
.p-3 {
  position: relative;
}
</style>