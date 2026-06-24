<template>
    <button
        class="tts-btn"
        :class="buttonClass"
        :disabled="loading"
        :title="buttonTitle"
        @click.stop="$emit('speak')"
    >
        <Icon v-if="loading" name="uil:spinner" class="text-base animate-spin" />
        <Icon v-else-if="playing" name="uil:pause-circle" class="text-base" />
        <Icon v-else name="uil:volume" class="text-base" />
    </button>
</template>

<script setup>
const props = defineProps({
    playing: { type: Boolean, default: false },
    loading: { type: Boolean, default: false }
})

defineEmits(['speak'])

const buttonClass = computed(() => {
    if (props.playing) return 'playing'
    if (props.loading) return 'loading'
    return ''
})

const buttonTitle = computed(() => {
    if (props.loading) return '正在合成语音...'
    if (props.playing) return '暂停播放'
    return '朗读'
})
</script>

<style scoped>
.tts-btn {
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: #6b7280;
    transition: background 0.15s ease, color 0.15s ease;
}

.tts-btn:hover:not(:disabled) {
    background: #e5e7eb;
    color: #374151;
}

.tts-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.tts-btn.playing {
    color: #6b7280;
}

.tts-btn.loading {
    color: #9ca3af;
}
</style>
