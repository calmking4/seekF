<template>
  <div v-if="sources && sources.length > 0" class="mt-2 border border-gray-200 rounded-lg bg-gray-50 overflow-hidden">
    <button
      @click="expanded = !expanded"
      class="w-full flex items-center justify-between px-3 py-2 text-xs text-gray-500 hover:bg-gray-100 transition-colors"
    >
      <span class="flex items-center gap-1">
        <Icon name="uil:search" class="text-sm" />
        已搜索 {{ sources.length }} 个来源
      </span>
      <Icon :name="expanded ? 'uil:angle-up' : 'uil:angle-down'" class="text-sm" />
    </button>
    <div v-show="expanded" class="px-3 pb-2 space-y-1">
      <a
        v-for="(source, idx) in sources"
        :key="idx"
        :href="source.url"
        target="_blank"
        rel="noopener noreferrer"
        class="flex items-start gap-2 p-2 rounded hover:bg-white transition-colors group"
      >
        <span class="flex-shrink-0 w-5 h-5 rounded bg-blue-100 text-blue-600 text-xs flex items-center justify-center font-medium">
          {{ idx + 1 }}
        </span>
        <div class="flex-1 min-w-0">
          <p class="text-xs font-medium text-gray-700 truncate group-hover:text-blue-600">
            {{ source.title }}
          </p>
          <p class="text-xs text-gray-400 truncate">{{ source.url }}</p>
        </div>
        <Icon name="uil:external-link-alt" class="text-xs text-gray-400 flex-shrink-0 mt-0.5" />
      </a>
    </div>
  </div>
</template>

<script setup>
defineProps({
  sources: { type: Array, default: () => [] }
})

const expanded = ref(false)
</script>
