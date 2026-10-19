<template>
  <section class="relative flex max-h-[calc(100dvh-40px)] w-full flex-col overflow-y-auto bg-slate-50 text-slate-800" aria-label="助力创作者">
    <header class="relative isolate flex min-h-40 shrink-0 items-center justify-between gap-5 overflow-hidden bg-gradient-to-br from-blue-50 via-indigo-50 to-sky-100 px-6 py-8 sm:px-10">
      <Icon name="mdi:lightning-bolt" class="pointer-events-none absolute right-20 top-0 -z-10 text-[180px] text-white/60" />
      <div class="min-w-0">
        <p class="text-xs font-medium tracking-widest text-blue-500">创作者助力</p>
        <h2 class="mt-2 truncate text-xl font-semibold sm:text-2xl">{{ creator?.nickname || nickname || '助力创作者' }}</h2>
        <p class="mt-2 text-sm text-slate-500">让喜欢的创作，继续发生</p>
      </div>
      <el-avatar :size="72" :src="creator?.avatar || avatar" class="shrink-0 !border-4 !border-white/80" />
      <button type="button" aria-label="关闭助力弹窗" class="absolute right-3 top-3 flex h-8 w-8 items-center justify-center rounded-full text-slate-400 hover:bg-white/70 hover:text-slate-700 focus-visible:outline-blue-500" @click="$emit('close')"><Icon name="mdi:close" class="text-xl" /></button>
    </header>

    <div class="min-h-48 flex-1 px-5 py-7 sm:px-10 sm:py-8">
      <div class="relative overflow-hidden rounded-2xl border-2 border-blue-400 bg-white p-5 shadow-[0_8px_32px_rgba(59,130,246,0.08)] sm:p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h3 class="text-lg font-semibold">30天创作助力</h3>
          <p class="text-blue-500"><span class="text-sm">¥</span><span class="text-3xl font-semibold">{{ price }}</span><span class="ml-1 text-sm text-slate-500">/ 30天</span></p>
        </div>
        <p class="mt-4 text-sm leading-7 text-slate-600">解锁这位创作者的全部助力专属图文、视频及评论。一次助力，30天内的新作品也可查看。</p>
        <div class="mt-5 flex flex-wrap gap-x-5 gap-y-2 text-xs text-slate-400">
          <span class="inline-flex items-center gap-1"><Icon name="mdi:play-box" class="text-blue-400" />专属图文与视频</span>
        </div>
        <div class="absolute bottom-0 right-0 flex h-7 w-9 items-center justify-center rounded-tl-2xl bg-blue-400 text-white"><Icon name="mdi:check" /></div>
      </div>
      <p v-if="order && creator && order.amount !== Number(creator.price_yuan).toFixed(2)" class="mt-3 text-xs leading-5 text-slate-500">已为你保留此前待支付订单的价格 ¥{{ price }}，本次付款以此金额为准。</p>
      <p v-if="error" class="mt-4 text-sm text-amber-700" role="alert">{{ error }}</p>
      <button v-if="error && sameUser()" type="button" class="mt-2 text-sm text-blue-500 hover:text-blue-600 focus-visible:outline-blue-500" :disabled="loading" @click="initialize">重新加载</button>
    </div>

    <footer class="shrink-0 border-t border-slate-100 bg-white px-5 py-5 sm:px-10 sm:py-6" aria-live="polite">
      <div class="mb-4 flex items-center justify-between gap-3 text-xs text-slate-400"><span class="inline-flex items-center gap-1"><Icon name="mdi:information-outline" />单次购买</span><span v-if="remaining && order?.status === 0">{{ remaining }}</span></div>
      <div v-if="loading" class="flex min-h-44 items-center justify-center gap-2 text-sm text-slate-500" role="status"><Icon name="uil:spinner" class="animate-spin text-xl" />正在准备支付</div>
      <div v-else-if="order?.status === 1" class="flex min-h-36 flex-col items-center justify-center gap-3 text-emerald-600"><Icon name="mdi:check-circle-outline" class="text-4xl" /><p>助力成功，正在解锁作品</p><button type="button" class="text-sm text-blue-500" @click="$emit('paid')">查看作品</button></div>
      <div v-else-if="order?.status === 2 || expired" class="flex min-h-40 flex-col items-center justify-center gap-3 text-sm text-slate-500"><Icon name="mdi:qrcode-remove" class="text-4xl text-slate-300" /><p>{{ order?.status === 2 ? '本次支付已关闭' : '二维码已过期，请先确认支付结果' }}</p><el-button :loading="checking || loading" @click="retryPayment">{{ order?.status === 2 ? '重新获取二维码' : '刷新支付结果' }}</el-button></div>
      <div v-else-if="urls && sameUser()" class="flex flex-col items-center justify-center gap-5 sm:flex-row sm:gap-6">
        <div class="relative h-[202px] w-[202px] shrink-0 overflow-hidden rounded-lg border border-slate-100 bg-white p-2">
          <div v-if="!qrLoaded" class="absolute inset-0 flex items-center justify-center bg-white text-slate-400" role="status"><Icon name="uil:spinner" class="animate-spin text-2xl" /></div>
          <!-- 为支付宝二维码页面的默认页边距预留空间，避免二维码被裁切或出现内层滚动条。 -->
          <iframe :key="urls.qr_url" :src="urls.qr_url" title="支付宝扫码支付二维码" class="block h-[184px] w-[184px] overflow-hidden border-0" scrolling="no" sandbox="allow-scripts allow-forms allow-same-origin" referrerpolicy="no-referrer" @load="qrLoaded = true" />
        </div>
        <div class="min-w-0 text-center sm:text-left">
          <p class="text-lg">扫码开通 <span class="ml-1 text-sm">¥</span><span class="text-3xl font-semibold">{{ price }}</span></p>
          <p class="mt-2 flex items-center justify-center gap-1 text-sm text-slate-400 sm:justify-start"><Icon name="ri:alipay-fill" class="text-lg text-blue-500" />使用支付宝扫码支付</p>
          <a :href="urls.pay_url" target="_blank" rel="noopener noreferrer" class="mt-4 inline-flex items-center gap-1 rounded-lg bg-blue-500 px-5 py-2.5 text-sm text-white hover:bg-blue-600 focus-visible:outline-blue-500">网页支付<Icon name="mdi:open-in-new" /></a>
          <button type="button" :disabled="checking" class="mx-auto mt-3 block text-xs text-blue-500 disabled:text-slate-300 sm:mx-0" @click="refresh">{{ checking ? '正在确认…' : '已支付？刷新结果' }}</button>
          <p v-if="pollingEnded" class="mt-2 text-xs text-slate-400">自动查询已结束，请手动刷新结果。</p>
          <p v-else-if="order?.has_error" class="mt-2 text-xs text-amber-600">支付结果确认中，请稍候或手动刷新。</p>
        </div>
      </div>
      <p v-else class="py-8 text-center text-sm text-slate-400">{{ error ? '暂时无法加载支付，请重试' : '正在读取创作者信息' }}</p>
    </footer>
  </section>
