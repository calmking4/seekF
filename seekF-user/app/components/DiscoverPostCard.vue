<template>
  <article :data-id="post.id" class="waterfall-item fade-in bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow" @click.stop="$emit('open', post)">
    <button type="button" class="w-full relative block overflow-hidden focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500" :style="{ aspectRatio: discoverAspectRatio(post) }" :aria-label="'查看动态：' + post.title" @click.stop="$emit('open', post)">
      <img v-if="imageSrc && !mediaFailed" :src="imageSrc" :alt="post.title" class="h-full w-full object-cover" loading="lazy" decoding="async" @error="mediaFailed = true" />
      <video v-else-if="post.type === 'video' && post.src && !mediaFailed" :src="post.src" class="h-full w-full object-cover" preload="metadata" muted playsinline @error="mediaFailed = true" />
      <div v-else class="flex h-full items-center justify-center text-gray-400"><Icon name="mdi:image-outline" class="text-3xl" /><span class="sr-only">媒体暂时无法显示</span></div>
    </button>
    <div class="p-3">
      <button type="button" class="text-sm font-medium line-clamp-2 mb-2 block w-full text-left focus-visible:outline-blue-500" @click.stop="$emit('open', post)"><span class="line-clamp-2">{{ post.title }}</span></button>
      <div class="flex items-center justify-between text-xs text-gray-500">
        <NuxtLink v-if="post.user_id" :to="'/user/' + encodeURIComponent(post.user_id)" class="flex min-w-0 items-center gap-2 text-left hover:text-blue-500" @click.stop>
          <img v-if="post.avatar" :src="post.avatar" alt="" class="w-7 h-7 rounded-full object-cover flex-shrink-0" />
          <span v-else class="w-7 h-7 rounded-full flex items-center justify-center text-white text-xs flex-shrink-0" :style="{ backgroundColor: getAvatarColor(post.id) }">{{ post.title.slice(0, 1) }}</span>
          <span class="truncate">{{ post.nickname }}</span>
        </NuxtLink>
        <span v-else class="truncate">{{ post.nickname }}</span>
        <button type="button" class="flex items-center gap-1 cursor-pointer focus-visible:outline-blue-500 disabled:opacity-50" :class="{ 'text-red-500': post.isLiked }" :aria-label="post.isLiked ? '取消点赞' : '点赞'" :aria-pressed="post.isLiked" :disabled="isPending(post.id)" @click.stop="like">
          <Icon :name="post.isLiked ? 'solar:heart-angle-bold' : 'mdi:heart-outline'" class="text-base" /><span>{{ post.likeCount }}</span>
        </button>
      </div>
    </div>
  </article>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { discoverAspectRatio } from '~/utils/discover'
const props = defineProps({ post: { type: Object, required: true } })
const emit = defineEmits(['open', 'like-updated'])
const { setLike, isPending } = useDiscoverLike()
// 沿用原卡片的确定性头像配色。
const avatarColors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8', '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA']
const getAvatarColor = id => {
  let hash = 0
  for (const char of String(id ?? '')) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return avatarColors[hash % avatarColors.length]
}
const mediaFailed = ref(false)
const imageSrc = computed(() => props.post.type === 'video' ? props.post.coverUrl : props.post.src)
watch(imageSrc, () => { mediaFailed.value = false })
const like = async () => {
  const result = await setLike(props.post.id, props.post.isLiked)
  if (result) emit('like-updated', { postId: props.post.id, likeCount: result.like_count, isLiked: result.is_liked })
}
</script>
