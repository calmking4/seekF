<template>
  <Teleport to="body" :disabled="standalone">
    <Transition name="discover">
      <div v-if="show" :class="standalone ? 'discover-standalone' : 'discover-overlay'" @click.self="handleClose">
      <button v-if="!standalone" type="button" aria-label="关闭详情" class="absolute right-4 top-4 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-white/90 text-gray-600 focus-visible:ring-2 focus-visible:ring-blue-500" @click="handleClose"><Icon name="mdi:close" /></button>
      <div ref="dialogRef" class="note-card" :class="{ '!h-auto w-[560px]': isSupportLocked }" tabindex="-1" :role="standalone ? 'region' : 'dialog'" :aria-modal="standalone ? undefined : true" :aria-label="isSupportLocked ? '助力后查看' : detail?.title || item?.title || '动态详情'" @keydown="handleKeydown">
        <div v-if="isSupportLocked" class="flex min-h-[360px] w-full flex-col items-center justify-center gap-4 px-6 py-12 text-center">
          <span class="flex h-16 w-16 items-center justify-center rounded-2xl bg-blue-50 text-blue-500"><Icon name="mdi:lock-outline" class="text-3xl" /></span>
          <h2 class="text-lg font-medium text-gray-800">{{ detail?.lock_reason || '助力创作者后可查看' }}</h2>
          <p class="max-w-sm text-sm leading-6 text-gray-500">助力后可查看该创作者的全部助力图文、视频及评论，权益有效期为30天。</p>
          <p v-if="detailError" class="text-xs text-gray-500" role="alert">{{ detailError }}<button type="button" class="ml-2 text-blue-500" @click="fetchDetail">重试</button></p>
          <el-button type="primary" :loading="supportInfoLoading || detailLoading" :disabled="!detail?.user_id" @click="showSupportInfo">查看助力</el-button>
        </div>
        <template v-else>
        <!-- 左侧媒体区：宽度由首张图测量锁定；切换时后续图宽度铺满，留白 bg-gray-50 -->
        <div
          ref="mediaContainerRef"
          class="image-container bg-gray-50"
          :class="{ 'image-container--locked': lockedWidthPx != null }"
          :style="lockedWidthPx != null ? { width: `${lockedWidthPx}px` } : undefined"
        >
          <!-- 视频播放 -->
          <div v-if="detail?.media_type === 1 && detail?.urls?.length && detail?.can_play !== false" class="video-wrapper">
            <video
              :key="detail.urls[0]"
              :src="detail.urls[0]"
              class="video-player"
              @loadedmetadata="onVideoMetadataLoad"
              @error="fetchDetail"
              controls
              preload="metadata"
              playsinline
            >
              您的浏览器不支持视频播放
            </video>
          </div>
          <!-- 图片轮播 -->
          <div v-else-if="detail?.urls?.length && detail?.media_type !== 1 && !imageError" class="carousel">
            <Transition name="carousel-fade" mode="out-in">
              <img
                :key="detail.urls[currentIndex]"
                :src="detail.urls[currentIndex]"
                :alt="detail.title"
                class="cover-image"
                :class="currentIndex > 0 ? 'cover-image--slide' : 'cover-image--first w-full h-full'"
                @load="onCarouselImageLoad"
                @error="imageError = true"
              />
            </Transition>
            <!-- 左右切换箭头 -->
            <button type="button" aria-label="上一张图片"
              v-if="detail.urls.length > 1"
              class="carousel-arrow carousel-prev"
              @click.stop="prevImage"
            >
              <Icon name="mdi:chevron-left" />
            </button>
            <button type="button" aria-label="下一张图片"
              v-if="detail.urls.length > 1"
              class="carousel-arrow carousel-next"
              @click.stop="nextImage"
            >
              <Icon name="mdi:chevron-right" />
            </button>
            <!-- 页码指示器 -->
            <div v-if="detail.urls.length > 1" class="carousel-indicator">
              {{ currentIndex + 1 }}/{{ detail.urls.length }}
            </div>
          </div>
          <!-- 回退：单图 -->
          <img
            v-else-if="item?.src && item?.type !== 'video' && !imageError"
            :src="item.src"
            :alt="item.title"
            class="cover-image cover-image--first w-full h-full"
            @load="onCarouselImageLoad"
                @error="imageError = true"
          />
          <!-- 回退：无媒体 -->
          <div v-else class="image-content">
            <div class="text-bg">
              <span class="text-line">暂无内容</span>
            </div>
          </div>
        </div>

        <!-- 右侧内容区 -->
        <div ref="contentContainerRef" class="content-container">
          <!-- 作者信息（固定顶部） -->
          <div class="author-header">
            <button type="button" class="author-info" :disabled="!authorId" @click="goToAuthor">
              <el-avatar :size="40" :src="detail?.avatar || item?.avatar" />
              <span class="author-name">{{ detail?.nickname || item?.nickname || '匿名用户' }}</span>
            </button>
            <el-button
              v-if="!isSelf"
              :type="isFollowing ? 'default' : 'primary'"
              size="small"
              round
              class="follow-btn"
              :loading="followLoading"
              :disabled="!authorId || !detail"
              @click="toggleFollow"
            >{{ isFollowing ? '已关注' : '关注' }}</el-button>
          </div>

          <!-- 可滚动的中间区域 -->
          <div class="scroll-area">
            <div v-if="detailLoading && !detail" class="py-6 flex items-center justify-center text-sm text-gray-500" role="status"><Icon name="uil:spinner" class="animate-spin text-xl mr-2" />正在加载动态</div>
            <div v-if="detailError" class="py-6 text-center text-sm text-gray-500" role="alert"><p>{{ detailError }}</p><button type="button" class="mt-3 text-blue-500" @click="fetchDetail">重试</button></div>
            <template v-else>

            <!-- 帖子内容 -->
            <div class="post-content">
              <h3 class="post-title">{{ detail?.title || item?.title }}</h3>
              <p v-if="detail?.content" class="post-text">{{ detail.content }}</p>
              <div v-if="detail?.tags?.length" class="post-tags">
                <NuxtLink v-for="tag in detail.tags" :key="tag" class="tag" :to="'/discover/search?q=' + encodeURIComponent(tag)">#{{ tag }}</NuxtLink>
              </div>
              <p class="post-time">{{ detail?.created_at || '' }}</p>
            </div>

            <!-- 评论区 -->
            <div class="comment-section">
              <div class="flex items-center justify-between">
                <p class="comment-title">共 {{ detail?.comment_count ?? 0 }} 条评论</p>
                <button type="button" class="mb-[15px] inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200 disabled:cursor-not-allowed disabled:opacity-50" :aria-label="commentsLoading ? '正在刷新评论' : '刷新评论'" :title="commentsLoading ? '正在刷新评论' : '刷新评论'" :aria-busy="commentsLoading" :disabled="!detail || commentsLoading || submitting" @click="fetchComments(true)">
                  <Icon name="mdi:refresh" class="text-lg" :class="{ 'animate-spin': commentsLoading }" />
                </button>
              </div>
              <div v-if="comments.length === 0 && !commentsLoading && !commentsError" class="no-comment">暂无评论</div>
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
                      <button type="button" class="comment-like disabled:opacity-50" :aria-label="c.is_liked ? '取消评论点赞' : '点赞评论'" :aria-pressed="c.is_liked" :disabled="isPending(c.uuid, true)"
                        :class="{ 'liked': c.is_liked }"
                        @click.stop="toggleCommentLike(c)"
                      >
                        <Icon v-if="c.is_liked" name="mdi:heart" />
                        <Icon v-else name="mdi:heart-outline" />
                        {{ c.like_count }}
                      </button>
                      <button type="button" class="reply-btn" :disabled="submitting" @click="startReply(c)">回复</button>
                    </div>
                  </div>
                </div>
                <!-- 回复沿用独立分页；网格行过渡支持不同回复数量，收起时禁止隐藏内容获得焦点。 -->
                <button v-if="c.reply_count || c.replies?.length" type="button" class="ml-[42px] mt-3 inline-flex items-center gap-1 rounded-full bg-slate-50 px-2.5 py-1 text-xs text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-200" :aria-expanded="c.expanded" :aria-controls="'discover-replies-' + c.uuid" @click="toggleReplies(c)">
                  <span>{{ c.expanded ? '收起回复' : '展开 ' + c.reply_count + ' 条回复' }}</span>
                  <Icon name="mdi:chevron-down" class="text-sm transition-transform duration-200 motion-reduce:transition-none" :class="{ 'rotate-180': c.expanded }" />
                </button>
                <div :id="'discover-replies-' + c.uuid" class="grid transition-[grid-template-rows,opacity] duration-300 ease-out motion-reduce:transition-none" :class="c.expanded ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'" :aria-hidden="!c.expanded" :inert="!c.expanded">
                  <div class="min-h-0 overflow-hidden">
                    <div class="reply-list">
                      <p v-if="c.replyLoading && !c.replies.length" class="flex items-center gap-1.5 py-1 text-xs text-slate-400" role="status"><Icon name="uil:spinner" class="animate-spin" />正在加载回复...</p>
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
                              <button type="button" class="comment-like disabled:opacity-50" :aria-label="reply.is_liked ? '取消评论点赞' : '点赞评论'" :aria-pressed="reply.is_liked" :disabled="isPending(reply.uuid, true)"
                                :class="{ 'liked': reply.is_liked }"
                                @click.stop="toggleCommentLike(reply)"
                              >
                                <Icon v-if="reply.is_liked" name="mdi:heart" />
                                <Icon v-else name="mdi:heart-outline" />
                                {{ reply.like_count }}
                              </button>
                              <button type="button" class="reply-btn" :disabled="submitting" @click="startReply(reply, c)">回复</button>
                            </div>
                          </div>
                        </div>
                      </div>
                      <p v-if="c.replyError" class="text-xs text-red-500" role="alert">{{ c.replyError }}</p>
                      <button v-if="c.replyHasMore || c.replyError" type="button" class="text-xs text-blue-500 disabled:opacity-50" :disabled="c.replyLoading" @click="loadReplies(c)">{{ c.replyLoading ? '正在加载回复...' : c.replyError ? '重试加载回复' : '加载更多回复' }}</button>
                    </div>
                  </div>
                </div>
              </div>
              <p v-if="commentsError" class="text-center text-xs text-red-500" role="alert">{{ commentsError }}</p>
              <button v-if="commentsHasMore || commentsError" type="button" class="w-full py-3 text-sm text-blue-500 disabled:opacity-50" :disabled="commentsLoading" @click="fetchComments()">{{ commentsLoading ? '正在加载评论...' : commentsError ? '重试加载评论' : '加载更多评论' }}</button>
            </div>
            </template>
          </div>

          <!-- 底部互动栏（固定底部） -->
          <div class="interaction-bar" :class="{ 'interaction-bar--active': showCommentInput }">
            <!-- 默认状态：输入提示 + 操作按钮 -->
            <button v-if="!showCommentInput" ref="commentPromptRef" type="button" class="input-area" :disabled="!detail" @click="openCommentInput">
              <span class="input-placeholder">说点什么...</span>
            </button>
            <div v-if="!showCommentInput" class="action-icons">
              <button type="button" class="action-item disabled:opacity-50" :aria-label="isLiked ? '取消点赞' : '点赞'" :aria-pressed="isLiked" :disabled="!detail || isPending(item?.id)" :class="{ 'liked': isLiked }" @click.stop="toggleLike">
                <Icon v-if="isLiked" name="solar:heart-angle-bold" />
                <Icon v-else name="mdi:heart-outline" />
                <span class="action-count">{{ likeCount }}</span>
              </button>
              <button type="button" class="action-item disabled:opacity-50" :aria-label="isCollected ? '取消收藏' : '收藏'" :disabled="!detail || collecting" :class="{ 'collected': isCollected }" @click.stop="toggleCollect">
                <Icon v-if="isCollected" name="mdi:bookmark" />
                <Icon v-else name="mdi:bookmark-outline" />
                <span class="action-count">{{ collectCount }}</span>
              </button>
              <button type="button" class="action-item" aria-label="写评论" :disabled="!detail" @click="openCommentInput">
                <Icon name="mdi:chat-outline" />
                <span class="action-count">{{ detail?.comment_count ?? 0 }}</span>
              </button>
              <button type="button" class="action-item" aria-label="复制分享链接" :disabled="!detail" @click="sharePost">
                <Icon name="ri:share-circle-fill" />
              </button>
            </div>

            <!-- 激活状态：输入框 + 发送/取消 -->
            <div v-if="showCommentInput" class="comment-input-wrapper">
              <div v-if="replyTarget" class="reply-hint">
                回复 {{ replyTarget.nickname }}
                <button type="button" class="cancel-reply" aria-label="取消回复" :disabled="submitting" @click="cancelReply">&times;</button>
              </div>
              <textarea
                ref="commentInputRef"
                v-model="commentText"
                class="comment-input"
                aria-label="评论内容"
                :disabled="submitting"
                :placeholder="replyTarget ? `回复 ${replyTarget.nickname}...` : '写评论...'"
                rows="3"
              ></textarea>
              <div class="comment-btn-group">
                <button
                  type="button"
                  class="ai-assist-btn"
                  :disabled="submitting"
                  :class="{ 'active': isAIComment }"
                  @click="toggleAIComment"
                >
                  <Icon name="mdi:robot-outline" /> @AI助手
                </button>
                <div class="comment-btn-right">
                  <button type="button" class="cancel-btn" :disabled="submitting" @click="closeCommentInput">取消</button>
                  <button
                    type="button"
                    class="send-btn"
                    :disabled="!detail || submitting || !commentText.trim()"
                    @click="submitComment"
                  >
                    {{ submitting ? '发送中...' : '发送' }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
        </template>
      </div>
      </div>
    </Transition>

    <!-- 收藏夹选择弹窗 -->
    <CollectDialog
      v-if="!isSupportLocked"
      :visible="showCollectDialog"
      :post-uuid="props.item?.id || ''"
      @close="showCollectDialog = false"
      @collected="handleCollected"
    />
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { mergeDiscoverPage, discoverMediaWidth } from '~/utils/discover'
const props = defineProps({ item: { type: Object, default: null }, standalone: Boolean })
const emit = defineEmits(['close', 'like-updated', 'collect-updated', 'follow-updated'])
const auth = useAuthState()
const show = ref(false)
const detail = ref(null)
const isSupportLocked = computed(() => (detail.value?.support_only ?? props.item?.support_only) === 1 && detail.value?.can_play !== true)
const detailLoading = ref(false)
const detailError = ref('')
const supportInfoLoading = ref(false)
const showSupportInfo = async () => {
  if (supportInfoLoading.value || !detail.value?.user_id) return
  const creatorId = detail.value.user_id
  const viewerId = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
  supportInfoLoading.value = true
  try {
    const response = await useApi$('/user/support/creator/info', { body: { creator_id: creatorId }, retry: 0 })
    if (disposed || detail.value?.user_id !== creatorId || viewerId !== (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')) return
    if (response.code !== 200 || !response.data?.activated) throw new Error(response.message || '创作者暂未开通助力')
    await ElMessageBox.alert(`每30天${response.data.price_yuan}元，可观看该创作者的全部助力图文与视频。助力购买暂未开放。`, '创作者助力', { confirmButtonText: '知道了' })
  } catch (error) { if (!disposed && error !== 'cancel' && error !== 'close') ElMessage.error(error?.data?.message || error.message || '获取助力信息失败') }
  finally { if (!disposed) supportInfoLoading.value = false }
}
const currentIndex = ref(0)
const imageError = ref(false)
const authorId = computed(() => detail.value?.user_id || props.item?.user_id || '')
const isSelf = computed(() => !!authorId.value && authorId.value === auth.getUser()?.uuid)
const isFollowing = ref(false)
const followLoading = ref(false)
const isLiked = ref(false)
const likeCount = ref(0)
const { setLike, isPending } = useDiscoverLike()
const isCollected = ref(false)
const collectCount = ref(0)
const collectedFolderUuid = ref('')
const showCollectDialog = ref(false)
const collecting = ref(false)
const showCommentInput = ref(false)
const commentText = ref('')
const commentInputRef = ref(null)
const replyTarget = ref(null)
const isAIComment = ref(false)
const submitting = ref(false)
const AI_PREFIX = '@AI助手 '
const comments = ref([])
const commentsPage = ref(1)
const commentsHasMore = ref(true)
const commentsLoading = ref(false)
const commentsError = ref('')
const commentsRetryReset = ref(false)
const dialogRef = ref(null)
const mediaContainerRef = ref(null)
const contentContainerRef = ref(null)
const commentPromptRef = ref(null)
const lockedWidthPx = ref(null)
const firstMediaSize = ref(null)
let enterFrame
let mediaFrame
let mediaResizeObserver

// 首图决定左栏宽度，右侧继续使用原来的 480px 内容栏；不再各占一半。
const updateMediaWidth = () => {
  if (!dialogRef.value || !contentContainerRef.value || !firstMediaSize.value || disposed) return
  if (window.innerWidth < 768) { lockedWidthPx.value = null; return }
  const parent = dialogRef.value.parentElement
  const style = window.getComputedStyle(parent)
  const availableWidth = parent.clientWidth - (parseFloat(style.paddingLeft) || 0) - (parseFloat(style.paddingRight) || 0) - contentContainerRef.value.offsetWidth
  lockedWidthPx.value = discoverMediaWidth(firstMediaSize.value, dialogRef.value.clientHeight, availableWidth)
}

const scheduleMediaWidth = () => {
  cancelAnimationFrame(mediaFrame)
  mediaFrame = requestAnimationFrame(updateMediaWidth)
}

// 仅首图可以修改左栏宽度，后续图片完整显示并沿用首图的布局。
const onCarouselImageLoad = event => {
  if (currentIndex.value !== 0) return
  const image = event?.target
  if (!image?.naturalWidth || !image?.naturalHeight) return
  firstMediaSize.value = { width: image.naturalWidth, height: image.naturalHeight }
  scheduleMediaWidth()
}

const onVideoMetadataLoad = event => {
  const video = event?.target
  if (!video?.videoWidth || !video?.videoHeight) return
  firstMediaSize.value = { width: video.videoWidth, height: video.videoHeight }
  scheduleMediaWidth()
}

watch(() => [detail.value?.urls?.[0], props.item?.src, props.item?.id], () => {
  currentIndex.value = 0
  lockedWidthPx.value = null
  firstMediaSize.value = null
  imageError.value = false
})
let previousFocus
let disposed = false
let closeTimer
let detailVersion = 0

const fetchDetail = async () => {
  if (!props.item?.id || detailLoading.value) return
  const version = ++detailVersion
  const viewerId = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
  detailLoading.value = true
  detailError.value = ''
  try {
    const response = await useApi$('/user/discover/detail', { body: { uuid: props.item.id }, retry: 0 })
    if (disposed || version !== detailVersion || viewerId !== (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')) return
    if (response.code !== 200 || !response.data) throw new Error(response.message || '动态暂时无法访问')
    detail.value = response.data
    if (isSupportLocked.value) {
      comments.value = []; commentsHasMore.value = false
      showCommentInput.value = false; commentText.value = ''; replyTarget.value = null
      showCollectDialog.value = false
      return
    }
    currentIndex.value = 0
    isLiked.value = !!response.data.is_liked
    likeCount.value = response.data.like_count || 0
    isCollected.value = !!response.data.is_collected
    collectCount.value = response.data.collect_count || 0
    fetchFollowState()
    if (isCollected.value) checkCollectedFolder()
    fetchComments(true)
    if (props.item.initialReply) {
      const comment = props.item.initialReply
      startReply(comment, comment.parent_id ? { uuid: comment.parent_id } : undefined)
    }
  } catch (err) {
    if (!disposed && version === detailVersion) detailError.value = err?.data?.message || '动态暂时无法访问，请稍后重试'
  } finally {
    if (!disposed && version === detailVersion) detailLoading.value = false
  }
}
// 切换账号时重新读取当前权限，并立即移除上一账号的媒体地址。
watch(() => auth.getUser()?.uuid || auth.getUser()?.Uuid || '', () => {
  detailVersion++
  detail.value = null
  comments.value = []; showCommentInput.value = false; commentText.value = ''; replyTarget.value = null; showCollectDialog.value = false
  detailLoading.value = false
  if (!disposed && show.value) void fetchDetail()
})
const fetchFollowState = async () => {
  if (!authorId.value || isSelf.value) return
  followLoading.value = true
  try {
    const response = await useApi$('/user/follow/counts', { body: { user_id: authorId.value } })
    if (!disposed && response.code === 200) isFollowing.value = !!response.data?.is_following
  } catch (err) { console.error('获取作者关注状态失败:', err) }
  finally { if (!disposed) followLoading.value = false }
}
const toggleFollow = async () => {
  if (!authorId.value || followLoading.value) return
  followLoading.value = true
  try {
    const response = await useApi$('/user/follow/toggle', { body: { follow_user_id: authorId.value }, retry: 0 })
    if (disposed) return
    if (response.code !== 200) throw new Error('关注操作失败')
    isFollowing.value = !!response.data.is_followed
    emit('follow-updated', { userId: authorId.value, isFollowing: isFollowing.value })
  } catch (err) { if (!disposed) ElMessage.error(err?.data?.message || '关注操作失败，请稍后重试') }
  finally { if (!disposed) followLoading.value = false }
}
const goToAuthor = () => { if (authorId.value) { emit('close', 'author'); navigateTo('/user/' + encodeURIComponent(authorId.value)) } }

const buildComment = comment => ({ ...comment, replies: [], expanded: false, replyPage: 1, replyHasMore: true, replyLoading: false, replyError: '' })
const fetchComments = async (reset = false) => {
  reset = reset || (!!commentsError.value && commentsRetryReset.value)
  if (!detail.value || isSupportLocked.value || !props.item?.id || commentsLoading.value || (!reset && !commentsHasMore.value && !commentsError.value)) return
  const version = detailVersion
  commentsLoading.value = true
  commentsError.value = ''
  const page = reset ? 1 : commentsPage.value
  try {
    const response = await useApi$('/user/discover/comment/threads', { body: { post_uuid: props.item.id, page, page_size: 20 }, retry: 0 })
    if (disposed || version !== detailVersion || isSupportLocked.value) return
    if (response.code !== 200 || !Array.isArray(response.data?.list)) throw new Error('获取评论失败')
    const list = response.data.list.map(buildComment)
    comments.value = reset ? list : mergeDiscoverPage(comments.value, list)
    commentsPage.value = page + 1
    commentsHasMore.value = !!response.data.has_more
    commentsRetryReset.value = false
  } catch (err) { if (!disposed) { commentsRetryReset.value = reset; commentsError.value = err?.data?.message || '获取评论失败，请重试' } }
  finally { if (!disposed) commentsLoading.value = false }
}
const loadReplies = async comment => {
  if (!detail.value || isSupportLocked.value || comment.replyLoading || !comment.replyHasMore) return
  const version = detailVersion
  comment.replyLoading = true
  comment.replyError = ''
  try {
    const response = await useApi$('/user/discover/comment/replies', { body: { post_uuid: props.item.id, parent_id: comment.uuid, page: comment.replyPage, page_size: 10 }, retry: 0 })
    if (disposed || version !== detailVersion || isSupportLocked.value) return
    if (response.code !== 200 || !Array.isArray(response.data?.list)) throw new Error('获取回复失败')
    comment.replies = mergeDiscoverPage(comment.replies, response.data.list)
    comment.replyPage++
    comment.replyHasMore = !!response.data.has_more
  } catch (err) { if (!disposed) comment.replyError = err?.data?.message || '获取回复失败，请重试' }
  finally { if (!disposed) comment.replyLoading = false }
}
const toggleReplies = comment => {
  comment.expanded = !comment.expanded
  if (comment.expanded && comment.replyPage === 1) loadReplies(comment)
}
const updateCommentLike = result => {
  for (const comment of comments.value) {
    const target = comment.uuid === result.uuid ? comment : comment.replies.find(reply => reply.uuid === result.uuid)
    if (target) { Object.assign(target, result); return }
  }
}
// 评论点赞继续使用阶段一的幂等接口，样式沿用原模板。
const toggleCommentLike = async comment => {
  const result = await setLike(comment.uuid, comment.is_liked, true)
  if (result && !disposed) updateCommentLike({ uuid: comment.uuid, is_liked: result.is_liked, like_count: result.like_count })
}
const toggleLike = async () => {
  const result = await setLike(props.item?.id, isLiked.value)
  if (!result || disposed) return
  isLiked.value = result.is_liked
  likeCount.value = result.like_count
  emit('like-updated', { postId: props.item.id, likeCount: likeCount.value, isLiked: isLiked.value })
}
const checkCollectedFolder = async () => {
  try {
    const response = await useApi$('/user/discover/check-collected', { body: { post_uuid: props.item.id } })
    if (!disposed && response.code === 200) collectedFolderUuid.value = response.data?.folder_uuid || ''
  } catch (err) { console.error('检查收藏状态失败:', err) }
}
const toggleCollect = async () => {
  if (collecting.value) return
  if (!isCollected.value) { showCollectDialog.value = true; return }
  collecting.value = true
  try {
    if (!collectedFolderUuid.value) await checkCollectedFolder()
    if (!collectedFolderUuid.value) throw new Error('未找到收藏夹')
    const response = await useApi$('/user/discover/uncollect', { body: { post_uuid: props.item.id, folder_uuid: collectedFolderUuid.value }, retry: 0 })
    if (disposed) return
    if (response.code !== 200) throw new Error('取消收藏失败')
    isCollected.value = false
    collectCount.value = response.data.collect_count
    collectedFolderUuid.value = ''
    emit('collect-updated', { postId: props.item.id, collectCount: collectCount.value, isCollected: false })
  } catch (err) { if (!disposed) ElMessage.error(err?.data?.message || '取消收藏失败，请重试') }
  finally { if (!disposed) collecting.value = false }
}
const handleCollected = ({ collectCount: count, folderUuid }) => {
  isCollected.value = true
  collectCount.value = count
  collectedFolderUuid.value = folderUuid
  emit('collect-updated', { postId: props.item.id, collectCount: count, isCollected: true })
}
const prevImage = () => { currentIndex.value = (currentIndex.value - 1 + detail.value.urls.length) % detail.value.urls.length }
const nextImage = () => { currentIndex.value = (currentIndex.value + 1) % detail.value.urls.length }
watch(currentIndex, () => { imageError.value = false })
const openCommentInput = () => { showCommentInput.value = true; nextTick(() => commentInputRef.value?.focus()) }
const closeCommentInput = () => {
  if (submitting.value) return
  showCommentInput.value = false
  commentText.value = ''
  replyTarget.value = null
  isAIComment.value = false
  nextTick(() => commentPromptRef.value?.focus({ preventScroll: true }))
}
const cancelReply = () => { if (!submitting.value) replyTarget.value = null }
const startReply = (comment, parentComment) => {
  if (submitting.value) return
  replyTarget.value = { uuid: comment.uuid, userId: comment.user_id, nickname: comment.nickname, content: comment.content, parentUuid: parentComment?.uuid || comment.uuid, isReplyToTopLevel: !parentComment }
  openCommentInput()
}
const toggleAIComment = () => {
  if (submitting.value) return
  isAIComment.value = !isAIComment.value
  if (isAIComment.value && !commentText.value.startsWith(AI_PREFIX)) commentText.value = AI_PREFIX + commentText.value
  else if (!isAIComment.value && commentText.value.startsWith(AI_PREFIX)) commentText.value = commentText.value.slice(AI_PREFIX.length)
  nextTick(() => commentInputRef.value?.focus())
}
const submitComment = async () => {
  if (!detail.value || isSupportLocked.value || submitting.value || !props.item?.id || !commentText.value.trim()) return
  const version = detailVersion
  const text = commentText.value.trim()
  const aiComment = text.startsWith('@AI助手') || isAIComment.value
  const question = aiComment ? text.replace(/^@AI助手\s*/, '').trim() : ''
  if (aiComment && !question) { ElMessage.warning('请输入你的问题'); return }
  submitting.value = true
  const target = replyTarget.value
  const body = { post_uuid: props.item.id, content: text }
  if (target) {
    body.parent_id = target.parentUuid
    if (!target.isReplyToTopLevel) body.reply_to_user_id = target.userId
  }
  if (aiComment) { body.ai_question = question; body.reply_to_content = target?.content || '' }
  try {
    const response = await useApi$(aiComment ? '/user/discover/comment/ai' : '/user/discover/comment/add', { body, retry: 0 })
    if (disposed || version !== detailVersion || isSupportLocked.value) return
    if (response.code !== 200) throw new Error('评论发送失败')
    const comment = response.data
    if (comment?.uuid) {
      const normalized = { ...comment, reply_to_nickname: target && !target.isReplyToTopLevel ? target.nickname : comment.reply_to_nickname }
      if (target) {
        const parent = comments.value.find(item => item.uuid === target.parentUuid)
        if (parent) { parent.reply_count = (parent.reply_count || 0) + 1; parent.replies = mergeDiscoverPage(parent.replies, [normalized]); parent.expanded = true }
      } else comments.value = mergeDiscoverPage(comments.value, [buildComment(normalized)])
    }
    detail.value.comment_count = (detail.value.comment_count || 0) + 1
    submitting.value = false
    closeCommentInput()
    ElMessage.success(aiComment ? '评论已发送，AI助手稍后回复，可刷新评论查看' : '评论成功')
  } catch (err) { if (!disposed) ElMessage.error(err?.data?.message || '评论发送失败，请重试') }
  finally { if (!disposed) submitting.value = false }
}
const sharePost = async () => {
  const url = new URL('/discover/post/' + encodeURIComponent(props.item.id), window.location.origin).href
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('分享链接已复制')
  } catch {
    // HTTP 或浏览器不支持剪贴板时，展示可手动复制的真实链接。
    try { await ElMessageBox.prompt('请复制下方链接，查看内容需要登录', '分享动态', { inputValue: url, confirmButtonText: '完成', cancelButtonText: '关闭', inputType: 'text' }) } catch { /* 用户关闭分享窗口。 */ }
  }
}
const handleClose = () => {
  if (props.standalone || showCollectDialog.value) return
  show.value = false
  clearTimeout(closeTimer)
  closeTimer = setTimeout(() => emit('close'), 300)
}
const handleKeydown = event => {
  if (props.standalone || showCollectDialog.value) return
  if (event.key === 'Escape') { event.preventDefault(); handleClose() }
  if (event.key !== 'Tab') return
  const elements = [...dialogRef.value.querySelectorAll('button:not(:disabled), a[href], textarea:not(:disabled), input:not(:disabled), [tabindex="0"]')].filter(element => element.getClientRects().length)
  if (!elements.length) { event.preventDefault(); dialogRef.value.focus(); return }
  const first = elements[0], last = elements[elements.length - 1]
  if (event.shiftKey && (document.activeElement === first || document.activeElement === dialogRef.value)) { event.preventDefault(); last.focus() }
  else if (!event.shiftKey && (document.activeElement === last || document.activeElement === dialogRef.value)) { event.preventDefault(); first.focus() }
}
onMounted(() => {
  previousFocus = document.activeElement
  // 保留原来的下一帧入场，让遮罩和卡片缩放过渡能够生效。
  enterFrame = requestAnimationFrame(async () => {
    if (disposed) return
    show.value = true
    await nextTick()
    if (!props.standalone) dialogRef.value?.focus({ preventScroll: true })
    mediaResizeObserver = new ResizeObserver(scheduleMediaWidth)
    if (dialogRef.value?.parentElement) mediaResizeObserver.observe(dialogRef.value.parentElement)
    scheduleMediaWidth()
  })
  window.addEventListener('resize', scheduleMediaWidth)
  fetchDetail()
})
onUnmounted(() => {
  disposed = true
  detailVersion++
  clearTimeout(closeTimer)
  cancelAnimationFrame(enterFrame)
  cancelAnimationFrame(mediaFrame)
  mediaResizeObserver?.disconnect()
  window.removeEventListener('resize', scheduleMediaWidth)
  if (!props.standalone && previousFocus?.isConnected) previousFocus.focus()
})
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

/* 视频播放器样式 */
.video-wrapper {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
}

.video-player {
  width: 100%;
  height: 100%;
  object-fit: contain;
  outline: none;
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
  border: none;
  padding: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.author-info:disabled {
  cursor: default;
}

.author-info:hover:not(:disabled) .author-name {
  color: #409eff;
}

.author-name {
  margin-left: 10px;
  font-size: 16px;
  font-weight: 500;
}

.follow-btn {
  border-radius: 20px;
  padding: 15px 30px;
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
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.comment-btn-right {
  display: flex;
  gap: 8px;
}

.ai-assist-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #f0f7ff;
  color: #60a5fa;
  border: 1px solid #bfdbfe;
  border-radius: 20px;
  padding: 6px 12px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.ai-assist-btn:hover {
  background: #dbeafe;
}

.ai-assist-btn.active {
  background: #60a5fa;
  color: #fff;
  border-color: #60a5fa;
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

.action-item.collected {
  color: #ff9500;
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

/* 阶段一的独立详情沿用原卡片外观；仅保留已有的窄屏兼容。 */
.discover-standalone {
  display: flex;
  justify-content: center;
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}
.note-card { max-width: 100%; min-width: 0; outline: none; }
.content-container, .scroll-area { min-height: 0; }
@media (max-width: 767px) {
  .discover-overlay, .discover-standalone { padding: 12px; }
  .note-card { flex-direction: column; width: 100%; height: 85dvh; }
  .image-container { flex: 0 0 auto; width: 100% !important; height: 30dvh; }
  .content-container { flex: 1; width: 100%; }
}
@media (prefers-reduced-motion: reduce) {
  .discover-enter-active, .discover-leave-active,
  .discover-enter-active .note-card, .discover-leave-active .note-card,
  .carousel-fade-enter-active, .carousel-fade-leave-active { transition: none; }
}
</style>
