// 支付状态只读取本站确认结果，关闭页面或切换账号后停止查询并丢弃旧响应。
export function useSupportPayment(onPaid) {
  const auth = useAuthState()
  const owner = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
  const order = ref(null), error = ref(''), checking = ref(false), pollingEnded = ref(false)
  const now = ref(Date.now())
  const expired = computed(() => !!order.value && new Date(order.value.expires_at).getTime() <= now.value)
  let disposed = false, timer, clock, deadline = 0, notified = ''
  const controller = new AbortController()
  const sameUser = () => !disposed && !controller.signal.aborted && !!owner && owner === (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')
  const request = async (path, body = {}) => {
    if (!sameUser()) throw new Error('账号已切换，请重新打开页面')
    const result = await useApi$(path, { body, signal: controller.signal, retry: 0 })
    if (!sameUser()) throw new Error('页面状态已更新')
    if (result.code !== 200) throw new Error(result.message || '助力请求失败')
    return result.data
  }
  const showError = cause => {
    if (sameUser() && cause?.name !== 'AbortError') error.value = cause?.data?.message || cause.message || '助力请求失败'
  }
  const schedule = () => {
    clearTimeout(timer)
    pollingEnded.value = Date.now() >= deadline
    if (!sameUser() || document.hidden || order.value?.status !== 0 || pollingEnded.value) return
    timer = setTimeout(refresh, 3000)
  }
  const acceptOrder = row => {
    if (!sameUser()) return
    // 手动刷新与轮询可能交错返回，已确认付款不能被较早的待支付响应覆盖。
    if (order.value?.order_no === row.order_no && ((order.value.status === 1 && row.status !== 1) || (order.value.status === 2 && row.status === 0))) return
    if (order.value?.order_no !== row.order_no) deadline = Date.now() + 15 * 60 * 1000
    order.value = row
    if (row.status === 1 && notified !== row.order_no) {
      notified = row.order_no
      try { localStorage.setItem('seekf-support-confirmed', JSON.stringify({ user_id: owner, creator_id: row.creator_id, at: Date.now() })) } catch { /* 存储受限时仍可通过窗口焦点重新检查作品权限。 */ }
      onPaid?.(row)
    }
    schedule()
  }
  const refresh = async () => {
    if (!order.value || checking.value || !sameUser()) return
    checking.value = true
    const no = order.value.order_no
    try {
      const row = await request('/user/support/order/info', { order_no: no })
      if (order.value?.order_no !== no) return
      error.value = ''
      acceptOrder(row)
    } catch (cause) { showError(cause) }
    finally { checking.value = false; schedule() }
  }
  const visibility = () => { if (document.hidden) clearTimeout(timer); else { now.value = Date.now(); schedule() } }
  watch(() => auth.getUser()?.uuid || auth.getUser()?.Uuid || '', () => {
    controller.abort(); clearTimeout(timer); clearInterval(clock); order.value = null
    error.value = '账号已切换，请重新打开页面'
  })
  onMounted(() => {
    clock = setInterval(() => { now.value = Date.now() }, 1000)
    document.addEventListener('visibilitychange', visibility)
  })
  onBeforeUnmount(() => {
    disposed = true; controller.abort(); clearTimeout(timer); clearInterval(clock)
    document.removeEventListener('visibilitychange', visibility)
  })
  return { order, error, checking, pollingEnded, expired, now, owner, sameUser, request, showError, acceptOrder, refresh }
}
