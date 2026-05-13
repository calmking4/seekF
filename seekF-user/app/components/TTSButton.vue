<template>
    <button
        class="inline-flex items-center justify-center w-8 h-8 rounded-lg transition-all duration-200"
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
    if (props.playing) return 'text-blue-500 hover:text-blue-600'
    if (props.loading) return 'text-blue-400 cursor-wait'
    return 'text-blue-500 hover:text-blue-600 hover:bg-blue-50 active:scale-95'
})

const buttonTitle = computed(() => {
    if (props.loading) return '正在合成语音...'
    if (props.playing) return '暂停播放'
    return '朗读'
})
</script>
