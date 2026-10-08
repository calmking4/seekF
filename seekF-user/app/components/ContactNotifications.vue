<template>
  <div :aria-busy="loading">
    <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-5 py-5 sm:px-7">
      <h2 class="flex items-center gap-2 text-base font-semibold text-gray-800">好友和群<span v-if="filteredItems.length" class="text-xs font-normal text-gray-400">{{ filteredItems.length }}</span></h2>
      <button type="button" aria-label="刷新申请通知" :disabled="loading" class="grid h-8 w-8 place-items-center rounded-full text-gray-400 transition-colors hover:bg-gray-50 hover:text-blue-400 disabled:opacity-50" @click="load">
        <Icon name="uil:sync" class="text-lg" :class="{ 'animate-spin': loading }" />
      </button>
    </div>
    <div class="flex flex-wrap gap-2 px-5 pt-5 sm:px-7" role="group" aria-label="筛选好友和群通知">
      <button v-for="filter in filters" :key="filter.key" type="button" :aria-pressed="activeFilter === filter.key"
        class="rounded-full border px-4 py-2 text-xs font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400"
        :class="activeFilter === filter.key ? 'border-blue-100 bg-blue-50 text-blue-500' : 'border-transparent bg-gray-50 text-gray-500 hover:bg-gray-100'"
        @click="activeFilter = filter.key">{{ filter.label }}</button>
    </div>
    <div v-if="loading && !items.length" class="flex min-h-60 items-center justify-center gap-2 text-sm text-gray-400"><Icon name="uil:spinner" class="animate-spin text-xl" />正在加载通知…</div>
    <div v-else-if="error && !items.length" role="alert" class="flex min-h-60 flex-col items-center justify-center gap-4 px-5 text-center text-sm text-gray-500">
      <Icon name="uil:cloud-slash" class="text-3xl text-gray-300" /><p>{{ error }}</p><button type="button" class="rounded-full bg-blue-50 px-5 py-2 text-blue-500 hover:bg-blue-100" @click="load">重新加载</button>
    </div>
    <div v-else-if="!filteredItems.length" class="flex min-h-60 items-center justify-center text-sm text-gray-400">{{ activeFilter === 'all' ? '暂无好友或群通知' : activeFilter === 'group' ? '暂无群通知' : '暂无好友通知' }}</div>
    <div v-else class="px-5 py-6 sm:px-7">
      <p v-if="error" role="alert" class="mb-5 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-500">{{ error }}<button type="button" class="ml-3 underline" @click="load">重试</button></p>
      <section v-for="section in sections" v-show="section.list.length" :key="section.key" class="mb-6 last:mb-0">
        <h3 class="mb-2 flex items-center gap-2 text-xs font-medium text-gray-400">{{ section.label }}<span>{{ section.list.length }}</span></h3>
        <div class="divide-y divide-gray-100">
          <article v-for="item in section.list" :key="item.key" class="flex flex-wrap items-start gap-3 py-5 sm:gap-4">
            <el-avatar :size="44" :src="item.avatar" class="shrink-0"><Icon :name="item.contact_type === 'group' && !item.is_received ? 'uil:users-alt' : 'uil:user'" /></el-avatar>
            <div class="min-w-0 flex-1">
              <p class="break-words text-sm font-medium text-gray-800">{{ item.name || '未知用户' }}</p>
              <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-gray-400"><span>{{ item.is_received ? (item.contact_type === 'group' ? '申请加入你的群聊' : '请求添加你为好友') : (item.contact_type === 'group' ? '你发出的入群申请' : '你发出的好友申请') }}</span><time>{{ formatTime(item.apply_time) }}</time></div>
              <p v-if="item.contact_type === 'group' && item.is_received" class="mt-2 text-xs text-gray-500">群聊：{{ item.contact_name }}</p>
              <p class="mt-2 break-words text-sm leading-6 text-gray-500">{{ remark(item.message) }}</p>
            </div>
            <div v-if="item.is_received && item.status === 0" class="ml-auto flex gap-2 self-center">
              <button type="button" :disabled="pending[item.key]" class="rounded-full border border-blue-100 bg-blue-50 px-4 py-1.5 text-xs text-blue-500 transition-colors hover:bg-blue-100 disabled:opacity-50" @click="respond(item, true)">{{ pending[item.key] ? '处理中…' : '同意' }}</button>
              <button type="button" :disabled="pending[item.key]" class="rounded-full border border-gray-200 px-4 py-1.5 text-xs text-gray-500 transition-colors hover:bg-gray-50 disabled:opacity-50" @click="respond(item, false)">拒绝</button>
            </div>
            <span v-else class="self-center rounded-full px-3 py-1 text-xs" :class="item.status === 1 ? 'bg-green-50 text-green-600' : item.status === 0 ? 'bg-blue-50 text-blue-500' : 'bg-gray-50 text-gray-400'">{{ statusText(item.status) }}</span>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'

