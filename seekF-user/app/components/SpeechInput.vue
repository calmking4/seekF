<template>
    <div class="speech-input" @keydown.esc.stop="cancel">
        <button type="button" class="microphone-button" :class="{ active: busy }" :disabled="disabled || busy"
            aria-label="语音输入" title="语音输入" @click="start">
            <Icon name="uil:microphone" class="text-xl" />
        </button>
        <Transition name="voice-panel">
            <section v-if="state !== 'idle'" class="voice-panel" aria-label="语音输入">
                <div class="voice-status" role="status" aria-live="polite">
                    <template v-if="state === 'recording'">
                        <span class="voice-wave" aria-hidden="true"><i v-for="(weight, index) in [.4, .7, 1, .6, .9, .5, .8]" :key="index" :style="{ height: `${5 + level * weight * 25}px` }" /></span>
                        <div><strong>正在聆听 <span class="voice-time">{{ Math.floor(duration).toString().padStart(2, '0') }} / 30 秒</span></strong><p>边说边识别，说完点击完成</p></div>
                    </template>
                    <template v-else-if="state === 'error'"><Icon name="uil:exclamation-circle" class="voice-error-icon" /><div><strong>{{ error }}</strong><p>输入框中的内容已保留</p></div></template>
                    <template v-else><Icon name="uil:spinner" class="voice-spinner" /><div><strong>{{ state === 'starting' ? '正在开启麦克风…' : '正在转成文字…' }}</strong><p>{{ state === 'starting' ? '请允许麦克风权限' : '识别文字正在逐步返回' }}</p></div></template>
                </div>
                <p v-if="transcript" class="voice-transcript">{{ transcript }}</p>
                <div class="voice-actions">
                    <button type="button" class="voice-cancel" @click="cancel">{{ state === 'error' ? '关闭' : '取消' }}</button>
                    <button v-if="state === 'recording'" type="button" class="voice-done" @click="finish"><Icon name="uil:check" />完成</button>
                    <template v-else-if="state === 'error'">
                        <button v-if="canRetry" type="button" class="voice-cancel" @click="retry">重试识别</button>
                        <button type="button" class="voice-done" :disabled="disabled" @click="start">重新录音</button>
                    </template>
                </div>
            </section>
        </Transition>
    </div>
</template>

<script setup>
import { watch, onUnmounted } from 'vue'
defineProps({ disabled: Boolean })
const emit = defineEmits(['transcript', 'busy-change'])
const { state, busy, canRetry, error, duration, level, transcript, start, finish, cancel, retry } = useASR(text => emit('transcript', text))
defineExpose({ cancel })
watch(busy, value => emit('busy-change', value), { flush: 'sync' })
onUnmounted(() => emit('busy-change', false))
</script>

<style scoped>
.microphone-button { display: grid; place-items: center; width: 40px; height: 40px; border: none; border-radius: 50%; background: #f5f5f5; color: #666; cursor: pointer; transition: background .18s, color .18s; }
.microphone-button:hover, .microphone-button.active { background: #eaf3ff; color: #0073ff; }
.microphone-button:disabled { cursor: default; opacity: .6; }
.voice-panel { position: absolute; bottom: calc(100% + 12px); left: 0; right: 0; display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 16px; padding: 20px 24px; border: 1px solid #e5eaf1; border-radius: 20px; background: white; box-shadow: 0 8px 32px #263a5510; }
.voice-status { display: flex; align-items: center; gap: 16px; min-width: 0; }
.voice-status strong { display: block; color: #374151; font-size: 14px; font-weight: 500; }
.voice-status p { margin: 5px 0 0; color: #9ca3af; font-size: 12px; }
.voice-time { margin-left: 8px; color: #8a98ab; font-size: 12px; font-variant-numeric: tabular-nums; }
.voice-wave { display: flex; align-items: center; justify-content: center; gap: 3px; width: 45px; height: 34px; flex-shrink: 0; }
.voice-wave i { display: block; width: 3px; border-radius: 4px; background: #60a5fa; transition: height .1s ease; }
.voice-spinner { color: #60a5fa; font-size: 26px; animation: voice-spin 1s linear infinite; }
.voice-error-icon { flex-shrink: 0; color: #df9076; font-size: 26px; }
.voice-actions { display: flex; align-items: center; gap: 8px; margin-left: auto; }
.voice-transcript { order: 1; flex-basis: 100%; max-height: 120px; overflow-y: auto; margin: 0; color: #374151; font-size: 14px; line-height: 1.7; white-space: pre-wrap; overflow-wrap: anywhere; }
.voice-actions button { display: flex; align-items: center; gap: 5px; border: none; border-radius: 20px; padding: 8px 14px; font-size: 13px; cursor: pointer; }
.voice-cancel { color: #6b7280; background: #f5f7fa; }
.voice-done { color: white; background: #0073ff; }
.voice-done:disabled { opacity: .5; cursor: default; }
button:focus-visible { outline: 2px solid #0073ff; outline-offset: 3px; }
.voice-panel-enter-active, .voice-panel-leave-active { transition: opacity .2s ease, transform .2s ease; }
.voice-panel-enter-from, .voice-panel-leave-to { opacity: 0; transform: translateY(6px); }
@keyframes voice-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .voice-panel-enter-active, .voice-panel-leave-active, .voice-wave i { transition: none; } .voice-spinner { animation: none; } }
</style>
