<template>
  <el-dialog
    v-model="visible"
    :title="title"
    align-center
    append-to-body
    destroy-on-close
    class="!w-[calc(100vw_-_2rem)] !max-w-[480px] !overflow-hidden !rounded-3xl !p-0 !shadow-2xl [&_.el-dialog\_\_header]:!m-0 [&_.el-dialog\_\_header]:!p-0 [&_.el-dialog\_\_body]:!p-0 [&_.el-dialog\_\_headerbtn]:!top-5 [&_.el-dialog\_\_headerbtn]:!right-4"
  >
    <template #header="{ titleId }">
      <div class="border-b border-blue-100/60 bg-gradient-to-br from-blue-50 via-white to-sky-50 px-6 pb-5 pt-6">
        <div class="mb-3 flex h-11 w-11 items-center justify-center rounded-2xl bg-white text-blue-400 shadow-sm ring-1 ring-blue-100">
          <Icon :name="type === 'following' ? 'uil:user-check' : 'uil:users-alt'" class="text-2xl" />
        </div>
        <h2 :id="titleId" class="text-lg font-semibold text-gray-900">{{ title }}</h2>
        <p class="mt-1 text-xs text-gray-400">{{ type === 'following' ? '在这里，遇见你感兴趣的人' : '每一份关注，都是一次相遇' }}</p>
      </div>
    </template>

    <div class="max-h-[min(52vh,440px)] min-h-[240px] overflow-y-auto overscroll-contain px-4 py-3 [scrollbar-gutter:stable]" :aria-busy="loading" aria-live="polite">
      <div v-if="loading && !list.length" class="space-y-5 px-2 py-4" aria-label="列表加载中">
        <div v-for="index in 4" :key="index" class="flex items-center gap-3 motion-safe:animate-pulse">
          <div class="h-12 w-12 shrink-0 rounded-full bg-blue-50" />
          <div class="flex-1 space-y-2"><div class="h-3 w-24 rounded bg-gray-100" /><div class="h-2.5 w-3/4 rounded bg-gray-50" /></div>
        </div>
      </div>
      <div v-else-if="!list.length && !error" class="flex min-h-[216px] flex-col items-center justify-center gap-3">
        <div class="flex h-16 w-16 items-center justify-center rounded-full bg-blue-50 text-blue-300"><Icon name="uil:users-alt" class="text-3xl" /></div>
        <p class="text-sm text-gray-500">{{ type === 'following' ? '还没有关注的人' : '还没有粉丝' }}</p>
        <p class="text-xs text-gray-400">{{ type === 'following' ? '去发现更多有趣的创作者吧' : '分享你的精彩，让更多人看见你' }}</p>
      </div>
      <TransitionGroup v-else tag="div" name="follow-list" appear class="space-y-1">
        <NuxtLink v-for="item in list" :key="item.uuid" :to="`/user/${encodeURIComponent(item.uuid)}`" class="group flex items-center gap-3 rounded-2xl p-3 transition-colors duration-200 hover:bg-blue-50/70 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" @click="visible = false">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-full bg-gradient-to-br from-blue-100 to-sky-50 text-lg font-medium text-blue-500 ring-1 ring-black/5">
            <img v-if="item.avatar" :src="item.avatar" :alt="`${item.nickname || '用户'}的头像`" class="h-full w-full object-cover" loading="lazy" />
            <span v-else>{{ (item.nickname || '用户').slice(0, 1) }}</span>
          </div>
          <div class="min-w-0 flex-1"><p class="truncate text-sm font-medium text-gray-800 group-hover:text-blue-500">{{ item.nickname || '用户' }}</p><p class="mt-1 truncate text-xs text-gray-400">{{ item.signature || '还没有简介' }}</p></div>
          <Icon name="uil:angle-right" class="shrink-0 text-lg text-gray-300 transition-transform group-hover:translate-x-0.5 group-hover:text-blue-400" />
        </NuxtLink>
      </TransitionGroup>
      <div v-if="error" class="py-6 text-center"><p class="mb-3 text-sm text-gray-500">{{ error }}</p><el-button round size="small" @click="fetchList">重新加载</el-button></div>
      <div v-else-if="list.length" class="py-4 text-center">
        <el-button v-if="!noMore" round size="small" :loading="loading" @click="fetchList">加载更多</el-button>
        <p v-else class="text-xs text-gray-300">已经看到全部了</p>
      </div>
    </div>
    <div class="border-t border-gray-100 bg-gray-50/60 px-6 py-3 text-xs text-gray-400">{{ total === null ? title : `共 ${total} 位${type === 'following' ? '关注' : '粉丝'}` }}</div>
  </el-dialog>
</template>

<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'

const props = defineProps({
  userId: { type: String, required: true },
  type: { type: String, default: 'following', validator: value => ['following', 'followers'].includes(value) },
})
const visible = defineModel({ type: Boolean, default: false })
const title = computed(() => props.type === 'following' ? '关注列表' : '粉丝列表')
const list = ref([])
const total = ref(null)
const loading = ref(false)
const error = ref('')
const noMore = ref(false)
const page = ref(1)
const pageSize = 20
let requestVersion = 0

// 分页成功后才推进页码，关闭或切换列表后丢弃旧响应。
const fetchList = async () => {
  if (loading.value || noMore.value || !visible.value || !props.userId) return
  const version = requestVersion
  loading.value = true
  error.value = ''
  try {
    const res = await useApi$(`/user/follow/list-${props.type}`, {
      body: { user_id: props.userId, page: page.value, page_size: pageSize },
    })
    if (version !== requestVersion) return
    if (res.code !== 200 || !res.data) throw new Error(res.message || '获取列表失败')
    const items = res.data.list || []
    const existing = new Set(list.value.map(item => item.uuid))
    list.value.push(...items.filter(item => !existing.has(item.uuid)))
    total.value = res.data.total ?? null
    noMore.value = items.length < pageSize || (total.value !== null && list.value.length >= total.value)
    page.value++
  } catch (err) {
    if (version !== requestVersion) return
    console.error('获取关注或粉丝列表失败:', err)
    error.value = '加载失败，请稍后重试'
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

watch([visible, () => props.userId, () => props.type], () => {
  requestVersion++
  list.value = []
  total.value = null
  loading.value = false
  error.value = ''
  noMore.value = false
  page.value = 1
  if (visible.value) fetchList()
}, { immediate: true })

onUnmounted(() => { requestVersion++ })
</script>

<style scoped>
/* TransitionGroup 的列表入场与位移动画，弹窗开合沿用 Element Plus 的淡入缩放。 */
.follow-list-enter-active,
.follow-list-leave-active,
.follow-list-move {
  transition: opacity 0.28s ease, transform 0.28s ease;
}
.follow-list-enter-from,
.follow-list-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
@media (prefers-reduced-motion: reduce) {
  .follow-list-enter-active,
  .follow-list-leave-active,
  .follow-list-move { transition: none; }
}
</style>
