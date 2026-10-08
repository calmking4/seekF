import { ref, computed, onUnmounted } from 'vue'
import { encodeRecordingWav } from '~/utils/recordingWav'
import { readASRStream } from '~/utils/asrStream'

export const useASR = (onText) => {
    const state = ref('idle')
    const error = ref('')
    const duration = ref(0)
    const level = ref(0)
    const canRetry = ref(false)
    const transcript = ref('')
    const busy = computed(() => ['starting', 'recording', 'transcribing'].includes(state.value))
    const config = useRuntimeConfig()
    let version = 0
    let stream, audioContext, source, recorder, abortController
    let chunks = [], sampleCount = 0, sampleRate = 16000, recordingBlob = null
    let flushComplete = null
    let segmentChunks = [], segmentSamples = 0, silentSamples = 0, segmentHasSpeech = false
    let queue = [], drainJob = null, recognizedText = ''

    const releaseMicrophone = () => {
        stream?.getTracks().forEach(track => track.stop())
        stream = null
        source?.disconnect(); recorder?.disconnect()
        if (recorder) recorder.port.onmessage = null
        source = recorder = null
        audioContext?.close().catch(() => {})
        audioContext = null
        level.value = 0
    }

    const cancel = () => {
        version++
        abortController?.abort()
        abortController = null
        flushComplete?.(); flushComplete = null
        releaseMicrophone()
        chunks = []; recordingBlob = null
        segmentChunks = []; segmentSamples = 0; silentSamples = 0; segmentHasSpeech = false
        queue = []; drainJob = null; recognizedText = ''
        state.value = 'idle'; error.value = ''; duration.value = 0; canRetry.value = false
        transcript.value = ''
    }

    // 每段音频独立使用SSE识别，单个请求超时不会影响下一次录音。
    const requestRecording = async (blob, current, onPartial) => {
        const requestController = new AbortController()
        abortController = requestController
        const timeout = setTimeout(() => requestController.abort(), 65000)
        try {
            const body = new FormData()
            body.append('file', blob, 'recording.wav')
            const response = await fetch(`${String(config.public.apiBase).replace(/\/$/, '')}/user/aichat/asr`, {
                method: 'POST', credentials: 'include', headers: { Accept: 'text/event-stream' }, body, signal: requestController.signal,
            })
            const text = await readASRStream(response, text => {
                if (current === version) onPartial(text)
            }, requestController.signal)
            return text
        } finally {
            clearTimeout(timeout)
            if (current === version) abortController = null
        }
    }

    // 保留完整录音，分段识别失败后可重试，不重复拼接已有文字。
    const failRecognition = (err, current) => {
        if (current !== version) return
        if (!recordingBlob && sampleCount >= sampleRate * .3) recordingBlob = encodeRecordingWav(chunks, sampleRate)
        releaseMicrophone()
        queue = []
        state.value = 'error'; canRetry.value = !!recordingBlob
        error.value = err.name === 'AbortError' ? '识别超时，请重试' : (err.message || '语音识别失败，请重试')
    }

    const joinText = (left, right) => left && /[a-zA-Z0-9]$/.test(left) && /^[a-zA-Z0-9]/.test(right) ? `${left} ${right}` : left + right

    // 串行处理音频段，避免后录制的文字抢先返回。
    const drainQueue = (current) => {
        if (drainJob) return drainJob
        drainJob = (async () => {
            try {
                while (queue.length && current === version) {
                    const blob = queue.shift()
                    const text = await requestRecording(blob, current, partial => {
                        transcript.value = joinText(recognizedText, partial)
                    })
                    if (current !== version) return
                    recognizedText = joinText(recognizedText, text)
                    transcript.value = recognizedText
                }
            } catch (err) { failRecognition(err, current) }
            finally { if (current === version) drainJob = null }
        })()
        return drainJob
    }

    // 停顿优先切段，连续讲话最长8秒切段，静音段不发送。
    const enqueueSegment = () => {
        if (segmentSamples > 0 && segmentHasSpeech) queue.push(encodeRecordingWav(segmentChunks, sampleRate))
        segmentChunks = []; segmentSamples = 0; silentSamples = 0; segmentHasSpeech = false
        if (queue.length) void drainQueue(version)
    }

    const transcribe = async () => {
        if (!recordingBlob || busy.value) return
        const current = version
        state.value = 'transcribing'; error.value = ''; transcript.value = ''
        try {
            const text = await requestRecording(recordingBlob, current, partial => { transcript.value = partial })
            if (current !== version) return
            if (!text) throw new Error('没有识别到讲话，请重新录音')
            state.value = 'idle'; recordingBlob = null; canRetry.value = false
            onText(text)
        } catch (err) { failRecognition(err, current) }
    }

    const finish = async () => {
        if (state.value !== 'recording') return
        const current = version
        state.value = 'transcribing'
        // 等待音频线程送出最后一批采样，避免丢失句尾。
        await new Promise(resolve => {
            const timer = setTimeout(resolve, 500)
            flushComplete = () => { clearTimeout(timer); resolve() }
            recorder.port.postMessage('stop')
        })
        if (current !== version) return
        flushComplete = null
        if (state.value === 'error') return
        releaseMicrophone()
        if (sampleCount < sampleRate * .3) {
            state.value = 'error'; error.value = '录音太短，请重新录音'; chunks = []
            return
        }
        recordingBlob = encodeRecordingWav(chunks, sampleRate)
        enqueueSegment()
        if (drainJob) await drainJob
        if (current !== version || state.value === 'error') return
        const text = recognizedText.trim()
        if (!text) { failRecognition(new Error('没有识别到讲话，请重新录音'), current); return }
        chunks = []; recordingBlob = null; canRetry.value = false
        state.value = 'idle'
        onText(text)
    }

    const start = async () => {
        if (busy.value) return
        cancel()
        const current = version
        if (!navigator.mediaDevices?.getUserMedia || !window.AudioWorkletNode) {
            state.value = 'error'; error.value = '当前环境不支持录音，请使用HTTPS或localhost访问并更新浏览器'
            return
        }
        state.value = 'starting'
        let grantedStream, context
        try {
            // AudioContext在用户点击时创建，满足浏览器音频启动要求。
            context = new (window.AudioContext || window.webkitAudioContext)()
            audioContext = context
            await context.resume()
            if (current !== version) return
            grantedStream = await navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true }, video: false })
            if (current !== version) { grantedStream.getTracks().forEach(track => track.stop()); return }
            stream = grantedStream
            sampleRate = context.sampleRate
            await context.audioWorklet.addModule(`${config.app.baseURL}audio/asr-recorder.js`)
            if (current !== version) return
            source = context.createMediaStreamSource(stream)
            recorder = new AudioWorkletNode(context, 'seekf-asr-recorder')
            sampleCount = 0; chunks = []
            recorder.port.onmessage = ({ data }) => {
                if (current !== version) return
                if (data.done) { flushComplete?.(); return }
                if (!data.samples) return
                const samples = data.samples.subarray(0, Math.max(0, sampleRate * 30 - sampleCount))
                if (samples.length) {
                    chunks.push(samples); sampleCount += samples.length
                    duration.value = sampleCount / sampleRate
                    let sum = 0
                    for (const value of samples) sum += value * value
                    const rms = Math.sqrt(sum / samples.length)
                    level.value = Math.min(1, rms * 6)
                    segmentChunks.push(samples); segmentSamples += samples.length
                    if (rms > .003) { segmentHasSpeech = true; silentSamples = 0 }
                    else silentSamples += samples.length
                    if (state.value === 'recording' && (segmentSamples >= sampleRate * 8 ||
                        (segmentSamples >= sampleRate * 2 && silentSamples >= sampleRate * .5))) enqueueSegment()
                }
                if (sampleCount >= sampleRate * 30 && state.value === 'recording') void finish()
            }
            state.value = 'recording'
            source.connect(recorder); recorder.connect(context.destination)
            stream.getAudioTracks()[0].onended = () => {
                if (current === version && state.value === 'recording') {
                    cancel(); state.value = 'error'; error.value = '麦克风已断开，请重新录音'
                }
            }
        } catch (err) {
            if (current !== version) return
            releaseMicrophone()
            state.value = 'error'
            error.value = err.name === 'NotAllowedError' ? '请允许使用麦克风后重新录音' : err.name === 'NotFoundError' ? '未找到麦克风，请检查设备' : '无法启动录音，请检查麦克风是否被占用'
        }
    }

    onUnmounted(cancel)
    return { state, busy, canRetry, error, duration, level, transcript, start, finish, cancel, retry: transcribe }
}
