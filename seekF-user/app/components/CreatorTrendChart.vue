<template>
  <div class="rounded-2xl border border-gray-100 p-4 sm:p-6">
    <div class="mb-5 flex min-h-7 flex-wrap items-center justify-between gap-2">
      <h3 class="text-sm font-medium text-gray-800 sm:text-base">{{ title }}</h3>
      <p class="text-xs text-gray-500"><span class="mr-2 inline-block h-0.5 w-4 align-middle bg-[#60a5fa]" />{{ selected ? `${selected.date} · ${displayValue(selected)}` : unit === '元' ? '创作者净收益（元）' : '帖子点击量（次）' }}</p>
    </div>
    <div class="w-full overflow-x-auto">
      <svg viewBox="0 0 1000 290" class="block min-w-[480px] w-full" role="group" :aria-label="title" @mouseleave="selectedIndex = null">
        <defs><linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#60a5fa" stop-opacity="0.18" /><stop offset="100%" stop-color="#60a5fa" stop-opacity="0.01" /></linearGradient></defs>
        <g v-for="tick in ticks" :key="tick.value">
          <line x1="65" x2="975" :y1="tick.y" :y2="tick.y" stroke="#e5e7eb" stroke-dasharray="4 5" />
          <text x="53" :y="tick.y + 4" text-anchor="end" fill="#9ca3af" font-size="12">{{ unit === '元' ? tick.value.toFixed(2) : tick.value }}</text>
        </g>
        <path :d="areaPath" :fill="`url(#${gradientId})`" />
        <path :d="linePath" fill="none" stroke="#60a5fa" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" />
        <g v-for="(point, index) in points" :key="point.date">
          <text v-if="showDate(index)" :x="point.x" y="278" :text-anchor="index === 0 ? 'start' : index === points.length - 1 ? 'end' : 'middle'" fill="#9ca3af" font-size="12">{{ point.date.slice(5).replace('-', '/') }}</text>
          <rect :x="point.x - 455 / Math.max(1, points.length - 1)" y="20" :width="910 / Math.max(1, points.length - 1)" height="230" fill="transparent" @mouseenter="selectedIndex = index" @click="selectedIndex = index" />
          <circle :cx="point.x" :cy="point.y" r="5" fill="white" stroke="#60a5fa" stroke-width="2" :opacity="selectedIndex === index ? 1 : 0" tabindex="0" :aria-label="`${point.date}，${displayValue(point)}`" class="outline-none focus:opacity-100" @focus="selectedIndex = index" @blur="selectedIndex = null"><title>{{ point.date }}：{{ displayValue(point) }}</title></circle>
        </g>
        <line v-if="selected" :x1="selected.x" :x2="selected.x" y1="20" y2="250" stroke="#93c5fd" stroke-dasharray="4 5" pointer-events="none" />
      </svg>
    </div>
    <details class="mt-4 border-t border-gray-50 pt-3 text-xs text-gray-500">
      <summary class="w-fit cursor-pointer rounded hover:text-blue-500 focus-visible:outline-blue-400">查看每日数据</summary>
      <div class="mt-3 max-h-64 overflow-auto"><table class="w-full text-left"><caption class="sr-only">{{ title }}每日明细</caption><thead class="sticky top-0 bg-white"><tr><th scope="col" class="py-2 font-medium">日期</th><th scope="col" class="py-2 text-right font-medium">{{ unit === '元' ? '净收益（元）' : '点击量（次）' }}</th></tr></thead><tbody><tr v-for="row in rows" :key="row.date" class="border-t border-gray-50"><td class="py-2">{{ row.date }}</td><td class="py-2 text-right tabular-nums">{{ row[field] }}</td></tr></tbody></table></div>
    </details>
  </div>
</template>

<script setup>
const props = defineProps({ title: { type: String, required: true }, rows: { type: Array, default: () => [] }, field: { type: String, required: true }, unit: { type: String, default: '次' } })
const gradientId = `creator-trend-${useId().replace(/[^a-zA-Z0-9-]/g, '')}`
const selectedIndex = ref(null)
const values = computed(() => props.rows.map(row => Number(row[props.field]) || 0))
const bounds = computed(() => {
  const min = Math.min(0, ...values.value)
  const max = Math.max(0, ...values.value)
  const step = props.unit === '元' ? Math.max(0.01, (max - min) / 4) : Math.max(1, Math.ceil((max - min) / 4))
  return { min: Math.floor(min / step) * step, max: Math.max(step * 4, Math.ceil(max / step) * step) }
})
const y = value => 250 - (value - bounds.value.min) / (bounds.value.max - bounds.value.min) * 230
const ticks = computed(() => Array.from({ length: 5 }, (_, index) => {
  const value = bounds.value.min + (bounds.value.max - bounds.value.min) * index / 4
  return { value, y: y(value) }
}))
const targetPoints = computed(() => props.rows.map((row, index) => ({ ...row, x: 65 + index * 910 / Math.max(1, props.rows.length - 1), y: y(values.value[index]) })))
const points = ref([])
const baseline = ref(y(0))
const { animate } = useCreatorAnimation()
// 天数改变时按横坐标采样当前曲线，使7/30/90个点之间也能连续过渡。
const sampleY = (previous, x, fallback) => {
  if (!previous.length) return fallback
  const rightIndex = previous.findIndex(point => point.x >= x)
  if (rightIndex <= 0) return (rightIndex === 0 ? previous[0] : previous[previous.length - 1]).y
  const left = previous[rightIndex - 1], right = previous[rightIndex]
  return left.y + (right.y - left.y) * (x - left.x) / (right.x - left.x)
}
watch(targetPoints, target => {
  selectedIndex.value = null
  const previous = points.value
  const previousBaseline = baseline.value
  const nextBaseline = y(0)
  const from = target.map(point => sampleY(previous, point.x, previousBaseline))
  animate(progress => {
    points.value = target.map((point, index) => ({ ...point, y: from[index] + (point.y - from[index]) * progress }))
    baseline.value = previousBaseline + (nextBaseline - previousBaseline) * progress
  })
}, { immediate: true })
const linePath = computed(() => points.value.map((point, index) => `${index ? 'L' : 'M'}${point.x},${point.y}`).join(' '))
const areaPath = computed(() => points.value.length ? `${linePath.value} L975,${baseline.value} L65,${baseline.value} Z` : '')
const selected = computed(() => selectedIndex.value == null ? null : points.value[selectedIndex.value])
const displayValue = row => `${row[props.field]} ${props.unit}`
const showDate = index => index === 0 || index === points.value.length - 1 || index % Math.max(1, Math.ceil((points.value.length - 1) / 5)) === 0
</script>
