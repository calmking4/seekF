<template>
  <div class="flex h-full bg-gray-100">
    <!-- 左侧：搜索栏 + 联系人/会话列表 -->
    <aside class="bg-white border-r border-gray-200 h-full flex flex-col flex-shrink-0 pr-3 relative" :style="{ width: sidebarWidth + 'px' }">
      <!-- 拖动条 -->
      <div class="sidebar-resize-handle" @mousedown="startSidebarResize"></div>
      <!-- 顶部搜索栏 -->
      <SearchBar @search-select="handleSearchSelect" />

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
          <el-avatar :size="40" :src="currentChat.avatar">
            {{ currentChat.name ? currentChat.name.charAt(0) : '?' }}
          </el-avatar>
          <span class="font-medium text-sm">{{ currentChat.name }}</span>
          <div class="flex-1"></div>
          <div class="flex gap-4 text-gray-500">
            <button><Icon name="uil:ellipsis-h" /></button>
          </div>
        </div>

        <!-- 聊天内容区 - 自适应高度 -->
        <div class="flex-1 min-h-0 overflow-hidden" :style="{ paddingBottom: '0' }">
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
                class="chat-item"
                :class="{ 'chat-item-self': msg.isSelf }"
              >
                <!-- 用户名 - 在消息行外面 -->
                <div class="user-name" :class="{ 'user-name-self': msg.isSelf }">
                  {{ msg.isSelf ? '我' : msg.senderName }}
                </div>

                <!-- 消息行：气泡 + 头像 -->
                <div class="message-row" :class="{ 'message-row-self': msg.isSelf }">
                  <!-- 对方消息：头像在左，气泡在右 -->
                  <template v-if="!msg.isSelf">
                    <el-avatar class="avatar" :size="50" :src="msg.avatar">
                      {{ msg.senderName ? msg.senderName.charAt(0) : '?' }}
                    </el-avatar>
                    <div class="bubble bubble-left">
                      <div v-if="msg.isImage || isImageUrl(msg.content)">
                        <img :src="msg.content" class="max-w-full rounded cursor-pointer hover:opacity-90" style="max-height: 200px;" @click="previewImage(msg.content)" />
                      </div>
                      <p v-else class="text-sm">{{ msg.content }}</p>
                      <p class="text-xs text-gray-400 text-right mt-1">{{ formatTime(msg.sendTime) }}</p>
                    </div>
                  </template>

                  <!-- 自己消息：气泡在左，头像在右 -->
                  <template v-else>
                    <div class="bubble bubble-right">
                      <div v-if="msg.isImage || isImageUrl(msg.content)">
                        <img :src="msg.content" class="max-w-full rounded cursor-pointer hover:opacity-90" style="max-height: 200px;" @click="previewImage(msg.content)" />
                      </div>
                      <p v-else class="text-sm">{{ msg.content }}</p>
                      <p class="msg-time text-xs text-right mt-1">{{ formatTime(msg.sendTime) }}</p>
                    </div>
                    <el-avatar class="avatar" :size="50" :src="currentUserAvatar">
                      我
                    </el-avatar>
                  </template>
                </div>
              </div>
            </div>
          </el-scrollbar>
        </div>

        <!-- 输入框区域 -->
        <div class="input-area flex-shrink-0" :style="{ height: inputAreaHeight + 'px' }">
          <!-- 拖动条 -->
          <div class="resize-handle" @mousedown="startResize">
            <div class="resize-line"></div>
          </div>

          <!-- 输入框和发送按钮 -->
          <div class="input-container">
            <textarea
              v-model="inputMessage"
              placeholder="输入消息..."
              class="message-input"
              @keydown.enter.prevent="sendMessage"
            ></textarea>
            <div class="toolbar-send">
              <div class="toolbar">
                <button class="toolbar-btn" title="表情">
                  <Icon name="uil:smile" class="text-base" />
                </button>
                <button class="toolbar-btn" title="发送图片" @click="triggerImageUpload">
                  <Icon name="tabler:photo" class="text-base" />
                </button>
                <input
                  ref="imageInputRef"
                  type="file"
                  accept="image/*"
                  class="hidden"
                  @change="handleImageUpload"
                />
                <button
                  v-if="currentChat && !currentChat.id?.startsWith('G')"
                  class="toolbar-btn"
                  title="语音通话"
                  @click="startCall"
                >
                  <Icon name="fluent:call-24-regular" class="text-base" />
                </button>
              </div>
              <button
                class="send-btn"
                :disabled="!inputMessage.trim() || !ws.isConnected"
                @click="sendMessage"
              >
                <Icon name="uil:message" class="text-base" />
                <span>发送</span>
              </button>
            </div>
          </div>
        </div>
      </template>
    </main>

    <!-- 图片预览组件 -->
    <el-image-viewer
      v-if="showImageViewer"
      :url-list="[previewImageUrl]"
      :initial-index="0"
      :hide-on-click-modal="true"
      @close="showImageViewer = false"
    />
  </div>
