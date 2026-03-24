<template>
  <div class="flex h-full bg-gray-100">
    <!-- 左侧：搜索栏 + 联系人/会话列表 -->
    <aside class="w-80 bg-white border-r border-gray-200 h-full flex flex-col flex-shrink-0 pr-3">
      <!-- 顶部搜索栏 -->
      <SearchBar />

      <!-- 联系人/会话列表 -->
      <div class="flex-1 overflow-y-auto">
        <div v-if="chatList.length === 0" class="p-8 text-center text-gray-400">
          暂无会话
        </div>
        <div
          v-for="(item, index) in chatList"
          :key="item.sessionId"
          class="flex items-center gap-3 px-3 py-3 hover:bg-gray-50 cursor-pointer transition-colors border-b border-gray-100"
          :class="{ 'bg-gray-100': activeIndex === index }"
          @click="selectSession(index)"
        >
          <!-- 头像 -->
          <el-avatar :size="48" :src="item.avatar" class="flex-shrink-0">
            {{ item.name ? item.name.charAt(0) : '?' }}
          </el-avatar>
          <!-- 消息内容 -->
          <div class="flex-1 min-w-0">
            <div class="flex justify-between items-start">
              <h3 class="font-medium text-sm truncate">{{ item.name }}</h3>
              <span class="text-xs text-gray-400">{{ item.time }}</span>
            </div>
            <p class="text-xs text-gray-500 truncate">{{ item.lastMsg }}</p>
          </div>
          <!-- 未读红点 -->
          <div v-if="item.unread" class="w-5 h-5 rounded-full bg-red-500 text-white text-xs flex items-center justify-center">
            {{ item.unread }}
          </div>
        </div>
      </div>
    </aside>

    <!-- 右侧：聊天窗口 -->
    <main class="flex-1 flex flex-col bg-[#f3f4f6] overflow-hidden">
      <!-- 未选择会话时的占位 -->
      <div v-if="activeIndex === -1" class="flex-1 flex flex-col items-center justify-center text-gray-400">
        <Icon name="uil:comment-alt" class="text-6xl mb-4" />
        <p class="text-lg">选择一个会话开始聊天</p>
        <p class="text-sm mt-2">seekF 消息</p>
      </div>

      <!-- 已选择会话的聊天界面 -->
      <div v-else class="flex flex-col h-full">
        <!-- 聊天头部 -->
        <div class="bg-white border-b border-gray-200 p-3 flex items-center gap-3">
          <el-avatar :size="40" :src="currentChat.avatar">
            {{ currentChat.name ? currentChat.name.charAt(0) : '?' }}
          </el-avatar>
          <h3 class="font-medium flex-1">{{ currentChat.name }}</h3>
          <!-- WebSocket 连接状态 -->
          <div class="flex items-center gap-2 text-xs mr-2">
            <span
              class="w-2 h-2 rounded-full"
              :class="ws.isConnected ? 'bg-green-500' : 'bg-red-500'"
            ></span>
            <span :class="ws.isConnected ? 'text-green-600' : 'text-red-600'">
              {{ ws.isConnected ? '已连接' : '未连接' }}
            </span>
          </div>
          <div class="flex gap-4 text-gray-500">
            <button><Icon name="uil:search" /></button>
            <button><Icon name="uil:ellipsis-h" /></button>
          </div>
        </div>

        <!-- 聊天内容区 -->
        <div class="flex-1 p-6 overflow-y-auto space-y-4">
          <div v-if="messageList.length === 0" class="flex items-center justify-center h-full text-gray-400">
            <p>暂无消息</p>
          </div>
          <div
            v-for="msg in messageList"
            :key="msg.messageId"
            class="flex items-start gap-3"
            :class="{ 'justify-end': msg.isSelf }"
          >
            <!-- 对方头像 -->
            <el-avatar v-if="!msg.isSelf" :size="32" :src="msg.avatar" class="flex-shrink-0">
              {{ msg.senderName ? msg.senderName.charAt(0) : '?' }}
            </el-avatar>
            <!-- 消息内容 -->
            <div
              class="rounded-lg px-4 py-2 max-w-[60%] shadow-sm"
              :class="msg.isSelf ? 'bg-[#D9FDD3]' : 'bg-white'"
            >
              <p class="text-sm">{{ msg.content }}</p>
              <p class="text-xs text-gray-400 text-right mt-1">{{ formatTime(msg.sendTime) }}</p>
            </div>
            <!-- 自己头像 -->
            <el-avatar v-if="msg.isSelf" :size="32" :src="currentUserAvatar" class="flex-shrink-0">
              我
            </el-avatar>
          </div>
        </div>

        <!-- 输入框区域 -->
        <div class="bg-white p-3 border-t border-gray-200">
          <div class="flex items-center gap-3 mb-3">
            <button class="text-gray-500"><Icon name="uil:smile" /></button>
            <button class="text-gray-500"><Icon name="uil:paperclip" /></button>
            <button class="text-gray-500"><Icon name="uil:mic" /></button>
          </div>
          <div class="flex gap-3">
            <textarea
              v-model="inputMessage"
              placeholder="请输入消息..."
              class="flex-1 border border-gray-300 rounded-lg p-2 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm resize-none"
              rows="2"
              @keydown.enter.prevent="sendMessage"
            ></textarea>
            <button
              class="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              :disabled="!ws.isConnected"
              @click="sendMessage"
            >
              发送
            </button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
