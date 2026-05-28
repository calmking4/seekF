/**
 * TTS (Text-to-Speech) composable
 * 使用 AudioContext 实现 PCM 流式播放
 * PCM 格式: 16-bit signed little-endian, 24000Hz, mono
 */
export const useTTS = () => {
    const SAMPLE_RATE = 24000
    // 缓冲多少字节后开始播放（~0.3秒音频）
    const MIN_BUFFER_BYTES = SAMPLE_RATE * 2 * 0.3

    const isPlaying = ref(false)
    const isLoading = ref(false)
    const currentMessageId = ref(null)

    let audioCtx = null
    let abortController = null
    let scheduledEndTime = 0
    let playing = false

    const { apiBase } = useRuntimeConfig().public

    // PCM 16-bit little-endian 转 Float32（须用 slice 的 byteOffset/byteLength，避免误读整块 ArrayBuffer）
    const pcm16ToFloat32 = (bytes) => {
        const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
        const float32 = new Float32Array(bytes.byteLength / 2)
        for (let i = 0; i < float32.length; i++) {
            // Int16 little-endian -> float [-1, 1]
            float32[i] = view.getInt16(i * 2, true) / 32768.0
        }
        return float32
    }

    // 调度一段 Float32 数据到 AudioContext 播放
    const scheduleBuffer = (float32Data) => {
        const buffer = audioCtx.createBuffer(1, float32Data.length, SAMPLE_RATE)
        buffer.getChannelData(0).set(float32Data)
        const source = audioCtx.createBufferSource()
        source.buffer = buffer
        source.connect(audioCtx.destination)
        const when = Math.max(scheduledEndTime, audioCtx.currentTime)
        source.start(when)
        scheduledEndTime = when + buffer.duration
    }

    // 清理
    const destroyAudio = () => {
        if (abortController) {
            abortController.abort()
            abortController = null
        }
        if (audioCtx) {
            audioCtx.close().catch(() => {})
            audioCtx = null
        }
        scheduledEndTime = 0
        playing = false
    }

    const speak = async (text, messageId) => {
        if (currentMessageId.value === messageId && isLoading.value) return
        if (currentMessageId.value === messageId && playing) {
            destroyAudio()
            isPlaying.value = false
            currentMessageId.value = null
            return
        }

        destroyAudio()
        isPlaying.value = false
        isLoading.value = true
        currentMessageId.value = messageId

        try {
            audioCtx = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: SAMPLE_RATE })
            scheduledEndTime = audioCtx.currentTime

            abortController = new AbortController()
            const response = await fetch(apiBase + 'user/aichat/tts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ content: text }),
                signal: abortController.signal
            })

            if (!response.ok) throw new Error('语音合成请求失败')

            const reader = response.body.getReader()
            // 累积缓冲区
            let pending = new Uint8Array(0)

            // 将 pending 中完整的采样帧（2字节）转为 float 并调度播放
            const flushPending = () => {
                const validBytes = pending.length - (pending.length % 2)
                if (validBytes < 2) return
                const chunk = pending.slice(0, validBytes)
                pending = pending.slice(validBytes)
                scheduleBuffer(pcm16ToFloat32(chunk))
            }

            while (true) {
                const { done, value } = await reader.read()
                if (done) {
                    flushPending()
                    break
                }

                // 合并到 pending
                const merged = new Uint8Array(pending.length + value.length)
                merged.set(pending)
                merged.set(value, pending.length)
                pending = merged

                // 缓冲够了开始播放
                if (!playing && pending.length >= MIN_BUFFER_BYTES) {
                    playing = true
                    isPlaying.value = true
                    isLoading.value = false
                    flushPending()
                } else if (playing) {
                    flushPending()
                }
            }

            // 等待播放结束
            if (playing) {
                await new Promise((resolve) => {
                    const check = () => {
                        if (!audioCtx || audioCtx.currentTime >= scheduledEndTime - 0.05) {
                            resolve()
                        } else {
                            requestAnimationFrame(check)
                        }
                    }
                    requestAnimationFrame(check)
                })
            }
        } catch (err) {
            if (err.name !== 'AbortError') {
                console.error('TTS failed:', err)
                ElMessage.error('语音合成失败')
            }
        } finally {
            destroyAudio()
            isPlaying.value = false
            isLoading.value = false
            currentMessageId.value = null
        }
    }

    const stop = () => {
        destroyAudio()
        isPlaying.value = false
        isLoading.value = false
        currentMessageId.value = null
    }

    const isMessagePlaying = (messageId) => {
        return currentMessageId.value === messageId && isPlaying.value
    }

    const isMessageLoading = (messageId) => {
        return currentMessageId.value === messageId && isLoading.value
    }

    onUnmounted(() => { stop() })

    return {
        isPlaying: readonly(isPlaying),
        isLoading: readonly(isLoading),
        currentMessageId: readonly(currentMessageId),
        speak,
        stop,
        isMessagePlaying,
        isMessageLoading
    }
}
