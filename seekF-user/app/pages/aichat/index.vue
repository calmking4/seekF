<template>
    <div class="flex h-full bg-gray-100">
        <!-- 左侧：AI 会话列表 -->
        <aside class="w-80 bg-white border-r border-gray-200 h-full flex flex-col flex-shrink-0 pr-3">
            <!-- 顶部：新建会话按钮 -->
            <div class="p-3 border-b border-gray-200">
                <el-button type="primary" class="w-full" @click="handleCreateSession">
                    <Icon name="uil:plus" class="mr-1" />
                    新建 AI 对话
                </el-button>
            </div>

            <!-- 会话列表 -->
            <div class="flex-1 overflow-y-auto">
                <div v-if="sessionList.length === 0" class="p-8 text-center text-gray-400">
                    暂无 AI 会话
                </div>
                <div
                    v-for="(item, index) in sessionList"
                    :key="item.sessionId"
                    class="flex items-center gap-3 px-3 py-3 hover:bg-gray-50 cursor-pointer transition-colors border-b border-gray-100"
                    :class="{ 'bg-gray-100': activeIndex === index }"
                    @click="selectSession(index)"
                >
                    <!-- AI 头像 -->
                    <div class="relative flex-shrink-0">
                        <el-avatar :size="48" :style="{ backgroundColor: getModelColor(item.modelType) }">
                            <Icon :name="getModelIcon(item.modelType)" class="text-white text-xl" />
                        </el-avatar>
                    </div>
                    <!-- 会话信息 -->
                    <div class="flex-1 min-w-0">
                        <div class="flex justify-between items-start">
                            <h3 class="font-medium text-sm truncate">AI 助手</h3>
                            <span class="text-xs text-gray-400">{{ item.createdAt }}</span>
                        </div>
                        <p class="text-xs text-gray-500 truncate">{{ item.firstMessage || '点击开始对话' }}</p>
                    </div>
                </div>
            </div>
        </aside>

        <!-- 右侧：聊天窗口 -->
        <main class="flex-1 flex flex-col bg-[#f3f4f6] h-full overflow-hidden">
            <!-- 未选择会话时的占位 -->
            <div v-if="activeIndex === -1" class="flex-1 flex flex-col items-center justify-center text-gray-400">
                <Icon name="uil:robot" class="text-6xl mb-4" />
                <p class="text-lg">选择一个 AI 会话开始对话</p>
                <p class="text-sm mt-2">支持 DeepSeek / Qwen / GLM</p>
            </div>

            <!-- 已选择会话的聊天界面 -->
            <template v-else>
                <!-- 聊天头部 -->
                <div class="bg-white border-b border-gray-200 p-3 flex items-center gap-2 flex-shrink-0">
                    <el-avatar :size="40" :style="{ backgroundColor: getModelColor(currentSession?.modelType) }">
                        <Icon :name="getModelIcon(currentSession?.modelType)" class="text-white" />
                    </el-avatar>
                    <div class="flex items-center gap-2">
                        <span class="text-base font-medium">AI 助手</span>
                        <el-select v-model="currentSession.modelType" size="small" class="!w-24" @change="handleModelChange">
                            <el-option label="DeepSeek" value="deepseek" />
                            <el-option label="Qwen" value="qwen" />
                            <el-option label="GLM" value="glm" />
                        </el-select>
                    </div>
                    <div class="flex-1"></div>
                    <!-- 流式状态指示 -->
                    <div v-if="isStreaming" class="flex items-center gap-2 text-xs mr-2">
                        <span class="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></span>
                        <span class="text-blue-600">AI 正在思考...</span>
                    </div>
                </div>

                <!-- 消息列表区域 -->
                <div class="flex-1 min-h-0 overflow-hidden">
                    <el-scrollbar
                        ref="scrollbarRef"
                        class="h-full"
                        @end-reached="handleEndReached"
                    >
                        <div class="p-4 space-y-3">
                            <div v-if="messageList.length === 0" class="flex items-center justify-center py-20 text-gray-400">
                                <p>暂无消息，开始对话吧</p>
                            </div>

                            <!-- 加载更多 -->
                            <div v-if="hasMore && messageList.length > 0" class="text-center py-2">
                                <span v-if="loadingMore" class="text-gray-400 text-sm">加载中...</span>
                                <button v-else class="text-blue-500 text-sm hover:underline" @click="loadMoreMessages">
                                    加载更多消息
                                </button>
                            </div>

                            <!-- 消息列表 -->
                            <div
                                v-for="(msg, idx) in messageList"
                                :key="msg.messageId || idx"
                                class="flex items-start gap-3"
                                :class="{ 'justify-end': msg.isSelf }"
                            >
                                <!-- AI 消息：头像在左 -->
                                <div v-if="!msg.isSelf" class="flex-shrink-0">
                                    <el-avatar :size="40" :style="{ backgroundColor: getModelColor(currentSession?.modelType) }">
                                        <Icon :name="getModelIcon(currentSession?.modelType)" class="text-white text-sm" />
                                    </el-avatar>
                                </div>

                                <!-- 消息气泡 -->
                                <div
                                    class="rounded-lg px-4 py-2 max-w-[60%] shadow-sm flex-shrink-0"
                                    :class="msg.isSelf ? 'bg-[#D9FDD3]' : 'bg-white'"
                                >
                                    <!-- 流式 AI 消息显示光标 -->
                                    <p v-if="msg.isStreaming" class="text-sm whitespace-pre-wrap">{{ msg.content }}<span class="inline-block w-0.5 h-4 bg-gray-400 ml-0.5 animate-pulse">|</span></p>
                                    <p v-else class="text-sm whitespace-pre-wrap">{{ msg.content }}</p>
                                    <p class="text-xs text-gray-400 text-right mt-1">{{ msg.sendTime }}</p>
                                </div>

                                <!-- 用户消息：头像在右 -->
                                <div v-if="msg.isSelf" class="flex-shrink-0">
                                    <el-avatar :size="40" :src="currentUserAvatar">
                                        我
                                    </el-avatar>
                                </div>
                            </div>
                        </div>
                    </el-scrollbar>
                </div>

                <!-- 输入框区域 -->
                <div class="bg-white p-3 border-t border-gray-200 flex-shrink-0">
                    <div class="flex gap-3">
                        <textarea
                            v-model="inputMessage"
                            placeholder="输入消息，按 Enter 发送..."
                            class="flex-1 border border-gray-300 rounded-lg p-2 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm resize-none"
                            rows="2"
                            @keydown.enter.prevent="sendMessage"
                        ></textarea>
                        <button
                            class="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            :disabled="isStreaming || !inputMessage.trim()"
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
const aiChat = useAIChat()

