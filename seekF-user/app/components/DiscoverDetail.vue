<template>
  <Teleport to="body">
    <Transition name="discover">
      <div v-if="show" class="discover-overlay" @click.self="handleClose">
      <div class="note-card">
        <!-- 左侧图片区：宽度由首张图测量锁定；切换时后续图宽度铺满，留白 bg-gray-50 -->
        <div
          class="image-container bg-gray-50"
          :class="{ 'image-container--locked': lockedWidthPx != null }"
          :style="lockedWidthPx != null ? { width: `${lockedWidthPx}px` } : undefined"
        >
          <!-- 图片轮播 -->
          <div v-if="detail?.urls?.length" class="carousel">
            <Transition name="carousel-fade" mode="out-in">
              <img
                :key="detail.urls[currentIndex]"
                :src="detail.urls[currentIndex]"
                :alt="detail.title"
                class="cover-image"
                :class="currentIndex > 0 ? 'cover-image--slide' : 'cover-image--first'"
                @load="onCarouselImageLoad"
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
            class="cover-image cover-image--first"
            @load="onCarouselImageLoad"
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
                <div class="comment-main">
                  <div class="comment-avatar-col">
                    <span class="comment-author">{{ c.nickname }}</span>
                    <el-avatar :size="32" :src="c.avatar" />
                  </div>
                  <div class="comment-content-col">
                    <p class="comment-text">{{ c.content }}</p>
                    <p class="comment-time">{{ c.created_at }}</p>
                    <div class="comment-actions">
                      <span
                        class="comment-like"
                        :class="{ 'liked': c.is_liked }"
                        @click.stop="toggleCommentLike(c)"
                      >
                        <Icon v-if="c.is_liked" name="mdi:heart" />
                        <Icon v-else name="mdi:heart-outline" />
                        {{ c.like_count }}
                      </span>
                      <span class="reply-btn" @click="startReply(c)">回复</span>
                    </div>
                  </div>
                </div>
                <!-- 回复列表 -->
                <div v-if="c.replies?.length" class="reply-list">
                  <div class="reply-item" v-for="reply in c.replies" :key="reply.uuid">
                    <div class="comment-main">
                      <div class="comment-avatar-col">
                        <span class="comment-author">{{ reply.nickname }}</span>
                        <el-avatar :size="28" :src="reply.avatar" />
                      </div>
                      <div class="comment-content-col">
                        <p class="comment-text"><span v-if="reply.reply_to_nickname" class="reply-to">回复 {{ reply.reply_to_nickname }}：</span>{{ reply.content }}</p>
                        <p class="comment-time">{{ reply.created_at }}</p>
                        <div class="comment-actions">
                          <span
                            class="comment-like"
                            :class="{ 'liked': reply.is_liked }"
                            @click.stop="toggleCommentLike(reply)"
                          >
                            <Icon v-if="reply.is_liked" name="mdi:heart" />
                            <Icon v-else name="mdi:heart-outline" />
                            {{ reply.like_count }}
                          </span>
                          <span class="reply-btn" @click="startReply(reply, c)">回复</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 底部互动栏（固定底部） -->
          <div class="interaction-bar" :class="{ 'interaction-bar--active': showCommentInput }">
            <!-- 默认状态：输入提示 + 操作按钮 -->
            <div v-if="!showCommentInput" class="input-area" @click="openCommentInput">
              <span class="input-placeholder">说点什么...</span>
            </div>
            <div v-if="!showCommentInput" class="action-icons">
              <span class="action-item" :class="{ 'liked': isLiked }" @click.stop="toggleLike">
                <Icon v-if="isLiked" name="solar:heart-angle-bold" />
                <Icon v-else name="mdi:heart-outline" />
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

            <!-- 激活状态：输入框 + 发送/取消 -->
            <div v-if="showCommentInput" class="comment-input-wrapper">
              <div v-if="replyTarget" class="reply-hint">
                回复 {{ replyTarget.nickname }}
                <span class="cancel-reply" @click="cancelReply">&times;</span>
              </div>
              <textarea
                ref="commentInputRef"
                v-model="commentText"
                class="comment-input"
                :placeholder="replyTarget ? `回复 ${replyTarget.nickname}...` : '写评论...'"
                rows="3"
              ></textarea>
              <div class="comment-btn-group">
                <button class="cancel-btn" @click="closeCommentInput">取消</button>
                <button
                  class="send-btn"
                  :disabled="!commentText.trim()"
                  @click="submitComment"
                >
                  发送
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, shallowRef, onMounted, watch, nextTick } from 'vue'

