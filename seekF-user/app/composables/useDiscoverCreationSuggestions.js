// 图片创作建议仅由用户主动请求，媒体或编辑目标变化时丢弃旧结果。
export function useDiscoverCreationSuggestions(getContext) {
  const titles = ref([])
  const suggestedTags = ref([])
  const loading = ref(false)
  const error = ref('')
  const open = ref(false)
  const apiBase = useRuntimeConfig().public.apiBase
  let generation = 0
  let controller
  const identity = computed(() => {
    const ctx = getContext()
    return JSON.stringify([ctx.mediaType, ctx.media.map(item => [item.uploadedUrl, item.uploadStatus]), ctx.editorKey, ctx.userId])
  })
  const ready = computed(() => {
    const ctx = getContext()
    return ctx.mediaType === 'image' && !ctx.disabled && !!ctx.userId && ctx.media.length > 0 && ctx.media.length <= 9 && ctx.media.every(item => item.uploadStatus === 'done' && item.uploadedUrl)
  })
  const reset = () => {
    generation++
    controller?.abort()
    loading.value = false
    titles.value = []
    suggestedTags.value = []
    error.value = ''
    open.value = false
  }
  const generate = async () => {
    if (!ready.value || loading.value) return
    const version = ++generation
    const ctx = getContext()
    controller = new AbortController()
    loading.value = true
    error.value = ''
    open.value = true
    try {
      const response = await $fetch('/user/discover/ai-suggestions', {
        baseURL: apiBase, method: 'POST', credentials: 'include', retry: 0, timeout: 65000, signal: controller.signal,
        body: { media_type: 0, urls: ctx.media.map(item => item.uploadedUrl), title: ctx.title, content: ctx.content, tags: [...ctx.tags] },
      })
      if (version !== generation) return
      if (response.code !== 200 || !Array.isArray(response.data?.titles) || !Array.isArray(response.data?.tags)) throw new Error(response.message || '建议加载失败，请重试')
      titles.value = response.data.titles
      suggestedTags.value = response.data.tags
      if (!titles.value.length && !suggestedTags.value.length) throw new Error('这次没有可用建议，试试重新生成')
    } catch (err) {
      if (version === generation) error.value = err?.data?.message || err.message || '生成失败，请重试或继续手动填写'
    } finally {
      if (version === generation) loading.value = false
    }
  }
  const toggle = () => {
    if (!ready.value || loading.value) return
    open.value = !open.value
    if (open.value && !titles.value.length && !suggestedTags.value.length) void generate()
  }
  watch(identity, reset, { flush: 'sync' })
  onScopeDispose(reset)
  return { titles, suggestedTags, loading, error, open, ready, generate, toggle }
}
