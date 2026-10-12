import { ref } from 'vue'

// 同一目标请求期间禁用重复操作；失败重试仍发送相同目标状态。
export function useDiscoverLike() {
  const pending = ref(new Set())
  const desiredStates = new Map()
  const setLike = async (targetId, currentState, isComment = false) => {
    const key = (isComment ? 'comment:' : 'post:') + targetId
    if (!targetId || pending.value.has(key)) return null
    const desired = desiredStates.has(key) ? desiredStates.get(key) : !currentState
    desiredStates.set(key, desired)
    pending.value = new Set([...pending.value, key])
    try {
      const response = await useApi$(isComment ? '/user/discover/comment/like/set' : '/user/discover/like/set', {
        body: { target_uuid: targetId, is_liked: desired }, retry: 0,
      })
      if (response.code !== 200) throw new Error(response.message || '点赞操作失败')
      desiredStates.delete(key)
      return response.data
    } catch (err) {
      ElMessage.error(err?.data?.message || '点赞操作失败，请重试')
      return null
    } finally {
      const next = new Set(pending.value)
      next.delete(key)
      pending.value = next
    }
  }
  const isPending = (id, isComment = false) => pending.value.has((isComment ? 'comment:' : 'post:') + id)
  return { setLike, isPending }
}
