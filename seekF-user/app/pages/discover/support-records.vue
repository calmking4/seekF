<template>
  <div class="mx-auto max-w-[1180px] space-y-6 pb-8 sm:pt-2">
    <header class="flex flex-wrap items-center justify-between gap-3 px-1">
      <div><h1 class="text-xl font-semibold tracking-tight text-gray-800">助力记录</h1><p class="mt-1.5 text-sm text-gray-400">查看收到的支持，以及你的助力订单与观看权益</p></div>
    </header>
    <section class="rounded-2xl border border-gray-100 bg-white p-5 sm:p-8">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-gray-100 pb-5">
        <div class="relative isolate flex flex-wrap gap-1" role="group" aria-label="助力记录类型">
          <LiquidSelectionIndicator :selected="active" fill-class="bg-blue-50" />
          <button v-for="tab in tabs" :key="tab.key" type="button" :data-liquid-key="tab.key" :aria-pressed="active === tab.key" class="relative rounded-full px-4 py-2.5 text-sm transition-colors duration-300 hover:text-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 sm:px-5" :class="active === tab.key ? 'font-medium text-blue-500' : 'text-gray-500'" @click="selectTab(tab.key)">{{ tab.label }}</button>
        </div>
        <el-button :loading="loading" :disabled="!sameUser()" @click="load">刷新记录</el-button>
      </div>
      <p class="mt-5 text-xs leading-6 text-gray-400">{{ active === 'received' ? '仅展示你作为创作者收到的已确认付款记录。' : active === 'orders' ? '仅展示已确认付款的助力订单，价格以创建订单时的快照为准。' : '权益按创作者独立计算，有效期内可查看对应创作者的全部助力专属作品。' }}</p>
      <div v-if="loading" class="flex min-h-64 items-center justify-center gap-2 text-sm text-gray-400" role="status"><Icon name="svg-spinners:180-ring" class="text-lg text-blue-400" />正在加载记录…</div>
      <div v-else-if="error" class="flex min-h-64 flex-col items-center justify-center gap-4 text-center" role="alert"><p class="text-sm text-gray-500">{{ error }}</p><el-button :disabled="!sameUser()" @click="load">重新加载</el-button></div>
      <div v-else-if="!rows.length" class="flex min-h-64 flex-col items-center justify-center text-gray-400"><Icon :name="active === 'entitlements' ? 'mdi:shield-check-outline' : 'mdi:clipboard-text-outline'" class="mb-4 text-5xl text-gray-200" /><p class="text-sm">{{ emptyText }}</p></div>
      <div v-else class="mt-5 space-y-4">
        <article v-for="row in rows" :key="row.order_no || row.creator_id" class="rounded-xl border border-gray-100 p-4 sm:p-5">
          <template v-if="active === 'received'">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <NuxtLink :to="`/user/${encodeURIComponent(row.buyer_id)}`" class="flex min-w-0 items-center gap-3 text-sm text-gray-700 hover:text-blue-500 focus-visible:outline-blue-400"><el-avatar :size="36" :src="row.buyer_avatar" /><span class="break-all font-medium">{{ row.buyer_nickname || '用户已注销' }}</span></NuxtLink>
              <div class="text-right"><p class="text-lg font-semibold text-gray-800">¥{{ money(row.amount) }}</p><span class="text-xs" :class="row.refund_status === 4 ? 'text-gray-400' : 'text-emerald-600'">{{ row.refund_status === 4 ? '已退款' : '付款已确认' }}</span></div>
            </div>
            <div class="mt-4 flex flex-wrap justify-between gap-2 text-xs leading-6 text-gray-400"><span class="break-all">订单号：{{ row.order_no }}</span><time>付款时间：{{ date(row.paid_at || row.confirmed_at) }}</time></div>
          </template>
          <template v-else-if="active === 'orders'">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0"><p class="break-all text-sm font-medium text-gray-700">订单号：{{ row.order_no }}</p><NuxtLink :to="`/user/${encodeURIComponent(row.creator_id)}`" class="mt-2 inline-block break-all text-xs text-blue-500 hover:text-blue-600 focus-visible:outline-blue-400">创作者：{{ row.creator_id }}</NuxtLink></div>
              <div class="text-right"><p class="text-lg font-semibold text-gray-800">¥{{ money(row.amount) }}</p><span class="text-xs" :class="orderStatusClass(row)">{{ orderStatus(row) }}</span></div>
            </div>
            <div class="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-xs leading-6 text-gray-400"><span>助力期限：{{ row.duration_hours / 24 }}天</span><time>创建时间：{{ date(row.created_at) }}</time><time v-if="row.paid_at">付款时间：{{ date(row.paid_at) }}</time></div>
            <ol v-if="row.refund_status" class="mt-6 grid gap-6 sm:grid-cols-3 sm:gap-4" aria-label="退款处理流程">
              <li v-for="(step, index) in refundSteps(row)" :key="index" class="relative flex min-w-0 items-start gap-3 sm:flex-col sm:items-center sm:gap-2 sm:text-center" :aria-current="step.state === 'current' || step.state === 'error' ? 'step' : undefined">
                <div v-if="index < 2" aria-hidden="true" class="absolute left-[17px] top-10 h-[calc(100%-16px)] w-px bg-gray-100 sm:left-[calc(50%+26px)] sm:top-[18px] sm:h-px sm:w-[calc(100%-36px)]">
                  <Icon name="mdi:chevron-down" class="absolute -left-[7px] top-1/2 -translate-y-1/2 text-base text-gray-300 sm:hidden" />
                  <Icon name="mdi:chevron-right" class="absolute left-1/2 top-1/2 hidden -translate-x-1/2 -translate-y-1/2 bg-white text-base text-gray-300 sm:block" />
                </div>
                <span class="relative z-[1] flex h-9 w-9 shrink-0 items-center justify-center rounded-full border text-sm" :class="step.state === 'done' ? 'border-emerald-100 bg-emerald-50 text-emerald-600' : step.state === 'current' ? 'border-blue-200 bg-blue-50 text-blue-500' : step.state === 'error' ? 'border-amber-200 bg-amber-50 text-amber-600' : 'border-gray-200 bg-white text-gray-300'">
                  <Icon v-if="step.state === 'done'" name="mdi:check" class="text-lg" /><Icon v-else-if="step.state === 'error'" name="mdi:close" class="text-lg" /><span v-else>{{ index + 1 }}</span>
                </span>
                <div class="min-w-0 flex-1 pt-1 sm:w-full sm:pt-0">
                  <p class="text-sm font-medium" :class="step.state === 'current' ? 'text-blue-500' : step.state === 'error' ? 'text-amber-600' : step.state === 'pending' ? 'text-gray-400' : 'text-gray-700'">{{ step.title }}</p>
                  <time v-if="step.time" :datetime="step.time" class="mt-1 block text-xs leading-5 text-gray-400">{{ date(step.time) }}</time>
                  <p v-if="step.description" class="mt-1 whitespace-pre-wrap break-words text-xs leading-5 text-gray-500">{{ step.description }}</p>
                </div>
              </li>
            </ol>
            <div v-if="row.can_refund && !row.refund_requested_at && now < new Date(row.paid_at).getTime() + 168 * 3600000" class="mt-3 text-xs leading-6 text-gray-500">
              <el-button :disabled="!sameUser()" @click="openRefund(row)">申请退款</el-button>
            </div>
          </template>
          <template v-else>
            <div class="flex flex-wrap items-center justify-between gap-3"><NuxtLink :to="`/user/${encodeURIComponent(row.creator_id)}`" class="break-all text-sm font-medium text-gray-700 hover:text-blue-500 focus-visible:outline-blue-400">创作者：{{ row.creator_id }}</NuxtLink><span class="rounded-full px-3 py-1 text-xs" :class="hasAccess(row) ? 'bg-emerald-50 text-emerald-600' : 'bg-gray-100 text-gray-400'">{{ hasAccess(row) ? '权益有效' : row.expires_at ? '已到期' : '暂无有效权益' }}</span></div>
            <div class="mt-4 flex flex-wrap items-center justify-between gap-3 text-xs text-gray-400"><span>权益截止：{{ date(row.expires_at) }}</span><NuxtLink :to="`/user/${encodeURIComponent(row.creator_id)}`" class="text-blue-500 hover:text-blue-600 focus-visible:outline-blue-400">查看创作者作品</NuxtLink></div>
          </template>
        </article>
      </div>
      <div v-if="!loading && !error" class="mt-6 flex flex-wrap items-center justify-center gap-3 text-sm text-gray-500">
        <el-button :disabled="page <= 1 || !sameUser()" @click="changePage(-1)">上一页</el-button><span>第 {{ page }} 页<span v-if="active !== 'entitlements'" class="ml-2 text-gray-400">共 {{ total }} 条</span></span><el-button :disabled="!hasNext || !sameUser()" @click="changePage(1)">下一页</el-button>
      </div>
    </section>
    <el-dialog v-model="refundOpen" title="申请全额退款" width="min(480px, calc(100vw - 32px))" :close-on-click-modal="!refundBusy" :close-on-press-escape="!refundBusy" :show-close="!refundBusy" @closed="refundReason = ''">
      <p class="mb-4 text-sm leading-6 text-gray-500">付款后7天内可申请，每单仅一次。管理员审核通过后发起支付宝原路退款，确认退款成功后重新计算观看权益。</p>
      <el-input v-model="refundReason" type="textarea" :rows="4" maxlength="500" show-word-limit aria-label="退款原因" placeholder="请说明退款原因" :disabled="refundBusy" />
      <p v-if="refundError" class="mt-3 text-sm text-red-500" role="alert">{{ refundError }}</p>
      <template #footer><el-button :disabled="refundBusy" @click="refundOpen = false">取消</el-button><el-button type="primary" :loading="refundBusy" :disabled="!sameUser() || !refundReason.trim()" @click="submitRefund">提交申请</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
