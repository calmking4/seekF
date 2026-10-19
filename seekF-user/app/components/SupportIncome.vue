<template>
  <section class="space-y-4 rounded-2xl border border-gray-100 bg-white p-6 sm:p-7">
    <div class="flex items-center justify-between gap-3"><h2 class="text-sm font-medium text-gray-700">助力收益</h2><el-button :loading="loading" @click="load">刷新</el-button></div>
    <p v-if="error" class="text-sm text-amber-600" role="alert">{{ error }}</p>
    <div v-if="income" class="grid gap-4 sm:grid-cols-3"><div v-for="item in metrics" :key="item.key"><p class="text-xs text-gray-400">{{ item.label }}</p><p class="mt-2 text-2xl font-semibold text-gray-700">¥{{ income[item.key] ?? '0.00' }}</p></div></div>
    <p class="text-xs leading-6 text-gray-400">按已确认付款订单统计。退款处理中仍计入实收，确认全额退款后计入退款金额；以上为助力收款统计，不是可提现余额。本期不支持提现和自动分账。</p>
  </section>
</template>
<script setup>
const auth = useAuthState(), income = ref(null), loading = ref(false), error = ref('')
const owner = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
const metrics = [{ key: 'gross_amount', label: '助力实收' }, { key: 'refund_amount', label: '已退款' }, { key: 'net_amount', label: '净额' }]
let disposed = false
const controller = new AbortController()
const sameUser = () => !disposed && owner && owner === (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')
const load = async () => {
  if (loading.value || !sameUser()) return
  loading.value = true; error.value = ''
  try { const result = await useApi$('/user/support/creator/income', { body: {}, retry: 0, signal: controller.signal }); if (!sameUser()) return; if (result.code !== 200) throw new Error(result.message || '查询助力收益失败'); income.value = result.data }
  catch (cause) { if (sameUser()) error.value = cause?.data?.message || cause.message || '查询助力收益失败' }
  finally { if (sameUser()) loading.value = false }
}
watch(() => auth.getUser()?.uuid || auth.getUser()?.Uuid || '', () => { controller.abort(); income.value = null; error.value = '账号已切换，请重新打开页面' })
onMounted(load)
onBeforeUnmount(() => { disposed = true; controller.abort() })
</script>
