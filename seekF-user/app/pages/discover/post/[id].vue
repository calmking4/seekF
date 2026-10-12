<template>
  <div class="min-h-full bg-gray-50">
    <DiscoverDetail :key="postId" :item="{ id: postId }" @close="closePost" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
const route = useRoute()
const postId = computed(() => String(route.params.id || ''))
// 作者入口自行跳转用户主页；点击遮罩关闭时返回发现页。
const closePost = reason => {
  if (reason !== 'author') navigateTo('/discover')
}
useSeoMeta({ title: '动态详情', description: '查看 seekF 社区动态及评论。' })
</script>
