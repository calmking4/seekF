// WebSocket 连接管理
let wsInstance = null
let messageCallbacks = []
let reconnectAttempts = 0
const maxReconnectAttempts = 5
const reconnectInterval = 3000

export const useWebSocket = () => {
    const config = useRuntimeConfig()
    const isConnected = ref(false)
    const user = useAuthState()

    // WebSocket URL
    const getWsUrl = () => {
        const wsBase = config.public.wsBase || 'ws://localhost:8080/'
        const userUuid = user.getUser()?.uuid || ''
        return `${wsBase}user/ws/login?client_id=${userUuid}`
    }

    // 连接 WebSocket
    const connect = () => {
        // 如果已经连接，直接返回
        if (wsInstance?.readyState === WebSocket.OPEN) {
            console.log('WebSocket 已连接')
            isConnected.value = true
            return
        }

        // 如果没有用户信息，不连接
        const currentUser = user.getUser()
        if (!currentUser?.uuid) {
            console.log('用户未登录，暂不连接 WebSocket')
            return
        }

        const wsUrl = getWsUrl()
        console.log('正在连接 WebSocket:', wsUrl)

        try {
            wsInstance = new WebSocket(wsUrl)

            wsInstance.onopen = () => {
                console.log('WebSocket 连接成功')
                isConnected.value = true
                reconnectAttempts = 0
            }

            wsInstance.onmessage = (event) => {
                console.log('收到 WebSocket 消息:', event.data)
                try {
                    const data = JSON.parse(event.data)
                    // 调用所有注册的回调函数
                    messageCallbacks.forEach(callback => callback(data))
                } catch (e) {
                    console.log('WebSocket 收到非 JSON 消息:', event.data)
                    messageCallbacks.forEach(callback => callback(event.data))
                }
            }

            wsInstance.onclose = () => {
                console.log('WebSocket 连接关闭')
                isConnected.value = false
                wsInstance = null
                attemptReconnect()
            }

            wsInstance.onerror = (error) => {
                console.error('WebSocket 错误:', error)
                isConnected.value = false
            }
        } catch (error) {
            console.error('创建 WebSocket 失败:', error)
            attemptReconnect()
        }
    }

    // 尝试重连
    const attemptReconnect = () => {
        // 检查用户是否已登录
        const currentUser = user.getUser()
        if (!currentUser?.uuid) {
            console.log('用户未登录，停止重连')
            return
        }

        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++
            console.log(`WebSocket 尝试重连 (${reconnectAttempts}/${maxReconnectAttempts})...`)
            setTimeout(() => {
                connect()
            }, reconnectInterval)
        } else {
            console.error('WebSocket 重连次数已达上限')
        }
    }

    // 断开连接
    const disconnect = () => {
        if (wsInstance) {
            wsInstance.close()
            wsInstance = null
            isConnected.value = false
            console.log('WebSocket 已断开')
        }
    }

    // 发送消息
    const send = (message) => {
        if (wsInstance?.readyState === WebSocket.OPEN) {
            const messageStr = typeof message === 'string' ? message : JSON.stringify(message)
            wsInstance.send(messageStr)
            console.log('发送 WebSocket 消息:', messageStr)
            return true
        } else {
            console.error('WebSocket 未连接，无法发送消息')
            return false
        }
    }

    // 注册消息回调
    const onMessage = (callback) => {
        messageCallbacks.push(callback)
        // 返回取消注册的函数
        return () => {
            const index = messageCallbacks.indexOf(callback)
            if (index > -1) {
                messageCallbacks.splice(index, 1)
            }
        }
    }

    // 清除所有回调
    const clearCallbacks = () => {
        messageCallbacks = []
    }

    // 发送文本消息
    const sendTextMessage = (sessionId, content, receiveId) => {
        const currentUser = user.getUser()
        if (!currentUser) {
            console.error('用户未登录')
            return false
        }

        return send({
            session_id: sessionId,
            type: 0, // 文本消息
            content: content,
            url: '',
            send_id: currentUser.uuid,
            send_name: currentUser.nickname || currentUser.uuid,
            send_avatar: currentUser.avatar || '',
            receive_id: receiveId,
            file_size: '',
            file_type: '',
            file_name: '',
            av_data: ''
        })
    }

    // 发送文件消息
    const sendFileMessage = (sessionId, url, fileName, fileSize, fileType, receiveId) => {
        const currentUser = user.getUser()
        if (!currentUser) {
            console.error('用户未登录')
            return false
        }

        return send({
            session_id: sessionId,
            type: 2, // 文件消息
            content: '',
            url: url,
            send_id: currentUser.uuid,
            send_name: currentUser.nickname || currentUser.uuid,
            send_avatar: currentUser.avatar || '',
            receive_id: receiveId,
            file_size: fileSize,
            file_type: fileType,
            file_name: fileName,
            av_data: ''
        })
    }

    // 发送音视频通话消息
    const sendAVCallMessage = (sessionId, avData, receiveId) => {
        const currentUser = user.getUser()
        if (!currentUser) {
            console.error('用户未登录')
            return false
        }

        return send({
            session_id: sessionId,
            type: 3, // 音视频通话消息
            content: '',
            url: '',
            send_id: currentUser.uuid,
            send_name: currentUser.nickname || currentUser.uuid,
            send_avatar: currentUser.avatar || '',
            receive_id: receiveId,
            file_size: '',
            file_type: '',
            file_name: '',
            av_data: JSON.stringify(avData)
        })
    }

    return {
        isConnected,
        connect,
        disconnect,
        send,
        onMessage,
        clearCallbacks,
        sendTextMessage,
        sendFileMessage,
        sendAVCallMessage
    }
}