// 获取路由和URL参数
const route = useRoute()
const router = useRouter()

// WebSocket
const ws = useWebSocket()

// 当前用户信息
const currentUserAvatar = ref('')
const currentUserId = ref('')

// 会话列表
const chatList = ref([])
// 当前选中的会话索引
const activeIndex = ref(-1)
// 消息列表
const messageList = ref([])
// 输入的消息
const inputMessage = ref('')

// 当前选中的聊天对象
const currentChat = computed(() => {
  if (activeIndex.value === -1) return {}
  return chatList.value[activeIndex.value]
})

// 格式化时间
const formatTime = (time) => {
  if (!time) return ''
  const date = new Date(time)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()

  if (isToday) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else {
    return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
  }
}

// 获取当前用户信息
const getCurrentUserInfo = async () => {
  try {
    const data = await useApi$('/user/userinfo/getMyInfo', {
      method: 'POST'
    })
    if (data && data.code === 200) {
      currentUserAvatar.value = data.data.avatar
      currentUserId.value = data.data.uuid
    }
  } catch (error) {
    console.error('获取用户信息失败:', error)
  }
}

// 加载会话列表
const loadSessionList = async () => {
  try {
    const data = await useApi$('/user/session/getSessionList', {
      method: 'POST'
    })
    if (data && data.code === 200) {
      const sessions = data.data || []
      chatList.value = sessions.map(session => ({
        sessionId: session.sessionId,
        id: session.id,
        name: session.name,
        avatar: session.avatar,
        lastMsg: '点击开始聊天',
        time: '',
        unread: 0
      }))

      // 如果有URL参数，尝试选中对应会话
      const { session_id, receive_id } = route.query
      if (session_id || receive_id) {
        selectSessionByParams(session_id, receive_id)
      }
    } else {
      ElMessage.error(data?.message || '获取会话列表失败')
    }
  } catch (error) {
    console.error('获取会话列表失败:', error)
    ElMessage.error('获取会话列表失败')
  }
}

// 根据参数选中会话
const selectSessionByParams = (sessionId, receiveId) => {
  let index = -1

  if (sessionId) {
    // 根据sessionId查找
    index = chatList.value.findIndex(item => item.sessionId === sessionId)
  }

  if (index === -1 && receiveId) {
    // 根据receiveId查找
    index = chatList.value.findIndex(item => item.id === receiveId)
  }

  if (index !== -1) {
    selectSession(index)
  } else if (receiveId) {
    // 如果找不到对应会话，但传了receiveId，说明是新创建的会话，需要刷新列表
    loadSessionList()
  }
}

