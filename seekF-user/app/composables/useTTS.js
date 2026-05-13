/**
 * TTS (Text-to-Speech) composable
 * 封装语音合成功能和音频播放控制
 */
export const useTTS = () => {
    const isPlaying = ref(false)
    const isLoading = ref(false)
    const currentMessageId = ref(null)

    let audio = null

    const { apiBase } = useRuntimeConfig().public

    // 彻底清理 audio 对象
    const destroyAudio = () => {
        if (audio) {
            // 先移除所有回调，防止 stop/pause 触发 onended/onerror
            audio.onplay = null
            audio.onended = null
            audio.onerror = null
            audio.onpause = null
            audio.pause()
            audio.src = ''
            audio = null
        }
    }

    const speak = async (text, messageId) => {
        // 同一条消息正在加载中 -> 忽略
        if (currentMessageId.value === messageId && isLoading.value) {
            return
        }
        // 同一条消息正在播放 -> 暂停
        if (currentMessageId.value === messageId && isPlaying.value) {
            audio?.pause()
            return
        }
        // 同一条消息已暂停 -> 恢复
        if (currentMessageId.value === messageId && audio && !isPlaying.value) {
            audio.play()
            return
        }

        // 彻底清理旧的 audio
        destroyAudio()
        isPlaying.value = false
        isLoading.value = true
        currentMessageId.value = messageId

        try {
            const response = await fetch(apiBase + 'user/aichat/tts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ content: text })
            })

            if (!response.ok) {
                throw new Error('语音合成失败')
            }

            const blob = await response.blob()
            const audioBlob = new Blob([blob], { type: 'audio/wav' })
            const url = URL.createObjectURL(audioBlob)

            audio = new Audio(url)
            audio.onplay = () => {
                isPlaying.value = true
            }
            audio.onended = () => {
                destroyAudio()
                isPlaying.value = false
                currentMessageId.value = null
            }
            audio.onerror = () => {
                destroyAudio()
                isPlaying.value = false
                currentMessageId.value = null
            }
            audio.onpause = () => {
                isPlaying.value = false
            }

            await audio.play()
        } catch (err) {
            console.error('TTS failed:', err)
            ElMessage.error('语音合成失败')
            destroyAudio()
            currentMessageId.value = null
        } finally {
            isLoading.value = false
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

    onUnmounted(() => {
        stop()
    })

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
