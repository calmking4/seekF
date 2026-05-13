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
     * @param {File} imageFile - 图片文件（可选）
     * @param {boolean} useKnowledge - 是否使用知识库
     * @param {boolean} useWebSearch - 是否启用联网搜索
     * @param {function} onChunk - 收到每个 chunk 时的回调
     * @param {function} onSources - 收到搜索来源时的回调
     * @param {function} onComplete - 完成时的回调
     * @param {function} onError - 错误回调
     * @returns {object} 返回 close 方法，可用于手动关闭
     */
    const sendMessage = (sessionId, content, modelType, imageFile, useKnowledge = false, useWebSearch = false, onChunk, onSources, onComplete, onError) => {
        const formData = new FormData()
        formData.append('session_id', sessionId)
        formData.append('content', content)
        formData.append('model_type', modelType)
        if (useKnowledge) {
            formData.append('use_knowledge', 'true')
        }
        if (useWebSearch) {
            formData.append('use_web_search', 'true')
        }
        if (imageFile) {
            formData.append('image', imageFile)
        }

        const url = `${apiBase}/user/aichat/sendMessage`

        const controller = new AbortController()
        
        fetch(url, {
            method: 'POST',
            body: formData,
            credentials: 'include',
            signal: controller.signal
        }).then(response => {
            if (!response.ok) {
                throw new Error('请求失败')
            }
            
            const reader = response.body.getReader()
            const decoder = new TextDecoder()
            let buffer = ''

            const readStream = () => {
                reader.read().then(({ done, value }) => {
                    if (done) {
                        if (buffer) {
                            try {
                                const data = JSON.parse(buffer)
                                if (data.error) {
                                    onError?.(data.error)
                                }
                            } catch (e) {}
                        }
                        onComplete?.()
                        return
                    }

                    buffer += decoder.decode(value, { stream: true })
                    const lines = buffer.split('\n')
                    buffer = lines.pop() || ''

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            try {
                                const data = JSON.parse(line.slice(6))
                                if (data.error) {
                                    onError?.(data.error)
                                    controller.abort()
                                    return
                                }
                                if (data.sources) {
                                    onSources?.(data.sources)
                                }
                                if (data.content) {
                                    onChunk?.(data.content)
                                }
                                if (data.done) {
                                    onComplete?.()
                                    controller.abort()
                                    return
                                }
                            } catch (e) {
                                console.error('解析 SSE 数据失败:', e)
                            }
                        }
                    }

                    readStream()
                })
            }

            readStream()
        }).catch(err => {
            if (err.name !== 'AbortError') {
                console.error('SSE 连接错误:', err)
                onError?.('AI 响应中断')
            }
        })

        return {
            close: () => controller.abort()
        }
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
        sendMessage,
        deleteSession
    }
}
