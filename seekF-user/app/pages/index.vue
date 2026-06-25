<template>
  <el-button @click="ElMessage('hello')">button</el-button>
  <ElButton :icon="ElIconEditPen" type="success">button</ElButton>
  <LazyElButton type="warning">lazy button</LazyElButton>
  <Icon name="uil:github" style="color: black" />
</template>

<script setup>
// 首页 SEO（不设置 title，使用全局默认）
useSeoMeta({
  description: 'seekF 是一个集成了 AI 能力的即时通讯平台，支持智能对话、知识库管理、社交发现等功能。',
})
</script>

<style lang="scss" scoped>

</style>