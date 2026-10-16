// 作者提醒独立刷新；请求只在页面可见时发出，失败保留上次结果。
export function useFollowingUpdates() {
  const authors = ref([])
  const unreadAuthors = ref(0)
  const loading = ref(false)
  const error = ref(false)
  const readError = ref(false)
  const selected = ref('')
  const user = useCookie('user')
  const apiBase = useRuntimeConfig().public.apiBase
  const pending = new Map()
  let selectionToken = ''
  let generation = 0
  let active = false
  let timer
  let controller
  let reading = false
  let revision = 0
  const request = (path, body, signal) => $fetch('/user/discover/following/' + path, { baseURL: apiBase, method: 'POST', credentials: 'include', body, signal, retry: 0 })

  const refresh = async () => {
    if (!active || !user.value || document.hidden || loading.value) return
    const version = generation
    const requestRevision = revision
    controller = new AbortController()
    loading.value = true
    try {
      const response = await request('updates', {}, controller.signal)
      if (version !== generation) return
      if (requestRevision !== revision) return
      if (response.code !== 200 || !Array.isArray(response.data?.authors) || !Number.isInteger(response.data?.unread_authors)) throw new Error('关注更新加载失败')
      // 维持当前作者顺序，红点消失时不让头像在用户点击下跳动。
      const order = new Map(authors.value.map((author, index) => [author.author_uuid, index]))
      authors.value = response.data.authors.sort((a, b) => (order.get(a.author_uuid) ?? Infinity) - (order.get(b.author_uuid) ?? Infinity))
      for (const [token, authorId] of pending) {
        if (!authors.value.some(author => author.author_uuid === authorId)) pending.delete(token)
      }
      if (!pending.size) readError.value = false
      unreadAuthors.value = response.data.unread_authors
      error.value = false
      if (selected.value && !authors.value.some(author => author.author_uuid === selected.value)) select('')
    } catch {
      if (version === generation) error.value = true
    } finally {
      if (version === generation) {
        loading.value = false
        if (requestRevision !== revision) void refresh()
      }
    }
  }

  const flushReads = async () => {
    if (reading || !active || document.hidden) return
    const version = generation
    reading = true
    try {
      for (const [token, authorId] of pending) {
        let response
        try {
          response = await request('read', { token })
        } catch (err) {
          if (version !== generation) return
          // 取消后重新关注或服务重启导致快照失效，丢弃旧确认并重新获取状态。
          if ((err?.statusCode || err?.status) === 410) { pending.delete(token); revision++; continue }
          throw err
        }
        if (version !== generation) return
        if (response.code !== 200) throw new Error('查看状态同步失败')
        pending.delete(token)
        revision++
        const author = authors.value.find(item => item.author_uuid === authorId)
        // 若已刷新到更新的快照，不能用旧确认清掉新的红点。
        if (author?.read_token === token) {
          author.has_update = false
          author.read_token = ''
          unreadAuthors.value = Math.max(0, unreadAuthors.value - 1)
        }
      }
      readError.value = false
      await refresh()
    } catch {
      if (version === generation) readError.value = true
    } finally { reading = false }
  }

  const select = authorId => {
    selected.value = authorId
    selectionToken = authors.value.find(author => author.author_uuid === authorId)?.read_token || ''
  }
  const loaded = authorId => {
    if (!authorId || authorId !== selected.value || !selectionToken) return
    pending.set(selectionToken, authorId)
    selectionToken = ''
    void flushReads()
  }
  const tick = () => { if (!document.hidden) { void refresh(); if (pending.size) void flushReads() } }
  const start = () => {
    if (active) return
    active = true
    document.addEventListener('visibilitychange', tick)
    window.addEventListener('focus', tick)
    timer = window.setInterval(tick, 30000)
    tick()
  }
  const stop = () => {
    active = false
    generation++
    controller?.abort()
    loading.value = false
    clearInterval(timer)
    document.removeEventListener('visibilitychange', tick)
    window.removeEventListener('focus', tick)
  }
  watch(user, () => {
    generation++
    controller?.abort()
    authors.value = []
    unreadAuthors.value = 0
    selected.value = ''
    selectionToken = ''
    pending.clear()
    error.value = false
    readError.value = false
    loading.value = false
    if (active) tick()
  })
  onMounted(start)
  onActivated(start)
  onDeactivated(stop)
  onUnmounted(stop)
  return { authors, unreadAuthors, loading, error, readError, selected, refresh, select, loaded }
}
