<template>
  <UserProfile :user-id="String(route.params.id || '')" />
</template>

<script setup>
const route = useRoute()
</script>
