import { ref, computed, onScopeDispose } from 'vue'
import { normalizeDiscoverPost, mergeDiscoverPage } from '~/utils/discover'

// 首页和搜索页共用分页、失败重试与过期响应保护。
export function useDiscoverFeed(getKeyword, getChannel) {
  const items = ref([])
  const loading = ref(false)
  const error = ref('')
  const noMore = ref(false)
  const page = ref(1)
  const total = ref(0)
  const cursor = ref('')
  const hasFollowing = ref(false)
  const pageSize = 12
  let generation = 0
  let controller = null
  const initialLoading = computed(() => loading.value && items.value.length === 0)

  const loadMore = async () => {
    if (loading.value || noMore.value) return
    const requestGeneration = generation
    const keyword = getKeyword?.() || ''
    if (getKeyword && !keyword.trim()) { noMore.value = true; return }
    controller = new AbortController()
    loading.value = true
    error.value = ''
    try {
      const response = await useApi$(getKeyword ? '/user/discover/search' : '/user/discover/feed', {
        body: getKeyword ? { page: page.value, page_size: pageSize, keyword } : { cursor: cursor.value, page_size: pageSize, source: getChannel?.() === 'following' ? 'following' : 'all', ...(getChannel?.() === 'image' ? { media_type: 0 } : getChannel?.() === 'video' ? { media_type: 1 } : {}) },
        signal: controller.signal,
        retry: 0,
      })
      if (requestGeneration !== generation) return
      if (response.code !== 200 || !response.data || (response.data.list != null && !Array.isArray(response.data.list))) {
        throw new Error(response.message || '获取内容失败，请稍后重试')
      }
      if (!getKeyword && (!Array.isArray(response.data.list) || typeof response.data.has_more !== 'boolean' || typeof response.data.next_cursor !== 'string' || typeof response.data.has_following !== 'boolean' || (response.data.has_more && (!response.data.next_cursor || response.data.next_cursor === cursor.value || !response.data.list?.length)))) {
        throw new Error('获取内容失败，请稍后重试')
      }
      const list = (response.data.list || []).map(normalizeDiscoverPost)
      items.value = mergeDiscoverPage(items.value, list)
      total.value = Number(response.data.total) || 0
      if (getKeyword) {
        page.value++
        noMore.value = list.length === 0 || list.length < pageSize || items.value.length >= total.value
      } else {
        cursor.value = response.data.next_cursor
        noMore.value = !response.data.has_more
        hasFollowing.value = response.data.has_following
      }
    } catch (err) {
      if (requestGeneration !== generation) return
      error.value = err?.data?.message || (err.message === '获取内容失败，请稍后重试' ? err.message : '获取内容失败，请稍后重试')
    } finally {
      if (requestGeneration === generation) loading.value = false
    }
  }

  const reset = () => {
    generation++
    controller?.abort()
    items.value = []
    page.value = 1
    cursor.value = ''
    hasFollowing.value = false
    total.value = 0
    noMore.value = false
    error.value = ''
    loading.value = false
    return loadMore()
  }

  const updateLike = ({ postId, likeCount, isLiked }) => {
    const item = items.value.find(post => post.id === postId)
    if (item) { item.likeCount = likeCount; item.isLiked = isLiked }
  }
  onScopeDispose(() => { generation++; controller?.abort() })
  return { items, loading, initialLoading, error, noMore, total, hasFollowing, loadMore, reset, updateLike }
}
