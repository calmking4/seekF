<template>
  <div class="notification-page" :style="{ '--switch-offset': `${switchDirection * 16}px` }">
    <div class="notification-shell">
      <header class="page-heading">
        <div>
          <h1>消息通知<span v-if="unreadCount" class="heading-count">{{ badge(unreadCount) }} 条未读</span></h1>
        </div>
      </header>
      <div class="category-grid" role="tablist" aria-label="通知分类" :style="{ '--active-index': activeIndex }">
        <span class="category-indicator" aria-hidden="true" />
        <button v-for="category in categories" :id="`tab-${category.key}`" :key="category.key"
          role="tab" :aria-selected="activeCategory === category.key" aria-controls="notification-list"
          class="category-card" :class="[{ active: activeCategory === category.key }, category.key]"
          @click="selectCategory(category.key)" @keydown="handleTabKey($event, category.key)">
          <span class="category-icon"><Icon :name="category.icon" /><Transition name="notification-badge"><span v-if="unreadCategories[category.key]" class="badge">{{ badge(unreadCategories[category.key]) }}</span></Transition></span>
          <span class="category-copy"><strong>{{ category.label }}</strong></span>
          <Icon name="uil:angle-right" class="category-arrow" />
        </button>
      </div>
      <Transition name="notification-panel" mode="out-in">
      <section :key="activeCategory" id="notification-list" class="list-panel" role="tabpanel" :aria-labelledby="`tab-${activeCategory}`" :aria-busy="loading">
        <div class="list-toolbar">
          <h2>{{ currentCategory.label }}<span v-if="total > 0">{{ total }}</span></h2>
          <div class="toolbar-actions">
            <button class="refresh-button" aria-label="刷新通知" :disabled="loading" @click="refresh"><Icon name="uil:sync" :class="{ spinning: loading }" /></button>
          </div>
        </div>
        <Transition name="notification-content" mode="out-in">
        <div :key="contentState" class="list-content">
        <div v-if="loading && !list.length" class="skeleton-list" aria-label="正在加载通知">
          <div v-for="n in 4" :key="n" class="skeleton-row"><span class="skeleton-avatar" /><div><span /><span /></div><span class="skeleton-cover" /></div>
        </div>
        <div v-else-if="error && !list.length" class="empty-state" role="alert">
          <div class="empty-icon"><Icon name="uil:cloud-slash" /></div>
          <h3>加载失败</h3><p>{{ error }}</p><button class="primary-button" @click="refresh">重试</button>
        </div>
        <div v-else-if="!list.length" class="empty-state">
          <span>暂无消息</span>
        </div>
        <div v-else class="notification-list">
          <article v-for="item in list" :key="item.id" class="notification-row" :class="{ unread: !item.is_read }">
            <button class="avatar-button" :aria-label="`查看${item.actor_name || '用户'}的主页`" :disabled="!item.actor_id" @click="openActor(item)">
              <img v-if="item.actor_avatar && !failedImages[item.actor_avatar]" :src="item.actor_avatar" alt="" @error="failedImages[item.actor_avatar] = true" />
              <span v-else class="avatar-fallback" :style="{ background: avatarColor(item.actor_id) }">{{ (item.actor_name || '用户').slice(0, 1) }}</span>
              <span v-if="!item.is_read" class="unread-dot" aria-label="未读" />
            </button>
            <div class="notification-content">
              <button class="actor-name" :disabled="!item.actor_id" @click="openActor(item)">{{ item.actor_name || '未知用户' }}</button>
              <div class="event-meta"><span>{{ actionText(item.type) }}</span><time :datetime="item.created_at" :title="fullTime(item.created_at)">{{ formatTime(item.created_at) }}</time></div>
              <button v-if="item.type !== 4" class="post-message" @click="openPost(item)">
                <span v-if="isComment(item)" class="comment-text">{{ commentContent(item) }}</span>
                <span class="post-reference"><Icon name="uil:file-alt" />{{ item.target_available ? (item.post_title || '查看原帖') : '原帖已删除或不可见' }}</span>
              </button>
            </div>
            <button v-if="item.type === 4" class="follow-button" :class="{ followed: item.is_following }"
              :disabled="followPending[item.actor_id] || !item.target_available" @click="toggleFollow(item)">
              <Icon :name="item.is_following ? 'uil:check' : 'uil:plus'" />{{ followPending[item.actor_id] ? '处理中…' : item.is_following ? '已关注' : '回关' }}
            </button>
            <div v-else class="post-actions">
              <button v-if="isComment(item) && item.target_available" class="reply-button" @click="openPost(item, true)">{{ item.comment_uuid ? '回复' : '查看评论' }}</button>
              <button class="post-cover" aria-label="查看原帖" @click="openPost(item)">
                <img v-if="item.post_cover && !failedImages[item.post_cover]" :src="item.post_cover" alt="" loading="lazy" @error="failedImages[item.post_cover] = true" />
                <Icon v-else :name="item.target_available ? 'uil:file-alt' : 'uil:file-block-alt'" />
              </button>
            </div>
          </article>
          <div class="list-footer" aria-live="polite">
            <template v-if="error"><span>{{ error }}</span><button @click="loadList()">重试</button></template>
            <span v-else-if="loading">正在加载更多…</span>
            <button v-else-if="hasMore" @click="loadList()">加载更多<Icon name="uil:angle-down" /></button>
          </div>
        </div>
        </div>
        </Transition>
      </section>
      </Transition>
      <div ref="loadTrigger" class="load-trigger" aria-hidden="true" />
    </div>
    <DiscoverDetail v-if="selectedPost" :key="selectedPost.id" :item="selectedPost" @close="selectedPost = null" />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'

