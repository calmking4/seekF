<template>
  <section class="mx-auto max-w-[1440px] space-y-6 pb-6">
    <header class="flex flex-wrap items-center justify-between gap-3 px-1 pt-2"><div><h1 class="text-xl font-semibold text-gray-800">数据中心</h1><p class="mt-1.5 text-sm text-gray-400">了解作品表现，记录每一份关注与支持</p></div><el-button :loading="loading" :disabled="!ownerId" class="!rounded-xl !border-blue-100 !px-4 !text-blue-500 hover:!bg-blue-50" @click="load"><Icon v-if="!loading" name="mdi:refresh" class="mr-1.5 text-base" />刷新数据</el-button></header>
    <div v-if="error && !data" role="alert" class="rounded-2xl border border-red-100 bg-white p-8 text-center"><p class="mb-4 text-sm text-gray-500">{{ error }}</p><el-button @click="load">重新加载</el-button></div>
    <div v-else-if="!data" class="space-y-6" aria-label="数据中心加载中" role="status"><div v-for="index in 2" :key="index" class="rounded-2xl bg-white p-6 sm:p-8"><el-skeleton :rows="8" animated /></div></div>
    <template v-else>
      <div v-if="error" role="alert" class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-100 bg-amber-50 px-5 py-3 text-sm text-amber-700"><span>更新失败，当前保留上次数据：{{ error }}</span><button type="button" class="rounded px-2 py-1 font-medium hover:bg-amber-100 focus-visible:outline-amber-500" @click="load">重试</button></div>
      <section class="rounded-2xl bg-white p-5 sm:p-8" aria-labelledby="creator-overview-title">
        <div class="mb-6 flex flex-wrap items-center justify-between gap-4"><div class="flex flex-wrap items-center gap-3"><h2 id="creator-overview-title" class="text-lg font-semibold text-gray-900">核心数据概览</h2><span class="text-xs text-gray-400">更新至 {{ data.updated_at }}</span><span v-if="loading" role="status" class="inline-flex items-center gap-1 text-xs text-blue-400"><Icon name="mdi:loading" class="animate-spin" />更新中</span></div><div class="flex flex-wrap items-center gap-3"><el-select id="core-days" v-model="days" aria-label="核心数据时间范围" class="!w-32 [&_.el-select__wrapper]:!min-h-10 [&_.el-select__wrapper]:!rounded-xl [&_.el-select__wrapper]:!bg-gray-50/70" :popper-class="periodPopperClass"><el-option v-for="value in periods" :key="value" :value="value" :label="`近${value}天`" /></el-select><button type="button" :disabled="loading" class="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-blue-100 bg-blue-50/70 px-4 text-sm font-medium text-blue-500 transition-colors hover:border-blue-200 hover:bg-blue-100/70 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-400 disabled:cursor-wait disabled:opacity-50" @click="exportOverview"><Icon name="mdi:export-variant" class="shrink-0 text-base" />导出数据</button></div></div>
        <div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4 xl:grid-cols-7">
          <div v-for="(metric, index) in metrics" :key="metric.key" class="relative min-w-0 overflow-hidden rounded-xl p-4 sm:py-5" :class="index === 0 ? 'bg-gradient-to-br from-[#60a5fa] to-blue-400 text-white' : 'bg-gray-50 text-gray-800'"><Icon :name="metric.icon" aria-hidden="true" class="pointer-events-none absolute bottom-3 right-3 text-4xl opacity-[0.10] sm:text-5xl" /><p class="relative text-xs font-medium sm:text-sm">{{ metric.label }}</p><p class="relative mt-2 text-2xl font-semibold tabular-nums" :aria-label="formatCount(data.metrics[metric.key])">{{ formatCount(animatedValues[metric.key]) }}</p></div>
        </div>
        <CreatorTrendChart :title="`近${loadedDays}天帖子点击量`" :rows="data.click_trend" field="clicks" />
        <details class="mt-4 text-xs leading-6 text-gray-400"><summary class="w-fit cursor-pointer hover:text-gray-600">统计说明</summary><p class="mt-2">按北京时间统计，包含今天。点击量按用户、帖子、自然日去重，不含作者自身点击。点赞、收藏为期间新增且当前仍有效的记录；评论包含回复，排除已删除记录。作品统计不含已删除作品。</p><p>净增粉丝为新增关注减取消关注。净增粉丝、分享从 {{ data.tracking_since }} 开始记录，之前的历史无法追溯。分享数为成功复制链接或确认手动复制的次数。</p><p>助力数为所选期间已付款且未全额退款的订单数，按当前支付环境统计。</p></details>
      </section>
      <section class="rounded-2xl bg-white p-5 sm:p-8" aria-labelledby="creator-revenue-title">
        <div class="mb-6 flex flex-wrap items-center justify-between gap-4"><h2 id="creator-revenue-title" class="text-lg font-semibold text-gray-900">收益数据</h2><div class="flex flex-wrap items-center gap-3"><el-select id="revenue-days" v-model="revenueDays" aria-label="收益时间范围" class="!w-32 [&_.el-select__wrapper]:!min-h-10 [&_.el-select__wrapper]:!rounded-xl [&_.el-select__wrapper]:!bg-gray-50/70" :popper-class="periodPopperClass"><el-option v-for="value in periods" :key="value" :value="value" :label="`近${value}天`" /></el-select><button type="button" :disabled="loading" class="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border border-blue-100 bg-blue-50/70 px-4 text-sm font-medium text-blue-500 transition-colors hover:border-blue-200 hover:bg-blue-100/70 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-400 disabled:cursor-wait disabled:opacity-50" @click="exportRevenue"><Icon name="mdi:export-variant" class="shrink-0 text-base" />导出数据</button></div></div>
        <div class="mb-6 flex flex-wrap items-center justify-between gap-4"><div class="min-w-52 rounded-xl bg-gradient-to-br from-[#60a5fa] to-blue-400 px-5 py-5 text-white"><p class="text-sm font-medium">净收益（元）</p><p class="mt-2 text-3xl font-semibold tabular-nums" :aria-label="data.revenue">{{ Number((animatedValues.revenue || 0).toFixed(2)).toFixed(2) }}</p></div><NuxtLink to="/discover/support-wallet" class="inline-flex items-center gap-1 rounded-full bg-blue-50 px-4 py-2 text-sm text-blue-500 hover:bg-blue-100 focus-visible:outline-blue-400">查看收益明细<Icon name="mdi:chevron-right" /></NuxtLink></div>
        <CreatorTrendChart :title="`近${loadedRevenueDays}天净收益（元）`" :rows="data.revenue_trend" field="revenue" unit="元" />
        <p class="mt-4 text-xs leading-6 text-gray-400">已扣除平台份额，按收益流水发生日计算入账减退款冲销；退款日可能为负数。提现、冻结与解冻不计入收益。{{ data.environment === 'sandbox' ? '当前为沙箱数据。' : '' }}</p>
      </section>
    </template>
  </section>
