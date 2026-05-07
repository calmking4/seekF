<template>
  <Teleport to="body">
    <Transition name="discover">
      <div v-if="show" class="discover-overlay" @click.self="handleClose">
      <div class="note-card">
        <!-- 左侧图片区 -->
        <div class="image-container">
          <!-- 图片轮播 -->
          <div v-if="detail?.urls?.length" class="carousel">
            <Transition name="carousel-fade" mode="out-in">
              <img
                :key="detail.urls[currentIndex]"
                :src="detail.urls[currentIndex]"
                :alt="detail.title"
                class="cover-image"
              />
            </Transition>
            <!-- 左右切换箭头 -->
            <div
              v-if="detail.urls.length > 1"
              class="carousel-arrow carousel-prev"
              @click.stop="prevImage"
            >
              <Icon name="mdi:chevron-left" />
            </div>
            <div
              v-if="detail.urls.length > 1"
              class="carousel-arrow carousel-next"
              @click.stop="nextImage"
            >
              <Icon name="mdi:chevron-right" />
            </div>
            <!-- 页码指示器 -->
            <div v-if="detail.urls.length > 1" class="carousel-indicator">
              {{ currentIndex + 1 }}/{{ detail.urls.length }}
            </div>
          </div>
          <!-- 回退：单图 -->
          <img
            v-else-if="item?.src"
            :src="item.src"
            :alt="item.title"
            class="cover-image"
          />
          <!-- 回退：无图 -->
          <div v-else class="image-content">
            <div class="text-bg">
              <span class="text-line">暂无图片</span>
            </div>
          </div>
        </div>

        <!-- 右侧内容区 -->
        <div class="content-container">
          <!-- 作者信息（固定顶部） -->
          <div class="author-header">
            <div class="author-info">
              <el-avatar :size="40" :src="detail?.avatar || item?.avatar" />
              <span class="author-name">{{ detail?.nickname || item?.nickname || '匿名用户' }}</span>
            </div>
            <el-button type="primary" size="small" round class="follow-btn" :style="{ backgroundColor: '#60a5fa', borderColor: '#60a5fa',padding: '15px 30px' }">关注</el-button>
          </div>

          <!-- 可滚动的中间区域 -->
          <div class="scroll-area">
            <!-- 帖子内容 -->
            <div class="post-content">
              <h3 class="post-title">{{ detail?.title || item?.title }}</h3>
              <p v-if="detail?.content" class="post-text">{{ detail.content }}</p>
              <div v-if="detail?.tags?.length" class="post-tags">
                <span v-for="tag in detail.tags" :key="tag" class="tag">#{{ tag }}</span>
              </div>
              <p class="post-time">{{ detail?.created_at || '' }}</p>
            </div>

            <!-- 评论区 -->
            <div class="comment-section">
              <p class="comment-title">共 {{ detail?.comment_count ?? 0 }} 条评论</p>
              <div v-if="comments.length === 0" class="no-comment">暂无评论</div>
              <div class="comment-item" v-for="c in comments" :key="c.uuid">
                <div class="comment-header">
                  <el-avatar :size="32" :src="c.avatar" />
                  <div class="comment-info">
                    <span class="comment-author">{{ c.nickname }}</span>
                    <p class="comment-time">{{ c.created_at }}</p>
                  </div>
                </div>
                <p class="comment-text">{{ c.content }}</p>
                <div class="comment-actions">
                  <span class="comment-like">
                    <Icon name="gravity-ui:heart" /> {{ c.like_count }}
                  </span>
                  <span class="reply-btn">回复</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 底部互动栏（固定底部） -->
          <div class="interaction-bar">
            <div class="input-area" @click="focusCommentInput">
              <span class="input-placeholder">说点什么...</span>
            </div>
            <div class="action-icons">
              <span class="action-item" :class="{ 'liked': isLiked }" @click.stop="toggleLike">
                <Icon :name="isLiked ? 'gravity-ui:heart-fill' : 'gravity-ui:heart'" />
                <span class="action-count">{{ likeCount }}</span>
              </span>
              <span class="action-item">
                <Icon name="i-line-md:star" />
                <span class="action-count">{{ detail?.comment_count ?? 0 }}</span>
              </span>
              <span class="action-item">
                <Icon name="mdi:chat-outline" />
                <span class="action-count">{{ detail?.comment_count ?? 0 }}</span>
              </span>
              <span class="action-item">
                <Icon name="ri:share-circle-fill" />
              </span>
            </div>
          </div>
        </div>
      </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  item: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close'])

const show = ref(false)
const detail = ref(null)
const comments = ref([])
const currentIndex = ref(0)
const isLiked = ref(false)
const likeCount = ref(0)

onMounted(() => {
  requestAnimationFrame(() => {
    show.value = true
  })
  fetchDetail()
})

const fetchDetail = async () => {
  if (!props.item?.id) return
  try {
    const res = await useApi$('/user/discover/detail', {
      body: { uuid: props.item.id },
    })
    if (res.code === 200 && res.data) {
      detail.value = res.data
      isLiked.value = res.data.is_liked || false
      likeCount.value = res.data.like_count || 0
      fetchComments()
    }
  } catch (e) {
    console.error('获取帖子详情失败:', e)
  }
}

const fetchComments = async () => {
  if (!props.item?.id) return
  try {
    const res = await useApi$('/user/discover/comment/list', {
      body: { post_uuid: props.item.id, page: 1, page_size: 50 },
    })
    if (res.code === 200 && res.data?.list) {
      comments.value = res.data.list
    }
  } catch (e) {
    console.error('获取评论失败:', e)
  }
}