useSeoMeta({ title: '消息通知', description: '查看评论、赞和收藏以及新增关注，与 seekF 社区保持连接。' })
const categories = [
  { key: 'comments', label: '评论', icon: 'uil:comment-dots' },
  { key: 'likes', label: '赞和收藏', icon: 'uil:heart' },
  { key: 'follows', label: '新增关注', icon: 'uil:user-plus' },
]
const activeCategory = ref('comments')
const activeIndex = computed(() => categories.findIndex(item => item.key === activeCategory.value))
const switchDirection = ref(1)
const currentCategory = computed(() => categories.find(item => item.key === activeCategory.value))
const { unreadCount, unreadCategories, readBeforeIds, markCategoryAsRead, decrementUnread } = useNotification()
const list = ref([])
const total = ref(0)
const page = ref(0)
const loading = ref(false)
const hasMore = ref(true)
const error = ref('')
const contentState = computed(() => loading.value && !list.value.length ? 'loading' : error.value && !list.value.length ? 'error' : list.value.length ? 'list' : 'empty')
const selectedPost = ref(null)
const failedImages = ref({})
const followPending = ref({})
const loadTrigger = ref(null)
const pendingReads = new Set()
const readIds = new Set()
let requestVersion = 0
let selectionVersion = 0
let observer
let disposed = false
const pageSize = 20

const badge = count => count > 99 ? '99+' : count
const isComment = item => [2, 3].includes(item.type)
const categoryOf = item => isComment(item) ? 'comments' : item.type === 4 ? 'follows' : 'likes'
const actionText = type => ({ 1: '赞了你的帖子', 2: '评论了你的帖子', 3: '回复了你的评论', 4: '成为了你的新关注者', 5: '收藏了你的帖子' }[type] || '与你有了新互动')
const commentContent = item => (item.content || '').replace(/^(评论了你的帖子|回复了你的评论)[：:]/, '') || '查看评论内容'
const avatarColor = id => ['#7ca5cf', '#b39ac8', '#d6a18f', '#83b6a4'][Array.from(String(id || '')).reduce((sum, char) => sum + char.charCodeAt(0), 0) % 4]
const parseTime = value => new Date(String(value || '').replace(/^(\d{4}-\d{2}-\d{2}) /, '$1T'))
const fullTime = value => { const date = parseTime(value); return Number.isNaN(date.getTime()) ? '' : date.toLocaleString('zh-CN', { hour12: false }) }
const formatTime = value => {
  const date = parseTime(value)
  if (Number.isNaN(date.getTime())) return ''
  const minutes = Math.max(0, Math.floor((Date.now() - date.getTime()) / 60000))
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (minutes < 1440) return `${Math.floor(minutes / 60)}小时前`
  if (minutes < 10080) return `${Math.floor(minutes / 1440)}天前`
  return date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric', ...(date.getFullYear() !== new Date().getFullYear() ? { year: 'numeric' } : {}) })
}