</template>

<script setup>
const route = useRoute()
const router = useRouter()

const ws = useWebSocket()
const avCall = useAVCall()

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
const imageInputRef = ref()

// 侧边栏宽度调整
const sidebarWidth = ref(320)
const isSidebarResizing = ref(false)
const sidebarStartX = ref(0)
const sidebarStartWidth = ref(0)

const startSidebarResize = (e) => {
  e.preventDefault()
  isSidebarResizing.value = true
  sidebarStartX.value = e.clientX
  sidebarStartWidth.value = sidebarWidth.value
  document.body.style.cursor = 'ew-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', handleSidebarResize)
  document.addEventListener('mouseup', stopSidebarResize)
}

const handleSidebarResize = (e) => {
  if (!isSidebarResizing.value) return
  e.preventDefault()
  const diff = e.clientX - sidebarStartX.value
  const newWidth = Math.max(240, Math.min(500, sidebarStartWidth.value + diff))
  sidebarWidth.value = newWidth
}

const stopSidebarResize = () => {
  isSidebarResizing.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  document.removeEventListener('mousemove', handleSidebarResize)
  document.removeEventListener('mouseup', stopSidebarResize)
}

// 输入区域高度调整
const inputAreaHeight = ref(180)
const isResizing = ref(false)
const startY = ref(0)
const startHeight = ref(0)

