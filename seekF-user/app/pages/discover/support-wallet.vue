<template>
  <div class="mx-auto max-w-[1180px] space-y-6 pb-8 sm:pt-2">
    <header class="flex flex-wrap items-center justify-between gap-3 px-1"><div><h1 class="text-xl font-semibold tracking-tight text-gray-800">收益与提现</h1><p class="mt-1.5 text-sm text-gray-400">收益自动入账，随时申请提现至本人支付宝</p></div><el-button :loading="loading" :disabled="!sameUser()" @click="load">刷新</el-button></header>
    <p v-if="error" role="alert" class="rounded-xl bg-amber-50 p-4 text-sm text-amber-700">{{ error }}</p>
    <section class="rounded-2xl border border-gray-100 bg-white p-6 sm:p-7">
      <div class="grid gap-5 sm:grid-cols-3"><div v-for="metric in metrics" :key="metric.key"><p class="text-xs text-gray-400">{{ metric.label }}</p><p class="mt-2 text-2xl font-semibold text-gray-700">¥{{ balance[metric.key] ?? '0.00' }}</p></div></div>
      <p class="mt-5 text-xs leading-6 text-gray-400">以上为已记入收益账户的金额。历史助力收款统计不会自动转为可提现余额。退款申请会预留相应收益，已提现后的退款不足部分优先从后续收益抵扣。</p>
    </section>
    <section class="space-y-4 rounded-2xl border border-gray-100 bg-white p-6 sm:p-7">
      <div class="flex flex-wrap items-center justify-between gap-3"><h2 class="text-sm font-medium text-gray-700">提现到支付宝</h2><el-button :disabled="busy || !!pending || !sameUser()" @click="bindOpen = true">{{ account.bound ? '更换收款账户' : '绑定收款账户' }}</el-button></div>
      <p class="text-sm text-gray-500">{{ account.bound ? `${account.account_mask} · ${account.name_mask}` : '请先绑定本人支付宝账号并确认收款规则' }}</p>
      <div class="flex flex-wrap items-center gap-3"><el-input v-model="amount" inputmode="decimal" placeholder="提现金额（元）" aria-label="提现金额" class="!w-[240px] max-w-full" :disabled="busy || !!pending" /><el-button :disabled="busy || !!pending || !account.bound" @click="amount = balance.available_amount || '0.00'">全部提现</el-button><el-button type="primary" :loading="busy" :disabled="loading || !sameUser() || (!pending && (!account.bound || !!error))" @click="withdraw">{{ pending ? '核对上次提现申请' : '申请提现' }}</el-button></div>
      <p v-if="pending" role="status" class="text-sm text-amber-600">上次申请结果尚未确认。请核对原申请，系统会复用原请求标识，避免重复提现。</p>
      <p v-if="withdrawError" role="alert" class="text-sm text-red-500">{{ withdrawError }}</p>
      <p class="text-xs leading-6 text-gray-400">提现无需人工审核。提交后以支付宝确认结果为准，处理中或结果待确认的金额保持冻结；渠道手续费由平台承担。</p>
    </section>
    <section class="space-y-4 rounded-2xl border border-gray-100 bg-white p-6 sm:p-7">
      <div class="flex flex-wrap gap-3"><el-button :type="tab === 'withdrawals' ? 'primary' : 'default'" @click="selectTab('withdrawals')">提现记录</el-button><el-button :type="tab === 'ledger' ? 'primary' : 'default'" @click="selectTab('ledger')">收益明细</el-button></div>
      <p v-if="loading" role="status" class="py-8 text-center text-sm text-gray-400">正在查询…</p>
      <p v-else-if="!rows.length" class="py-8 text-center text-sm text-gray-400">暂无记录</p>
      <article v-for="row in rows" :key="row.withdraw_no || row.id" class="space-y-2 rounded-xl border border-gray-100 p-4 text-sm"><div class="flex flex-wrap items-center justify-between gap-2"><span class="font-medium text-gray-700">{{ tab === 'ledger' ? ledgerLabel(row.kind) : statuses[row.status] || '待核对' }}</span><span class="text-gray-700">¥{{ row.amount }}</span></div><p class="break-all text-xs text-gray-400">{{ row.withdraw_no || row.order_no }}</p><p v-if="tab === 'ledger'" class="text-xs text-gray-500">可用余额变化 ¥{{ row.available_delta }} · 变更后 ¥{{ row.available_after }} · 本次抵扣 ¥{{ row.offset_amount }}</p><p v-else class="text-xs text-gray-500">{{ row.account_mask }} · {{ row.status === 3 ? '已确认失败，金额已退回账户或抵扣欠款' : row.has_error ? '渠道结果待查询，请稍后刷新' : '以渠道确认结果为准' }}</p><p class="text-xs text-gray-400">{{ date(row.created_at) }}</p></article>
      <div class="flex flex-wrap items-center justify-center gap-3 text-sm text-gray-500"><el-button :disabled="loading || page <= 1" @click="changePage(-1)">上一页</el-button><span>第 {{ page }} 页 · 共 {{ total }} 条</span><el-button :disabled="loading || page * 20 >= total" @click="changePage(1)">下一页</el-button></div>
    </section>
    <el-dialog v-model="bindOpen" title="绑定本人支付宝" width="min(480px, calc(100vw - 32px))" :close-on-click-modal="!binding" :close-on-press-escape="!binding" :show-close="!binding" destroy-on-close @open="clearPayee" @closed="clearPayee">
      <el-form label-position="top" autocomplete="off" @submit.prevent="bind"><el-form-item label="支付宝手机号或邮箱"><el-input v-model="payee.account" name="settlement-payee-account" maxlength="128" autocomplete="off" :disabled="binding" /></el-form-item><el-form-item label="收款人真实姓名"><el-input v-model="payee.name" name="settlement-payee-name" maxlength="64" autocomplete="off" :disabled="binding" /></el-form-item><el-form-item v-if="account.requires_password" label="当前账号登录密码"><el-input v-model="payee.password" name="settlement-password-confirmation" type="password" show-password autocomplete="new-password" :disabled="binding" /></el-form-item><p class="text-xs leading-6 text-gray-500">新订单平台抽成 {{ (account.platform_rate_bps ?? 1000) / 100 }}%，按订单快照计算。退款余额不足由平台先垫付，后续收益优先抵扣，补足前不可提现。换绑只影响后续申请。</p><el-checkbox v-model="payee.accepted" :disabled="binding" class="mt-4 !h-auto [&_.el-checkbox__label]:!whitespace-normal">我确认这是本人收款账户，并同意上述收益与退款抵扣规则</el-checkbox><p v-if="bindError" role="alert" class="mt-3 text-sm text-red-500">{{ bindError }}</p><div class="mt-5 flex justify-end gap-3"><el-button :disabled="binding" @click="bindOpen = false">取消</el-button><el-button type="primary" native-type="submit" :loading="binding" :disabled="!payee.accepted || !sameUser()">确认绑定</el-button></div></el-form>
    </el-dialog>
  </div>