async function loadList(reset = false) {
  if (!reset && (loading.value || !hasMore.value)) return
  const version = ++requestVersion
  const nextPage = reset ? 1 : page.value + 1
  const category = activeCategory.value
  if (reset) { list.value = []; page.value = 0; total.value = 0; hasMore.value = true }
  loading.value = true
  error.value = ''
  try {
    const res = await useApi$('/user/notification/list', { body: { page: nextPage, page_size: pageSize, category } })
    if (res.code !== 200) throw new Error(res.message || '获取通知失败')
    if (version !== requestVersion || disposed) return
    const incoming = res.data?.list || []
    const existing = new Set(list.value.map(item => item.id))
    list.value.push(...incoming.filter(item => !existing.has(item.id)).map(item =>
      readIds.has(item.id) || item.id <= readBeforeIds.value[category] ? { ...item, is_read: true } : item))
    total.value = res.data?.total || 0
    page.value = nextPage
    hasMore.value = incoming.length === pageSize && list.value.length < total.value
  } catch (cause) {
    if (version === requestVersion && !disposed) error.value = cause.data?.message || cause.message || '加载失败，请稍后重试'
  } finally {
    if (version === requestVersion && !disposed) loading.value = false
  }
}
async function selectCategory(category) {
  const nextIndex = categories.findIndex(item => item.key === category)
  if (nextIndex !== activeIndex.value) switchDirection.value = nextIndex > activeIndex.value ? 1 : -1
  const selection = ++selectionVersion
  requestVersion++
  activeCategory.value = category
  list.value = []
  total.value = 0
  loading.value = true
  error.value = ''
  try {
    // 后端标记整个分类，包含尚未加载的分页记录。
    await markCategoryAsRead(category)
  } catch (cause) {
    if (!disposed) ElMessage.error(cause.data?.message || cause.message || '标记分类通知已读失败')
  }
  if (selection === selectionVersion && !disposed) await loadList(true)
}
function handleTabKey(event, key) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const index = categories.findIndex(category => category.key === key)
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? 2 : (index + (event.key === 'ArrowRight' ? 1 : 2)) % 3
  selectCategory(categories[next].key)
  document.getElementById(`tab-${categories[next].key}`)?.focus()
}
function refresh() { return selectCategory(activeCategory.value) }

