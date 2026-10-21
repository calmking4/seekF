<template>
  <button
    v-if="visible"
    type="button"
    :aria-label="label"
    :title="label"
    class="z-20 flex h-11 w-11 items-center justify-center rounded-full border border-blue-100 bg-white text-blue-500 shadow-lg transition-colors hover:border-blue-300 hover:bg-blue-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-400"
    @click="scrollToEdge"
  >
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="h-5 w-5" aria-hidden="true">
      <path v-if="direction === 'top'" d="M5 4h14M12 20V8m-5 5 5-5 5 5" />
      <path v-else d="M5 20h14M12 4v12m-5-5 5 5 5-5" />
    </svg>
  </button>
</template>

<script setup>
const props = defineProps({
  target: { type: Function, required: true },
  direction: { type: String, default: 'top' },
  threshold: { type: Number, default: 400 }
})
const visible = ref(false)
const label = computed(() => props.direction === 'top' ? '回到顶部' : '回到底部')
let scrollTarget
let resizeObserver
let stopWatching

// updateVisibility 根据实际滚动容器与目标边缘的距离决定按钮是否显示。
const updateVisibility = () => {
  if (!scrollTarget) { visible.value = false; return }
  const distance = props.direction === 'top'
    ? scrollTarget.scrollTop
    : scrollTarget.scrollHeight - scrollTarget.clientHeight - scrollTarget.scrollTop
  visible.value = distance > props.threshold
}

// detachTarget 释放旧滚动容器的监听，避免切换会话或页面后残留。
const detachTarget = () => {
  scrollTarget?.removeEventListener('scroll', updateVisibility)
  resizeObserver?.disconnect()
  scrollTarget = null
}

// scrollToEdge 只滚动指定内容区，并遵循系统减少动态效果的设置。
const scrollToEdge = () => {
  if (!scrollTarget) return
  scrollTarget.scrollTo({
    top: props.direction === 'top' ? 0 : Math.max(0, scrollTarget.scrollHeight - scrollTarget.clientHeight),
    behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth'
  })
}

onMounted(() => {
  stopWatching = watch(() => props.target(), target => {
    detachTarget()
    scrollTarget = target
    if (scrollTarget) {
      scrollTarget.addEventListener('scroll', updateVisibility, { passive: true })
      // 同时观察内容和视口，覆盖图片加载、历史消息追加及输入区尺寸变化。
      resizeObserver = new ResizeObserver(updateVisibility)
      resizeObserver.observe(scrollTarget)
      if (scrollTarget.firstElementChild) resizeObserver.observe(scrollTarget.firstElementChild)
    }
    updateVisibility()
  }, { immediate: true, flush: 'post' })
})
onBeforeUnmount(() => { stopWatching?.(); detachTarget() })
</script>