</template>

<script setup>
const props = defineProps({ creatorId: { type: String, required: true }, nickname: String, avatar: String })
const emit = defineEmits(['paid', 'close'])
const creator = ref(null), urls = ref(null), loading = ref(false), qrLoaded = ref(false)
const { order, error, checking, pollingEnded, expired, now, owner, sameUser, request, showError, acceptOrder, refresh } = useSupportPayment(() => emit('paid'))
const price = computed(() => order.value ? order.value.amount : creator.value ? Number(creator.value.price_yuan).toFixed(2) : '—')
const remaining = computed(() => {
  if (!order.value || expired.value) return ''
  const seconds = Math.max(0, Math.ceil((new Date(order.value.expires_at).getTime() - now.value) / 1000))
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')} 内有效`
})
// 弹窗自动准备同一笔待支付订单，不要求用户进入订单页。
const initialize = async () => {
  if (loading.value || !sameUser()) return
  loading.value = true; error.value = ''; urls.value = null; qrLoaded.value = false
  try {
    creator.value = await request('/user/support/creator/info', { creator_id: props.creatorId })
    if (creator.value.has_support) { emit('paid'); return }
    if (!creator.value.activated) throw new Error('创作者暂未开通助力')
    const key = `seekf-support-request:${owner}:${props.creatorId}`
    let requestKey
    try { requestKey = sessionStorage.getItem(key) } catch { /* 存储受限时由服务端待支付订单复用防重。 */ }
    if (!requestKey) {
      requestKey = crypto.randomUUID()
      try { sessionStorage.setItem(key, requestKey) } catch { /* 服务端仍会复用有效待支付订单。 */ }
    }
    const row = await request('/user/support/order/create', { creator_id: props.creatorId, request_key: requestKey })
    try { sessionStorage.removeItem(key) } catch { /* 不影响已创建订单继续付款。 */ }
    acceptOrder(row)
    if (row.status !== 0 || new Date(row.expires_at).getTime() <= Date.now()) return
    const result = await request('/user/support/order/pay', { order_no: row.order_no })
    if (order.value?.status === 0) urls.value = result
  } catch (cause) { showError(cause) }
  finally { loading.value = false }
}
const retryPayment = async () => {
  await refresh()
  if (order.value?.status === 2) await initialize()
}
onMounted(initialize)
</script>
