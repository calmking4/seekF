<template>
  <div class="min-h-screen bg-gray-50">
    <DiscoverHeader v-model:channel="channel" immersive />
    <DiscoverFeed :key="channel" :channel="channel" />
  </div>
</template>

<script setup>
const channel = ref('recommended')
useSeoMeta({ title: '发现', description: '发现有趣的人和内容，探索 seekF 社区的精彩动态。' })
</script>
