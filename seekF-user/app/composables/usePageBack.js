// usePageBack 优先返回站内上一页，无可用历史时替换为默认页面。
export const usePageBack = (fallback = '/discover') => {
  const router = useRouter()
  return () => {
    const previous = router.options.history.state.back
    if (typeof previous === 'string' && previous.startsWith('/') && !previous.startsWith('//') && previous !== router.currentRoute.value.fullPath) {
      router.back()
      return
    }
    return navigateTo(typeof fallback === 'function' ? fallback() : fallback, { replace: true })
  }
}