const startResize = (e) => {
  isResizing.value = true
  startY.value = e.clientY
  startHeight.value = inputAreaHeight.value
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

const handleResize = (e) => {
  if (!isResizing.value) return
  e.preventDefault()
  const diff = startY.value - e.clientY
  const newHeight = Math.max(120, Math.min(400, startHeight.value + diff))
  inputAreaHeight.value = newHeight
}

const stopResize = () => {
  isResizing.value = false
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

// 图片预览状态
const showImageViewer = ref(false)
const previewImageUrl = ref('')
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

// 处理搜索联想选中（跳转到对应会话）
const handleSearchSelect = (item) => {
  if (typeof item === 'object' && item.session_id) {
    // 联想结果：跳转到对应会话
    const index = chatList.value.findIndex(s => s.sessionId === item.session_id)
    if (index !== -1) {
      selectSession(index)
    }
  } else if (typeof item === 'string') {
    // 搜索历史关键词：可以触发搜索（当前不处理）
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

      const messages = list.map(msg => {
        // 文件消息(type=2)使用 url 作为内容，文本消息使用 content
        const isFileMsg = msg.type === 2
        const messageContent = isFileMsg ? (msg.url || msg.content) : msg.content
        return {
          messageId: msg.uuid || msg.messageId,
          content: messageContent,
          senderName: msg.send_name || msg.senderName,
          avatar: msg.send_avatar || msg.avatar,
          sendTime: msg.created_at || msg.createdAt,
          isSelf: msg.send_id === currentUserId.value,
          isImage: isFileMsg && isImageUrl(messageContent)
        }
      })

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

// 触发图片上传
const triggerImageUpload = () => {
  imageInputRef.value?.click()
}

// 判断是否是图片URL
const isImageUrl = (url) => {
  if (!url || typeof url !== 'string') return false
  return /\.(jpg|jpeg|png|gif|webp)$/i.test(url) || url.includes('message_image')
}

// 预览图片
const previewImage = (url) => {
  previewImageUrl.value = url
  showImageViewer.value = true
}

// 处理图片上传
const handleImageUpload = async (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  // 验证文件类型
  const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
  if (!validTypes.includes(file.type)) {
    ElMessage.error('只支持 JPG、PNG、GIF、WebP 格式的图片')
    return
  }

  // 验证文件大小 (5MB)
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error('图片大小不能超过 5MB')
    return
  }

  if (activeIndex.value === -1) return

  const session = currentChat.value
  if (!session) return

  try {
    // 上传文件到服务器
    const formData = new FormData()
    formData.append('file', file)
    formData.append('fileType', 'message_image')

    const data = await useApi$('/user/file/upload', {
      method: 'POST',
      body: formData
    })

    if (data?.code === 200) {
      const imageUrl = typeof data.data === 'object' ? data.data.url : data.data

      // 发送图片消息
      const success = ws.sendFileMessage(
        session.sessionId,
        imageUrl,
        file.name,
        file.size.toString(),
        file.type,
        session.id
      )

      if (success) {
        // 添加临时消息到列表
        const tempId = 'temp_' + Date.now()
        messageList.value.push({
          messageId: tempId,
          content: imageUrl,
          senderName: '我',
          avatar: currentUserAvatar.value,
          sendTime: new Date().toISOString(),
          isSelf: true,
          isImage: true
        })

        session.lastMsg = '[图片]'
        session.time = '刚刚'

        scrollToBottom()
        ElMessage.success('图片发送成功')
      }
    } else {
      ElMessage.error(data?.message || '图片上传失败')
    }
  } catch (error) {
    console.error('图片上传失败:', error)
    ElMessage.error('图片上传失败')
  }

  // 清空文件输入框，允许重复选择同一文件
  event.target.value = ''
}

// 发起通话
const startCall = () => {
  if (activeIndex.value === -1) return
  
  const session = currentChat.value
  if (!session) return
  
  // 只有单聊才能发起通话
  if (session.id.startsWith('G')) {
    alert('暂不支持群聊通话')
    return
  }
  
  avCall.startCall(session.sessionId, session.id, {
    name: session.name,
    avatar: session.avatar
  })
}

const handleWebSocketMessage = (data) => {
  console.log('handleWebSocketMessage 收到数据:', data)
  
  if (typeof data !== 'object') {
    console.log('数据不是对象，跳过')
    return
  }
  
  // 处理文本消息(type=0)和文件消息(type=2)
  if (data.type !== 0 && data.type !== 2) {
    console.log('数据类型不是文本或文件消息:', data.type)
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
    // 文件消息使用 url 作为内容，文本消息使用 content
    const messageContent = data.type === 2 ? data.url : data.content
    const realMessage = {
      messageId: data.uuid,
      content: messageContent,
      senderName: data.send_name,
      avatar: data.send_avatar,
      sendTime: new Date().toISOString(),
      isSelf: isSelf,
      isImage: data.type === 2 && isImageUrl(data.url)
    }

    if (isSelf) {
      const tempIndex = messageList.value.findIndex(m => m.messageId.startsWith('temp_') && m.content === messageContent)
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

    // 文件消息显示[图片]，文本消息显示内容
    const lastMsg = data.type === 2 ? '[图片]' : data.content

    if (sessionIndex !== -1) {
      chatList.value[sessionIndex].unread++
      chatList.value[sessionIndex].lastMsg = lastMsg
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
})

onUnmounted(() => {
  ws.removeMessageCallback(handleWebSocketMessage)
})
</script>

<style scoped>
/* 一条消息 */
.chat-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  margin-bottom: 24px;
}

/* 自己的消息右对齐 */
.chat-item-self {
  align-items: flex-end;
}

/* 用户名 */
.user-name {
  width: 50px;
  margin-bottom: 6px;
  text-align: center;
  font-size: 12px;
  color: #666;
  line-height: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 气泡和头像 */
.message-row {
  display: flex;
  align-items: flex-start; /* 关键：顶部对齐 */
  gap: 15px;
}

/* 自己的消息行 */
.message-row-self {
  justify-content: flex-end;
}

/* 聊天气泡 */
.bubble {
  position: relative;
  max-width: 500px;
  padding: 12px 16px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  color: #333;
  word-break: break-word;
  box-sizing: border-box;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

/* 对方气泡 - 白色背景 */
.bubble-left {
  background: #fff;
}

/* 自己气泡 - 蓝色背景 */
.bubble-right {
  background: #7ab5fe;
  color: #fff;
}

.bubble-right .msg-time {
  color: rgba(255, 255, 255, 0.8);
}

/* 对方气泡箭头 - 指向左边头像 */
.bubble-left::after {
  content: "";
  position: absolute;
  left: -6px;
  top: 15px;
  width: 16px;
  height: 16px;
  background: #fff;
  transform: rotate(45deg);
}

/* 自己气泡箭头 - 指向右边头像 */
.bubble-right::after {
  content: "";
  position: absolute;
  right: -6px;
  top: 15px;
  width: 16px;
  height: 16px;
  background: #7ab5fe;
  transform: rotate(45deg);
}

/* 头像 */
.avatar {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

/* 输入区域 */
.input-area {
  background: #fff;
  border-top: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  position: relative;
}

/* 拖动条 */
.resize-handle {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 8px;
  cursor: ns-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.resize-handle:hover {
  background: rgba(59, 130, 246, 0.1);
}

.resize-line {
  width: 40px;
  height: 3px;
  background: #d1d5db;
  border-radius: 2px;
  transition: all 0.2s ease;
}

.resize-handle:hover .resize-line {
  background: #3b82f6;
  width: 50px;
}

/* 输入容器 */
.input-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 12px 16px;
  padding-top: 16px;
}

/* 消息输入框 */
.message-input {
  flex: 1;
  border: none;
  background: transparent;
  padding: 0;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  outline: none;
  min-height: 40px;
}

.message-input::placeholder {
  color: #9ca3af;
}

/* 工具栏和发送按钮容器 */
.toolbar-send {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
}

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  color: #9ca3af;
  transition: all 0.2s ease;
}

.toolbar-btn:hover {
  background: #f3f4f6;
  color: #6b7280;
}

/* 发送按钮 */
.send-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 24px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.send-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.3);
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 侧边栏拖动条 */
.sidebar-resize-handle {
  position: absolute;
  top: 0;
  right: 0;
  width: 4px;
  height: 100%;
  cursor: ew-resize;
  z-index: 10;
  transition: background 0.2s ease;
}

.sidebar-resize-handle:hover {
  background: rgba(59, 130, 246, 0.3);
}
</style>