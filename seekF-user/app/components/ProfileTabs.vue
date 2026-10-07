<template>
  <div class="profile-tabs">
    <div class="profile-tab-nav" role="tablist" aria-label="主页内容">
      <button
        v-for="(tab, index) in tabs"
        :id="`${tabsId}-tab-${tab.name}`"
        :key="tab.name"
        :ref="element => { tabButtons[index] = element }"
        type="button"
        role="tab"
        class="profile-tab"
        :class="{ 'is-active': modelValue === tab.name }"
        :aria-selected="modelValue === tab.name"
        :aria-controls="`${tabsId}-panel-${tab.name}`"
        :tabindex="modelValue === tab.name ? 0 : -1"
        @click="selectTab(index)"
        @keydown.right.prevent="selectTab((index + 1) % tabs.length, true)"
        @keydown.left.prevent="selectTab((index - 1 + tabs.length) % tabs.length, true)"
        @keydown.home.prevent="selectTab(0, true)"
        @keydown.end.prevent="selectTab(tabs.length - 1, true)"
      >{{ tab.label }}</button>
    </div>
    <div class="profile-tab-content">
      <Transition :name="transitionName" mode="out-in" @after-enter="$emit('after-enter')">
        <div
          :id="`${tabsId}-panel-${modelValue}`"
          :key="modelValue"
          role="tabpanel"
          :aria-labelledby="`${tabsId}-tab-${modelValue}`"
        >
          <slot :name="modelValue" />
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, useId } from 'vue'

const props = defineProps({
  modelValue: { type: String, required: true },
  tabs: { type: Array, required: true },
})
const emit = defineEmits(['update:modelValue', 'tab-change', 'after-enter'])
const tabsId = useId()
const tabButtons = ref([])
const transitionName = ref('profile-slide-right')

watch(() => props.modelValue, (current, previous) => {
  const nextIndex = props.tabs.findIndex(tab => tab.name === current)
  const previousIndex = props.tabs.findIndex(tab => tab.name === previous)
  transitionName.value = nextIndex < previousIndex ? 'profile-slide-left' : 'profile-slide-right'
}, { flush: 'sync' })

const selectTab = (index, focus = false) => {
  const tab = props.tabs[index]
  if (!tab) return
  if (tab.name !== props.modelValue) {
    emit('update:modelValue', tab.name)
    emit('tab-change', tab.name)
  }
  if (focus) nextTick(() => tabButtons.value[index]?.focus())
}
</script>

<style scoped>
.profile-tab-nav {
  display: flex;
  justify-content: center;
  border-bottom: 1px solid #e5e7eb;
}
.profile-tab {
  position: relative;
  height: 48px;
  padding: 0 24px;
  border: none;
  background: transparent;
  color: #6b7280;
  font-size: 15px;
  line-height: 48px;
  cursor: pointer;
  transition: color 0.2s;
}
.profile-tab:hover,
.profile-tab.is-active {
  color: #111827;
}
.profile-tab.is-active {
  font-weight: 500;
}
.profile-tab.is-active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 2px;
  background: #111827;
}
.profile-tab:focus-visible {
  outline: 2px solid #60a5fa;
  outline-offset: -4px;
  border-radius: 4px;
}
.profile-tab-content {
  overflow: hidden;
}
.profile-slide-left-enter-active,
.profile-slide-right-enter-active {
  transition: transform 0.22s ease-out, opacity 0.22s ease-out;
}
.profile-slide-left-leave-active,
.profile-slide-right-leave-active {
  transition: transform 0.16s ease-in, opacity 0.16s ease-in;
  pointer-events: none;
}
.profile-slide-left-enter-from,
.profile-slide-right-leave-to {
  transform: translateX(-32px);
  opacity: 0;
}
.profile-slide-right-enter-from,
.profile-slide-left-leave-to {
  transform: translateX(32px);
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .profile-tab-content > div {
    transition-duration: 0.01ms;
    transform: none;
  }
}
</style>
