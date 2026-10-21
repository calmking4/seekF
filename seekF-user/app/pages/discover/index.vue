<template>
  <div ref="pageRef" class="min-h-screen bg-gray-50">
    <DiscoverHeader v-model:channel="channel" :unread-authors="unreadAuthors" immersive />
    <div class="grid transition-[grid-template-rows,opacity] duration-300 ease-out motion-reduce:transition-none" :class="channel === 'following' ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'" :inert="channel !== 'following'">
      <div class="min-h-0 overflow-hidden"><DiscoverFollowingAuthors :authors="authors" :unread-authors="unreadAuthors" :selected="selected" :loading="loading" :error="error" :read-error="readError" @select="selectAuthor" @retry="refresh" /></div>
    </div>
    <Transition mode="out-in" enter-active-class="transition-[opacity,transform] duration-200 ease-out motion-reduce:transition-none" enter-from-class="translate-y-2 opacity-0" leave-active-class="transition-opacity duration-150 motion-reduce:transition-none" leave-to-class="opacity-0">
      <DiscoverFeed :key="channel + ':' + selected + ':' + feedRevision" :channel="channel" :author-uuid="channel === 'following' ? selected : ''" @loaded="loaded" />
    </Transition>
    <ScrollEdgeButton :target="() => pageRef?.closest('main')" direction="top" class="fixed bottom-6 right-6" />
  </div>
</template>

<script setup>
const pageRef = ref(null)
const channel = ref('recommended')
const feedRevision = ref(0)
const { authors, unreadAuthors, loading, error, readError, selected, refresh, select, loaded } = useFollowingUpdates()
// 再次点击当前作者时重新加载，以便主动查看刚发布的新作品。
const selectAuthor = authorId => {
  if (selected.value === authorId) feedRevision.value++
  select(authorId)
}
watch(channel, value => { select(''); if (value === 'following') void refresh() })
useSeoMeta({ title: '发现', description: '发现有趣的人和内容，探索 seekF 社区的精彩动态。' })
</script>