const currentUserAvatar = ref('')
const sessionList = ref([])
const activeIndex = ref(-1)
const messageList = ref([])
const inputMessage = ref('')
const isStreaming = ref(false)

const scrollbarRef = ref()
const hasMore = ref(true)
const loadingMore = ref(false)
const currentPage = ref(1)
const pageSize = 20
const totalMessages = ref(0)
let currentEventSource = null

const currentSession = computed(() => {
    if (activeIndex.value === -1) return null
    return sessionList.value[activeIndex.value]
})

// 模型图标（统一机器人头像）
const getModelIcon = () => {
    return 'uil:robot'
}

// 模型颜色（统一颜色）
const getModelColor = () => {
    return '#6B7280'
}

// 滚动到底部
const scrollToBottom = () => {
    nextTick(() => {
        if (scrollbarRef.value?.wrapRef) {
            const wrap = scrollbarRef.value.wrapRef
            scrollbarRef.value.setScrollTop(wrap.scrollHeight)
        }
    })
}

// 滚动到顶部加载更多
const handleEndReached = (direction) => {
    if (direction === 'top') {
        loadMoreMessages()
    }
}

// 获取当前用户信息
const getCurrentUserInfo = async () => {
    try {
        const data = await useApi$('/user/userinfo/getMyInfo', { method: 'POST' })
        if (data?.code === 200) {
            currentUserAvatar.value = data.data.avatar
        }
    } catch (error) {
        console.error('获取用户信息失败:', error)
    }
}

// 加载会话列表
const loadSessionList = async () => {
    try {
        const data = await aiChat.getSessionList()
        sessionList.value = (data.list || []).map(item => ({
            sessionId: item.session_id,
            modelType: 'deepseek',
            firstMessage: item.first_message || '',
            createdAt: item.created_at || ''
        }))
    } catch (error) {
        console.error('获取AI会话列表失败:', error)
    }
}

