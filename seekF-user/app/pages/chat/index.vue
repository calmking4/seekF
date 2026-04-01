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
    <main class="flex-1 flex flex-col bg-[#f3f4f6] h-full overflow-hidden">
      <!-- 未选择会话时的占位 -->
      <div v-if="activeIndex === -1" class="flex-1 flex flex-col items-center justify-center text-gray-400">
        <Icon name="uil:comment-alt" class="text-6xl mb-4" />
        <p class="text-lg">选择一个会话开始聊天</p>
        <p class="text-sm mt-2">seekF 消息</p>
      </div>

      <!-- 已选择会话的聊天界面 -->
      <template v-else>
        <!-- 聊天头部 -->
        <div class="bg-white border-b border-gray-200 p-3 flex items-center gap-3 flex-shrink-0">
          <div class="flex flex-col items-center">
            <span class="text-xs mb-1">{{ currentChat.name }}</span>
            <el-avatar :size="40" :src="currentChat.avatar">
              {{ currentChat.name ? currentChat.name.charAt(0) : '?' }}
            </el-avatar>
          </div>
          <div class="flex-1"></div>
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

        <!-- 聊天内容区 - 固定高度 -->
        <div class="flex-1 min-h-0 overflow-hidden">
          <el-scrollbar
            ref="scrollbarRef"
            class="h-full"
            @end-reached="handleEndReached"
          >
            <div class="p-4 space-y-3">
              <div v-if="messageList.length === 0" class="flex items-center justify-center py-20 text-gray-400">
                <p>暂无消息</p>
              </div>

              <!-- 加载更多 - 放在顶部 -->
              <div v-if="hasMore && messageList.length > 0" class="text-center py-2">
                <span v-if="loadingMore" class="text-gray-400 text-sm">加载中...</span>
                <button v-else class="text-blue-500 text-sm hover:underline" @click="loadMoreMessages">
                  加载更多消息
                </button>
              </div>

              <!-- 消息列表 - 倒序显示，最新消息在底部 -->
              <div
                v-for="msg in messageList"
                :key="msg.messageId"
                class="flex items-center gap-3"
                :class="{ 'justify-end': msg.isSelf }"
              >
                <!-- 对方消息 -->
                <div v-if="!msg.isSelf" class="flex-shrink-0">
                  <div class="flex flex-col items-center">
                    <span class="text-xs mb-1 text-gray-500">{{ msg.senderName }}</span>
                    <el-avatar :size="48" :src="msg.avatar">
                      {{ msg.senderName ? msg.senderName.charAt(0) : '?' }}
                    </el-avatar>
                  </div>
                </div>
                <!-- 消息内容 -->
                <div
                  class="rounded-lg px-4 py-2 max-w-[60%] shadow-sm flex-shrink-0"
                  :class="msg.isSelf ? 'bg-[#D9FDD3]' : 'bg-white'"
                >
                  <p class="text-sm">{{ msg.content }}</p>
                  <p class="text-xs text-gray-400 text-right mt-1">{{ formatTime(msg.sendTime) }}</p>
                </div>
                <!-- 自己消息 -->
                <div v-if="msg.isSelf" class="flex-shrink-0">
                  <div class="flex flex-col items-center">
                    <span class="text-xs mb-1 text-gray-500">我</span>
                    <el-avatar :size="48" :src="currentUserAvatar">
                      我
                    </el-avatar>
                  </div>
                </div>
              </div>
            </div>
          </el-scrollbar>
        </div>

        <!-- 输入框区域 -->
        <div class="bg-white p-3 border-t border-gray-200 flex-shrink-0">
          <div class="flex items-center gap-3 mb-3">
            <button class="text-gray-500"><Icon name="uil:smile" /></button>
            <button class="text-gray-500"><Icon name="uil:paperclip" /></button>
            <button class="text-gray-500"><Icon name="fluent:call-24-regular" /></button>
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
      </template>
    </main>
  </div>
</template>

<script setup>
const route = useRoute()
const router = useRouter()

const ws = useWebSocket()

const currentUserAvatar = ref('')
const currentUserId = ref('')

const chatList = ref([])
const activeIndex = ref(-1)
const messageList = ref([])
const inputMessage = ref('')

