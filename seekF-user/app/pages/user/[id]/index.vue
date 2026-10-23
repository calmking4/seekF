<template>
  <UserProfile :user-id="String(route.params.id || '')" :from-notification="route.query.from === 'notification'" />
</template>

<script setup>
const route = useRoute()
</script>
