// 将浏览器单声道浮点采样转换为16kHz、16位PCM WAV。
export function encodeRecordingWav(chunks, sampleRate) {
    const length = chunks.reduce((sum, chunk) => sum + chunk.length, 0)
    const input = new Float32Array(length)
    let offset = 0
    for (const chunk of chunks) { input.set(chunk, offset); offset += chunk.length }
    const outputLength = Math.min(16000 * 30, Math.floor(length * 16000 / sampleRate))
    const buffer = new ArrayBuffer(44 + outputLength * 2)
    const view = new DataView(buffer)
    const writeString = (at, value) => {
        for (let i = 0; i < value.length; i++) view.setUint8(at + i, value.charCodeAt(i))
    }
    writeString(0, 'RIFF'); view.setUint32(4, buffer.byteLength - 8, true)
    writeString(8, 'WAVE'); writeString(12, 'fmt '); view.setUint32(16, 16, true)
    view.setUint16(20, 1, true); view.setUint16(22, 1, true)
    view.setUint32(24, 16000, true); view.setUint32(28, 32000, true)
    view.setUint16(32, 2, true); view.setUint16(34, 16, true)
    writeString(36, 'data'); view.setUint32(40, outputLength * 2, true)
    const ratio = sampleRate / 16000
    for (let i = 0; i < outputLength; i++) {
        const from = Math.floor(i * ratio)
        const to = Math.min(length, Math.max(from + 1, Math.floor((i + 1) * ratio)))
        let sum = 0
        for (let j = from; j < to; j++) sum += input[j]
        const sample = Math.max(-1, Math.min(1, sum / (to - from)))
        view.setInt16(44 + i * 2, Math.round(sample * (sample < 0 ? 32768 : 32767)), true)
    }
    return new Blob([buffer], { type: 'audio/wav' })
}