async function readItem(item) {
  if (item.is_read || pendingReads.has(item.id)) return
  pendingReads.add(item.id)
  try {
    const res = await useApi$('/user/notification/read', { body: { ids: [item.id] } })
    if (res.code !== 200) throw new Error(res.message || '标记已读失败')
    readIds.add(item.id)
    if (!item.is_read) {
      item.is_read = true
      if (item.id > readBeforeIds.value[categoryOf(item)]) decrementUnread(1, categoryOf(item))
      const displayed = list.value.find(row => row.id === item.id)
      if (displayed) displayed.is_read = true
    }
  } catch (cause) { ElMessage.error(cause.data?.message || cause.message || '标记已读失败') }
  finally { pendingReads.delete(item.id) }
}
function openActor(item) {
  if (!item.actor_id) return
  readItem(item)
  navigateTo(`/user/${encodeURIComponent(item.actor_id)}`)
}
function openPost(item, reply = false) {
  readItem(item)
  if (!item.target_available || !item.target_uuid) { ElMessage.info('原帖已删除或不可见'); return }
  selectedPost.value = {
    id: item.target_uuid, title: item.post_title, src: item.post_cover,
    initialReply: reply && item.comment_uuid ? {
      uuid: item.comment_uuid, parent_id: item.comment_parent_uuid,
      user_id: item.actor_id, nickname: item.actor_name, content: commentContent(item),
    } : null,
  }
}
async function toggleFollow(item) {
  if (followPending.value[item.actor_id]) return
  followPending.value[item.actor_id] = true
  try {
    const res = await useApi$('/user/follow/toggle', { body: { follow_user_id: item.actor_id } })
    if (res.code !== 200) throw new Error(res.message || '关注操作失败')
    list.value.forEach(row => { if (row.actor_id === item.actor_id) row.is_following = !!res.data?.is_followed })
    readItem(item)
    ElMessage.success(res.data?.is_followed ? '回关成功' : '已取消关注')
  } catch (cause) { ElMessage.error(cause.data?.message || cause.message || '关注操作失败') }
  finally { followPending.value[item.actor_id] = false }
}
onMounted(() => {
  refresh()
  observer = new IntersectionObserver(entries => {
    if (entries.some(entry => entry.isIntersecting) && list.value.length && !error.value) loadList()
  }, { root: loadTrigger.value?.closest('main'), rootMargin: '0px 0px 160px 0px' })
  if (loadTrigger.value) observer.observe(loadTrigger.value)
})
onBeforeUnmount(() => { disposed = true; selectionVersion++; requestVersion++; observer?.disconnect() })
</script>

