// 在音频线程批量收集采样，避免每个音频帧都触发主线程更新。
class ASRRecorder extends AudioWorkletProcessor {
    constructor() {
        super()
        this.buffer = new Float32Array(2048)
        this.offset = 0
        this.stopped = false
        this.port.onmessage = ({ data }) => {
            if (data === 'stop') {
                this.stopped = true
                this.flush()
                this.port.postMessage({ done: true })
            }
        }
    }
    flush() {
        if (!this.offset) return
        const chunk = this.buffer.slice(0, this.offset)
        this.port.postMessage({ samples: chunk }, [chunk.buffer])
        this.offset = 0
    }
    process(inputs) {
        if (this.stopped) return false
        const channels = inputs[0]
        if (!channels?.length) return true
        for (let i = 0; i < channels[0].length; i++) {
            let sample = 0
            for (const channel of channels) sample += channel[i]
            this.buffer[this.offset++] = sample / channels.length
            if (this.offset === this.buffer.length) this.flush()
        }
        // 不写入输出，防止麦克风声音回放产生回声。
        return true
    }
}
registerProcessor('seekf-asr-recorder', ASRRecorder)
