/**
 * AI Chat composable
 * 封装 AI 聊天的 SSE 流式请求和会话管理
 */
export const useAIChat = () => {
    const config = useRuntimeConfig()
    const apiBase = (config.public.apiBase || 'http://localhost:8080').replace(/\/+$/, '')

    /**
     * 创建 AI 会话
     */
    const createSession = async (modelType) => {
        try {
            const res = await useApi$('/user/aichat/createSession', {
                method: 'POST',
                body: { model_type: modelType }
            })
            if (res?.code === 200) {
                return res.data
            }
            return null
        } catch (error) {
            console.error('创建AI会话失败:', error)
            return null
        }
    }

    /**
     * 获取 AI 会话列表
     */
    const getSessionList = async (page = 1, pageSize = 20) => {
        try {
            const res = await useApi$('/user/aichat/getSessionList', {
                method: 'POST',
                body: { page, page_size: pageSize }
            })
            if (res?.code === 200) {
                return res.data
            }
            throw new Error(res?.message || '获取AI会话列表失败')
        } catch (error) {
            console.error('获取AI会话列表失败:', error)
            throw error
        }
    }

    /**
     * 获取 AI 消息历史（游标分页）
     * @param {string} sessionId - 会话 ID
     * @param {number} pageSize - 每页数量
     * @param {string} cursor - 游标（最后一条消息的时间戳），为空则获取最新消息
     * @param {string} direction - 翻页方向：prev=向前(更旧), next=向后(更新)
     */
    const getMessageHistory = async (sessionId, pageSize = 20, cursor = '', direction = 'prev') => {
        try {
            const res = await useApi$('/user/aichat/getMessageHistory', {
                method: 'POST',
                body: { session_id: sessionId, page_size: pageSize, cursor, direction }
            })
            if (res?.code === 200) {
                return res.data
            }
            throw new Error(res?.message || '获取消息历史失败')
        } catch (error) {
            console.error('获取消息历史失败:', error)
            throw error
        }
    }

    // 发起请求不自动重试，刷新恢复只订阅已有任务，避免重复调用模型。
    const startGeneration = async (sessionId, content, modelType, imageFile, useKnowledge, useWebSearch) => {
        const body = new FormData()
        body.append('session_id', sessionId)
        body.append('content', content)
        body.append('model_type', modelType)
        if (imageFile) body.append('image', imageFile)
        if (useKnowledge) body.append('use_knowledge', 'true')
        if (useWebSearch) body.append('use_web_search', 'true')
        const response = await fetch(`${apiBase}/user/aichat/startGeneration`, { method: 'POST', body, credentials: 'include' })
        const result = await response.json()
        if (!response.ok || result.code !== 200) throw new Error(result.message || '启动AI回答失败')
        return result.data
    }

    // 恢复前查询归属及终态，缓存过期由页面回退已保存的历史消息。
    const getStreamInfo = async (sessionId, streamId) => {
        const query = new URLSearchParams({ session_id: sessionId, stream_id: streamId })
        const response = await fetch(`${apiBase}/user/aichat/streamInfo?${query}`, { credentials: 'include' })
        const result = await response.json()
        if (!response.ok || result.code !== 200) {
            const error = new Error(result.message || '查询AI回答状态失败')
            error.status = response.status
            throw error
        }
        return result.data
    }

    // 每次订阅持有自己的游标，整页刷新从头回放，连接中断从已处理事件之后重连。
    const subscribeStream = (sessionId, streamId, callbacks) => {
        const controller = new AbortController()
        let after = '0-0'
        const newer = id => {
            const [ms, seq] = id.split('-').map(BigInt)
            const [lastMS, lastSeq] = after.split('-').map(BigInt)
            return ms > lastMS || (ms === lastMS && seq > lastSeq)
        }
        const consume = async () => {
            let retries = 0
            while (!controller.signal.aborted) {
                let reader
                try {
                    const query = new URLSearchParams({ session_id: sessionId, stream_id: streamId, after })
                    const response = await fetch(`${apiBase}/user/aichat/stream?${query}`, { credentials: 'include', signal: controller.signal })
                    if (!response.ok) {
                        const result = await response.json().catch(() => ({}))
                        const error = new Error(result.message || '连接AI回答失败')
                        error.status = response.status
                        throw error
                    }
                    if (!response.body) throw new Error('AI回答连接没有响应流')
                    reader = response.body.getReader()
                    const decoder = new TextDecoder()
                    let buffer = ''
                    while (!controller.signal.aborted) {
                        const { value, done } = await reader.read()
                        if (done) throw new Error('AI回答连接已断开')
                        buffer += decoder.decode(value, { stream: true })
                        let boundary
                        while ((boundary = /\r?\n\r?\n/.exec(buffer))) {
                            const block = buffer.slice(0, boundary.index)
                            buffer = buffer.slice(boundary.index + boundary[0].length)
                            let id = '', event = 'message'
                            const dataLines = []
                            for (const line of block.split(/\r?\n/)) {
                                if (line.startsWith('id:')) id = line.slice(3).trim()
                                else if (line.startsWith('event:')) event = line.slice(6).trim()
                                else if (line.startsWith('data:')) dataLines.push(line.slice(5).trimStart())
                            }
                            if (!dataLines.length) continue
                            if (id && (!/^\d+-\d+$/.test(id) || !newer(id))) continue
                            const data = JSON.parse(dataLines.join('\n'))
                            if (event === 'delta') callbacks.onChunk?.(data.content || '')
                            else if (event === 'sources') callbacks.onSources?.(data.sources || [])
                            else if (event === 'posts') callbacks.onPosts?.(data.posts || [])
                            else if (event === 'done') {
                                controller.abort()
                                await callbacks.onComplete?.(data)
                                return
                            } else if (event === 'error') {
                                controller.abort()
                                callbacks.onError?.(data.error || 'AI回答生成失败')
                                return
                            }
                            if (id) after = id
                            retries = 0
                        }
                    }
                } catch (error) {
                    if (controller.signal.aborted) return
                    if ((error.status && error.status < 500) || retries >= 5) {
                        callbacks.onError?.(error.message || 'AI回答连接中断，请刷新重试')
                        return
                    }
                    // 只重试订阅，不重新提交提问；关闭页面时可立即取消等待。
                    await new Promise(resolve => {
                        const onAbort = () => { clearTimeout(timer); resolve() }
                        const timer = setTimeout(() => { controller.signal.removeEventListener('abort', onAbort); resolve() }, Math.min(1000 * 2 ** retries++, 5000))
                        controller.signal.addEventListener('abort', onAbort, { once: true })
                    })
                } finally {
                    await reader?.cancel().catch(() => {})
                }
            }
        }
        void consume()
        return { close: () => controller.abort() }
    }

    /**
     * 删除 AI 会话
     */
    const deleteSession = async (sessionId) => {
        try {
            const res = await useApi$('/user/aichat/deleteSession', {
                method: 'POST',
                body: { session_id: sessionId }
            })
            if (res?.code === 200) {
                return true
            }
            return false
        } catch (error) {
            console.error('删除会话失败:', error)
            return false
        }
    }

    return {
        createSession,
        getSessionList,
        getMessageHistory,
        startGeneration,
        getStreamInfo,
        subscribeStream,
        deleteSession
    }
}
