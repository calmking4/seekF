<template>
  <Teleport to="body">
    <div v-if="visible" ref="menuRef" role="menu" aria-label="消息操作" class="fixed z-[3000] min-w-36 overflow-hidden rounded-lg border border-gray-200 bg-[#f0f1f3] py-1 text-sm text-gray-600 shadow-lg outline-none" :style="{ left: `${position.x}px`, top: `${position.y}px` }" @contextmenu.prevent @keydown="handleKeydown">
      <button v-if="canRecall" type="button" role="menuitem" :disabled="busy" class="block w-full px-4 py-2 text-left outline-none hover:bg-gray-200 focus:bg-gray-200 disabled:opacity-50" @click="choose('recall')">撤回</button>
      <button type="button" role="menuitem" :disabled="busy" class="block w-full px-4 py-2 text-left outline-none hover:bg-gray-200 focus:bg-gray-200 disabled:opacity-50" @click="choose('delete')">删除</button>
    </div>
  </Teleport>
</template>

<script setup>
const props = defineProps({ canRecall: Boolean, busy: Boolean })
const emit = defineEmits(['command', 'close'])
const visible = ref(false)
const menuRef = ref(null)
const position = ref({ x: 0, y: 0 })
let anchor
let version = 0

// close 关闭菜单，并在键盘取消时恢复原消息焦点。
const close = (restoreFocus = false) => {
  version++
  if (!visible.value) return
  visible.value = false
  if (restoreFocus && anchor?.isConnected) anchor.focus({ preventScroll: true })
  anchor = null
  emit('close')
}

// open 在指针位置打开菜单，完成布局后避让视口边缘。
const open = async (event) => {
  const request = ++version
  anchor = event.currentTarget
  const rect = anchor.getBoundingClientRect()
  const keyboard = event.type === 'keydown'
  const x = keyboard ? rect.left : event.clientX
  const y = keyboard ? rect.bottom : event.clientY
  position.value = { x, y }
  visible.value = true
  await nextTick()
  if (request !== version || !menuRef.value) return
  const bounds = menuRef.value.getBoundingClientRect()
  position.value = {
    x: Math.max(8, Math.min(x, window.innerWidth - bounds.width - 8)),
    y: Math.max(8, Math.min(y, window.innerHeight - bounds.height - 8))
  }
  menuRef.value.querySelector('button:not(:disabled)')?.focus({ preventScroll: true })
}

// choose 发出操作后收起菜单，防止重复点击。
const choose = command => {
  if (props.busy) return
  emit('command', command)
  close(true)
}

// handleKeydown 支持方向键、首尾导航、Esc取消及Tab离开。
const handleKeydown = event => {
  if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close(true); return }
  if (event.key === 'Tab') { close(true); return }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const buttons = [...menuRef.value.querySelectorAll('button:not(:disabled)')]
  if (!buttons.length) return
  const current = buttons.indexOf(document.activeElement)
  const index = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1 : (current + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length
  buttons[index].focus({ preventScroll: true })
}

// handleOutside 点击菜单外部时关闭，保留用户点击目标的焦点行为。
const handleOutside = event => { if (visible.value && !menuRef.value?.contains(event.target)) close() }
const handleViewportChange = () => close()
onMounted(() => {
  document.addEventListener('pointerdown', handleOutside, true)
  document.addEventListener('scroll', handleViewportChange, true)
  window.addEventListener('resize', handleViewportChange)
  window.addEventListener('blur', handleViewportChange)
})
onUnmounted(() => {
  close()
  document.removeEventListener('pointerdown', handleOutside, true)
  document.removeEventListener('scroll', handleViewportChange, true)
  window.removeEventListener('resize', handleViewportChange)
  window.removeEventListener('blur', handleViewportChange)
})
defineExpose({ open, close })
</script>