const toggleLike = async () => {
  if (!props.item?.id) return
  try {
    const res = await useApi$('/user/discover/like', {
      body: { target_uuid: props.item.id },
    })
    if (res.code === 200) {
      isLiked.value = res.data.is_liked
      likeCount.value += isLiked.value ? 1 : -1
    }
  } catch (e) {
    console.error('点赞失败:', e)
  }
}

const prevImage = () => {
  if (!detail.value?.urls?.length) return
  currentIndex.value = (currentIndex.value - 1 + detail.value.urls.length) % detail.value.urls.length
}

const nextImage = () => {
  if (!detail.value?.urls?.length) return
  currentIndex.value = (currentIndex.value + 1) % detail.value.urls.length
}

const focusCommentInput = () => {
  // TODO: 聚焦评论输入框
}

const handleClose = () => {
  show.value = false
  setTimeout(() => emit('close'), 300)
}
</script>

<style scoped>
.discover-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.note-card {
  position: relative;
  display: flex;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  height: 80vh;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.15);
}

.image-container {
  flex: 1;
  min-width: 0;
  background: #f9fafb;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cover-image {
  display: block;
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

/* 轮播 */
.carousel {
  width: 100%;
  height: 100%;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.carousel-arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 20px;
  color: #333;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transition: background 0.2s;
  z-index: 5;
}

.carousel-arrow:hover {
  background: #fff;
}

.carousel-prev {
  left: 12px;
}

.carousel-next {
  right: 12px;
}

.carousel-indicator {
  position: absolute;
  bottom: 12px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 10px;
}

.image-content {
  background: #fff;
  border: 18px solid #fde6e9;
  border-radius: 12px;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.text-bg {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.text-line {
  font-size: 48px;
  font-weight: bold;
  line-height: 1.2;
  color: #ccc;
}

.content-container {
  flex-shrink: 0;
  width: 480px;
  display: flex;
  flex-direction: column;
}

.scroll-area {
  flex: 1;
  overflow-y: auto;
  padding: 0 20px;
  scrollbar-width: none;
}

.scroll-area::-webkit-scrollbar {
  display: none;
}

.author-header {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.author-info {
  display: flex;
  align-items: center;
}

.author-name {
  margin-left: 10px;
  font-size: 16px;
  font-weight: 500;
}

.follow-btn {
  border-radius: 20px;
  padding: 0 15px;
}

.post-title {
  font-size: 18px;
  font-weight: 600;
  margin: 16px 0 10px 0;
}

.post-text {
  font-size: 14px;
  line-height: 1.8;
  color: #333;
  margin: 0 0 12px 0;
  white-space: pre-wrap;
}

.post-tags {
  margin: 10px 0;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag {
  color: #60a5fa;
  font-size: 13px;
  background: #eff6ff;
  padding: 2px 10px;
  border-radius: 12px;
}

.post-time {
  font-size: 12px;
  color: #999;
  margin: 12px 0;
}

.comment-section {
  margin-top: 20px;
  padding-bottom: 16px;
}

.comment-title {
  font-size: 14px;
  color: #333;
  font-weight: 500;
  margin-bottom: 15px;
}

.no-comment {
  text-align: center;
  color: #999;
  font-size: 13px;
  padding: 20px 0;
}

.comment-item {
  margin-bottom: 16px;
}

.comment-header {
  display: flex;
  align-items: center;
}

.comment-info {
  margin-left: 10px;
}

.comment-author {
  font-size: 14px;
  font-weight: 500;
}

.comment-time {
  font-size: 12px;
  color: #999;
  margin: 2px 0 0 0;
}

.comment-text {
  font-size: 14px;
  margin: 8px 0 5px 42px;
  line-height: 1.6;
}

.comment-actions {
  font-size: 12px;
  color: #999;
  margin-left: 42px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.comment-like {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.reply-btn {
  cursor: pointer;
}

.reply-btn:hover {
  color: #60a5fa;
}

.interaction-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-top: 1px solid #eee;
}

.input-area {
  display: flex;
  align-items: center;
  background: #f5f5f5;
  border-radius: 20px;
  padding: 8px 15px;
  flex: 1;
  margin-right: 10px;
  cursor: text;
}

.input-placeholder {
  margin-left: 8px;
  color: #999;
  font-size: 14px;
}

.action-icons {
  display: flex;
  gap: 14px;
  font-size: 14px;
  color: #666;
}

.action-item {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  font-size: 20px;
  transition: color 0.2s;
}

.action-item:hover {
  color: #ff2442;
}

.action-item.liked {
  color: #ff2442;
}

.action-count {
  font-size: 13px;
}

/* 弹窗动画 */
.discover-enter-active {
  transition: opacity 0.35s ease;
}

.discover-enter-active .note-card {
  transition: transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1),
              opacity 0.25s ease;
}

.discover-leave-active {
  transition: opacity 0.25s ease;
}

.discover-leave-active .note-card {
  transition: transform 0.25s cubic-bezier(0.4, 0, 1, 1),
              opacity 0.2s ease;
}

.discover-enter-from {
  opacity: 0;
}

.discover-enter-from .note-card {
  transform: scale(0.85);
  opacity: 0;
}

.discover-leave-to {
  opacity: 0;
}

.discover-leave-to .note-card {
  transform: scale(0.9);
  opacity: 0;
}

/* 图片轮播动画 */
.carousel-fade-enter-active {
  transition: opacity 0.3s ease;
}

.carousel-fade-leave-active {
  transition: opacity 0.2s ease;
}

.carousel-fade-enter-from {
  opacity: 0;
}

.carousel-fade-leave-to {
  opacity: 0;
}
</style>
