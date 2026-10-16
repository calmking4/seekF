<template>
  <div ref="container" class="flex gap-4">
    <div v-for="(column, index) in columns" :key="index" class="flex-1 min-w-0 flex flex-col space-y-4">
      <DiscoverPostCard v-for="post in column" :key="post.id" :post="post" :feedback-enabled="feedbackEnabled" :source="source" @open="$emit('open', $event)" @like-updated="$emit('like-updated', $event)" @feedback="$emit('feedback', $event)" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { discoverAspectRatio } from '~/utils/discover'
const props = defineProps({ posts: { type: Array, default: () => [] }, feedbackEnabled: { type: Boolean, default: false }, source: { type: String, default: 'profile' } })
defineEmits(['open', 'like-updated', 'feedback'])
const container = ref(null)
const width = ref(0)
const viewportWidth = ref(0)
const visibleIds = new Set()
const observedElements = new Map()
let resizeObserver
let animationObserver
let disposed = false

// 保留原页面的列数断点，媒体高度仍按阶段一保存的真实比例计算。
const columns = computed(() => {
  const count = viewportWidth.value >= 1024 ? 4 : viewportWidth.value >= 768 ? 3 : viewportWidth.value >= 640 ? 2 : 1
  const result = Array.from({ length: count }, () => [])
  const heights = Array(count).fill(0)
  const cardWidth = Math.max(1, (width.value - (count - 1) * 16) / count)
  for (const post of props.posts) {
    const index = heights.indexOf(Math.min(...heights))
    result[index].push(post)
    heights[index] += cardWidth / discoverAspectRatio(post) + 112
  }
  return result
})

// 沿用原先 50ms 错开、进入视区后渐显上移的入场逻辑；重排不重播旧帖子。
const observeNewItems = () => {
  if (!container.value || disposed) return
  const elements = [...container.value.querySelectorAll('.fade-in')]
  const currentElements = new Set(elements)
  for (const [id, element] of observedElements) {
    if (!currentElements.has(element)) { animationObserver?.unobserve(element); observedElements.delete(id) }
  }
  let delayIndex = 0
  for (const element of elements) {
    const id = element.dataset.id
    if (visibleIds.has(id)) {
      element.classList.add('visible')
      element.style.transitionDelay = '0ms'
      continue
    }
    if (observedElements.get(id) === element) continue
    element.style.transitionDelay = (delayIndex++ * 50) + 'ms'
    if (!animationObserver) { element.classList.add('visible'); visibleIds.add(id); continue }
    observedElements.set(id, element)
    animationObserver.observe(element)
  }
}

const handleResize = () => {
  viewportWidth.value = window.innerWidth
  width.value = container.value?.clientWidth || 0
}
watch(columns, async () => { await nextTick(); observeNewItems() }, { flush: 'post' })
onMounted(async () => {
  if (typeof IntersectionObserver !== 'undefined') {
    animationObserver = new IntersectionObserver(entries => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue
        entry.target.classList.add('visible')
        visibleIds.add(entry.target.dataset.id)
        animationObserver.unobserve(entry.target)
        observedElements.delete(entry.target.dataset.id)
      }
    }, { threshold: 0.1 })
  }
  handleResize()
  resizeObserver = new ResizeObserver(handleResize)
  if (container.value) resizeObserver.observe(container.value)
  window.addEventListener('resize', handleResize)
  await nextTick()
  observeNewItems()
})
onUnmounted(() => {
  disposed = true
  resizeObserver?.disconnect()
  animationObserver?.disconnect()
  observedElements.clear()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
/* 恢复原发现页的动画参数，组件抽取不能删除或替换这些效果。 */
.fade-in {
  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.fade-in.visible {
  opacity: 1;
  transform: translateY(0);
}
</style>
