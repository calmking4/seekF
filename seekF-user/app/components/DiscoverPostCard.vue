<template>
  <article :data-id="post.id" class="waterfall-item fade-in bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow" @click.stop="openPost">
    <button type="button" class="w-full relative block overflow-hidden focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500" :style="{ aspectRatio: discoverAspectRatio(post) }" :aria-label="'查看动态：' + post.title" @click.stop="openPost">
      <div v-if="post.is_blocked_by" class="flex h-full flex-col items-center justify-center gap-2 bg-gray-100 text-gray-400"><Icon name="mdi:account-lock-outline" class="text-3xl" /><span class="text-xs">内容暂不可见</span></div>
      <img v-else-if="imageSrc && !mediaFailed" :src="imageSrc" :alt="post.title" class="h-full w-full object-cover" loading="lazy" decoding="async" @error="mediaFailed = true" />
      <video v-else-if="post.type === 'video' && post.src && !mediaFailed" :src="post.src" class="h-full w-full object-cover" preload="metadata" muted playsinline @error="mediaFailed = true" />
      <div v-else class="flex h-full items-center justify-center text-gray-400"><Icon name="mdi:image-outline" class="text-3xl" /><span class="sr-only">媒体暂时无法显示</span></div>
      <span v-if="post.type === 'video'" aria-hidden="true" class="pointer-events-none absolute right-2 flex h-6 w-6 items-center justify-center rounded-full bg-black/40 text-white" :class="post.support_only === 1 ? 'bottom-2' : 'top-2'"><Icon name="mdi:play" class="text-base" /></span>
      <span v-if="post.support_only === 1" class="pointer-events-none absolute right-2 top-2 inline-flex h-6 items-center justify-center gap-1 rounded-md bg-[#60A5FA] pl-1.5 pr-2 text-xs font-semibold text-white"><Icon name="mdi:lightning-bolt" class="block shrink-0 text-base" aria-hidden="true" /><span class="block leading-none">{{ discoverSupportLabel(post) }}</span></span>
    </button>
    <div class="p-3">
      <button type="button" class="text-sm font-medium line-clamp-2 mb-2 block w-full text-left focus-visible:outline-blue-500" @click.stop="openPost"><span class="line-clamp-2">{{ post.title }}</span></button>
      <div class="flex items-center justify-between text-xs text-gray-500">
        <NuxtLink v-if="post.user_id" :to="'/user/' + encodeURIComponent(post.user_id)" class="flex min-w-0 items-center gap-2 text-left hover:text-blue-500" @click.stop>
          <img v-if="post.avatar" :src="post.avatar" alt="" class="w-7 h-7 rounded-full object-cover flex-shrink-0" />
          <span v-else class="w-7 h-7 rounded-full flex items-center justify-center text-white text-xs flex-shrink-0" :style="{ backgroundColor: getAvatarColor(post.id) }">{{ post.title.slice(0, 1) }}</span>
          <span class="truncate">{{ post.nickname }}</span>
        </NuxtLink>
        <span v-else class="truncate">{{ post.nickname }}</span>
        <div class="flex shrink-0 items-center gap-2" @click.stop>
          <button type="button" class="flex items-center gap-1 cursor-pointer focus-visible:outline-blue-500 disabled:opacity-50" :class="{ 'text-red-500': post.isLiked }" :aria-label="post.isLiked ? '取消点赞' : '点赞'" :aria-pressed="post.isLiked" :disabled="post.is_blocked_by || isPending(post.id)" @click.stop="like">
            <Icon :name="post.isLiked ? 'solar:heart-angle-bold' : 'mdi:heart-outline'" class="text-base" /><span>{{ post.likeCount }}</span>
          </button>
          <el-dropdown v-if="!post.is_blocked_by && feedbackEnabled && post.user_id !== auth.getUser()?.uuid" trigger="click" placement="bottom-end" @command="feedback">
            <button type="button" aria-label="更多推荐选项" class="flex h-6 w-6 items-center justify-center rounded text-gray-500 hover:bg-gray-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500 disabled:opacity-50" :disabled="feedbackPending" @click.stop><Icon name="mdi:dots-vertical" class="text-base" /></button>
            <template #dropdown>
              <el-dropdown-menu @click.stop>
                <el-dropdown-item command="post" :disabled="feedbackPending">对该帖子不感兴趣</el-dropdown-item>
                <el-dropdown-item command="author" :disabled="feedbackPending">对该作者不感兴趣</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>
  </article>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { discoverAspectRatio, discoverSupportLabel } from '~/utils/discover'
const props = defineProps({ post: { type: Object, required: true }, feedbackEnabled: { type: Boolean, default: false }, source: { type: String, default: 'profile' } })
const emit = defineEmits(['open', 'like-updated', 'feedback'])
const { setLike, isPending } = useDiscoverLike()
const { pending: feedbackPending, setFeedback } = useDiscoverFeedback()
const { recordClick } = useDiscoverClick()
const auth = useAuthState()
const openPost = () => { if (props.post.is_blocked_by) { ElMessage.warning('已被拉黑'); return }; emit('open', props.post); void recordClick(props.post.id, props.source) }
const feedback = async targetType => {
  const result = await setFeedback(props.post.id, targetType)
  if (result) { emit('feedback', result); ElMessage.success('已设置不感兴趣') }
}
// 沿用原卡片的确定性头像配色。
const avatarColors = ['#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8', '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA']
const getAvatarColor = id => {
  let hash = 0
  for (const char of String(id ?? '')) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return avatarColors[hash % avatarColors.length]
}
const mediaFailed = ref(false)
// 助力图文隐藏正文图片地址，卡片仍可使用公开封面。
const imageSrc = computed(() => props.post.type === 'video' ? props.post.coverUrl : props.post.src || props.post.coverUrl)
watch(imageSrc, () => { mediaFailed.value = false })
const like = async () => {
  const result = await setLike(props.post.id, props.post.isLiked)
  if (result) emit('like-updated', { postId: props.post.id, likeCount: result.like_count, isLiked: result.is_liked })
}
</script>
