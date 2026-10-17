<template>
  <section class="flex h-full min-h-0 flex-col overflow-hidden rounded-2xl bg-white p-5 sm:p-8">
    <div class="mb-6 flex shrink-0 flex-wrap items-center justify-between gap-4">
      <div class="relative isolate flex flex-wrap items-center gap-1" aria-label="作品状态筛选">
        <LiquidSelectionIndicator :selected="filter" />
        <button v-for="tab in tabs" :key="tab.key" :data-liquid-key="tab.key" type="button" class="relative rounded-full transition-colors duration-300 px-5 py-2.5 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500" :class="filter === tab.key ? 'font-medium text-gray-800' : 'text-gray-500 hover:text-gray-700'" :aria-pressed="filter === tab.key" @click="selectFilter(tab.key)">{{ tab.label }}<span class="ml-1">{{ statusCounts[tab.key] || 0 }}</span></button>
      </div>
      <form class="flex w-full items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 sm:w-72" @submit.prevent="searchWorks"><Icon name="mdi:magnify" class="text-xl text-gray-400" /><input v-model="keyword" type="search" maxlength="100" placeholder="搜索作品标题" aria-label="搜索作品标题" class="min-w-0 flex-1 bg-transparent text-sm outline-none" /><button type="submit" class="text-sm text-blue-500">搜索</button></form>
    </div>
    <div class="relative grid min-h-0 flex-1 overflow-hidden">
    <Transition :name="slideDirection">
    <div :key="displayedFilter" ref="worksScroller" :inert="loading && displayedFilter !== filter" class="col-start-1 row-start-1 min-h-0 min-w-0 overflow-y-auto overscroll-contain">
    <div v-if="loading && displayedFilter === filter" class="flex justify-center gap-2 py-20 text-sm text-gray-400" role="status"><Icon name="mdi:loading" class="animate-spin text-xl" />加载作品...</div>
    <div v-else-if="error" class="py-20 text-center"><p class="mb-4 text-sm text-gray-500">{{ error }}</p><el-button @click="loadWorks">重新加载</el-button></div>
    <div v-else-if="!posts.length" class="flex flex-col items-center py-20 text-gray-400"><Icon name="mdi:clipboard-text-outline" class="mb-4 text-5xl text-gray-200" /><p class="text-sm">{{ appliedKeyword ? '没有找到相关作品' : '暂无作品' }}</p><NuxtLink to="/discover/create" class="mt-5 rounded-full bg-[#60a5fa] px-6 py-2.5 text-sm text-white">去发布</NuxtLink></div>
    <div v-else class="w-full space-y-4">
      <article v-for="post in posts" :key="post.uuid" class="flex flex-col gap-4 rounded-2xl border border-gray-100 bg-white p-4 sm:flex-row sm:gap-5 sm:p-5">
        <NuxtLink :to="'/discover/post/' + encodeURIComponent(post.uuid)" target="_blank" rel="noopener" :aria-label="'查看作品：' + post.title" class="relative h-32 w-36 shrink-0 overflow-hidden rounded-xl bg-gray-100"><img :src="workCover(post)" :alt="post.title" class="h-full w-full object-cover" @error="useDefaultCover" /><span v-if="post.media_type === 1" aria-hidden="true" class="absolute bottom-2 right-2 flex h-6 w-6 items-center justify-center rounded-full bg-black/40 text-white"><Icon name="mdi:play" class="text-base" /></span></NuxtLink>
        <div class="flex min-w-0 flex-1 flex-col">
          <NuxtLink :to="'/discover/post/' + encodeURIComponent(post.uuid)" target="_blank" rel="noopener" class="line-clamp-2 text-base font-medium text-gray-800 hover:text-blue-500">{{ post.title }}</NuxtLink>
          <p class="mt-2 text-xs text-gray-400">{{ post.created_at }}</p>
          <span v-if="post.support_only === 1" class="mt-2 w-fit rounded-full bg-blue-50 px-2 py-1 text-xs text-blue-500">助力专属</span>
          <span class="mt-2 w-fit rounded-full px-2 py-1 text-xs" :class="post.visibility === 1 ? 'bg-gray-100 text-gray-500' : 'bg-blue-50 text-blue-500'">{{ post.visibility === 1 ? '仅自己可见' : '公开' }}</span>
          <div class="mt-4 flex flex-wrap items-center gap-4 text-xs text-gray-400 sm:mt-auto sm:pt-4"><span v-for="stat in stats" :key="stat.field" class="flex items-center gap-1" :title="stat.label" :aria-label="stat.label + '：' + (post[stat.field] || 0)"><Icon :name="stat.icon" class="text-base" />{{ post[stat.field] || 0 }}</span></div>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-2 sm:self-start">
          <el-select :model-value="post.visibility" :disabled="!!pending[post.uuid]" :aria-label="'谁可以看：' + post.title" class="!w-36" @change="value => changeVisibility(post, value)"><el-option :value="0" label="公开" /><el-option :value="1" label="仅自己可见" /></el-select>
          <NuxtLink :to="'/discover/create?edit=' + encodeURIComponent(post.uuid)" title="编辑作品" :aria-disabled="!!pending[post.uuid]" :tabindex="pending[post.uuid] ? -1 : undefined" @click="pending[post.uuid] && $event.preventDefault()" :aria-label="'编辑作品：' + post.title" class="flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 hover:bg-blue-50 hover:text-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" :class="pending[post.uuid] ? 'pointer-events-none opacity-50' : ''"><Icon name="uil:edit-alt" class="text-lg" /></NuxtLink>
          <button type="button" title="删除作品" :aria-label="'删除作品：' + post.title" :disabled="!!pending[post.uuid]" class="flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 hover:bg-red-50 hover:text-red-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-red-400 disabled:opacity-50" @click="deleteWork(post)"><Icon name="uil:trash-alt" class="text-lg" /></button>
        </div>
      </article>
    </div>
    </div>
    </Transition>
    <div v-if="loading && displayedFilter !== filter" class="absolute inset-0 flex items-center justify-center gap-2 bg-white/80 text-sm text-gray-400" role="status"><Icon name="mdi:loading" class="animate-spin text-xl" />加载作品...</div>
    </div>
    <div v-if="total > pageSize && !error && !loading" class="mt-8 flex shrink-0 justify-center"><el-pagination :current-page="page" :page-size="pageSize" :total="total" :disabled="loading" layout="prev, pager, next" @current-change="changePage" /></div>
  </section>
