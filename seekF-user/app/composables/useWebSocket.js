// 网页共享一条连接，所有调用方观察同一连接状态。
let wsInstance = null
let socketOwner = ''
let messageCallbacks = []
let reconnectAttempts = 0
let pingInterval = null
let reconnectTimer = null
let manuallyClosed = false
let connectionGeneration = 0
const pendingMessages = new Map()
const maxReconnectAttempts = 5
const reconnectInterval = 3000
const pingIntervalTime = 25000
const messageConfirmTimeout = 15000

export const useWebSocket = () => {
    const config = useRuntimeConfig()
    const isConnected = useState('private-chat-connected', () => false)
    const user = useAuthState()
    const ownerId = () => user.getUser()?.uuid || ''
    const sendFailures = useState('private-chat-send-failures', () => ({}))

    // dispatch 只向当前账号的订阅者分发，单个回调失败不影响其他页面。
    const dispatch = data => {
        for (const entry of [...messageCallbacks]) {
            if (entry.owner !== ownerId()) continue
            try { entry.callback(data) } catch (error) { console.error('处理网页消息失败:', error) }
        }
    }

    const stopPing = () => { clearInterval(pingInterval); pingInterval = null }
    const clearReconnect = () => { clearTimeout(reconnectTimer); reconnectTimer = null }

    // clearPending 清理待确认请求的计时器，避免旧账号继续更新发送状态。
    const clearPending = () => {
        for (const job of pendingMessages.values()) clearTimeout(job.timer)
        pendingMessages.clear()
        sendFailures.value = {}
    }

    // restorePending 恢复本标签页当前账号的请求，连接恢复后由用户手动重试。
    const restorePending = () => {
        clearPending()
        try {
            const stored = JSON.parse(sessionStorage.getItem(`private-chat-pending:${ownerId()}`) || '[]')
            for (const message of stored.slice(0, 100)) {
                const request = message.message || message
                if (request.send_id !== ownerId() || typeof request.client_msg_id !== 'string') continue
                pendingMessages.set(request.client_msg_id, {
                    message: request,
                    createdAt: message.createdAt || new Date().toISOString(),
                    failed: true,
                    error: message.error || '消息尚未确认保存，点击重试'
                })
            }
            savePending()
        } catch (error) { console.error('恢复待确认私信失败:', error) }
    }

    const savePending = () => {
        sendFailures.value = Object.fromEntries([...pendingMessages].filter(([, job]) => job.failed).map(([id, job]) => [id, job.error]))
        try { sessionStorage.setItem(`private-chat-pending:${socketOwner}`, JSON.stringify([...pendingMessages.values()].map(({ message, createdAt, error }) => ({ message, createdAt, error })))) }
        catch (error) { console.error('保存待确认私信失败:', error) }
    }

    // failPending 保留失败消息与原请求编号，仅标记失败，不自动重发。
    const failPending = (id, reason) => {
        const job = pendingMessages.get(id)
        if (!job) return
        clearTimeout(job.timer)
        job.failed = true
        job.error = reason
        savePending()
    }

    const send = message => {
        if (socketOwner !== ownerId() || wsInstance?.readyState !== WebSocket.OPEN) return false
        try {
            wsInstance.send(typeof message === 'string' ? message : JSON.stringify(message))
            return true
        } catch (error) {
            console.error('发送网页消息失败:', error)
            return false
        }
    }

    // sendPending 首次发送或手动重试同一请求，超时只显示失败提示。
    const sendPending = id => {
        const job = pendingMessages.get(id)
        if (!job || socketOwner !== ownerId()) return false
        clearTimeout(job.timer)
        job.failed = false
        job.error = ''
        if (!send(job.message)) {
            failPending(id, '连接不可用，消息发送失败，点击重试')
            return false
        }
        const owner = socketOwner
        job.timer = setTimeout(() => {
            if (owner === ownerId() && pendingMessages.get(id) === job) failPending(id, '消息尚未确认保存，点击重试')
        }, messageConfirmTimeout)
        savePending()
        return true
    }

    // retryPrivateMessage 用户点击失败图标后，复用原请求编号和内容重试。
    const retryPrivateMessage = id => {
        if (socketOwner !== ownerId() || wsInstance?.readyState !== WebSocket.OPEN || !pendingMessages.get(id)?.failed) return false
        return sendPending(id)
    }

    // getPendingMessages 切换会话时恢复该用户尚未确认的本地气泡。
    const getPendingMessages = receiveId => socketOwner === ownerId()
        ? [...pendingMessages.values()].filter(job => job.message.receive_id === receiveId).map(job => ({ ...job.message, created_at: job.createdAt }))
        : []

    // reconcilePrivateMessages 用服务端相同的编号规则核对补查结果，避免回显丢失后出现双气泡。
    const reconcilePrivateMessages = async messages => {
        const owner = ownerId()
        if (socketOwner !== owner || !messages.length || !globalThis.crypto?.subtle) return {}
        const savedIds = new Set(messages.map(message => message.uuid))
        const confirmed = {}
        for (const [id, job] of [...pendingMessages]) {
            const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(`${owner}:${id}`))
            if (owner !== ownerId() || socketOwner !== owner) return {}
            const uuid = 'M' + Array.from(new Uint8Array(hash), byte => byte.toString(16).padStart(2, '0')).join('').slice(0, 19)
            if (!savedIds.has(uuid) || pendingMessages.get(id) !== job) continue
            clearTimeout(job.timer)
            pendingMessages.delete(id)
            confirmed[uuid] = id
        }
        savePending()
        return confirmed
    }

    const attemptReconnect = () => {
        if (manuallyClosed || !ownerId() || reconnectTimer || reconnectAttempts >= maxReconnectAttempts) return
        reconnectAttempts++
        reconnectTimer = setTimeout(() => { reconnectTimer = null; connect() }, reconnectInterval)
    }

    // connect 用连接代次隔离旧回调，主动退出后不自动重连。
    const connect = () => {
        const owner = ownerId()
        if (!owner) return
        if (socketOwner === owner && wsInstance && [WebSocket.OPEN, WebSocket.CONNECTING].includes(wsInstance.readyState)) return
        clearReconnect()
        stopPing()
        const generation = ++connectionGeneration
        wsInstance?.close()
        if (socketOwner !== owner) { clearPending(); reconnectAttempts = 0 }
        socketOwner = owner
        restorePending()
        manuallyClosed = false
        isConnected.value = false
        const wsBase = config.public.wsBase || 'ws://localhost:8080/'
        const socket = new WebSocket(`${wsBase}user/ws/login?client_id=${encodeURIComponent(owner)}`)
        wsInstance = socket
        const isCurrent = () => generation === connectionGeneration && wsInstance === socket && owner === ownerId()
        socket.onopen = () => {
            if (!isCurrent()) { socket.close(); return }
            isConnected.value = true
            reconnectAttempts = 0
            pingInterval = setInterval(() => {
                if (!isCurrent()) return
                send({ type: 'ping' })
            }, pingIntervalTime)
            dispatch({ type: 'connection_restored' })
        }
        socket.onmessage = event => {
            if (!isCurrent()) return
            try {
                const data = JSON.parse(event.data)
                if (data.type === 'pong' || data.type === 'connected') return
                if (data.client_msg_id && (data.send_id === owner || ['error', 'message_deleted', 'message_recalled'].includes(data.type))) {
                    if (data.type === 'error') failPending(data.client_msg_id, data.message || '私聊消息发送失败，点击重试')
                    else {
                        clearTimeout(pendingMessages.get(data.client_msg_id)?.timer)
                        pendingMessages.delete(data.client_msg_id)
                        savePending()
                    }
                }
                dispatch(data)
            } catch (error) { console.error('解析网页消息失败:', error) }
        }
        socket.onclose = () => {
            if (!isCurrent()) return
            isConnected.value = false
            wsInstance = null
            stopPing()
            for (const [id, job] of pendingMessages) {
                if (!job.failed) failPending(id, '连接已断开，消息尚未确认保存，点击重试')
            }
            attemptReconnect()
        }
        socket.onerror = () => { if (isCurrent()) socket.close() }
    }

    // disconnect 使旧连接与定时器失效，退出登录时通知服务端清理该用户连接。
    const disconnect = async () => {
        const owner = socketOwner
        manuallyClosed = true
        ++connectionGeneration
        clearReconnect()
        stopPing()
        wsInstance?.close()
        wsInstance = null
        isConnected.value = false
        if (socketOwner) {
            try { sessionStorage.removeItem(`private-chat-pending:${socketOwner}`) } catch {}
        }
        clearPending()
        socketOwner = ''
        reconnectAttempts = 0
        if (owner && owner === ownerId()) {
            try { await useApi$('/user/ws/logout', { method: 'POST' }) }
            catch (error) { console.error('清理网页连接失败:', error) }
        }
    }

    const onMessage = callback => {
        const entry = { callback, owner: ownerId() }
        messageCallbacks.push(entry)
        const remove = () => { messageCallbacks = messageCallbacks.filter(item => item !== entry) }
        if (getCurrentInstance()) onUnmounted(remove)
        return remove
    }
    const clearCallbacks = () => { messageCallbacks = [] }
    const removeMessageCallback = callback => { messageCallbacks = messageCallbacks.filter(entry => entry.callback !== callback) }

    // sendPrivateMessage 返回请求编号，网页用它关联临时消息与保存结果。
    const sendPrivateMessage = message => {
        if (!isConnected.value || socketOwner !== ownerId() || pendingMessages.size >= 100) return false
        const id = Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('')
        const request = { ...message, send_id: ownerId(), client_msg_id: id }
        pendingMessages.set(id, { message: request, createdAt: new Date().toISOString(), failed: false, error: '' })
        sendPending(id)
        return id
    }
    const sendTextMessage = (sessionId, content, receiveId) => sendPrivateMessage({ session_id: sessionId, type: 0, content, receive_id: receiveId })
    const sendFileMessage = (sessionId, url, fileName, fileSize, fileType, receiveId) => sendPrivateMessage({ session_id: sessionId, type: 2, url, file_name: fileName, file_size: fileSize, file_type: fileType, receive_id: receiveId })

    return { isConnected, sendFailures, connect, disconnect, send, onMessage, clearCallbacks, removeMessageCallback, sendTextMessage, sendFileMessage, retryPrivateMessage, getPendingMessages, reconcilePrivateMessages }
}
