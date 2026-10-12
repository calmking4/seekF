<template>
  <div class="min-h-screen bg-gray-50">
    <DiscoverHeader :initial-keyword="keyword" />
    <DiscoverFeed search :keyword="keyword" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
const route = useRoute()
const keyword = computed(() => typeof route.query.q === 'string' ? route.query.q.trim() : '')
useSeoMeta({ title: '搜索发现', description: '搜索 seekF 社区动态和话题。' })
</script>
