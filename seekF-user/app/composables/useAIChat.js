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
            return { list: [], total: 0 }
        } catch (error) {
            console.error('获取AI会话列表失败:', error)
            return { list: [], total: 0 }
        }
    }

    /**
     * 获取 AI 消息历史
     */
    const getMessageHistory = async (sessionId, page = 1, pageSize = 20) => {
        try {
            const res = await useApi$('/user/aichat/getMessageHistory', {
                method: 'POST',
                body: { session_id: sessionId, page, page_size: pageSize }
            })
            if (res?.code === 200) {
                return res.data
            }
            return { list: [], total: 0 }
        } catch (error) {
            console.error('获取消息历史失败:', error)
            return { list: [], total: 0 }
        }
    }

    /**
     * 发送消息（SSE 流式）
     * @param {string} sessionId - 会话 ID
     * @param {string} content - 消息内容
     * @param {string} modelType - 模型类型
     * @param {function} onChunk - 收到每个 chunk 时的回调
     * @param {function} onComplete - 完成时的回调
     * @param {function} onError - 错误回调
     * @returns {EventSource} 返回 EventSource 实例，可用于手动关闭
     */
    const sendMessage = (sessionId, content, modelType, onChunk, onComplete, onError) => {
        const encodedContent = encodeURIComponent(content)
        const url = `${apiBase}/user/aichat/sendMessage?session_id=${sessionId}&content=${encodedContent}&model_type=${modelType}`

        const evtSource = new EventSource(url, { withCredentials: true })

        evtSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data)
                if (data.error) {
                    onError?.(data.error)
                    evtSource.close()
                    return
                }
                if (data.content) {
                    onChunk?.(data.content)
                }
                if (data.done) {
                    onComplete?.()
                    evtSource.close()
                }
            } catch (e) {
                console.error('解析 SSE 数据失败:', e)
            }
        }

        evtSource.onerror = (err) => {
            console.error('SSE 连接错误:', err)
            onError?.('AI 响应中断')
            evtSource.close()
        }

        return evtSource
    }

    return {
        createSession,
        getSessionList,
        getMessageHistory,
        sendMessage
    }
}
