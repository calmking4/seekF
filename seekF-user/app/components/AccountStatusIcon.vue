<template>
  <!-- 本地 SVG 避免按钮依赖图标接口，转圈只代表真实请求。 -->
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" class="h-[1em] w-[1em] shrink-0" :class="{ 'animate-spin': loading }" aria-hidden="true" focusable="false">
    <template v-if="loading"><circle cx="12" cy="12" r="9" opacity=".2" /><path d="M12 3a9 9 0 0 1 9 9" /></template>
    <template v-else-if="mode === 'retry'"><path d="M20 7v5h-5M4 17v-5h5M6.1 6.1A8 8 0 0 1 20 12M4 12a8 8 0 0 0 13.9 5.9" /></template>
    <template v-else>
      <circle cx="9" cy="7" r="3.5" /><path d="M3 21v-3a6 6 0 0 1 9-5.2" />
      <path v-if="mode === 'unblock'" d="m14 17 3 3 5-6" />
      <template v-else><circle cx="17.5" cy="17.5" r="4.5" /><path d="m14.3 14.3 6.4 6.4" /></template>
    </template>
  </svg>
</template>

<script setup>
defineProps({
  loading: { type: Boolean, default: false },
  mode: { type: String, default: 'block' },
})
</script>
