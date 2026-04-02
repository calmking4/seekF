<template>
  <div class="flex h-screen bg-white">
    <!-- 左侧侧边栏 -->
    <aside class="w-[200px] bg-white border-r border-gray-100 flex flex-col flex-shrink-0">
      <!-- 侧边栏头部 Logo -->
      <div class="px-5 py-4 mb-4">
        <div class="text-xl font-bold text-[#60a5fa]">seekF</div>
      </div>

      <!-- 侧边栏导航 -->
      <nav class="flex-1 px-2">
        <NuxtLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-3 py-3 rounded-lg mb-1 transition-colors hover:bg-gray-50"
          :class="{ 'bg-blue-50 text-[#0073ff]': route.path === item.path }"
        >
          <Icon :name="item.icon" class="text-lg" />
          <span class="font-medium">{{ item.label }}</span>
        </NuxtLink>
      </nav>

      <!-- 侧边栏底部 -->
      <div class="px-2 pb-4">
        <button
          @click="logout"
          class="flex items-center gap-3 w-full px-3 py-3 rounded-lg border border-gray-200 hover:bg-gray-50 transition-colors"
        >
          <Icon name="uil:signout" class="text-lg" />
          <span class="font-medium">退出登录</span>
        </button>
      </div>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="flex-1 h-full overflow-y-auto">
      <slot />
    </main>

    <!-- 来电弹窗 - 全局 -->
    <AVCallDialog
      :visible="avCall.callStatus.value === 'ringing' && avCall.isIncoming.value"
      :caller-info="avCall.callerInfo.value || { id: '', name: '', avatar: '' }"
      @accept="acceptCall"
      @reject="rejectCall"
    />

    <!-- 通话界面 - 全局 -->
    <AVCallOverlay
      :visible="avCall.callStatus.value === 'calling' || avCall.callStatus.value === 'connected'"
      :local-stream="avCall.localStream.value"
      :remote-stream="avCall.remoteStream.value"
      :remote-name="avCall.callerInfo.value?.name || ''"
      :remote-avatar="avCall.callerInfo.value?.avatar || ''"
      :format-duration="avCall.formatDuration.value"
      :is-muted="avCall.isMuted.value"
      :is-camera-off="avCall.isCameraOff.value"
      @end-call="endCall"
      @toggle-mute="avCall.toggleMute"
      @toggle-camera="avCall.toggleCamera"
    />
  </div>
</template>

<script setup>

// 引入路由对象，用于判断当前激活的导航项
const route = useRoute()

// 引入WebSocket和通话管理
const ws = useWebSocket()
const avCall = useAVCall()

// 侧边栏导航数据
const navItems = [
  { path: '/chat', label: '消息', icon: 'uil:comment-alt' },
  { path: '/contact', label: '联系人', icon: 'uil:users-alt' },
  { path: '/discover', label: '发现', icon: 'uil:plus-square' },
  { path: '/aichat', label: 'AIChat', icon: 'uil:robot' },
  { path: '/my', label: '我', icon: 'uil:user' }
]

// 处理音视频通话消息
const handleAVCallMessage = (data) => {
  console.log('收到音视频通话消息:', data)
  avCall.handleSignal(data)
}

// 接受通话
const acceptCall = () => {
  avCall.acceptCall()
}

// 拒绝通话
const rejectCall = () => {
  avCall.rejectCall()
}

// 挂断通话
const endCall = () => {
  avCall.endCall()
}

// 页面加载时连接WebSocket
onMounted(async () => {
  console.log('WebSocket 连接状态:', ws.isConnected.value)
  
  if (!ws.isConnected.value) {
    console.log('WebSocket 未连接，尝试连接...')
    ws.connect()
  }

  // 注册音视频通话回调
  ws.onAVCall(handleAVCallMessage)
})

const logout = async () => {
  try {
    // 先断开WebSocket连接
    try {
      await ws.disconnect();
    } catch (wsErr) {
      console.error('WebSocket断开失败:', wsErr);
    }
    
    // 向后端发送退出登录请求
    const res = await useApi$('/user/logout', {
      method: 'POST'
    });
    
    if (res && res.code === 200) {
      ElMessage.success(res.message || '退出登录成功');
    } else {
      console.error('登出请求失败:', res);
    }
  } catch (err) {
    console.error('登出请求异常:', err);
    ElMessage.error(err?.data?.message || err?.message || '退出登录失败');
    // 即使后端请求失败，也要清除本地信息
  } finally {
    // 结束通话
    avCall.endCall();
    // 清除回调
    ws.clearCallbacks();
    // 清除用户信息和token
    const authState = useAuthState();
    authState.clear();
    
    // 跳转到登录页
    navigateTo('/login');
  }
}
</script>