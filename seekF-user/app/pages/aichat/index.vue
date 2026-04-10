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
                    class="flex items-center gap-3 px-3 py-3 hover:bg-gray-50 cursor-pointer transition-colors border-b border-gray-100 group"
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
                            <div class="flex items-center gap-1">
                                <span class="text-xs text-gray-400">{{ item.createdAt }}</span>
                                <button
                                    class="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-500 transition-opacity p-1"
                                    @click.stop="handleDeleteSession(item, index)"
                                >
                                    <Icon name="tabler:trash" class="text-sm" />
                                </button>
                            </div>
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
                            <el-option label="GLM-4.6V(图片识别)" value="glm-4v" />
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
                                    <!-- 用户图片消息 -->
                                    <div v-if="msg.isSelf && msg.type === 2 && msg.url && isImageUrl(msg.url)">
                                        <img :src="msg.url" class="max-w-full rounded cursor-pointer mb-1" @click="previewImage(msg.url)" />
                                    </div>
                                    <!-- 文本消息 -->
                                    <p v-if="msg.content && msg.content !== '图片'" class="text-sm whitespace-pre-wrap">
                                        <span v-if="!msg.isSelf && msg.isStreaming">{{ msg.content }}<span class="inline-block w-0.5 h-4 bg-gray-400 ml-0.5 animate-pulse">|</span></span>
                                        <span v-else>{{ msg.content }}</span>
                                    </p>
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
                    <!-- 图片预览 -->
                    <div v-if="selectedImage" class="mb-2 flex items-center gap-2">
                        <div class="relative inline-block">
                            <img :src="selectedImagePreview" class="h-16 w-16 object-cover rounded border" />
                            <button
                                class="absolute -top-1 -right-1 bg-red-500 text-white rounded-full w-5 h-5 flex items-center justify-center text-xs hover:bg-red-600"
                                @click="removeImage"
                            >
                                ×
                            </button>
                        </div>
                    </div>
                    <div class="flex gap-3">
                        <!-- 图片上传按钮：仅 GLM-4.6V 模型显示 -->
                        <el-upload
                            v-if="currentSession?.modelType === 'glm-4v'"
                            :show-file-list="false"
                            :auto-upload="false"
                            accept="image/*"
                            :limit="1"
                            @change="handleImageChange"
                        >
                            <el-button slot="trigger" size="small" type="text" class="!text-gray-500 !hover:text-blue-500">
                                <Icon name="uil:image" class="text-xl" />
                            </el-button>
                        </el-upload>
                        <textarea
                            v-model="inputMessage"
                            placeholder="输入消息，按 Enter 发送..."
                            class="flex-1 border border-gray-300 rounded-lg p-2 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm resize-none"
                            rows="2"
                            @keydown.enter.prevent="sendMessage"
                        ></textarea>
                        <button
                            class="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            :disabled="isStreaming || (!inputMessage.trim() && !selectedImage)"
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
const selectedImage = ref(null)

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

const selectedImagePreview = computed(() => {
    if (!selectedImage.value) return ''
    return URL.createObjectURL(selectedImage.value)
})

// 图片选择
const handleImageChange = (file) => {
    if (file.raw) {
        selectedImage.value = file.raw
    }
}

// 移除图片
const removeImage = () => {
    if (selectedImage.value) {
        URL.revokeObjectURL(selectedImagePreview.value)
        selectedImage.value = null
    }
}

// 判断 URL 是否为图片（支持 blob、data、http URL）
const isImageUrl = (url) => {
    if (!url) return false
    // blob URL
    if (url.startsWith('blob:')) return true
    // base64 数据 URL
    if (url.startsWith('data:image/')) return true
    // http/https 图片 URL
    return /\.(jpg|jpeg|png|gif|webp|bmp)$/i.test(url)
}

// 图片预览
const previewImage = (url) => {
    window.open(url, '_blank')
}

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
            isSelf: msg.send_id && !msg.send_id.startsWith('A'),
            type: msg.type,
            url: msg.url || ''
        }))

        if (page === 1) {
            // 后端返回倒序（最新在前），反转后最新在最后
            messageList.value = messages.reverse()
        } else {
            // 加载更多时，反转后追加到前面
            messageList.value = [...messages.reverse(), ...messageList.value]
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
    if (activeIndex.value === -1 || isStreaming.value) return

    const session = currentSession.value
    if (!session) return

    const content = inputMessage.value.trim()
    const imageFile = selectedImage.value

    if (!content && !imageFile) return

    // 添加用户消息（如果是图片消息，保存图片 URL 用于显示）
    const userMsgData = {
        messageId: 'user_' + Date.now(),
        content: content || '图片',
        senderName: '我',
        sendTime: new Date().toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }),
        isSelf: true,
        type: imageFile ? 2 : 0,
        url: imageFile ? URL.createObjectURL(imageFile) : ''
    }
    messageList.value.push(userMsgData)

    // 清除已选择的图片
    if (imageFile) {
        removeImage()
    }

    inputMessage.value = ''

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
        imageFile,
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

// 删除会话
const handleDeleteSession = async (item, index) => {
    try {
        await ElMessageBox.confirm('确定要删除这个会话吗？', '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
        })

        const success = await aiChat.deleteSession(item.sessionId)
        if (success) {
            sessionList.value.splice(index, 1)
            if (activeIndex.value === index) {
                activeIndex.value = -1
                messageList.value = []
            } else if (activeIndex.value > index) {
                activeIndex.value--
            }
            ElMessage.success('删除成功')
        } else {
            ElMessage.error('删除失败')
        }
    } catch {
        // 用户取消
    }
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