</template>
<script setup>
definePageMeta({ layout: 'creator', keepalive: false, pageTransition: { name: 'creator-page', mode: 'out-in' } })
useSeoMeta({ title: '作品管理', description: '管理自己发布的作品和可见范围。' })
const tabs = [{ key: 0, label: '已发布' }, { key: 2, label: '审核中' }, { key: 3, label: '未通过' }, { key: 1, label: '已下架' }]
const stats = [{ field: 'view_count', label: '浏览', icon: 'mdi:eye-outline' }, { field: 'comment_count', label: '评论', icon: 'mdi:comment-outline' }, { field: 'like_count', label: '点赞', icon: 'mdi:heart-outline' }, { field: 'collect_count', label: '收藏', icon: 'mdi:star-outline' }]
const defaultWorkCover = '/images/discover-cover-default.svg'
// 视频首个媒体地址是视频文件，缺少封面时直接使用默认图片。
const workCover = post => post.cover_url || (post.media_type === 1 ? '' : post.first_url) || defaultWorkCover
// 原封面加载失败时使用本地默认图，默认图失败不重复设置以免循环。
const useDefaultCover = event => {
 const image = event.currentTarget
 if (image.getAttribute('src') !== defaultWorkCover) image.src = defaultWorkCover
}
const posts = ref([])
const filter = ref(0)
const slideDirection = ref('works-slide-next')
const displayedFilter = ref(0)
// 等目标状态请求返回后再切换内容，让真实作品或空状态参与轮播。
const revealFilter = () => {
 slideDirection.value = tabs.findIndex(tab => tab.key === filter.value) > tabs.findIndex(tab => tab.key === displayedFilter.value) ? 'works-slide-next' : 'works-slide-prev'
 displayedFilter.value = filter.value
}
const keyword = ref('')
const appliedKeyword = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const statusCounts = ref({ 0: 0, 1: 0, 2: 0, 3: 0 })
const worksScroller = ref(null)
const resetListScroll = () => { if (worksScroller.value) worksScroller.value.scrollTop = 0 }
const loading = ref(false)
const error = ref('')
const pending = ref({})
let requestId = 0
let disposed = false

