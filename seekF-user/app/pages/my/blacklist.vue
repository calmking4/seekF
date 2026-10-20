<template>
  <div class="mx-auto min-h-screen max-w-3xl bg-white px-6 py-8">
    <NuxtLink to="/my" class="mb-6 inline-flex items-center gap-2 text-sm text-gray-500 hover:text-blue-500"><Icon name="mdi:arrow-left" />返回我的主页</NuxtLink>
    <h1 class="text-xl font-semibold text-gray-900">黑名单</h1>
    <p class="mt-3 text-sm leading-6 text-gray-500">拉黑后，首页不再展示对方的帖子，对方无法查看你的帖子或给你发私信。解除拉黑后恢复；若对方也拉黑了你，仍受对方限制。</p>
    <div v-if="loading && !rows.length" class="py-16 text-center text-gray-400" role="status">加载中...</div>
    <div v-else-if="!rows.length && !error" class="py-16 text-center text-sm text-gray-400">黑名单为空</div>
    <ul v-if="rows.length" class="mt-6 divide-y divide-gray-100">
      <li v-for="row in rows" :key="row.uuid" class="flex items-center gap-3 py-4">
        <NuxtLink :to="`/user/${encodeURIComponent(row.uuid)}`" class="flex min-w-0 flex-1 items-center gap-3 rounded-lg focus-visible:outline-blue-400">
          <img v-if="row.avatar" :src="row.avatar" alt="用户头像" class="h-12 w-12 shrink-0 rounded-full object-cover" />
          <span v-else class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-gray-50 text-gray-400"><Icon name="uil:user" /></span>
          <span class="min-w-0"><span class="block truncate text-sm font-medium text-gray-800">{{ row.nickname }}</span><span class="mt-1 block break-all text-xs text-gray-400">{{ row.uuid }}</span></span>
        </NuxtLink>
        <el-button round :loading="removing.has(row.uuid)" @click="remove(row)">解除拉黑</el-button>
      </li>
    </ul>
    <div v-if="error" class="py-6 text-center text-sm text-gray-500" role="alert"><p>{{ error }}</p><el-button class="mt-3" @click="load">重试</el-button></div>
    <div v-if="total > pageSize" class="mt-6 flex items-center justify-center gap-4">
      <el-button :disabled="page === 1 || loading" @click="changePage(-1)">上一页</el-button>
      <span class="text-sm text-gray-500">{{ page }} / {{ Math.ceil(total / pageSize) }}</span>
      <el-button :disabled="page * pageSize >= total || loading" @click="changePage(1)">下一页</el-button>
    </div>
  </div>
</template>

<script setup>
import { ElMessage } from 'element-plus'
definePageMeta({ keepalive: false })
useSeoMeta({ title: '黑名单' })
const rows = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const error = ref('')
const removing = reactive(new Set())
const blockVersion = useState('user-block-version', () => 0)
const account = useCookie('user')
const accountId = computed(() => account.value?.uuid || '')
let generation = 0
let disposed = false

const load = async () => {
  const version = ++generation
  loading.value = true
  error.value = ''
  try {
    const res = await useApi$('/user/block/list', { body: { page: page.value, page_size: pageSize } })
    if (disposed || version !== generation) return
    if (res.code !== 200) throw new Error(res.message || '加载黑名单失败')
    total.value = res.data.total
    if (page.value > 1 && (page.value - 1) * pageSize >= total.value) {
      page.value = Math.max(1, Math.ceil(total.value / pageSize))
      return await load()
    }
    rows.value = res.data.list || []
  } catch (e) {
    if (!disposed && version === generation) error.value = e.message || '加载黑名单失败，请重试'
  } finally {
    if (!disposed && version === generation) loading.value = false
  }
}

const changePage = (delta) => { page.value += delta; rows.value = []; load() }
const remove = async (row) => {
  if (removing.has(row.uuid)) return
  removing.add(row.uuid)
  const owner = accountId.value
  try {
    const res = await useApi$('/user/block/set', { body: { target_user_id: row.uuid, blocked: false } })
    if (disposed || owner !== accountId.value) return
    if (res.code !== 200) throw new Error(res.message || '解除拉黑失败')
    blockVersion.value++
    ElMessage.success('已解除拉黑')
    await load()
  } catch (e) {
    if (!disposed && owner === accountId.value) ElMessage.error(e.message || '解除拉黑失败，请重试')
  } finally { removing.delete(row.uuid) }
}
onMounted(load)
watch(accountId, () => { rows.value = []; page.value = 1; total.value = 0; load() })
onUnmounted(() => { disposed = true; generation++ })
</script>