const props = defineProps({
  item: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'like-updated'])

const show = ref(false)
const detail = ref(null)
const comments = shallowRef([])
const currentIndex = ref(0)
const isLiked = ref(false)
const likeCount = ref(0)
const showCommentInput = ref(false)
const commentText = ref('')
const commentInputRef = ref(null)
const replyTarget = ref(null) // { uuid, nickname, parentUuid }

/** 左侧栏宽度（px），取首张图在当前 flex 布局下 contain 后的 offsetWidth，切换幻灯片不变 */
const lockedWidthPx = ref(null)

function onCarouselImageLoad(ev) {
  if (currentIndex.value !== 0) return
  const img = ev?.target
  if (!img || !(img instanceof HTMLImageElement)) return
  requestAnimationFrame(() => {
    const w = img.offsetWidth
    if (w > 0) lockedWidthPx.value = w
  })
}

watch(
  () => [detail.value?.urls?.[0], props.item?.src],
  () => {
    currentIndex.value = 0
    lockedWidthPx.value = null
  },
)

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
      currentIndex.value = 0
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
      // 构建评论树：顶级评论 + 回复
      const commentMap = {}
      const topLevel = []

      // 先收集所有评论
      for (const c of res.data.list) {
        commentMap[c.uuid] = { ...c, replies: [] }
      }

      // 组装树结构
      for (const c of res.data.list) {
        if (c.parent_id && commentMap[c.parent_id]) {
          // 是回复，添加到父评论的replies中
          commentMap[c.parent_id].replies.push(commentMap[c.uuid])
        } else {
          // 是顶级评论
          topLevel.push(commentMap[c.uuid])
        }
      }

      comments.value = topLevel
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
      likeCount.value = res.data.like_count
      // 通知父组件更新赞数
      emit('like-updated', {
        postId: props.item.id,
        likeCount: likeCount.value,
        isLiked: isLiked.value,
      })
    }
  } catch (e) {
    console.error('点赞失败:', e)
  }
}

