// useChatUnread 在导航与聊天页面间共享服务端未读快照，不在本地累加实时消息以免重复计数。
export const useChatUnread = () => {
  const user = useState('user', () => null)
  const ownerId = computed(() => user.value?.uuid || '')
  const snapshot = useState('chat-unread-snapshot', () => ({ owner: '', list: [], total: 0, revision: 0 }))
  const requestVersion = useState('chat-unread-request', () => 0)
  const pendingReads = new Map()
  const sessions = computed(() => snapshot.value.owner === ownerId.value ? snapshot.value.list : [])
  const totalUnread = computed(() => snapshot.value.owner === ownerId.value ? snapshot.value.total : 0)
  const unreadRevision = computed(() => snapshot.value.revision)

  // resetUnread 切换账号时清空统计并使旧请求失效。
  const resetUnread = () => {
    requestVersion.value++
    snapshot.value = { owner: ownerId.value, list: [], total: 0, revision: snapshot.value.revision + 1 }
  }

  // refreshUnread 获取同一服务端快照的会话计数和总数，隔离乱序及跨账号响应。
  const refreshUnread = async () => {
    const owner = ownerId.value
    if (!owner) return
    const request = ++requestVersion.value
    try {
      const res = await useApi$('/user/session/unread')
      if (owner !== ownerId.value || request !== requestVersion.value) return
      if (res.code !== 200) throw new Error(res.message || '获取私聊未读数失败')
      snapshot.value = { owner, list: res.data?.list || [], total: res.data?.total || 0, revision: snapshot.value.revision + 1 }
    } catch (error) {
      if (owner === ownerId.value && request === requestVersion.value) console.error('获取私聊未读数失败:', error)
    }
  }

  // unreadFor 返回会话角标，与导航使用同一份快照。
  const unreadFor = sessionId => sessions.value.find(item => item.session_id === sessionId)?.unread_count || 0

  // removeUnreadSession 删除会话后同步移除其角标，不影响其余会话。
  const removeUnreadSession = sessionId => {
    requestVersion.value++
    const list = sessions.value.filter(item => item.session_id !== sessionId)
    snapshot.value = { owner: ownerId.value, list, total: list.reduce((sum, item) => sum + item.unread_count, 0), revision: snapshot.value.revision + 1 }
  }

  // markRead 合并同会话的并发确认，只确认已展示的消息主键，成功后重新获取未读数。
  const markRead = (sessionId, messageId) => {
    const owner = ownerId.value
    const id = String(messageId || '0')
    if (!owner || !sessionId || !/^\d+$/.test(id) || BigInt(id) <= 0n) return Promise.resolve()
    const known = sessions.value.find(item => item.session_id === sessionId)?.last_read_message_id || '0'
    if (BigInt(id) <= BigInt(known)) return Promise.resolve()
    const key = `${owner}:${sessionId}`
    const existing = pendingReads.get(key)
    if (existing) {
      if (BigInt(id) > BigInt(existing.id)) existing.id = id
      return existing.promise
    }
    const job = { id, promise: null }
    pendingReads.set(key, job)
    job.promise = (async () => {
      try {
        let confirmed = '0'
        while (owner === ownerId.value && BigInt(job.id) > BigInt(confirmed)) {
          const boundary = job.id
          requestVersion.value++
          const res = await useApi$('/user/session/read', { body: { session_id: sessionId, message_id: boundary } })
          if (owner !== ownerId.value) return
          if (res.code !== 200) throw new Error(res.message || '确认私聊已读失败')
          confirmed = boundary
          requestVersion.value++
          await refreshUnread()
        }
      } catch (error) {
        if (owner === ownerId.value) console.error('确认私聊已读失败:', error)
      }
    })().finally(() => pendingReads.delete(key))
    return job.promise
  }

  return { ownerId, totalUnread, unreadRevision, sessions, unreadFor, refreshUnread, resetUnread, removeUnreadSession, markRead }
}
