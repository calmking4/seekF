<template>
  <div class="min-h-full bg-gray-50">
    <DiscoverDetail :key="postId" :item="{ id: postId }" @close="closePost" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
const route = useRoute()
const postId = computed(() => String(route.params.id || ''))
const goBack = usePageBack('/discover')
// 作者入口自行跳转用户主页；关闭详情时返回上一页。
const closePost = reason => {
  if (reason !== 'author') goBack()
}
useSeoMeta({ title: '动态详情', description: '查看 seekF 社区动态及评论。' })
</script>