// 选择会话
const selectSession = async (index) => {
    activeIndex.value = index
    const session = sessionList.value[index]
    if (!session) return

    // 关闭之前的 SSE 连接
    if (currentEventSource) {
        currentEventSource.close()
        currentEventSource = null
    }

    currentPage.value = 1
    hasMore.value = true
    loadingMore.value = false
    messageList.value = []

    await loadMessageList(session.sessionId)
    scrollToBottom()
}

// 加载消息历史
const loadMessageList = async (sessionId, page = 1) => {
    try {
        if (!sessionId) return

        const data = await aiChat.getMessageHistory(sessionId, page, pageSize)
        const list = data.list || []
        totalMessages.value = data.total || 0

        const messages = list.map(msg => ({
            messageId: msg.session_id + '_' + msg.created_at,
            content: msg.content,
            senderName: msg.send_name,
            sendTime: msg.created_at,
            isSelf: msg.send_id && !msg.send_id.startsWith('A')
        }))

        if (page === 1) {
            messageList.value = messages
        } else {
            messageList.value = [...messages, ...messageList.value]
        }

        hasMore.value = messageList.value.length < totalMessages.value
    } catch (error) {
        console.error('获取消息历史失败:', error)
    }
}

// 加载更多消息
const loadMoreMessages = async () => {
    if (loadingMore.value || !hasMore.value || !currentSession.value) return
    loadingMore.value = true

    const oldScrollHeight = scrollbarRef.value?.wrapRef?.scrollHeight || 0

    currentPage.value++
    await loadMessageList(currentSession.value.sessionId, currentPage.value)

    nextTick(() => {
        if (scrollbarRef.value?.wrapRef) {
            const newScrollHeight = scrollbarRef.value.wrapRef.scrollHeight
            scrollbarRef.value.setScrollTop(newScrollHeight - oldScrollHeight)
        }
    })

    loadingMore.value = false
}

// 发送消息
const sendMessage = async () => {
    if (!inputMessage.value.trim() || activeIndex.value === -1 || isStreaming.value) return

    const session = currentSession.value
    if (!session) return

    const content = inputMessage.value.trim()
    inputMessage.value = ''

    // 添加用户消息
    messageList.value.push({
        messageId: 'user_' + Date.now(),
        content: content,
        senderName: '我',
        sendTime: new Date().toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }),
        isSelf: true
    })

    // 添加 AI 流式消息占位
    const aiMsgIndex = messageList.value.length
    messageList.value.push({
        messageId: 'ai_' + Date.now(),
        content: '',
        senderName: 'AI 助手',
        sendTime: '',
        isSelf: false,
        isStreaming: true
    })

    isStreaming.value = true
    scrollToBottom()

    // 关闭之前的 SSE 连接
    if (currentEventSource) {
        currentEventSource.close()
    }

    currentEventSource = aiChat.sendMessage(
        session.sessionId,
        content,
        session.modelType,
        // onChunk
        (chunk) => {
            const aiMsg = messageList.value[aiMsgIndex]
            if (aiMsg) {
                aiMsg.content += chunk
                scrollToBottom()
            }
        },
        // onComplete
        () => {
            const aiMsg = messageList.value[aiMsgIndex]
            if (aiMsg) {
                aiMsg.isStreaming = false
                aiMsg.sendTime = new Date().toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
            }
            isStreaming.value = false
            session.lastMessage = aiMsg.content
            currentEventSource = null
            scrollToBottom()
        },
        // onError
        (error) => {
            const aiMsg = messageList.value[aiMsgIndex]
            if (aiMsg) {
                aiMsg.isStreaming = false
                if (!aiMsg.content) {
                    aiMsg.content = '抱歉，响应出现错误：' + error
                }
            }
            isStreaming.value = false
            currentEventSource = null
            ElMessage.error('AI 响应失败')
        }
    )
}

// 创建新会话
const handleCreateSession = async () => {
    const result = await aiChat.createSession('deepseek')
    if (result) {
        await loadSessionList()
        // 自动选中新创建的会话
        const idx = sessionList.value.findIndex(s => s.sessionId === result.session_id)
        if (idx !== -1) {
            selectSession(idx)
        }
        ElMessage.success('创建会话成功')
    } else {
        ElMessage.error('创建会话失败')
    }
}

// 切换模型
const handleModelChange = (newModel) => {
    const session = currentSession.value
    if (!session) return
    session.modelType = newModel
}

onMounted(async () => {
    await getCurrentUserInfo()
    await loadSessionList()
})

onUnmounted(() => {
    if (currentEventSource) {
        currentEventSource.close()
        currentEventSource = null
    }
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