// 评论点赞
const toggleCommentLike = async (comment) => {
  try {
    const res = await useApi$('/user/discover/comment/like', {
      body: { comment_uuid: comment.uuid },
    })
    if (res.code === 200) {
      // 在 comments 数组中找到对应的评论并更新
      const updateComment = (list) => {
        for (const c of list) {
          if (c.uuid === comment.uuid) {
            c.is_liked = res.data.is_liked
            c.like_count = res.data.like_count
            return true
          }
          if (c.replies && updateComment(c.replies)) {
            return true
          }
        }
        return false
      }
      updateComment(comments.value)
      // 手动触发 shallowRef 更新
      comments.value = [...comments.value]
    }
  } catch (e) {
    console.error('评论点赞失败:', e)
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

const openCommentInput = () => {
  showCommentInput.value = true
  nextTick(() => {
    commentInputRef.value?.focus()
  })
}

const closeCommentInput = () => {
  showCommentInput.value = false
  commentText.value = ''
  replyTarget.value = null
}

const startReply = (comment, parentComment) => {
  // 始终将新评论嵌套在顶级评论下
  // parentComment存在说明回复的是非顶级评论，否则回复的是顶级评论
  const topLevelUuid = parentComment ? parentComment.uuid : comment.uuid
  replyTarget.value = {
    uuid: comment.uuid,
    userId: comment.user_id,
    nickname: comment.nickname,
    parentUuid: topLevelUuid,
    isReplyToTopLevel: !parentComment,
  }
  openCommentInput()
}

const cancelReply = () => {
  replyTarget.value = null
}

const submitComment = async () => {
  if (!commentText.value.trim() || !props.item?.id) return
  try {
    const body = {
      post_uuid: props.item.id,
      content: commentText.value.trim(),
    }
    // 如果是回复评论
    if (replyTarget.value) {
      body.parent_id = replyTarget.value.parentUuid
      // 仅回复非顶级评论时才设置reply_to_user_id，用于显示"回复 XXX:"
      if (!replyTarget.value.isReplyToTopLevel) {
        body.reply_to_user_id = replyTarget.value.userId
      }
    }

    const res = await useApi$('/user/discover/comment/add', {
      body,
    })
    if (res.code === 200) {
      commentText.value = ''
      showCommentInput.value = false
      replyTarget.value = null
      // 刷新评论列表
      await fetchComments()
      // 更新评论数
      if (detail.value) {
        detail.value.comment_count = (detail.value.comment_count || 0) + 1
      }
      ElMessage.success('评论成功')
    } else {
      ElMessage.error(res.message || '评论失败')
    }
  } catch (e) {
    console.error('评论失败:', e)
    ElMessage.error('评论失败')
  }
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
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-container--locked {
  flex: 0 0 auto;
  min-width: 0;
}

.cover-image {
  display: block;
  object-fit: contain;
}

.cover-image--first {
  max-width: 100%;
  max-height: 100%;
}

/* 非首张：宽度铺满左侧栏，上下留白由容器 bg-gray-50 透出 */
.cover-image--slide {
  width: 100%;
  height: auto;
  max-height: 100%;
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

.comment-main {
  display: flex;
  gap: 12px;
}

.comment-avatar-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.comment-author {
  font-size: 12px;
  color: #999;
  max-width: 48px;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.comment-content-col {
  flex: 1;
  min-width: 0;
  padding-top: 16px;
}

.reply-to {
  font-size: 14px;
  color: #999;
}

.comment-text {
  font-size: 14px;
  margin: 0;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.comment-time {
  font-size: 12px;
  color: #999;
  margin: 4px 0 0 0;
}

.comment-actions {
  font-size: 12px;
  color: #999;
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 4px;
}

.comment-like {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.comment-like.liked {
  color: #ff2442;
}

.reply-btn {
  cursor: pointer;
}

.reply-btn:hover {
  color: #60a5fa;
}

.reply-list {
  margin-left: 42px;
  margin-top: 12px;
  padding-left: 12px;
  border-left: 2px solid #f0f0f0;
}

.reply-item {
  margin-bottom: 12px;
}

.reply-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #60a5fa;
  margin-bottom: 4px;
}

.cancel-reply {
  cursor: pointer;
  font-size: 14px;
  color: #999;
}

.cancel-reply:hover {
  color: #333;
}

.interaction-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-top: 1px solid #eee;
  transition: all 0.3s ease;
}

.interaction-bar--active {
  padding: 12px 20px;
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
  transition: all 0.3s ease;
}

.input-placeholder {
  margin-left: 8px;
  color: #999;
  font-size: 14px;
}

.comment-input-wrapper {
  display: flex;
  flex-direction: column;
  flex: 1;
  gap: 8px;
  animation: slideUp 0.3s ease;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.comment-input {
  width: 100%;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 10px 15px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
  resize: vertical;
  font-family: inherit;
  line-height: 1.5;
}

.comment-input:focus {
  border-color: #60a5fa;
}

.comment-btn-group {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.cancel-btn {
  background: #f5f5f5;
  color: #666;
  border: none;
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.cancel-btn:hover {
  background: #e5e7eb;
}

.send-btn {
  background: #60a5fa;
  color: #fff;
  border: none;
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.send-btn:hover:not(:disabled) {
  background: #3b82f6;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-icons {
  display: flex;
  gap: 14px;
  font-size: 14px;
  color: #666;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
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