// 每次筛选生成独立请求标识，丢弃先前请求的晚到响应。
const loadWorks = async () => {
 const current = ++requestId
 loading.value = true; error.value = ''
 try {
  const body = { page: page.value, page_size: pageSize, keyword: appliedKeyword.value, status: filter.value }
  const response = await useApi$('/user/discover/works', { body, retry: 0 })
  if (disposed || current !== requestId) return
  if (response.code !== 200) throw new Error(response.message || '加载作品失败')
  revealFilter()
  posts.value = response.data?.list || []; total.value = response.data?.total || 0
  statusCounts.value = { 0: 0, 1: 0, 2: 0, 3: 0, ...(response.data?.counts || {}) }
  if (!posts.value.length && page.value > 1 && total.value > 0) { page.value = Math.ceil(total.value / pageSize); await loadWorks() }
 } catch (cause) { if (!disposed && current === requestId) { revealFilter(); error.value = cause?.data?.message || cause.message || '加载作品失败' } }
 finally { if (!disposed && current === requestId) loading.value = false }
}
const selectFilter = value => { if (filter.value === value) return; filter.value = value; page.value = 1; void loadWorks() }
const searchWorks = () => { appliedKeyword.value = keyword.value.trim(); page.value = 1; resetListScroll(); void loadWorks() }
const changePage = value => { page.value = value; resetListScroll(); void loadWorks() }

// 可见性独立更新，只发送目标范围及当前版本，冲突时重新读取列表。
const changeVisibility = async (post, visibility) => {
 if (pending.value[post.uuid] || post.visibility === visibility) return
 pending.value[post.uuid] = true
 try {
  const response = await useApi$('/user/discover/visibility', { body: { post_uuid: post.uuid, visibility, version: post.version }, retry: 0 })
  if (response.code !== 200) throw new Error(response.message || '设置可见性失败')
  post.visibility = visibility; post.version = response.data.version
  ElMessage.success(visibility === 1 ? '已设为仅自己可见' : '已设为公开')
  await loadWorks()
 } catch (cause) { ElMessage.error(cause?.data?.message || cause.message || '设置可见性失败'); await loadWorks() }
 finally { pending.value[post.uuid] = false }
}
const deleteWork = async post => {
 if (pending.value[post.uuid]) return
 try { await ElMessageBox.confirm('删除后该作品将无法访问，确定删除吗？', '删除作品', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }) } catch { return }
 pending.value[post.uuid] = true
 try {
  const response = await useApi$('/user/discover/delete', { body: { post_uuid: post.uuid }, retry: 0 })
  if (response.code !== 200) throw new Error(response.message || '删除失败')
  ElMessage.success('作品已删除'); await loadWorks()
 } catch (cause) { ElMessage.error(cause?.data?.message || cause.message || '删除失败') }
 finally { pending.value[post.uuid] = false }
}
onMounted(loadWorks)
onUnmounted(() => { disposed = true; requestId++ })
</script>

<style scoped>
/* 同格叠放新旧列表，按页签顺序水平轮播，列表内部仍可独立滚动。 */
.works-slide-next-enter-active, .works-slide-next-leave-active,
.works-slide-prev-enter-active, .works-slide-prev-leave-active {
  transition: transform 420ms cubic-bezier(.22,.75,.25,1), opacity 320ms ease;
  will-change: transform, opacity;
}
.works-slide-next-leave-active, .works-slide-prev-leave-active { pointer-events: none; }
.works-slide-next-enter-from, .works-slide-prev-leave-to { transform: translateX(100%); opacity: 0; }
.works-slide-next-leave-to, .works-slide-prev-enter-from { transform: translateX(-100%); opacity: 0; }
@media (prefers-reduced-motion: reduce) {
  .works-slide-next-enter-active, .works-slide-next-leave-active,
  .works-slide-prev-enter-active, .works-slide-prev-leave-active { transition: none; }
  .works-slide-next-enter-from, .works-slide-next-leave-to,
  .works-slide-prev-enter-from, .works-slide-prev-leave-to { transform: none; }
}
</style>
