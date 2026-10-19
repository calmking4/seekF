<template>
  <div class="mx-auto flex min-h-[70vh] max-w-lg flex-col items-center justify-center px-6 py-12 text-center">
    <span class="flex h-20 w-20 items-center justify-center rounded-full" :class="order?.status === 1 ? 'bg-emerald-50 text-emerald-500' : 'bg-blue-50 text-blue-500'"><Icon :name="order?.status === 1 ? 'mdi:check' : 'mdi:clock-outline'" class="text-4xl" /></span>
    <h1 class="mt-6 text-2xl font-semibold text-slate-800">{{ order?.status === 1 ? '助力成功' : order?.status === 2 ? '支付已关闭' : '正在确认支付结果' }}</h1>
    <p class="mt-3 text-sm leading-7 text-slate-500">{{ order?.status === 1 ? '30天观看权益已生效，可以回到原帖子继续查看。' : '支付结果以服务端确认为准，你也可以手动刷新。' }}</p>
    <p v-if="error" class="mt-4 text-sm text-amber-700" role="alert">{{ error }}</p>
    <p v-if="pollingEnded && order?.status === 0" class="mt-3 text-xs text-slate-400">自动查询已结束，请手动刷新。</p>
    <div class="mt-6 flex flex-wrap justify-center gap-3">
      <el-button v-if="order?.status !== 1" :loading="checking || loading" @click="load">刷新支付结果</el-button>
      <NuxtLink v-if="order" :to="`/user/${encodeURIComponent(order.creator_id)}`" class="rounded-lg bg-blue-500 px-5 py-2.5 text-sm text-white hover:bg-blue-600">查看创作者作品</NuxtLink>
      <NuxtLink to="/discover" class="rounded-lg border border-slate-200 px-5 py-2.5 text-sm text-slate-500 hover:bg-slate-50">返回发现</NuxtLink>
    </div>
  </div>
</template>
<script setup>
definePageMeta({ keepalive: false })
useSeoMeta({ title: '助力支付结果' })
const route = useRoute()
const loading = ref(false)
const { order, error, checking, pollingEnded, request, showError, acceptOrder } = useSupportPayment(() => ElMessage.success('支付成功，助力权益已生效'))
const load = async () => {
  if (loading.value) return
  loading.value = true
  try {
    if (typeof route.query.order_no !== 'string' || !route.query.order_no) throw new Error('缺少订单号，请返回原帖子查看支付结果')
    const no = route.query.order_no
    const row = await request('/user/support/order/info', { order_no: no })
    if (route.query.order_no !== no) return
    error.value = ''; acceptOrder(row)
  } catch (cause) { showError(cause) }
  finally { loading.value = false }
}
onMounted(load)
</script>
