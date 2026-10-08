// 消费ASR事件流，保留跨网络数据块的UTF-8字符和SSE事件边界。
export async function readASRStream(response, onText, signal) {
    if (!response.ok) {
        const result = await response.json().catch(() => ({}))
        throw new Error(result.message || '语音识别失败，请重试')
    }
    if (!response.body || !response.headers.get('content-type')?.includes('text/event-stream')) {
        throw new Error('语音识别服务未返回事件流')
    }
    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let pending = '', eventLines = [], text = '', finished = false
    // 完整文本事件用于校准结果；增量事件直接追加，不合并重复词语。
    const consumeEvent = () => {
        if (!eventLines.length) return
        const payload = eventLines.join('\n')
        eventLines = []
        if (payload.trim() === '[DONE]') { finished = true; return }
        let data
        try { data = JSON.parse(payload) }
        catch { throw new Error('语音识别事件格式无效') }
        if (data.error) throw new Error(data.error.message || (typeof data.error === 'string' ? data.error : '语音识别失败'))
        if (data.type?.endsWith('.done')) {
            if (typeof data.text === 'string') { text = data.text; onText(text) }
            finished = true
            return
        }
        const delta = typeof data.delta === 'string' ? data.delta :
            (data.choices?.[0]?.delta?.content ?? data.choices?.[0]?.delta?.text ?? data.text ?? '')
        if (typeof delta === 'string' && delta) { text += delta; onText(text) }
    }
    const consumeLines = () => {
        let index
        while ((index = pending.indexOf('\n')) !== -1) {
            const line = pending.slice(0, index).replace(/\r$/, '')
            pending = pending.slice(index + 1)
            if (!line) consumeEvent()
            else if (line.startsWith('data:')) eventLines.push(line.slice(5).replace(/^ /, ''))
            if (finished) return
        }
        if (pending.length + eventLines.join('').length > 1024 * 1024) throw new Error('语音识别事件过大')
    }
    try {
        while (!finished) {
            signal?.throwIfAborted()
            const { value, done } = await reader.read()
            if (done) {
                pending += decoder.decode() + '\n\n'
                consumeLines()
                if (!finished) throw new Error('语音识别连接中断，请重试')
                break
            }
            pending += decoder.decode(value, { stream: true })
            consumeLines()
        }
        signal?.throwIfAborted()
        return text.trim()
    } finally {
        await reader.cancel().catch(() => {})
        reader.releaseLock()
    }
}
