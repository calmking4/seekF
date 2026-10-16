import { ref } from 'vue'

// 三点菜单反馈成功后才通知列表移除，失败保留内容以便重试。
export function useDiscoverFeedback() {
  const pending = ref(false)
  const setFeedback = async (postId, targetType, active = true) => {
    if (pending.value) return null
    pending.value = true
    try {
      const response = await useApi$('/user/discover/feedback', { body: { post_uuid: postId, target_type: targetType, active }, retry: 0 })
      if (response.code !== 200 || !response.data) throw new Error('保存不感兴趣失败')
      return response.data
    } catch {
      ElMessage.error('保存不感兴趣失败，请重试')
      return null
    } finally {
      pending.value = false
    }
  }
  return { pending, setFeedback }
}

// 点击只由卡片的主动打开触发，不阻塞详情，也不因统计失败打断浏览。
export function useDiscoverClick() {
  const apiBase = useRuntimeConfig().public.apiBase
  const pending = new Set()
  const recordClick = async (postId, source) => {
    if (pending.has(postId)) return
    pending.add(postId)
    try {
      await $fetch('/user/discover/click', { baseURL: apiBase, method: 'POST', credentials: 'include', body: { post_uuid: postId, source }, retry: 0, timeout: 5000 })
    } catch {
      // 统计失败不伪造成功记录，下一次主动打开时仍可重试。
    } finally {
      pending.delete(postId)
    }
  }
  return { recordClick }
}
