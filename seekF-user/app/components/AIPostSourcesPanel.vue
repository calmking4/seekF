<template>
    <aside class="post-sources-panel" aria-label="回答参考的帖子">
        <header class="sources-header">
            <div>
                <h2>参考来源</h2>
                <p>{{ posts.length }} 篇相关帖子</p>
            </div>
            <button ref="closeButton" class="close-sources" aria-label="关闭参考来源" @click="$emit('close')">
                <Icon name="uil:times" class="text-xl" />
            </button>
        </header>
        <div class="sources-scroll">
            <div class="sources-grid">
                <button v-for="post in posts" :key="post.id" class="source-card" @click="$emit('post-click', post)">
                    <div class="source-cover" :class="{ 'without-cover': !getPostCover(post) }">
                        <img v-if="getPostCover(post)" :src="getPostCover(post)" :alt="post.title || '帖子封面'" loading="lazy" @error="markCoverFailed(post)" />
                        <Icon v-else :name="isVideoPost(post) ? 'uil:video' : 'uil:image'" class="text-4xl text-gray-300" />
                        <span v-if="isVideoPost(post)" class="video-badge" aria-label="视频帖子"><Icon name="mdi:play" class="block text-lg" /></span>
                    </div>
                    <p class="source-title">{{ post.title || '未命名帖子' }}</p>
                    <div class="source-footer">
                        <span class="source-author">
                            <img v-if="post.avatar" :src="post.avatar" alt="" loading="lazy" />
                            <span v-else class="avatar-placeholder"><Icon name="uil:user" /></span>
                            <span class="source-nickname">{{ post.nickname || '匿名用户' }}</span>
                        </span>
                        <span class="source-likes" :aria-label="`${post.like_count || 0} 个赞`">
                            <Icon name="uil:heart" />{{ post.like_count || 0 }}
                        </span>
                    </div>
                </button>
            </div>
            <p v-if="!posts.length" class="sources-empty">暂无参考帖子</p>
        </div>
    </aside>
</template>

<script setup>
import { ref, onMounted } from 'vue'

defineProps({ posts: { type: Array, default: () => [] } })
defineEmits(['close', 'post-click'])

const closeButton = ref(null)
onMounted(() => closeButton.value?.focus({ preventScroll: true }))

const failedCoverUrls = ref(new Set())
// 旧聊天记录没有媒体类型，兼容通过视频地址判断。
const isVideoPost = post => Number(post.media_type) === 1 || post.type === 'video' ||
    /\.(mp4|webm|mov|m4v|avi|mkv|m3u8)(?:$|[?#])/i.test(post.src || '')

const getPostCover = post => {
    const cover = post.cover_url || (isVideoPost(post) ? '' : post.src) || ''
    return failedCoverUrls.value.has(cover) ? '' : cover
}

const markCoverFailed = post => {
    const cover = getPostCover(post)
    if (cover) failedCoverUrls.value.add(cover)
}
</script>

<style scoped>
.post-sources-panel {
    display: flex;
    flex-direction: column;
    width: var(--post-sources-width, clamp(320px, 30vw, 420px));
    height: 100%;
    min-height: 0;
    border-left: 1px solid #ececec;
    background: white;
    z-index: 30;
}
.sources-header { display: flex; align-items: flex-start; justify-content: space-between; flex-shrink: 0; padding: 24px 22px 18px; }
.sources-header h2 { margin: 0; font-size: 17px; font-weight: 600; color: #333; }
.sources-header p { margin: 5px 0 0; font-size: 12px; color: #9ca3af; }
.close-sources { display: flex; align-items: center; justify-content: center; width: 30px; height: 30px; border: none; border-radius: 50%; background: transparent; color: #666; cursor: pointer; }
.close-sources:hover { background: #f3f4f6; }
.sources-scroll { flex: 1; min-height: 0; overflow-y: auto; padding: 0 22px 24px; overscroll-behavior: contain; }
.sources-grid { columns: 2; column-gap: 16px; }
.source-card { display: inline-block; width: 100%; margin: 0 0 22px; padding: 0; border: none; background: transparent; text-align: left; cursor: pointer; break-inside: avoid; vertical-align: top; }
.source-cover { position: relative; overflow: hidden; border: 1px solid #f0f0f0; border-radius: 12px; background: #f8f9fa; }
.source-cover img { display: block; width: 100%; height: auto; max-height: 300px; object-fit: cover; transition: transform .2s ease; }
.source-card:hover .source-cover img { transform: scale(1.025); }
.without-cover { display: flex; align-items: center; justify-content: center; aspect-ratio: 4 / 3; background: #f3f4f6; }
.video-badge { position: absolute; top: 8px; right: 8px; padding: 4px; border-radius: 50%; background: rgba(0, 0, 0, .4); color: white; }
.source-title { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; margin: 9px 0; color: #333; font-size: 14px; line-height: 1.5; overflow-wrap: anywhere; }
.source-card:hover .source-title { color: #0073ff; }
.source-footer { display: flex; align-items: center; justify-content: space-between; gap: 6px; color: #8a8a8a; font-size: 11px; }
.source-author { display: flex; align-items: center; gap: 5px; min-width: 0; }
.source-author img, .avatar-placeholder { display: flex; align-items: center; justify-content: center; width: 20px; height: 20px; border-radius: 50%; object-fit: cover; flex-shrink: 0; background: #f3f4f6; }
.source-nickname { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.source-likes { display: flex; align-items: center; gap: 3px; flex-shrink: 0; }
.source-likes :deep(svg) { width: 14px; height: 14px; }
.sources-empty { text-align: center; color: #9ca3af; font-size: 13px; padding: 40px 0; }
button:focus-visible { outline: 2px solid #0073ff; outline-offset: 3px; }
@media (prefers-reduced-motion: reduce) {
    .source-cover img { transition: none; }
}
</style>