definePageMeta({ layout: 'creator', keepalive: false, pageTransition: { name: 'creator-page', mode: 'out-in' } })
useSeoMeta({ title: '助力记录' })
const tabs = [{ key: 'received', label: '收到的助力' }, { key: 'orders', label: '我的订单' }, { key: 'entitlements', label: '我的权益' }]
const endpoints = { received: '/user/support/creator/records', orders: '/user/support/order/list', entitlements: '/user/support/entitlement/list' }
const refundOpen = ref(false), refundReason = ref(''), refundError = ref(''), refundBusy = ref(false), refundOrder = ref('')
// 退款进度只使用服务端确认状态，不把审核通过显示为退款完成。
const refundSteps = row => {
  const status = row.refund_status
  const channelRefund = !row.refund_requested_at && !row.refund_reason
  const channelReview = [4, 5].includes(status) && !row.refund_reviewed_at
  return [
    {
      title: channelRefund ? '渠道记录' : '提交申请', state: 'done', time: row.refund_requested_at,
      description: row.refund_reason ? `申请原因：${row.refund_reason}` : channelRefund ? '支付宝交易核对' : '',
    },
    {
      title: channelReview ? '渠道核对' : status === 2 ? '审核未通过' : status === 1 ? '等待审核' : '审核通过',
      state: status === 2 ? 'error' : status === 1 ? 'current' : 'done', time: row.refund_reviewed_at,
      description: [row.refund_review_note, status === 2 ? '如有异议请联系平台。' : status === 1 ? '等待管理员审核' : channelReview ? '已核对支付宝退款记录' : ''].filter(Boolean).join('\n'),
    },
    {
      title: status === 3 ? '退款处理中' : status === 5 ? '退款异常' : status === 2 ? '退款未执行' : '退款完成',
      state: status === 4 ? 'done' : status === 3 ? 'current' : status === 5 ? 'error' : 'pending', time: row.refunded_at,
      description: status === 4 ? '已确认全额退款，观看权益已重新计算' : status === 3 ? '确认退款前保留原有权益' : status === 5 ? '等待平台处理，确认退款前保留原有权益' : status === 2 ? '申请已驳回' : '审核通过后发起退款',
    },
  ]
}
let refundController
const openRefund = row => { refundOrder.value = row.order_no; refundReason.value = ''; refundError.value = ''; refundOpen.value = true }
const submitRefund = async () => {
  if (!sameUser() || refundBusy.value || !refundReason.value.trim()) return
  refundBusy.value = true; refundError.value = ''
  refundController = new AbortController()
  try {
    const response = await useApi$('/user/support/refund/apply', { body: { order_no: refundOrder.value, reason: refundReason.value.trim() }, signal: refundController.signal, retry: 0 })
    if (!sameUser()) return
    if (response.code !== 200) throw new Error(response.message || '申请退款失败')
    refundOpen.value = false; ElMessage.success('退款申请已提交'); await load()
  } catch (cause) {
    if (sameUser() && cause?.name !== 'AbortError') { refundError.value = cause?.data?.message || cause.message || '申请结果未确认，请刷新订单后查看'; await load() }
  } finally { refundBusy.value = false }
}
const auth = useAuthState()
const owner = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
const active = ref('received'), rows = ref([]), total = ref(0), page = ref(1), loading = ref(true), error = ref('')
const now = ref(Date.now())
const hasNext = computed(() => active.value === 'entitlements' ? rows.value.length === 20 : page.value * 20 < total.value)
const emptyText = computed(() => ({ received: '还没有收到助力', orders: '暂无助力订单', entitlements: '暂无助力权益' }[active.value]))
const money = value => value ?? '0.00'
const date = value => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '—'
const hasAccess = row => !!row.expires_at && new Date(row.expires_at).getTime() > now.value
const orderStatus = row => ({ 1: '退款申请中', 2: '退款已驳回', 3: '退款处理中', 4: '已退款', 5: '退款异常' }[row.refund_status] || (row.status === 1 ? '支付成功' : row.status === 2 ? '订单已关闭' : row.has_error ? '支付结果待确认' : '待支付确认'))
const orderStatusClass = row => row.refund_status ? [2, 4].includes(row.refund_status) ? 'text-gray-400' : 'text-amber-600' : row.status === 1 ? 'text-emerald-600' : row.status === 2 ? 'text-gray-400' : 'text-amber-600'
let disposed = false, version = 0, controller, clock
const sameUser = () => !disposed && !!owner && owner === (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')
// 切换标签或分页时取消旧查询，仅接受当前账号、当前页的最新结果。
const load = async () => {
  if (!sameUser()) return
  const current = ++version
  controller?.abort(); controller = new AbortController()
  loading.value = true; error.value = ''; rows.value = []; total.value = 0; now.value = Date.now()
  try {
    const response = await useApi$(endpoints[active.value], { body: { page: page.value }, signal: controller.signal, retry: 0 })
    if (!sameUser() || current !== version) return
    if (response.code !== 200) throw new Error(response.message || '查询助力记录失败')
    rows.value = active.value === 'entitlements' ? response.data : response.data.list
    total.value = active.value === 'entitlements' ? 0 : response.data.total
  } catch (cause) {
    if (sameUser() && current === version && cause?.name !== 'AbortError') error.value = cause?.data?.message || cause.message || '查询助力记录失败'
  } finally { if (sameUser() && current === version) loading.value = false }
}
const selectTab = key => { if (active.value !== key) { active.value = key; page.value = 1; load() } }
const changePage = delta => { if (!loading.value && page.value + delta >= 1 && (delta < 0 || hasNext.value)) { page.value += delta; load() } }
watch(() => auth.getUser()?.uuid || auth.getUser()?.Uuid || '', () => {
  version++; controller?.abort(); refundController?.abort(); refundOpen.value = false; clearInterval(clock); rows.value = []; total.value = 0; loading.value = false
  error.value = '账号已切换，请重新打开助力记录页'
})
onMounted(() => { load(); clock = setInterval(() => { now.value = Date.now() }, 1000) })
onBeforeUnmount(() => { disposed = true; version++; controller?.abort(); refundController?.abort(); clearInterval(clock) })
</script>