</template>
<script setup>
definePageMeta({ layout: 'creator', keepalive: false, pageTransition: { name: 'creator-page', mode: 'out-in' } })
useSeoMeta({ title: '收益与提现' })
const auth = useAuthState(), owner = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
const account = ref({}), balance = ref({}), rows = ref([]), total = ref(0), tab = ref('withdrawals'), page = ref(1), loading = ref(false), error = ref('')
const amount = ref(''), busy = ref(false), pending = ref(null), withdrawError = ref(''), bindOpen = ref(false), binding = ref(false), bindError = ref('')
const payee = reactive({ account: '', name: '', password: '', accepted: false })
const metrics = [{ key: 'income_amount', label: '累计收益' }, { key: 'available_amount', label: '可提现' }, { key: 'withdraw_frozen_amount', label: '提现中' }, { key: 'withdrawn_amount', label: '已提现' }, { key: 'refund_frozen_amount', label: '退款冻结' }, { key: 'debt_amount', label: '待抵扣' }]
const statuses = ['待处理', '处理中', '已到账', '已确认失败']
const ledgerLabel = key => ({ income: '收益入账', refund_reserve: '退款预留', refund_audit_reserve: '渠道退款预留', refund_release: '退款预留释放', refund: '退款冲销', withdraw_freeze: '提现冻结', withdraw_success: '提现成功', withdraw_fail: '提现失败解冻' }[key] || key)
const date = value => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '—'
let disposed = false, generation = 0, queryController, poll
const actionController = new AbortController()
const sameUser = () => !disposed && !!owner && owner === (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')
const storageKey = `support-withdraw:${owner}`
const clearPayee = () => { payee.account = ''; payee.name = ''; payee.password = ''; payee.accepted = false; bindError.value = '' }
const apiBase = useRuntimeConfig().public.apiBase
// 收款请求使用无正文日志的独立调用，避免通用错误钩子输出账号或密码。
const request = async (url, body, signal) => { const result = await $fetch(url, { baseURL: apiBase, method: 'POST', credentials: 'include', body, signal, retry: 0 }); if (result.code !== 200) { const failure = new Error(result.message || '操作失败'); failure.confirmedRejection = true; throw failure }; return result.data }
const load = async () => {
 if (!sameUser()) return
 const current = ++generation; queryController?.abort(); queryController = new AbortController(); loading.value = true; error.value = ''
 try { const [summary, bound, list] = await Promise.all([request('/user/support/settlement/summary', {}, queryController.signal), request('/user/support/settlement/account', {}, queryController.signal), request(tab.value === 'ledger' ? '/user/support/settlement/ledger' : '/user/support/withdraw/list', { page: page.value }, queryController.signal)]); if (!sameUser() || current !== generation) return; balance.value = summary; account.value = bound; rows.value = list.list; total.value = list.total }
 catch (cause) { if (sameUser() && current === generation && cause?.name !== 'AbortError') error.value = cause?.data?.message || cause.message || '查询失败' }
 finally { if (sameUser() && current === generation) loading.value = false }
}
const bind = async () => { if (!sameUser() || binding.value || pending.value) return; binding.value = true; bindError.value = ''; try { await request('/user/support/settlement/bind', { ...payee, terms_version: account.value.terms_version }, actionController.signal); if (!sameUser()) return; bindOpen.value = false; clearPayee(); ElMessage.success('收款信息已保存'); await load() } catch (cause) { if (sameUser()) bindError.value = cause?.data?.message || cause.message || '绑定结果未确认，请刷新后核对' } finally { binding.value = false } }
// 先保存原请求，再提交；网络失败和页面重开均复用同一请求键及收款版本。
const withdraw = async () => {
 if (!sameUser() || busy.value) return
 if (!pending.value) { if (!/^\d+(\.\d{1,2})?$/.test(amount.value) || !/[1-9]/.test(amount.value)) { withdrawError.value = '请输入大于0、最多两位小数的金额'; return }; pending.value = { amount: amount.value, account_version: account.value.account_version, request_key: crypto.randomUUID() }; try { localStorage.setItem(storageKey, JSON.stringify(pending.value)) } catch { pending.value = null; withdrawError.value = '无法保存提现请求，请允许本站本地存储后重试'; return } }
 busy.value = true; withdrawError.value = ''
 try { await request('/user/support/withdraw/create', pending.value, actionController.signal); if (!sameUser()) return; pending.value = null; localStorage.removeItem(storageKey); amount.value = ''; ElMessage.success('提现已受理，请查看渠道确认结果'); await load() }
 catch (cause) { if (sameUser()) { withdrawError.value = cause?.data?.message || cause.message || '申请结果尚未确认，请核对原申请'; if (cause.confirmedRejection || (cause?.data?.code >= 400 && cause?.data?.code < 500)) { pending.value = null; localStorage.removeItem(storageKey) } } }
 finally { busy.value = false }
}
const selectTab = value => { if (value !== tab.value) { tab.value = value; page.value = 1; load() } }
const changePage = delta => { page.value += delta; load() }
watch(() => auth.getUser()?.uuid || auth.getUser()?.Uuid || '', () => { generation++; queryController?.abort(); actionController.abort(); clearInterval(poll); account.value = {}; balance.value = {}; rows.value = []; pending.value = null; bindOpen.value = false; clearPayee(); error.value = '账号已切换，请重新打开收益页面' })
onMounted(() => { try { const saved = JSON.parse(localStorage.getItem(storageKey) || 'null'); if (saved?.request_key && typeof saved.amount === 'string' && Number.isInteger(saved.account_version)) { pending.value = saved; amount.value = saved.amount } } catch {} load(); poll = setInterval(() => { if (!document.hidden && !loading.value && sameUser()) load() }, 15000) })
onBeforeUnmount(() => { disposed = true; generation++; queryController?.abort(); actionController.abort(); clearInterval(poll); clearPayee() })
</script>
