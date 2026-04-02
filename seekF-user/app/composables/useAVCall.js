// 音视频通话管理 - 使用模块级变量实现单例
let peerConnection = null
let callTimer = null

export const useAVCall = () => {
  // 使用 useState 实现全局共享的响应式状态
  const callStatus = useState('avCallStatus', () => 'idle') // idle, calling, ringing, connected
  const isIncoming = useState('avCallIsIncoming', () => false)
  const callerInfo = useState('avCallCallerInfo', () => null)
  const sessionInfo = useState('avCallSessionInfo', () => null)
  const localStream = useState('avCallLocalStream', () => null)
  const remoteStream = useState('avCallRemoteStream', () => null)
  const callDuration = useState('avCallDuration', () => 0)
  const isMuted = useState('avCallIsMuted', () => false)
  const isCameraOff = useState('avCallIsCameraOff', () => false)

  // WebSocket
  const ws = useWebSocket()
  const user = useAuthState()

  // ICE服务器配置（使用免费STUN服务器）
  const iceServers = {
    iceServers: [
      { urls: 'stun:stun.l.google.com:19302' },
      { urls: 'stun:stun1.l.google.com:19302' },
      { urls: 'stun:stun2.l.google.com:19302' },
      { urls: 'stun:stun3.l.google.com:19302' },
      { urls: 'stun:stun4.l.google.com:19302' }
    ]
  }

  // 创建RTCPeerConnection
  const createPeerConnection = () => {
    console.log('创建RTCPeerConnection')
    peerConnection = new RTCPeerConnection(iceServers)

    // 添加本地流到连接
    if (localStream.value) {
      localStream.value.getTracks().forEach(track => {
        console.log('添加本地轨道:', track.kind)
        peerConnection.addTrack(track, localStream.value)
      })
    }

    // 处理远程流
    peerConnection.ontrack = (event) => {
      console.log('收到远程轨道:', event.track.kind)
      if (event.streams && event.streams[0]) {
        remoteStream.value = event.streams[0]
      }
    }

    // 处理ICE候选
    peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        console.log('发送ICE候选:', event.candidate)
        sendSignal('ice_candidate', {
          candidate: event.candidate.candidate,
          sdpMLineIndex: event.candidate.sdpMLineIndex,
          sdpMid: event.candidate.sdpMid
        })
      }
    }

    // 连接状态变化
    peerConnection.onconnectionstatechange = () => {
      console.log('连接状态:', peerConnection.connectionState)
      if (peerConnection.connectionState === 'connected') {
        callStatus.value = 'connected'
        startCallTimer()
      } else if (peerConnection.connectionState === 'disconnected' ||
                 peerConnection.connectionState === 'failed') {
        endCall()
      }
    }

    return peerConnection
  }

  // 获取本地媒体流
  const getLocalStream = async () => {
    try {
      // 如果已有本地流，先停止
      if (localStream.value) {
        localStream.value.getTracks().forEach(track => track.stop())
        localStream.value = null
      }

      const stream = await navigator.mediaDevices.getUserMedia({
        video: true,
        audio: true
      })
      localStream.value = stream
      return stream
    } catch (error) {
      console.error('获取媒体流失败:', error)
      // 提供更清晰的错误信息
      if (error.name === 'NotReadableError') {
        throw new Error('设备被占用，请确保没有其他应用或标签页正在使用摄像头/麦克风')
      } else if (error.name === 'NotAllowedError') {
        throw new Error('请允许访问摄像头和麦克风权限')
      } else if (error.name === 'NotFoundError') {
        throw new Error('未找到摄像头或麦克风设备')
      }
      throw error
    }
  }

  // 发送信令消息
  const sendSignal = (type, data = {}) => {
    if (!sessionInfo.value) {
      console.error('会话信息不存在')
      return false
    }

    const avData = {
      message_id: 'SIGNAL',
      type: type,
      ...data
    }

    return ws.sendAVCallMessage(
      sessionInfo.value.sessionId,
      avData,
      sessionInfo.value.receiveId
    )
  }

  // 发起通话
  const startCall = async (sessionId, receiveId, receiveInfo) => {
    try {
      console.log('发起通话:', { sessionId, receiveId, receiveInfo })

      // 保存会话信息
      sessionInfo.value = {
        sessionId,
        receiveId,
        receiveInfo
      }

      // 获取本地媒体流
      await getLocalStream()

      // 创建PeerConnection
      createPeerConnection()

      // 发送通话邀请
      sendSignal('start_call')

      // 设置状态为呼叫中
      callStatus.value = 'calling'
      isIncoming.value = false

      return true
    } catch (error) {
      console.error('发起通话失败:', error)
      endCall()
      return false
    }
  }

  // 接受通话
  const acceptCall = async () => {
    try {
      console.log('接受通话')

      // 获取本地媒体流
      await getLocalStream()

      // 创建PeerConnection
      createPeerConnection()

      // 发送接受通话信号
      sendSignal('accept_call')

      // 等待接收offer
      callStatus.value = 'calling'

      return true
    } catch (error) {
      console.error('接受通话失败:', error)
      // 显示错误信息，不自动拒绝
      alert(error.message || '无法接受通话，请检查设备权限')
      // 重置状态但不发送拒绝信号
      resetCall()
      return false
    }
  }

  // 拒绝通话
  const rejectCall = () => {
    console.log('拒绝通话')
    sendSignal('reject_call')
    resetCall()
  }

  // 挂断通话
  const endCall = () => {
    console.log('挂断通话')
    if (callStatus.value !== 'idle') {
      sendSignal('end_call')
    }
    resetCall()
  }

  // 重置通话状态
  const resetCall = () => {
    // 停止计时
    stopCallTimer()

    // 关闭PeerConnection
    if (peerConnection) {
      peerConnection.close()
      peerConnection = null
    }

    // 停止本地流
    if (localStream.value) {
      localStream.value.getTracks().forEach(track => track.stop())
      localStream.value = null
    }

    // 清空远程流
    remoteStream.value = null

    // 重置状态
    callStatus.value = 'idle'
    isIncoming.value = false
    callerInfo.value = null
    sessionInfo.value = null
    callDuration.value = 0
    isMuted.value = false
    isCameraOff.value = false
  }

  // 处理信令消息
  const handleSignal = async (data) => {
    console.log('处理信令:', data)

    // 检查是否是音视频通话消息
    if (data.type !== 3) return

    let avData
    try {
      avData = typeof data.av_data === 'string' ? JSON.parse(data.av_data) : data.av_data
    } catch (e) {
      console.error('解析av_data失败:', e)
      return
    }

    if (!avData || !avData.type) {
      console.log('不是音视频信令消息')
      return
    }

    console.log('信令类型:', avData.type)

    switch (avData.type) {
      case 'start_call':
        // 收到通话邀请
        isIncoming.value = true
        callerInfo.value = {
          id: data.send_id,
          name: data.send_name,
          avatar: data.send_avatar
        }
        sessionInfo.value = {
          sessionId: data.session_id,
          receiveId: data.send_id
        }
        callStatus.value = 'ringing'
        break

      case 'accept_call':
        // 对方接受通话，创建offer
        console.log('对方接受通话，创建offer')
        if (peerConnection) {
          try {
            const offer = await peerConnection.createOffer()
            await peerConnection.setLocalDescription(offer)
            sendSignal('offer', {
              sdp: {
                type: offer.type,
                sdp: offer.sdp
              }
            })
          } catch (error) {
            console.error('创建offer失败:', error)
            endCall()
          }
        }
        break

      case 'reject_call':
        // 对方拒绝通话
        console.log('对方拒绝通话')
        alert('对方已拒绝通话')
        resetCall()
        break

      case 'offer':
        // 收到offer，创建answer
        console.log('收到offer')
        if (avData.sdp) {
          try {
            // 如果没有peerConnection，先创建（可能是因为acceptCall时设备被占用）
            if (!peerConnection) {
              console.log('peerConnection不存在，尝试创建')
              // 如果没有本地流，尝试获取
              if (!localStream.value) {
                await getLocalStream()
              }
              createPeerConnection()
            }
            await peerConnection.setRemoteDescription(
              new RTCSessionDescription(avData.sdp)
            )
            const answer = await peerConnection.createAnswer()
            await peerConnection.setLocalDescription(answer)
            sendSignal('answer', {
              sdp: {
                type: answer.type,
                sdp: answer.sdp
              }
            })
          } catch (error) {
            console.error('处理offer失败:', error)
            endCall()
          }
        }
        break

      case 'answer':
        // 收到answer
        console.log('收到answer')
        if (peerConnection && avData.sdp) {
          try {
            await peerConnection.setRemoteDescription(
              new RTCSessionDescription(avData.sdp)
            )
          } catch (error) {
            console.error('处理answer失败:', error)
            endCall()
          }
        }
        break

      case 'ice_candidate':
        // 收到ICE候选
        console.log('收到ICE候选')
        if (peerConnection && avData.candidate) {
          try {
            await peerConnection.addIceCandidate(
              new RTCIceCandidate(avData.candidate)
            )
          } catch (error) {
            console.error('添加ICE候选失败:', error)
          }
        }
        break

      case 'end_call':
        // 对方挂断
        console.log('对方挂断通话')
        alert('通话已结束')
        resetCall()
        break
    }
  }

  // 切换静音
  const toggleMute = () => {
    if (localStream.value) {
      const audioTrack = localStream.value.getAudioTracks()[0]
      if (audioTrack) {
        audioTrack.enabled = !audioTrack.enabled
        isMuted.value = !audioTrack.enabled
      }
    }
  }

  // 切换摄像头
  const toggleCamera = () => {
    if (localStream.value) {
      const videoTrack = localStream.value.getVideoTracks()[0]
      if (videoTrack) {
        videoTrack.enabled = !videoTrack.enabled
        isCameraOff.value = !videoTrack.enabled
      }
    }
  }

  // 开始通话计时
  const startCallTimer = () => {
    callDuration.value = 0
    callTimer = setInterval(() => {
      callDuration.value++
    }, 1000)
  }

  // 停止通话计时
  const stopCallTimer = () => {
    if (callTimer) {
      clearInterval(callTimer)
      callTimer = null
    }
  }

  // 格式化通话时长
  const formatDuration = computed(() => {
    const hours = Math.floor(callDuration.value / 3600)
    const minutes = Math.floor((callDuration.value % 3600) / 60)
    const seconds = callDuration.value % 60
    if (hours > 0) {
      return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
    }
    return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
  })

  return {
    callStatus,
    isIncoming,
    callerInfo,
    localStream,
    remoteStream,
    isMuted,
    isCameraOff,
    callDuration,
    formatDuration,
    startCall,
    acceptCall,
    rejectCall,
    endCall,
    handleSignal,
    toggleMute,
    toggleCamera
  }
}
