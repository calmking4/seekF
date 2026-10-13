<template>
  <div ref="layer" aria-hidden="true" class="pointer-events-none absolute inset-0">
    <svg class="absolute h-0 w-0" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <filter :id="filterId" x="-50%" y="-100%" width="200%" height="300%" color-interpolation-filters="sRGB">
          <feGaussianBlur in="SourceGraphic" stdDeviation="4" result="blur" />
          <feColorMatrix in="blur" mode="matrix" values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 20 -9" />
        </filter>
      </defs>
    </svg>
    <span v-if="position" class="absolute" :class="fillClass" :style="[positionStyle, { opacity: flowing ? 0 : 1 }]" />
    <!-- 滤镜仅作用于高亮，让两团水滴拉伸融合，文字始终保持清晰。 -->
    <div v-if="position" class="absolute inset-0" :style="{ filter: 'url(#' + filterId + ')', opacity: flowing ? 1 : 0 }">
      <span ref="tail" class="absolute" :class="fillClass" :style="positionStyle" />
      <span ref="head" class="absolute" :class="fillClass" :style="positionStyle" />
    </div>
  </div>
</template>
<script setup>
const props = defineProps({ selected: { type: [String, Number], required: true }, fillClass: { type: String, default: 'bg-gray-100' } })
const filterId = 'liquid-selection-' + useId().replace(/[^a-zA-Z0-9_-]/g, '')
const layer = ref(null)
const head = ref(null)
const tail = ref(null)
const position = ref(null)
const flowing = ref(false)
const positionStyle = computed(() => position.value ? {
  left: position.value.x + 'px', top: position.value.y + 'px',
  width: position.value.width + 'px', height: position.value.height + 'px', borderRadius: position.value.radius,
} : {})
let observer
let animations = []
let generation = 0

const cancelFlow = () => { generation++; animations.forEach(animation => animation.cancel()); animations = []; flowing.value = false }
const measure = () => {
  const container = layer.value?.parentElement
  const target = Array.from(container?.querySelectorAll('[data-liquid-key]') || []).find(element => element.dataset.liquidKey === String(props.selected))
  if (!target) return null
  const bounds = target.getBoundingClientRect()
  const origin = layer.value.getBoundingClientRect()
  return { x: bounds.left - origin.left, y: bounds.top - origin.top, width: bounds.width, height: bounds.height, radius: getComputedStyle(target).borderRadius }
}
const move = async (animate = true) => {
  const destination = measure()
  if (!destination) return
  // 连续点击时从正在运动的水滴继续，避免跳回上一个按钮。
  const origin = layer.value.getBoundingClientRect()
  const current = flowing.value && head.value ? head.value.getBoundingClientRect() : null
  const previous = current ? { ...position.value, x: current.left - origin.left, y: current.top - origin.top, width: current.width, height: current.height } : position.value
  cancelFlow()
  position.value = destination
  if (!animate || !previous || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  const token = generation
  flowing.value = true
  await nextTick()
  if (token !== generation || !head.value || !tail.value) return
  const frame = point => ({ left: point.x + 'px', top: point.y + 'px', width: point.width + 'px', height: point.height + 'px', borderRadius: point.radius })
  const middle = progress => ({ ...destination, x: previous.x + (destination.x - previous.x) * progress, y: previous.y + (destination.y - previous.y) * progress, width: previous.width + (destination.width - previous.width) * progress, height: previous.height + (destination.height - previous.height) * progress })
  animations = [
    head.value.animate([{ ...frame(previous), offset: 0 }, { ...frame(middle(0.88)), offset: 0.55 }, { ...frame(destination), offset: 1 }], { duration: 520, easing: 'cubic-bezier(.22,.75,.25,1)' }),
    tail.value.animate([{ ...frame(previous), offset: 0 }, { ...frame(middle(0.12)), offset: 0.35 }, { ...frame(destination), offset: 1 }], { duration: 520, easing: 'cubic-bezier(.45,0,.25,1)' }),
  ]
  await Promise.all(animations.map(animation => animation.finished.catch(() => {})))
  if (token === generation) cancelFlow()
}
watch(() => props.selected, () => { void move() }, { flush: 'post' })
onMounted(() => {
  void move(false)
  observer = new ResizeObserver(() => {
    const next = measure()
    // 字重或计数变化会触发观察器，相同几何位置不打断正在进行的流动。
    if (next && position.value && ['x', 'y', 'width', 'height', 'radius'].every(key => next[key] === position.value[key])) return
    void move(false)
  })
  observer.observe(layer.value.parentElement)
  layer.value.parentElement.querySelectorAll('[data-liquid-key]').forEach(element => observer.observe(element))
})
onBeforeUnmount(() => { observer?.disconnect(); cancelFlow() })
</script>