const items = ref([])
const activeFilter = ref('all')
const filters = [{ key: 'all', label: '全部' }, { key: 'user', label: '好友通知' }, { key: 'group', label: '群通知' }]
// 本地筛选同一批申请，切换好友和群时保留处理状态。
const filteredItems = computed(() => activeFilter.value === 'all' ? items.value : items.value.filter(item => item.contact_type === activeFilter.value))
const loading = ref(false)
const error = ref('')
const pending = ref({})
let requestVersion = 0
let disposed = false
const sections = computed(() => [
  { key: 'received', label: '收到的请求', list: filteredItems.value.filter(item => item.is_received) },
  { key: 'sent', label: '发出的请求', list: filteredItems.value.filter(item => !item.is_received) },
])
const statusText = status => ({ 0: '等待验证', 1: '已同意', 2: '已拒绝', 3: '已拉黑' }[status] || '未知状态')
const remark = message => String(message || '').replace(/^申请理由[：:]\s*/, '') || '未填写申请理由'
const formatTime = value => String(value || '').replace(/-/g, '/').slice(0, 16)

// 申请人的身份与目标群分开保存，审核时使用申请人的ID。
async function load() {
  const version = ++requestVersion
  loading.value = true
  error.value = ''
  try {
    const res = await useApi$('/user/contact/getMyApplyList')
    if (version !== requestVersion || disposed) return
    if (res.code !== 200) throw new Error(res.message || '获取申请通知失败')
    items.value = (res.data || []).filter(item => ['user', 'group'].includes(item.contact_type)).map(item => ({
      ...item, status: Number(item.status),
      key: `${item.contact_type}:${item.user_id}:${item.contact_id}:${item.is_received}`,
      name: item.is_received ? (item.user_name || item.contact_name) : item.contact_name,
      avatar: item.is_received ? (item.user_avatar || item.contact_avatar) : item.contact_avatar,
    })).sort((a, b) => String(b.apply_time || '').localeCompare(String(a.apply_time || '')))
  } catch (cause) {
    if (version === requestVersion && !disposed) error.value = cause.data?.message || cause.message || '加载失败，请稍后重试'
  } finally {
    if (version === requestVersion && !disposed) loading.value = false
  }
}

async function respond(item, accepted) {
  if (pending.value[item.key] || !item.is_received || item.status !== 0) return
  pending.value[item.key] = true
  try {
    const body = { contact_id: item.user_id }
    if (item.contact_type === 'group') body.group_id = item.contact_id
    const res = await useApi$(`/user/contact/${accepted ? 'passContactApply' : 'refuseContactApply'}`, { body })
    if (disposed) return
    if (res.code !== 200) throw new Error(res.message || '处理申请失败')
    item.status = accepted ? 1 : 2
    ElMessage.success(accepted ? '已同意申请' : '已拒绝申请')
    await load()
  } catch (cause) {
    if (!disposed) ElMessage.error(cause.data?.message || cause.message || '处理申请失败，请重试')
  } finally {
    if (!disposed) delete pending.value[item.key]
  }
}

onMounted(load)
onBeforeUnmount(() => { disposed = true; requestVersion++ })
</script>
