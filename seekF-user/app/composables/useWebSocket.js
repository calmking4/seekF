// WebSocket 连接管理
let wsInstance = null
let messageCallbacks = []
let avCallCallbacks = [] // 音视频通话回调
let reconnectAttempts = 0
let pingInterval = null // 心跳定时器
const maxReconnectAttempts = 5
const reconnectInterval = 3000
const pingIntervalTime = 25000 // 25秒发送一次心跳
export const useWebSocket = () => {
    const config = useRuntimeConfig()
    const isConnected = ref(false)
    const user = useAuthState()

    // 自动清理回调的辅助函数
    const autoCleanupOnUnmount = (callback, callbackArray) => {
        // 检查是否在组件上下文中
        if (getCurrentInstance()) {
            onUnmounted(() => {
                const index = callbackArray.indexOf(callback)
                if (index > -1) {
                    callbackArray.splice(index, 1)
                }
            })
        }
    }

    // WebSocket URL
    const getWsUrl = () => {
        const wsBase = config.public.wsBase || 'ws://localhost:8080/'
        const userUuid = user.getUser()?.uuid || ''
        return `${wsBase}user/ws/login?client_id=${userUuid}`
    }

    // 启动心跳检测
    const startPing = () => {
        stopPing()
        pingInterval = setInterval(() => {
            if (wsInstance?.readyState === WebSocket.OPEN) {
                send({ type: 'ping' })
            }
        }, pingIntervalTime)
    }

    // 停止心跳检测
    const stopPing = () => {
        if (pingInterval) {
            clearInterval(pingInterval)
            pingInterval = null
        }
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
                startPing() // 启动心跳
            }

            wsInstance.onmessage = (event) => {
                console.log('收到 WebSocket 消息:', event.data)
                try {
                    const data = JSON.parse(event.data)
                    // 处理心跳 pong 响应
                    if (data.type === 'pong') {
                        return
                    }
                    // 调用所有注册的回调函数
                    messageCallbacks.forEach(callback => callback(data))
                    // 如果是音视频消息，调用音视频回调
                    if (data.type === 3) {
                        avCallCallbacks.forEach(callback => callback(data))
                    }
                } catch (e) {
                    console.log('WebSocket 收到非 JSON 消息:', event.data)
                    messageCallbacks.forEach(callback => callback(event.data))
                }
            }

            wsInstance.onclose = () => {
                console.log('WebSocket 连接关闭')
                isConnected.value = false
                wsInstance = null
                stopPing() // 停止心跳
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
    const disconnect = async () => {
        stopPing() // 停止心跳
        if (wsInstance) {
            wsInstance.close()
            wsInstance = null
            isConnected.value = false
            console.log('WebSocket 已断开')

            // 调用WebSocket登出接口
            try {
                await useApi$('/user/ws/logout', {
                    method: 'POST'
                })
                console.log('WebSocket登出接口调用成功')
            } catch (error) {
                console.error('WebSocket登出接口调用异常:', error)
            }
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

    // 注册消息回调（组件卸载时自动清理）
    const onMessage = (callback) => {
        messageCallbacks.push(callback)
        // 自动清理
        autoCleanupOnUnmount(callback, messageCallbacks)
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
        avCallCallbacks = []
    }

    // 移除指定的消息回调
    const removeMessageCallback = (callback) => {
        const index = messageCallbacks.indexOf(callback)
        if (index > -1) {
            messageCallbacks.splice(index, 1)
        }
    }

    // 注册音视频通话回调（组件卸载时自动清理）
    const onAVCall = (callback) => {
        avCallCallbacks.push(callback)
        // 自动清理
        autoCleanupOnUnmount(callback, avCallCallbacks)
        // 返回取消注册的函数
        return () => {
            const index = avCallCallbacks.indexOf(callback)
            if (index > -1) {
                avCallCallbacks.splice(index, 1)
            }
        }
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
        onAVCall,
        clearCallbacks,
        removeMessageCallback,
        sendTextMessage,
        sendFileMessage,
        sendAVCallMessage
    }
}