const currentChat = computed(() => {
  if (activeIndex.value === -1) return {}
  return chatList.value[activeIndex.value]
})

const scrollbarRef = ref()
const hasMore = ref(true)
const loadingMore = ref(false)
const currentPage = ref(1)
const pageSize = 20
const totalMessages = ref(0)

const scrollToBottom = () => {
  nextTick(() => {
    if (scrollbarRef.value?.wrapRef) {
      const wrap = scrollbarRef.value.wrapRef
      scrollbarRef.value.setScrollTop(wrap.scrollHeight)
    }
  })
}

const handleEndReached = (direction) => {
  if (direction === 'top') {
    loadMoreMessages()
  }
}

const formatTime = (time) => {
  if (!time) return ''
  const date = new Date(time)
  return date.toLocaleString('zh-CN', { 
    year: 'numeric', 
    month: '2-digit', 
    day: '2-digit', 
    hour: '2-digit', 
    minute: '2-digit' 
  })
}

const getCurrentUserInfo = async () => {
  try {
    const data = await useApi$('/user/userinfo/getMyInfo', { method: 'POST' })
    if (data?.code === 200) {
      currentUserAvatar.value = data.data.avatar
      currentUserId.value = data.data.uuid
    }
  } catch (error) {
    console.error('获取用户信息失败:', error)
  }
}

const loadSessionList = async () => {
  try {
    const data = await useApi$('/user/session/getSessionList', { method: 'POST' })
    if (data?.code === 200) {
      const sessions = data.data || []
      chatList.value = sessions.map(session => ({
        sessionId: session.session_id,
        id: session.id,
        name: session.name,
        avatar: session.avatar,
        lastMsg: session.last_message || '点击开始聊天',
        time: session.last_message_at || '',
        unread: 0
      }))

      const { session_id, receive_id } = route.query
      if (session_id || receive_id) {
        selectSessionByParams(session_id, receive_id)
      }
    }
  } catch (error) {
    console.error('获取会话列表失败:', error)
  }
}

const selectSessionByParams = (sessionId, receiveId) => {
  let index = -1
  if (sessionId) {
    index = chatList.value.findIndex(item => item.sessionId === sessionId)
  }
  if (index === -1 && receiveId) {
    index = chatList.value.findIndex(item => item.id === receiveId)
  }
  if (index !== -1) {
    selectSession(index)
  } else if (receiveId) {
    loadSessionList()
  }
}

const selectSession = async (index) => {
  activeIndex.value = index
  const session = chatList.value[index]
  if (!session) return

  currentPage.value = 1
  hasMore.value = true
  loadingMore.value = false
  messageList.value = []

  await loadMessageList(session.id)
  scrollToBottom()

  if (route.query.session_id || route.query.receive_id) {
    router.replace({ path: '/chat' })
  }
}

const loadMessageList = async (receiveId, page = 1) => {
  try {
    if (!receiveId) return

    const isGroup = receiveId.startsWith('G')
    const url = isGroup
      ? '/user/message/getGroupMessageList'
      : '/user/message/getUserMessageList'

    const body = isGroup
      ? { group_id: receiveId, page, page_size: pageSize }
      : { user_one_id: currentUserId.value, user_two_id: receiveId, page, page_size: pageSize }

    const data = await useApi$(url, { method: 'POST', body })

    if (data?.code === 200) {
      const list = data.data?.list || []
      totalMessages.value = data.data?.total || 0

      const messages = list.map(msg => ({
        messageId: msg.uuid || msg.messageId,
        content: msg.content,
        senderName: msg.send_name || msg.senderName,
        avatar: msg.send_avatar || msg.avatar,
        sendTime: msg.created_at || msg.createdAt,
        isSelf: msg.send_id === currentUserId.value
      }))

      if (page === 1) {
        messageList.value = messages.reverse()
      } else {
        messageList.value = [...messages.reverse(), ...messageList.value]
      }

      hasMore.value = messageList.value.length < totalMessages.value
    }
  } catch (error) {
    console.error('获取消息列表失败:', error)
  }
}