// 选择会话
const selectSession = async (index) => {
  activeIndex.value = index
  const session = chatList.value[index]
  if (!session) return

  // 加载消息列表
  await loadMessageList(session.id)

  // 清除URL参数
  if (route.query.session_id || route.query.receive_id) {
    router.replace({ path: '/chat' })
  }
}

// 加载消息列表
const loadMessageList = async (receiveId) => {
  try {
    if (!receiveId) return

    const isGroup = receiveId.startsWith('G')
    const url = isGroup
      ? '/user/message/getGroupMessageList'
      : '/user/message/getUserMessageList'

    const body = isGroup
      ? { group_id: receiveId }
      : { user_one_id: currentUserId.value, user_two_id: receiveId }

    const data = await useApi$(url, {
      method: 'POST',
      body
    })

    if (data && data.code === 200) {
      const messages = data.data || []
      messageList.value = messages.map(msg => ({
        messageId: msg.messageId || msg.uuid,
        content: msg.content,
        senderName: msg.senderName,
        avatar: msg.avatar,
        sendTime: msg.sendTime || msg.createdAt,
        isSelf: msg.senderId === currentUserId.value || msg.isSelf
      }))
    } else {
      ElMessage.error(data?.message || '获取消息列表失败')
    }
  } catch (error) {
    console.error('获取消息列表失败:', error)
    ElMessage.error('获取消息列表失败')
  }
}

// 发送消息
const sendMessage = async () => {
  if (!inputMessage.value.trim()) {
    ElMessage.warning('请输入消息内容')
    return
  }

  if (activeIndex.value === -1) {
    ElMessage.warning('请先选择一个会话')
    return
  }

  const session = currentChat.value
  if (!session) return

  // 使用 WebSocket 发送消息
  const success = ws.sendTextMessage(
    session.sessionId,
    inputMessage.value.trim(),
    session.id
  )

  if (success) {
    // 添加到本地消息列表
    messageList.value.push({
      messageId: Date.now().toString(),
      content: inputMessage.value.trim(),
      senderName: '我',
      avatar: currentUserAvatar.value,
      sendTime: new Date().toISOString(),
      isSelf: true
    })

    // 更新会话列表的最后消息
    session.lastMsg = inputMessage.value.trim()
    session.time = '刚刚'

    inputMessage.value = ''
  } else {
    ElMessage.error('发送失败，请检查网络连接')
  }
}

// 处理收到的 WebSocket 消息
const handleWebSocketMessage = (data) => {
  console.log('收到消息:', data)

  // 处理不同类型的消息
  if (typeof data === 'object') {
    // 文本消息
    if (data.type === 0) {
      // 检查是否是当前会话的消息
      const currentSession = currentChat.value
      if (currentSession && data.session_id === currentSession.sessionId) {
        messageList.value.push({
          messageId: data.uuid || Date.now().toString(),
          content: data.content,
          senderName: data.send_name,
          avatar: data.send_avatar,
          sendTime: new Date().toISOString(),
          isSelf: data.send_id === currentUserId.value
        })
      } else {
        // 更新其他会话的未读消息数
        const sessionIndex = chatList.value.findIndex(item => item.sessionId === data.session_id)
        if (sessionIndex !== -1) {
          chatList.value[sessionIndex].unread++
          chatList.value[sessionIndex].lastMsg = data.content
          chatList.value[sessionIndex].time = '刚刚'
        }
      }
    }
  }
}

// 页面加载时
onMounted(async () => {
  await getCurrentUserInfo()
  await loadSessionList()

  // 注册消息监听
  ws.onMessage(handleWebSocketMessage)

  // 如果 WebSocket 未连接，尝试连接
  if (!ws.isConnected.value) {
    ws.connect()
  }
})
</script>

<style scoped>
/* 滚动条美化，模拟微信风格 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
</style>
