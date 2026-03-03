<template>
  <div class="min-h-screen bg-white">
    <!-- 顶部用户信息区 -->
    <div class="max-w-3xl mx-auto pt-8 pb-6 px-6">
      <div class="flex items-center gap-6">
        <!-- 头像 -->
        <div class="w-24 h-24 rounded-full border border-gray-100 flex items-center justify-center overflow-hidden bg-gray-50">
          <Icon name="uil:book-alt" class="text-4xl text-blue-500" />
        </div>

        <!-- 用户信息 -->
        <div class="flex flex-col gap-2">
          <h1 class="text-xl font-semibold text-gray-900">小红薯665AE357</h1>
          <p class="text-gray-500 text-sm">小红书号: 11678956516</p>
          <p class="text-gray-600 text-sm">还没有简介</p>
          <div class="flex gap-6 text-sm text-gray-600 mt-1">
            <span><span class="font-medium">0</span> 关注</span>
            <span><span class="font-medium">0</span> 粉丝</span>
            <span><span class="font-medium">0</span> 获赞与收藏</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 标签切换区 -->
    <div class="max-w-3xl mx-auto">
      <el-tabs v-model="activeTab" class="w-full">
        <!-- 笔记（发布）标签 -->
        <el-tab-pane label="笔记" name="notes">
          <div class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:smile" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">你还没有发布任何内容哦</p>
          </div>
        </el-tab-pane>

        <!-- 收藏标签 -->
        <el-tab-pane label="收藏" name="collections">
          <div class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:bookmark" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">暂无收藏内容</p>
          </div>
        </el-tab-pane>

        <!-- 点赞标签 -->
        <el-tab-pane label="点赞" name="likes">
          <div class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:heart" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">暂无点赞内容</p>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const activeTab = ref('notes')
</script>

<style scoped>
/* 自定义 Element Plus 标签样式，匹配小红书风格 */
:deep(.el-tabs__header) {
  margin: 0;
  border-bottom: 1px solid #e5e7eb;
}

:deep(.el-tabs__nav-wrap) {
  padding: 0;
}

:deep(.el-tabs__nav) {
  display: flex;
  border: none;
}

:deep(.el-tabs__item) {
  padding: 0 24px;
  height: 48px;
  line-height: 48px;
  color: #6b7280;
  font-size: 15px;
  border: none;
  position: relative;
}

:deep(.el-tabs__item.is-active) {
  color: #111827;
  font-weight: 500;
}

:deep(.el-tabs__item.is-active::after) {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 24px;
  height: 2px;
  background-color: #111827;
}

:deep(.el-tabs__active-bar) {
  display: none;
}
</style>