const loadMoreMessages = async () => {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true

  const oldScrollHeight = scrollbarRef.value?.wrapRef?.scrollHeight || 0

  currentPage.value++
  await loadMessageList(currentChat.value.id, currentPage.value)

  nextTick(() => {
    if (scrollbarRef.value?.wrapRef) {
      const newScrollHeight = scrollbarRef.value.wrapRef.scrollHeight
      scrollbarRef.value.setScrollTop(newScrollHeight - oldScrollHeight)
    }
  })

  loadingMore.value = false
}

const sendMessage = async () => {
  if (!inputMessage.value.trim()) return
  if (activeIndex.value === -1) return

  const session = currentChat.value
  if (!session) return

  const success = ws.sendTextMessage(
    session.sessionId,
    inputMessage.value.trim(),
    session.id
  )

  if (success) {
    const tempId = 'temp_' + Date.now()
    messageList.value.push({
      messageId: tempId,
      content: inputMessage.value.trim(),
      senderName: '我',
      avatar: currentUserAvatar.value,
      sendTime: new Date().toISOString(),
      isSelf: true
    })

    session.lastMsg = inputMessage.value.trim()
    session.time = '刚刚'
    inputMessage.value = ''

    scrollToBottom()
  }
}

const handleWebSocketMessage = (data) => {
  console.log('handleWebSocketMessage 收到数据:', data)
  
  if (typeof data !== 'object') {
    console.log('数据不是对象，跳过')
    return
  }
  
  if (data.type !== 0) {
    console.log('数据类型不是文本消息(type !== 0):', data.type)
    return
  }

  const currentSession = currentChat.value
  const isSelf = data.send_id === currentUserId.value
  
  // 判断是否是群聊消息
  const isGroupMessage = data.receive_id.startsWith('G')

  // 判断是否是当前会话
  let isCurrentSession = false
  if (currentSession) {
    if (isGroupMessage) {
      // 群聊消息：检查当前会话是否是该群聊
      isCurrentSession = data.receive_id === currentSession.id
    } else {
      // 单聊消息：检查发送者/接收者是否匹配当前会话
      if (isSelf) {
        // 自己发的消息，检查接收者是否是当前聊天对象
        isCurrentSession = data.receive_id === currentSession.id
      } else {
        // 别人发的消息，检查发送者是否是当前聊天对象
        isCurrentSession = data.send_id === currentSession.id
      }
    }
  }

  console.log('当前会话:', currentSession?.id, '发送者:', data.send_id, '接收者:', data.receive_id, '是否群聊:', isGroupMessage, '是否当前会话:', isCurrentSession)

  if (isCurrentSession) {
    const realMessage = {
      messageId: data.uuid,
      content: data.content,
      senderName: data.send_name,
      avatar: data.send_avatar,
      sendTime: new Date().toISOString(),
      isSelf: isSelf
    }

    if (isSelf) {
      const tempIndex = messageList.value.findIndex(m => m.messageId.startsWith('temp_') && m.content === data.content)
      if (tempIndex !== -1) {
        messageList.value[tempIndex] = realMessage
      } else {
        messageList.value.push(realMessage)
      }
    } else {
      messageList.value.push(realMessage)
    }

    scrollToBottom()
  } else {
    console.log('不是当前会话，更新会话列表')
    const sessionIndex = chatList.value.findIndex(item => item.sessionId === data.session_id)

    if (sessionIndex !== -1) {
      chatList.value[sessionIndex].unread++
      chatList.value[sessionIndex].lastMsg = data.content
      chatList.value[sessionIndex].time = '刚刚'
      // 将会话移到顶部
      const updatedSession = chatList.value[sessionIndex]
      chatList.value.splice(sessionIndex, 1)
      chatList.value.unshift(updatedSession)
    } else {
      // 会话不存在，重新加载会话列表
      console.log('会话不存在，重新加载会话列表')
      loadSessionList()
    }
  }
}

onMounted(async () => {
  await getCurrentUserInfo()
  await loadSessionList()

  ws.onMessage(handleWebSocketMessage)

  console.log('WebSocket 连接状态:', ws.isConnected.value)
  
  if (!ws.isConnected.value) {
    console.log('WebSocket 未连接，尝试连接...')
    ws.connect()
  }
})

onUnmounted(() => {
  ws.clearCallbacks()
})
</script>

<style scoped>
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