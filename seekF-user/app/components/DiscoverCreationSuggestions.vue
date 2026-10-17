<template>
  <div v-if="mode === 'titles'" ref="container" class="relative mb-2 flex items-center justify-between gap-3" @keydown.esc.stop="$emit('close')">
    <span class="text-xs text-gray-400">{{ loading ? '正在根据图片寻找灵感…' : '让图片带来一点创作灵感' }}</span>
    <button ref="trigger" type="button" class="inline-flex shrink-0 items-center gap-1.5 rounded-full bg-gray-100 px-3.5 py-2 text-xs text-gray-600 transition-colors hover:bg-gray-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 disabled:cursor-not-allowed disabled:opacity-50 motion-reduce:transition-none" :disabled="!ready || loading" :aria-expanded="open" :aria-controls="panelId" :title="!ready ? '图片全部上传完成后可生成标题和标签' : '根据图片推荐标题和标签'" @click="$emit('toggle')">
      <Icon :name="loading ? 'uil:spinner' : 'mdi:auto-fix'" :class="loading ? 'animate-spin' : 'text-blue-400'" />{{ loading ? '生成中' : '智能标题' }}<Icon name="mdi:chevron-down" class="transition-transform duration-200 motion-reduce:transition-none" :class="open ? 'rotate-180' : ''" />
    </button>
    <Transition enter-active-class="transition-[opacity,transform] duration-200 ease-out motion-reduce:transition-none" enter-from-class="-translate-y-1 opacity-0" leave-active-class="transition-[opacity,transform] duration-150 motion-reduce:transition-none" leave-to-class="-translate-y-1 opacity-0">
      <section v-if="open" :id="panelId" class="absolute right-0 top-full z-30 mt-2 w-full overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-[0_8px_28px_rgba(0,0,0,0.08)] sm:max-w-md" aria-label="推荐标题" :aria-busy="loading">
        <header class="flex items-center justify-between gap-2 border-b border-gray-100 px-4 py-3"><span class="text-xs font-medium text-gray-700">{{ loading ? '正在构思标题与标签' : '选一句，作为你的标题' }}</span><div class="flex items-center gap-2"><button type="button" class="inline-flex items-center gap-1 rounded px-1 py-0.5 text-xs text-gray-400 transition-colors hover:text-blue-500 disabled:opacity-40" :disabled="loading || !ready" @click="$emit('generate')"><Icon name="mdi:refresh" />换一组</button><button type="button" class="flex size-6 items-center justify-center rounded-full text-gray-400 hover:bg-gray-100 focus-visible:outline-blue-400" aria-label="关闭标题建议" @click="$emit('close')"><Icon name="mdi:close" /></button></div></header>
        <div v-if="loading" class="space-y-4 px-4 py-5" role="status"><div v-for="i in 3" :key="i" class="h-3 rounded bg-gray-100 motion-safe:animate-pulse" :class="i === 2 ? 'w-3/4' : 'w-11/12'" /><p class="text-xs text-gray-400">图片越丰富，灵感越具体，请稍候</p></div>
        <div v-else>
          <p v-if="error" class="px-4 py-4 text-xs leading-6 text-gray-500" role="alert">{{ error }}<button type="button" class="ml-2 text-blue-500 hover:text-blue-600 disabled:opacity-40" :disabled="!ready" @click="$emit('generate')">重新生成</button></p>
          <div v-if="titles.length" class="max-h-72 overflow-y-auto p-1.5"><button v-for="(suggestion, index) in titles" :key="suggestion" type="button" class="group flex w-full items-start gap-3 rounded-xl px-3 py-3 text-left text-sm leading-6 text-gray-700 transition-colors hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 disabled:opacity-40" :disabled="!ready" @click="$emit('select-title', suggestion)"><span class="mt-0.5 text-[10px] tabular-nums text-gray-300">{{ String(index + 1).padStart(2, '0') }}</span><span class="min-w-0 flex-1 break-words">{{ suggestion }}</span><Icon name="mdi:arrow-top-left" class="mt-1 shrink-0 text-gray-300 transition-colors group-hover:text-blue-400" /></button></div>
          <p v-else-if="!error" class="px-4 py-5 text-xs text-gray-400">暂无标题建议，可换一组或继续手动填写</p>
        </div>
        <p class="border-t border-gray-100 px-4 py-2.5 text-[11px] leading-5 text-gray-400">点击才会填入标题，推荐标签在下方选择添加</p>
      </section>
    </Transition>
  </div>
  <div v-else-if="suggestedTags.length" class="mb-3" aria-label="推荐标签">
    <div class="mb-2 flex items-center gap-1.5 text-xs text-gray-400"><Icon name="mdi:sparkles" class="text-blue-400" />推荐标签<span class="ml-1 text-[11px]">点击添加</span></div>
    <TransitionGroup tag="div" class="flex flex-wrap gap-2" enter-active-class="transition-[opacity,transform] duration-200 motion-reduce:transition-none" enter-from-class="translate-y-1 opacity-0" leave-active-class="transition-opacity duration-150 motion-reduce:transition-none" leave-to-class="opacity-0">
      <button v-for="tag in suggestedTags" :key="tag" type="button" class="inline-flex max-w-full items-center gap-1 rounded-full border px-3 py-1.5 text-xs transition-colors duration-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400 disabled:cursor-default motion-reduce:transition-none" :class="selectedTags.includes(tag) ? 'border-blue-100 bg-blue-50/60 text-blue-500' : 'border-transparent bg-gray-100 text-gray-500 hover:bg-gray-200 disabled:opacity-40'" :disabled="!ready || selectedTags.includes(tag) || selectedTags.length >= 10" :aria-pressed="selectedTags.includes(tag)" @click="$emit('select-tag', tag)"><span class="truncate">#{{ tag }}</span><Icon v-if="selectedTags.includes(tag)" name="mdi:check" class="shrink-0 text-xs" /></button>
    </TransitionGroup>
  </div>
</template>

<script setup>
const props = defineProps({ mode: { type: String, default: 'titles' }, titles: { type: Array, default: () => [] }, suggestedTags: { type: Array, default: () => [] }, selectedTags: { type: Array, default: () => [] }, ready: Boolean, open: Boolean, loading: Boolean, error: { type: String, default: '' } })
const emit = defineEmits(['toggle', 'close', 'generate', 'select-title', 'select-tag'])
const container = ref(null)
const trigger = ref(null)
const panelId = useId()
// 关闭或采纳后恢复面板内的键盘焦点，点击外部输入时不抢回焦点。
watch(() => props.open, (value, previous) => {
  if (value || !previous || !container.value?.contains(document.activeElement)) return
  nextTick(() => { if (document.activeElement === document.body || container.value?.contains(document.activeElement)) trigger.value?.focus() })
})
const closeOutside = event => { if (props.mode === 'titles' && props.open && !container.value?.contains(event.target)) emit('close') }
onMounted(() => { if (props.mode === 'titles') document.addEventListener('pointerdown', closeOutside) })
onUnmounted(() => document.removeEventListener('pointerdown', closeOutside))
</script>
