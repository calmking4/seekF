<template>
  <div v-if="posts && posts.length > 0" class="mt-2 border border-gray-200 rounded-lg bg-gray-50 overflow-hidden">
    <button
      @click="expanded = !expanded"
      class="w-full flex items-center justify-between px-3 py-2 text-xs text-gray-500 hover:bg-gray-100 transition-colors"
    >
      <span class="flex items-center gap-1">
        <Icon name="uil:compass" class="text-sm" />
        已找到 {{ posts.length }} 个帖子
      </span>
      <Icon :name="expanded ? 'uil:angle-up' : 'uil:angle-down'" class="text-sm" />
    </button>
    <div v-show="expanded" class="px-3 pb-2 space-y-2">
      <div
        v-for="post in posts"
        :key="post.id"
        @click="$emit('post-click', post)"
        class="flex items-start gap-3 p-2 rounded hover:bg-white cursor-pointer transition-colors group"
      >
        <!-- 缩略图 -->
        <img
          v-if="post.src"
          :src="post.src"
          class="w-16 h-16 rounded object-cover flex-shrink-0"
        />
        <div
          v-else
          class="w-16 h-16 rounded bg-gray-200 flex items-center justify-center flex-shrink-0"
        >
          <Icon name="uil:image" class="text-gray-400 text-lg" />
        </div>
        <!-- 帖子信息 -->
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-gray-700 truncate group-hover:text-blue-600">
            {{ post.title }}
          </p>
          <div class="flex items-center gap-1 mt-1">
            <el-avatar v-if="post.avatar" :size="20" :src="post.avatar" />
            <span class="text-xs text-gray-500">{{ post.nickname }}</span>
          </div>
          <!-- 标签 -->
          <div v-if="post.tags && post.tags.length > 0" class="flex flex-wrap gap-1 mt-1">
            <el-tag
              v-for="tag in post.tags"
              :key="tag"
              type="info"
              size="small"
              class="!text-xs !h-5 !leading-5"
            >
              {{ tag }}
            </el-tag>
          </div>
          <div class="flex items-center gap-3 mt-1 text-xs text-gray-400">
            <span class="flex items-center gap-0.5">
              <Icon name="uil:thumbs-up" class="text-xs" />
              {{ post.like_count }}
            </span>
            <span class="flex items-center gap-0.5">
              <Icon name="uil:comment-dots" class="text-xs" />
              {{ post.comment_count }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  posts: { type: Array, default: () => [] }
})

defineEmits(['post-click'])

const expanded = ref(false)
</script>
