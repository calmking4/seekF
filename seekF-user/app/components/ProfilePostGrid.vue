<template>
  <div class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
    <article v-for="post in posts" :key="post.uuid" role="button" tabindex="0" :aria-label="`查看帖子：${post.title || '未命名帖子'}`"
      class="profile-card overflow-hidden rounded-xl border border-gray-100 bg-white shadow-sm cursor-pointer transition-shadow hover:shadow-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400"
      @click="openPost(post)" @keydown.enter.self.prevent="openPost(post)" @keydown.space.self.prevent="openPost(post)">
      <div class="relative aspect-[4/3] bg-gray-100">
        <span v-if="post.visibility === 1" class="absolute left-2 top-2 z-10 rounded bg-black/40 px-2 py-1 text-xs text-white">仅自己可见</span>
        <div v-if="post.is_blocked_by" class="flex h-full flex-col items-center justify-center gap-2 text-gray-400"><Icon name="mdi:account-lock-outline" class="text-3xl" /><span class="text-xs">内容暂不可见</span></div>
        <img v-else-if="post.cover_url || (post.media_type !== 1 && post.first_url)" :src="post.cover_url || post.first_url" :alt="post.title" class="h-full w-full object-cover" loading="lazy" />
        <div v-else class="flex h-full items-center justify-center text-gray-300"><Icon :name="post.media_type === 1 ? 'uil:video' : 'uil:image'" class="text-4xl" /></div>
        <span v-if="post.media_type === 1" class="absolute right-2 rounded-full bg-black/40 p-1 text-white" :class="post.support_only === 1 ? 'bottom-2' : 'top-2'"><Icon name="mdi:play" class="block text-lg" /></span>
        <span v-if="post.support_only === 1" class="pointer-events-none absolute right-2 top-2 inline-flex h-6 items-center justify-center gap-1 rounded-md bg-[#60A5FA] pl-1.5 pr-2 text-xs font-semibold text-white"><Icon name="mdi:lightning-bolt" class="block shrink-0 text-base" aria-hidden="true" /><span class="block leading-none">助力专属</span></span>
      </div>
      <div class="p-3">
        <h3 class="mb-3 line-clamp-2 text-sm font-medium text-gray-800">{{ post.title || '未命名帖子' }}</h3>
        <div class="flex items-center justify-between gap-2 text-xs text-gray-500">
          <NuxtLink v-if="post.user_id" :to="`/user/${encodeURIComponent(post.user_id)}`" class="flex min-w-0 items-center gap-2 hover:text-blue-500" @click.stop>
            <el-avatar :size="24" :src="post.avatar"><Icon name="uil:user" /></el-avatar><span class="truncate">{{ post.nickname || '用户' }}</span>
          </NuxtLink>
          <span v-else class="truncate">{{ post.nickname || '用户' }}</span>
          <button type="button" class="flex shrink-0 items-center gap-1 hover:text-red-500" :class="{ 'text-red-500': post.is_liked }" :aria-label="post.is_liked ? '取消点赞' : '点赞'" :disabled="post.is_blocked_by" @click.stop="$emit('like', post)">
            <Icon :name="post.is_liked ? 'mdi:heart' : 'mdi:heart-outline'" class="text-base" /><span>{{ post.like_count || 0 }}</span>
          </button>
        </div>
      </div>
    </article>
  </div>
</template>

<script setup>
import { ElMessage } from 'element-plus'
defineProps({ posts: { type: Array, default: () => [] } })
const emit = defineEmits(['open', 'like'])
const openPost = post => { if (post.is_blocked_by) { ElMessage.warning('已被拉黑'); return }; emit('open',post) }
</script>
