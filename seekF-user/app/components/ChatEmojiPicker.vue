<template>
  <el-popover v-model:visible="visible" trigger="click" placement="top-start" :offset="24" :width="320" :persistent="false" popper-class="!max-w-[calc(100vw-24px)]" @show="loadRecent">
    <template #reference>
      <button ref="triggerRef" type="button" class="flex h-8 w-8 items-center justify-center rounded-md text-gray-400 transition-all duration-200 hover:bg-gray-100 hover:text-gray-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" title="表情" aria-label="选择表情" :aria-expanded="visible" aria-haspopup="dialog">
        <Icon name="uil:smile" class="text-2xl" />
      </button>
    </template>
    <section role="dialog" aria-label="选择表情" @keydown.esc.stop.prevent="closeWithFocus">
      <div class="mb-3 flex flex-wrap gap-1 border-b border-gray-100 pb-2" aria-label="表情分类">
        <button v-for="category in categories" :key="category.id" type="button" class="rounded-md px-2 py-1.5 text-xs transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" :class="activeCategory === category.id ? 'bg-blue-50 text-blue-500' : 'text-gray-500 hover:bg-gray-100'" :aria-pressed="activeCategory === category.id" @click="activeCategory = category.id">{{ category.label }}</button>
      </div>
      <div class="max-h-[min(240px,40vh)] overflow-y-auto overscroll-contain" :aria-label="categories.find(category => category.id === activeCategory)?.label">
        <div v-if="displayedEmojis.length" class="grid grid-cols-8 gap-0.5">
          <button v-for="emoji in displayedEmojis" :key="emoji" type="button" :aria-label="`插入表情 ${emoji}`" class="flex h-9 min-w-0 items-center justify-center rounded-md text-2xl leading-none hover:bg-gray-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400" @click="selectEmoji(emoji)">{{ emoji }}</button>
        </div>
        <p v-else class="py-8 text-center text-xs text-gray-400">还没有使用过表情</p>
      </div>
    </section>
  </el-popover>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { emojiCategories, normalizeRecentEmojis } from '~/utils/emoji'

const props = defineProps({ userId: { type: String, default: '' }, contextId: { type: String, default: '' } })
const emit = defineEmits(['select'])
const visible = ref(false)
const triggerRef = ref(null)
const activeCategory = ref('faces')
const recent = ref([])
const recentDisplay = ref([])
const categories = [{ id: 'recent', label: '最近' }, ...emojiCategories]
const displayedEmojis = computed(() => activeCategory.value === 'recent' ? recentDisplay.value : emojiCategories.find(category => category.id === activeCategory.value)?.emojis || [])
const storageKey = computed(() => `chat_emoji_recent:${props.userId}`)

// loadRecent 按当前账号读取最近使用，存储不可用时保留本次会话记录。
const loadRecent = () => {
  if (props.userId) {
    try { recent.value = normalizeRecentEmojis(JSON.parse(localStorage.getItem(storageKey.value) || '[]')) } catch { /* 忽略损坏或不可用的本地缓存。 */ }
  }
  recentDisplay.value = [...recent.value]
}

// selectEmoji 保持面板焦点及最近列表位置不变，便于连续点击或按键插入。
const selectEmoji = emoji => {
  recent.value = normalizeRecentEmojis([emoji, ...recent.value])
  if (props.userId) {
    try { localStorage.setItem(storageKey.value, JSON.stringify(recent.value)) } catch { /* 存储不可用不影响发送表情。 */ }
  }
  emit('select', emoji)
}

// closeWithFocus 通过Esc关闭面板并将键盘焦点还给表情按钮。
const closeWithFocus = () => {
  visible.value = false
  triggerRef.value?.focus({ preventScroll: true })
}
const onEscape = event => {
  if (visible.value && event.key === 'Escape') { event.preventDefault(); closeWithFocus() }
}

watch(() => props.contextId, () => { visible.value = false })
watch(activeCategory, category => { if (category === 'recent') recentDisplay.value = [...recent.value] })
watch(() => props.userId, () => { visible.value = false; recent.value = []; recentDisplay.value = []; activeCategory.value = 'faces' })
onMounted(() => document.addEventListener('keydown', onEscape))
onUnmounted(() => document.removeEventListener('keydown', onEscape))
</script>
