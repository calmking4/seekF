<template>
  <el-button @click="ElMessage('hello')">button</el-button>
  <ElButton :icon="ElIconEditPen" type="success">button</ElButton>
  <LazyElButton type="warning">lazy button</LazyElButton>
  <Icon name="uil:github" style="color: black" />
</template>

<script setup>

</script>

<style lang="scss" scoped>

</style>