</template>

<script setup>
definePageMeta({ layout: 'creator', keepalive: false, pageTransition: { name: 'creator-page', mode: 'out-in' } })
useSeoMeta({ title: '数据中心', description: '查看自己的作品点击、互动、粉丝变化与创作收益。' })
const auth = useAuthState()
const ownerId = computed(() => auth.getUser()?.uuid || auth.getUser()?.Uuid || '')
const periods = [7, 30, 90]
const days = ref(7)
const revenueDays = ref(30)
const loadedDays = ref(days.value)
const loadedRevenueDays = ref(revenueDays.value)
const periodPopperClass = '!rounded-xl !border-blue-100 !shadow-lg [&_.el-select-dropdown__item]:!mx-1 [&_.el-select-dropdown__item]:!rounded-lg [&_.el-select-dropdown__item.is-selected]:!bg-blue-50 [&_.el-select-dropdown__item.is-selected]:!text-blue-500'
const data = ref(null)
const loading = ref(false)
const error = ref('')
const metrics = [
  { key: 'clicks', label: '帖子点击量', icon: 'mdi:cursor-default-click-outline' },
  { key: 'net_followers', label: '净增粉丝', icon: 'mdi:account-plus-outline' },
  { key: 'likes', label: '点赞', icon: 'mdi:heart-outline' },
  { key: 'collections', label: '收藏', icon: 'mdi:bookmark-outline' },
  { key: 'supports', label: '助力数', icon: 'mdi:lightning-bolt-outline' },
  { key: 'comments', label: '评论', icon: 'mdi:comment-text-outline' },
  { key: 'shares', label: '分享', icon: 'mdi:share-variant-outline' },
]
let version = 0
let controller
let dataOwner = ''
const animatedValues = ref({})
const { animate, cancel } = useCreatorAnimation()
const formatCount = value => new Intl.NumberFormat('zh-CN').format(Math.round(value ?? 0))
// 数字从当前显示值过渡到新结果，真实数据仍用于导出和无障碍标签。
watch(data, result => {
  if (!result) { cancel(); animatedValues.value = {}; return }
  const target = { ...result.metrics, revenue: Number(result.revenue) || 0 }
  const previous = { ...animatedValues.value }
  animate(progress => {
    animatedValues.value = Object.fromEntries(Object.entries(target).map(([key, value]) => [key, (previous[key] || 0) + (value - (previous[key] || 0)) * progress]))
  })
}, { flush: 'sync' })
// 同账号刷新时保留卡片和图表，仅替换数据；切换账号立即移除旧数据。
const load = async () => {
  const requestVersion = ++version
  const owner = ownerId.value
  controller?.abort()
  if (dataOwner !== owner) { data.value = null; dataOwner = owner }
  error.value = ''
  if (!owner) { loading.value = false; error.value = '请先登录后查看数据'; return }
  controller = new AbortController()
  const requestedDays = days.value
  const requestedRevenueDays = revenueDays.value
  loading.value = true
  try {
    const res = await useApi$('/user/discover/analytics', { body: { days: requestedDays, revenue_days: requestedRevenueDays }, signal: controller.signal, retry: 0, timeout: 15000 })
    if (requestVersion !== version || owner !== ownerId.value) return
    if (res.code !== 200 || !res.data) throw new Error(res.message || '加载数据中心失败')
    loadedDays.value = requestedDays
    loadedRevenueDays.value = requestedRevenueDays
    data.value = res.data
  } catch (err) {
    if (requestVersion === version && owner === ownerId.value) error.value = err?.data?.message || err?.message || '加载失败，请稍后重试'
  } finally { if (requestVersion === version) loading.value = false }
}
// 导出后释放对象URL，金额保留接口原始两位小数字符串。
const download = (name, rows) => {
  const csv = rows.map(row => row.map(cell => `"${String(cell).replaceAll('"', '""')}"`).join(',')).join('\r\n')
  const url = URL.createObjectURL(new Blob(['\uFEFF', csv], { type: 'text/csv;charset=utf-8;' }))
  const anchor = document.createElement('a')
  anchor.href = url; anchor.download = name; document.body.appendChild(anchor); anchor.click(); anchor.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
const exportOverview = () => {
  if (!data.value) return
  download(`核心数据-近${loadedDays.value}天.csv`, [['指标', '数值'], ...metrics.map(metric => [metric.label, data.value.metrics[metric.key]]), ['事件统计起始时间', data.value.tracking_since], [], ['日期', '点击量'], ...data.value.click_trend.map(row => [row.date, row.clicks])])
}
const exportRevenue = () => {
  if (!data.value) return
  download(`收益数据-近${loadedRevenueDays.value}天.csv`, [['日期', '净收益（元）'], ...data.value.revenue_trend.map(row => [row.date, row.revenue]), ['合计', data.value.revenue]])
}
watch([days, revenueDays], load)
watch(ownerId, load, { flush: 'sync' })
onMounted(load)
onUnmounted(() => { version++; controller?.abort() })
</script>