<style scoped>
.notification-page { min-height: 100%; background: #fafbfc; color: #202938; padding: 44px 36px 24px; }
.notification-shell { max-width: 960px; margin: 0 auto; }
.page-heading { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-bottom: 30px; }
h1 { display: flex; align-items: center; flex-wrap: wrap; gap: 14px; font-size: 28px; font-weight: 650; letter-spacing: -.6px; margin: 0; }
.heading-count { color: #6391c5; background: #ecf4ff; font-size: 12px; font-weight: 500; padding: 5px 10px; border-radius: 20px; letter-spacing: 0; }
button { transition: background .18s, border-color .18s, color .18s; }
button:focus-visible, a:focus-visible { outline: 3px solid #a8d1ff; outline-offset: 4px; }
button:disabled { cursor: default; opacity: .5; }
.category-grid { --category-gap: 16px; position: relative; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--category-gap); margin-bottom: 28px; }
.category-indicator { position: absolute; z-index: 1; pointer-events: none; bottom: -1px; left: 0; width: calc((100% - var(--category-gap) * 2) / 3); height: 3px; transform: translateX(calc(var(--active-index) * (100% + var(--category-gap)))); transition: transform .36s cubic-bezier(.22, 1, .36, 1); }
.category-indicator::after { content: ''; display: block; width: 28px; height: 3px; margin: 0 auto; background: #60a5fa; border-radius: 3px; }
.category-card { --accent: #5798e8; display: flex; align-items: center; gap: 14px; text-align: left; padding: 22px 20px; border: 1px solid #e9edf2; border-radius: 18px; background: white; position: relative; }
.category-card { transition: background .24s ease, border-color .24s ease, box-shadow .24s ease, transform .24s cubic-bezier(.22, 1, .36, 1); }
.category-card:active { transform: scale(.98); }
.category-card.likes { --accent: #e57989; }
.category-card.follows { --accent: #9b86d5; }
.category-card:hover { border-color: #c9d9ed; background: #fcfdff; }
.category-card.active { border-color: #9dc4f5; box-shadow: 0 4px 18px #609ae90a; background: #f7faff; }
.category-icon { position: relative; flex-shrink: 0; width: 48px; height: 48px; display: grid; place-items: center; color: var(--accent); font-size: 40px; }
.category-icon { transition: transform .32s cubic-bezier(.34, 1.56, .64, 1); }
.category-card.active .category-icon { transform: translateY(-2px) scale(1.06); }
.badge { position: absolute; top: -7px; right: -7px; background: #ee7482; border: 2px solid white; border-radius: 20px; min-width: 20px; height: 20px; padding: 0 4px; display: grid; place-items: center; color: white; font-size: 10px; }
.category-copy { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.category-copy strong { font-size: 15px; font-weight: 600; white-space: nowrap; }
.category-arrow { color: #b3becc; margin-left: auto; flex-shrink: 0; }
.list-panel { background: white; border: 1px solid #e9edf2; border-radius: 20px; overflow: hidden; }
.list-content { min-height: 240px; }
.notification-panel-enter-active { transition: opacity .24s ease, transform .28s cubic-bezier(.22, 1, .36, 1); }
.notification-panel-leave-active { transition: opacity .12s ease, transform .14s ease; pointer-events: none; }
.notification-panel-enter-from { opacity: 0; transform: translateX(var(--switch-offset)); }
.notification-panel-leave-to { opacity: 0; transform: translateX(calc(var(--switch-offset) * -1)); }
.notification-content-enter-active { transition: opacity .2s ease, transform .24s cubic-bezier(.22, 1, .36, 1); }
.notification-content-leave-active { transition: opacity .1s ease; pointer-events: none; }
.notification-content-enter-from { opacity: 0; transform: translateY(6px); }
.notification-content-leave-to { opacity: 0; }
.notification-badge-enter-active, .notification-badge-leave-active { transition: opacity .18s ease, transform .2s ease; }
.notification-badge-enter-from, .notification-badge-leave-to { opacity: 0; transform: scale(.5); }
.list-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 22px 28px; border-bottom: 1px solid #f0f2f5; gap: 12px; }
h2 { display: flex; align-items: center; gap: 10px; font-size: 16px; font-weight: 600; margin: 0; }
h2 span { font-size: 12px; font-weight: 400; color: #9aa4b0; }
.toolbar-actions { display: flex; gap: 12px; align-items: center; }
.refresh-button { color: #a0aab7; display: flex; padding: 5px; font-size: 17px; }
.notification-row { display: flex; align-items: flex-start; gap: 16px; padding: 25px 28px; position: relative; }
.notification-row + .notification-row::before { content: ''; position: absolute; top: 0; left: 92px; right: 28px; height: 1px; background: #f1f3f6; }
.notification-row.unread { background: #f9fbff; }
.avatar-button { position: relative; flex: 0 0 48px; height: 48px; border-radius: 50%; }
.avatar-button img, .avatar-fallback { width: 48px; height: 48px; border-radius: 50%; object-fit: cover; }
.avatar-fallback { display: grid; place-items: center; color: white; font-size: 19px; }
.unread-dot { position: absolute; right: 0; top: 0; width: 9px; height: 9px; background: #ed7986; border: 2px solid white; border-radius: 50%; }
.notification-content { flex: 1; min-width: 0; padding-top: 1px; }
.actor-name { font-size: 14px; font-weight: 600; text-align: left; overflow-wrap: anywhere; }
.actor-name:hover { color: #5798e8; }
.event-meta { display: flex; gap: 14px; flex-wrap: wrap; align-items: center; margin-top: 5px; font-size: 12px; color: #8f99a7; }
.event-meta time { color: #b0b7c1; font-size: 11px; }
.post-message { display: flex; flex-direction: column; align-items: flex-start; text-align: left; width: 100%; margin-top: 10px; gap: 9px; }
.comment-text { font-size: 14px; line-height: 1.6; color: #475364; overflow-wrap: anywhere; white-space: pre-wrap; }
.post-reference { display: flex; align-items: center; gap: 6px; background: #f5f7fa; color: #94a0ae; font-size: 12px; padding: 6px 10px; border-radius: 7px; max-width: 100%; overflow-wrap: anywhere; }
.post-reference .iconify { flex-shrink: 0; }
.post-message:hover .post-reference { color: #5798e8; }
.post-actions { display: flex; align-items: center; gap: 16px; flex-shrink: 0; align-self: center; }
.post-cover { width: 62px; height: 68px; border-radius: 10px; background: #f1f5f9; overflow: hidden; display: grid; place-items: center; color: #b9c9dc; font-size: 26px; }
.post-cover img { width: 100%; height: 100%; object-fit: cover; }
.reply-button { color: #6e8dad; border: 1px solid #e5ecf3; font-size: 12px; padding: 6px 13px; border-radius: 16px; }
.reply-button:hover { border-color: #a8cdff; color: #409eff; }
.follow-button { display: inline-flex; align-items: center; justify-content: center; gap: 4px; border-radius: 20px; background: #60a5fa; color: white; padding: 8px 16px; font-size: 12px; min-width: 84px; align-self: center; flex-shrink: 0; }
.follow-button:not(:disabled):hover { background: #4a95f2; }
.follow-button.followed { background: #f5f7fa; color: #9aa5b3; }
.empty-state { min-height: 240px; padding: 48px 24px; display: flex; align-items: center; justify-content: center; flex-direction: column; text-align: center; color: #9aa4b1; font-size: 14px; }
.empty-state h3 { font-size: 17px; font-weight: 550; margin: 0 0 10px; }
.empty-state p { font-size: 13px; color: #9aa4b1; line-height: 1.8; max-width: 320px; margin: 0 0 24px; }
.primary-button { display: inline-flex; align-items: center; gap: 9px; font-size: 12px; border: 1px solid #deebfc; color: #649bde; background: #f6faff; border-radius: 20px; padding: 8px 18px; }
.list-footer { display: flex; align-items: center; justify-content: center; gap: 12px; padding: 25px; color: #b0b8c3; font-size: 12px; }
.list-footer button { display: flex; align-items: center; gap: 5px; color: #6b9edb; }
.load-trigger { height: 1px; }
.skeleton-list { padding: 0 28px; }
.skeleton-row { display: flex; gap: 16px; padding: 28px 0; }
.skeleton-row span { display: block; background: #f1f4f8; animation: pulse 1.5s ease-in-out infinite; border-radius: 5px; }
.skeleton-row .skeleton-avatar { width: 48px; height: 48px; border-radius: 50%; flex-shrink: 0; }
.skeleton-row > div { flex: 1; padding-top: 4px; }
.skeleton-row > div span:first-child { width: 110px; height: 12px; margin-bottom: 14px; }
.skeleton-row > div span:last-child { width: 60%; height: 10px; }
.skeleton-row .skeleton-cover { width: 62px; height: 62px; border-radius: 10px; }
.spinning { animation: spin 1s linear infinite; }
@keyframes pulse { 50% { opacity: .45; } }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 1000px) {
  .notification-page { padding: 28px 22px; }
  .category-card { padding: 18px 14px; gap: 10px; }
  .category-icon { width: 40px; height: 40px; font-size: 34px; }
  .category-arrow { display: none; }
}
@media (max-width: 720px) {
  .notification-page { padding: 24px 14px; }
  .page-heading { align-items: flex-start; gap: 10px; }
  h1 { font-size: 23px; gap: 8px; }
  .category-grid { --category-gap: 8px; margin-bottom: 18px; }
  .category-card { flex-direction: column; align-items: center; padding: 16px 6px; gap: 10px; text-align: center; border-radius: 14px; }
  .category-copy strong { font-size: 13px; }
  .list-toolbar { padding: 18px 16px; }
  .notification-row { padding: 20px 16px; gap: 12px; }
  .notification-row + .notification-row::before { left: 68px; right: 16px; }
  .avatar-button { flex-basis: 40px; height: 40px; }
  .avatar-button img, .avatar-fallback { width: 40px; height: 40px; }
  .post-actions { flex-direction: column-reverse; gap: 8px; }
  .post-cover { width: 48px; height: 54px; }
  .reply-button { padding: 3px 9px; }
  .follow-button { padding: 7px 10px; min-width: 66px; }
  .follow-button .iconify { display: none; }
}
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation: none !important; transition: none !important; } }
</style>
