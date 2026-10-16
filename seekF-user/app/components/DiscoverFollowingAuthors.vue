<template>
  <section class="mx-4 mb-2 rounded-2xl border border-gray-100 bg-white/90 px-3 py-4 shadow-[0_4px_24px_rgba(15,23,42,0.025)] sm:px-5" aria-label="关注的作者">
    <div class="mb-3 flex items-center justify-between gap-3 px-1">
      <div class="flex items-center gap-2"><span class="text-sm font-medium text-gray-800">我的关注</span><span class="text-xs text-gray-400">{{ unreadAuthors ? unreadAuthors + ' 位作者有更新' : '不错过每一次分享' }}</span></div>
      <button v-if="error" type="button" class="text-xs text-blue-500 hover:text-blue-600 focus-visible:outline-blue-500" @click="$emit('retry')">加载失败，重试</button>
      <Icon v-else-if="loading" name="uil:spinner" class="animate-spin text-sm text-gray-400" aria-label="刷新作者列表" />
    </div>
    <div class="flex items-start gap-2 overflow-x-auto overscroll-x-contain py-1 [scrollbar-width:thin] sm:gap-3">
      <button type="button" class="group flex w-[4.5rem] shrink-0 flex-col items-center gap-2 rounded-xl py-1 outline-offset-2 focus-visible:outline-blue-500" :aria-pressed="!selected" @click="$emit('select', '')">
        <span class="flex size-14 shrink-0 items-center justify-center rounded-full border transition-all duration-200 group-hover:-translate-y-0.5 motion-reduce:transform-none motion-reduce:transition-none" :class="!selected ? 'border-blue-300 bg-blue-50 text-blue-500' : 'border-gray-100 bg-gray-50 text-gray-400 group-hover:bg-blue-50 group-hover:text-blue-500'"><Icon name="mdi:view-grid-outline" class="text-2xl" /></span>
        <span class="text-xs" :class="!selected ? 'font-medium text-blue-600' : 'text-gray-500'">全部动态</span>
      </button>
      <button v-for="author in authors" :key="author.author_uuid" type="button" class="group flex w-[4.5rem] shrink-0 flex-col items-center gap-2 rounded-xl py-1 outline-offset-2 focus-visible:outline-blue-500" :aria-pressed="selected === author.author_uuid" :aria-label="author.nickname + (author.has_update ? '，有新作品' : '')" @click="$emit('select', author.author_uuid)">
        <span class="relative flex size-14 shrink-0 items-center justify-center rounded-full border-2 transition-all duration-200 group-hover:-translate-y-0.5 motion-reduce:transform-none motion-reduce:transition-none" :class="selected === author.author_uuid ? 'border-blue-400' : 'border-transparent group-hover:border-blue-200'">
          <el-avatar :size="48" shape="circle" :src="author.avatar" class="!size-12 shrink-0 !rounded-full !bg-blue-50 !text-blue-400">{{ author.nickname?.slice(0, 1) }}</el-avatar>
          <Transition enter-active-class="transition duration-200 motion-reduce:transition-none" enter-from-class="scale-0 opacity-0" leave-active-class="transition duration-200 motion-reduce:transition-none" leave-to-class="scale-0 opacity-0"><span v-if="author.has_update" class="absolute right-0.5 top-0.5 size-2.5 rounded-full bg-rose-500 ring-[3px] ring-white" aria-hidden="true" /></Transition>
        </span>
        <span class="w-full truncate px-1 text-center text-xs transition-colors" :class="selected === author.author_uuid ? 'font-medium text-blue-600' : 'text-gray-500 group-hover:text-gray-800'">{{ author.nickname }}</span>
      </button>
      <div v-if="loading && !authors.length" class="flex gap-3" aria-hidden="true"><div v-for="i in 4" :key="i" class="flex w-[4.5rem] shrink-0 flex-col items-center gap-3 py-1 motion-safe:animate-pulse"><span class="size-14 rounded-full bg-gray-100" /><span class="h-2 w-10 rounded-full bg-gray-100" /></div></div>
      <p v-else-if="!authors.length && !error" class="self-center px-3 text-xs leading-6 text-gray-400">关注喜欢的作者<br />在这里查看他们的新作品</p>
    </div>
    <p v-if="readError" class="mt-3 px-1 text-xs text-amber-600" role="status">更新提醒暂未同步，稍后会自动重试</p>
  </section>
</template>

<script setup>
defineProps({ authors: { type: Array, default: () => [] }, selected: { type: String, default: '' }, unreadAuthors: { type: Number, default: 0 }, loading: Boolean, error: Boolean, readError: Boolean })
defineEmits(['select', 'retry'])
</script>
