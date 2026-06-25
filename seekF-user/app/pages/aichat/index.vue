<template>
    <div class="flex h-screen bg-[#f7f7f7]">
        <!-- 会话栏 -->
        <aside
            class="session-sidebar"
            :class="{ collapsed: !expanded }"
        >
            <div class="session-header">
                <button class="new-chat" @click="handleCreateSession">
                    <Icon name="uil:plus" class="mr-1" />
                    新建会话
                </button>

                <button
                    class="collapse-btn"
                    @click="toggleSidebar"
                >
                    <Icon name="uil:columns" class="text-lg" />
                </button>
            </div>

            <div class="divider"></div>

            <div class="session-list">
                <div v-if="sessionList.length === 0" class="text-center text-gray-400 mt-10 text-sm">
                    暂无 AI 会话
                </div>

                <template v-else>
                    <div class="text-gray-400 text-sm mb-3">会话列表</div>

                    <div
                        v-for="(item, index) in sessionList"
                        :key="item.sessionId"
                        class="session-item"
                        :class="{ active: activeIndex === index }"
                        @click="selectSession(index)"
                    >
                        <div class="flex items-center justify-between">
                            <span class="flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-sm">
                                <span v-if="activeStreamSessions.has(item.sessionId)" class="streaming-dot"></span>
                                {{ item.firstMessage || '新会话' }}
                            </span>
                            <button
                                class="delete-btn"
                                @click.stop="handleDeleteSession(item, index)"
                            >
                                <Icon name="uil:trash-alt" class="text-sm" />
                            </button>
                        </div>
                    </div>
                </template>

                <div v-if="sessionList.length > 0" class="text-center text-gray-400 mt-10 text-sm">
                    没有更多了
                </div>
            </div>
        </aside>

        <!-- 聊天区 -->
        <main class="flex-1 relative flex flex-col min-w-0">
            <!-- 聊天头部 -->
            <div class="h-[60px] bg-[#fafafa] border-b border-gray-200 flex items-center justify-between px-5 flex-shrink-0">
                <div class="flex items-center gap-2 min-w-[120px]">
                    <button
                        v-if="!expanded"
                        class="action-icon"
                        @click="toggleSidebar"
                    >
                        <Icon name="uil:columns" class="text-xl" />
                    </button>
                    <button
                        v-if="!expanded"
                        class="action-icon"
                        @click="handleCreateSession"
                    >
                        <Icon name="uil:plus" class="text-xl" />
                    </button>
                </div>

                <div class="flex-1 flex justify-center">
                    <span class="text-base font-semibold text-gray-700 max-w-[300px] overflow-hidden text-ellipsis whitespace-nowrap">
                        {{ currentSessionTitle }}
                    </span>
                </div>

                <div class="flex items-center gap-4 min-w-[120px] justify-end">
                    <div v-if="isStreaming" class="streaming-indicator">
                        <span class="streaming-dot"></span>
                        <span>AI 正在思考...</span>
                    </div>
                    <button class="header-btn" @click="goToKnowledge">
                        <Icon name="uil:book-open" class="mr-1" />
                        知识库管理
                    </button>
                </div>
            </div>

            <!-- 未选择会话时的占位 -->
            <div v-if="activeIndex === -1" class="flex-1 flex flex-col items-center justify-center text-gray-400">
                <Icon name="uil:robot" class="text-6xl w-16 h-16 mb-4 text-gray-300" />
                <p class="text-lg text-gray-700 mb-2">选择一个 AI 会话开始对话</p>
                <p class="text-sm text-gray-400">支持 DeepSeek / Qwen / GLM</p>
            </div>

            <!-- 已选择会话的聊天界面 -->
            <template v-else>

                <!-- 消息列表区域 -->
                <div class="chat-container" ref="chatContainerRef" @scroll="handleScroll">
                    <div class="w-full max-w-[1000px] mx-auto px-10">
                        <!-- 加载更多 -->
                        <div v-if="hasMore && messageList.length > 0" class="text-center py-4">
                            <span v-if="loadingMore" class="text-gray-400 text-sm">加载中...</span>
                            <button v-else class="bg-transparent border-none text-[#0073ff] cursor-pointer text-sm hover:underline" @click="loadMoreMessages">加载更多消息</button>
                        </div>

                        <!-- 消息列表 -->
                        <div
                            v-for="(msg, idx) in messageList"
                            :key="msg.messageId || idx"
                            class="w-full flex mb-[30px]"
                            :class="msg.isSelf ? 'justify-end' : 'justify-start'"
                        >
                            <!-- AI 消息 -->
                            <div v-if="!msg.isSelf" class="ai-message-wrapper">
                                <!-- 思考中动画 -->
                                <div v-if="msg.isStreaming && !msg.content" class="thinking-animation">
                                    <div class="thinking-dots">
                                        <span class="dot"></span>
                                        <span class="dot"></span>
                                        <span class="dot"></span>
                                    </div>
                                    <span class="thinking-text">正在思考</span>
                                </div>
                                <!-- Markdown渲染 -->
                                <MarkdownRenderer
                                    v-else-if="msg.content && msg.content !== '图片'"
                                    :content="msg.content"
                                    :is-streaming="msg.isStreaming"
                                />
                                <!-- 搜索来源 -->
                                <SearchSources v-if="msg.sources && msg.sources.length > 0" :sources="msg.sources" />
                                <!-- 帖子列表 -->
                                <DiscoverPosts
                                    v-if="msg.posts && msg.posts.length > 0"
                                    :posts="msg.posts"
                                    @post-click="openDiscoverDetail"
                                />
                                <!-- 操作按钮 -->
                                <div v-if="!msg.isStreaming && msg.content" class="message-actions">
                                    <button
                                        class="action-btn"
                                        :class="{ 'copy-success': copiedMap[msg.messageId] }"
                                        @click="copyMessage(msg)"
                                    >
                                        <Icon v-if="copiedMap[msg.messageId]" name="uil:check" class="text-base" />
                                        <Icon v-else name="ph:copy-simple-bold" class="text-base" />
                                    </button>
                                    <TTSButton
                                        :playing="tts.isMessagePlaying(msg.messageId)"
                                        :loading="tts.isMessageLoading(msg.messageId)"
                                        @speak="tts.speak(msg.content, msg.messageId)"
                                    />
                                    <span class="text-xs text-gray-400">{{ msg.sendTime }}</span>
                                </div>
                            </div>

                            <!-- 用户消息 -->
                            <div v-else class="user-message">
                                <!-- 图片消息 -->
                                <div v-if="msg.type === 2 && msg.url && isImageUrl(msg.url)">
                                    <img :src="msg.url" class="max-w-[200px] rounded-lg cursor-pointer" @click="previewImage(msg.url)" />
                                </div>
                                <!-- 文本消息 -->
                                <p v-else-if="msg.content && msg.content !== '图片'" class="m-0 whitespace-pre-wrap">
                                    {{ msg.content }}
                                </p>
                            </div>
                        </div>

                        <div v-if="messageList.length === 0" class="flex items-center justify-center h-[200px] text-gray-400">
                            <p>暂无消息，开始对话吧</p>
                        </div>
                    </div>
                </div>

                <!-- 输入框 -->
                <div class="input-wrapper">
                    <div class="input-box">
                        <!-- 图片预览 -->
                        <div v-if="selectedImage" class="relative inline-block mb-3">
                            <img :src="selectedImagePreview" class="h-16 w-16 object-cover rounded-lg border border-gray-200" />
                            <button class="remove-img-btn" @click="removeImage">×</button>
                        </div>

                        <textarea
                            ref="textareaRef"
                            v-model="inputMessage"
                            placeholder="搜索或者输入任何问题"
                            @keydown.enter.prevent="sendMessage"
                            @input="autoResize"
                        />

                        <div class="input-footer">
                            <div class="left-actions">
                                <!-- 上传图片 -->
                                <div class="dropdown">
                                    <button
                                        class="footer-btn"
                                        @click.stop="toggleUpload"
                                    >
                                        <Icon name="uil:plus" class="text-lg dropdown-arrow" :class="{ 'arrow-up': showUpload }" />
                                    </button>

                                    <div v-if="showUpload" class="dropdown-menu">
                                        <div
                                            class="menu-item"
                                            :class="{ 'menu-item-disabled': !canUploadImage }"
                                            @click="canUploadImage && chooseImage()"
                                        >
                                            <Icon name="uil:image" class="mr-2" />
                                            本地图片
                                            <span v-if="!canUploadImage" class="text-xs text-gray-400 ml-2">仅支持图片模型</span>
                                        </div>
                                    </div>
                                </div>

                                <!-- 工具 -->
                                <div class="dropdown">
                                    <button
                                        class="footer-btn"
                                        @click.stop="toggleTool"
                                    >
                                        <Icon name="uil:wrench" class="mr-1" />
                                        工具
                                        <Icon name="uil:angle-down" class="ml-1 dropdown-arrow" :class="{ 'arrow-up': showTool }" />
                                    </button>

                                    <div v-if="showTool" class="dropdown-menu" @click.stop>
                                        <div
                                            class="menu-item"
                                            :class="{ active: useKnowledgeBase }"
                                            @click.stop="useKnowledgeBase = !useKnowledgeBase"
                                        >
                                            <Icon name="uil:books" class="mr-2" />
                                            知识库
                                            <Icon v-if="useKnowledgeBase" name="uil:check" class="ml-auto text-green-500" />
                                        </div>

                                        <div
                                            class="menu-item"
                                            :class="{ active: useWebSearch }"
                                            @click.stop="useWebSearch = !useWebSearch"
                                        >
                                            <Icon name="uil:globe" class="mr-2" />
                                            联网搜索
                                            <Icon v-if="useWebSearch" name="uil:check" class="ml-auto text-green-500" />
                                        </div>
                                    </div>
                                </div>

                                <!-- 模型 -->
                                <div class="dropdown">
                                    <button
                                        class="footer-btn"
                                        @click.stop="toggleModel"
                                    >
                                        {{ currentModelName }}
                                        <Icon name="uil:angle-down" class="ml-1 dropdown-arrow" :class="{ 'arrow-up': showModel }" />
                                    </button>

                                    <div v-if="showModel" class="dropdown-menu">
                                        <div class="menu-item" @click="selectModel('deepseek')">DeepSeek</div>
                                        <div class="menu-item" @click="selectModel('qwen')">Qwen</div>
                                        <div class="menu-item" @click="selectModel('glm')">GLM</div>
                                        <div class="menu-item" @click="selectModel('glm-4v')">GLM-4.6V(图片)</div>
                                        <div class="menu-item" @click="selectModel('qwen-local')">Qwen3.5-9B(本地)</div>
                                    </div>
                                </div>
                            </div>

                            <button
                                class="send-btn"
                                :disabled="isStreaming || (!inputMessage.trim() && !selectedImage)"
                                @click="sendMessage"
                            >
                                <Icon name="uil:message" class="text-xl" />
                            </button>
                        </div>

                        <input
                            ref="fileInput"
                            type="file"
                            accept="image/*"
                            style="display:none"
                            @change="handleFileChange"
                        />
                    </div>
                </div>
            </template>
        </main>

        <!-- 帖子详情弹窗 -->
        <DiscoverDetail
            v-if="showDiscoverDetail && selectedDiscoverItem"
            :item="selectedDiscoverItem"
            @close="showDiscoverDetail = false"
        />

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
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'

// 页面级 SEO
useSeoMeta({
  title: 'AIChat',
  description: '与多种 AI 模型进行智能对话，支持 DeepSeek、通义千问、智谱等。',
})

const aiChat = useAIChat()
const knowledge = useKnowledge()
const tts = useTTS()
const aiAvatarUrl = 'https://seekf.oss-cn-shenzhen.aliyuncs.com/common/ai_avatar/AI%E5%8A%A9%E6%89%8B%E5%A4%B4%E5%83%8F.png'

// 布局状态
const expanded = ref(true)
const toggleSidebar = () => {
    expanded.value = !expanded.value
}

// 下拉菜单状态
const showTool = ref(false)
const showModel = ref(false)
const showUpload = ref(false)

const toggleTool = () => {
    showTool.value = !showTool.value
    showModel.value = false
    showUpload.value = false
}

const toggleModel = () => {
    showModel.value = !showModel.value
    showTool.value = false
    showUpload.value = false
}

const toggleUpload = () => {
    showUpload.value = !showUpload.value
    showTool.value = false
    showModel.value = false
}

const closeMenus = () => {
    showTool.value = false
    showModel.value = false
    showUpload.value = false
}

// AI消息复制功能
const copiedMap = ref({})
function copyMessage(msg) {
    navigator.clipboard.writeText(msg.content).then(() => {
        copiedMap.value[msg.messageId] = true
        setTimeout(() => { copiedMap.value[msg.messageId] = false }, 1500)
    })
}

const currentUserAvatar = ref('')
const sessionList = ref([])
const activeIndex = ref(-1)
const messageList = ref([])
const inputMessage = ref('')
const activeStreamSessions = ref(new Set())
const isStreaming = computed(() => {
    const sid = currentSession.value?.sessionId
    return sid ? activeStreamSessions.value.has(sid) : false
})
const selectedImage = ref(null)
const useKnowledgeBase = ref(false)
const useWebSearch = ref(false)
const showDiscoverDetail = ref(false)
const selectedDiscoverItem = ref(null)

// 图片预览状态
const showImageViewer = ref(false)
const previewImageUrl = ref('')
const fileInput = ref(null)
const textareaRef = ref(null)

// 自动调整 textarea 高度
const autoResize = () => {
    const textarea = textareaRef.value
    if (!textarea) return
    textarea.style.height = 'auto'
    // 加 1px 缓冲解决小数点精度问题
    textarea.style.height = Math.min(textarea.scrollHeight + 1, 155) + 'px'
}

// 当前模型名称显示
const currentModelName = computed(() => {
    const modelMap = {
        'deepseek': 'DeepSeek',
        'qwen': 'Qwen',
        'glm': 'GLM',
        'glm-4v': 'GLM-4.6V',
        'qwen-local': 'Qwen3.5-9B'
    }
    return modelMap[currentSession.value?.modelType] || 'DeepSeek'
})

// 当前会话标题
const currentSessionTitle = computed(() => {
    if (activeIndex.value === -1) return 'AI 助手'
    const session = sessionList.value[activeIndex.value]
    return session?.firstMessage || '新会话'
})

// 是否可以上传图片（仅 GLM-4.6V 和 Qwen3.5-9B 支持）
const canUploadImage = computed(() => {
    const modelType = currentSession.value?.modelType
    return modelType === 'glm-4v' || modelType === 'qwen-local'
})

const goToKnowledge = () => {
    navigateTo('/knowledge')
}

const chatContainerRef = ref(null)
const hasMore = ref(true)
const loadingMore = ref(false)
const pageSize = 20
const totalMessages = ref(0)
const oldestCursor = ref('') // 最旧消息的时间戳游标
/** @type {Map<string, { close: () => void }>} */
const activeStreams = new Map()
/** @type {Map<string, { messages: any[], oldestCursor: string, hasMore: boolean, totalMessages: number }>} */
const sessionCaches = new Map()

const cloneSessionMessages = (messages) =>
    messages.map(m => ({
        ...m,
        sources: m.sources ? [...m.sources] : [],
        posts: m.posts ? [...m.posts] : []
    }))

const saveCurrentSessionCache = () => {
    const session = currentSession.value
    if (!session) return
    sessionCaches.set(session.sessionId, {
        messages: cloneSessionMessages(messageList.value),
        oldestCursor: oldestCursor.value,
        hasMore: hasMore.value,
        totalMessages: totalMessages.value
    })
}

const restoreSessionCache = (sessionId) => {
    const cache = sessionCaches.get(sessionId)
    if (!cache) return false
    messageList.value = cloneSessionMessages(cache.messages)
    oldestCursor.value = cache.oldestCursor
    hasMore.value = cache.hasMore
    totalMessages.value = cache.totalMessages
    return true
}

const mergeCacheTailIfNewer = (sessionId) => {
    const cache = sessionCaches.get(sessionId)
    if (!cache?.messages?.length) return
    const lastCache = cache.messages[cache.messages.length - 1]
    if (!lastCache || lastCache.isSelf || !lastCache.content) return
    const lastLocal = messageList.value[messageList.value.length - 1]
    const cacheLen = lastCache.content.length
    const localLen = lastLocal?.isSelf ? 0 : (lastLocal?.content?.length || 0)
    if (cacheLen > localLen) {
        if (lastLocal && !lastLocal.isSelf) {
            messageList.value[messageList.value.length - 1] = { ...lastCache }
        } else {
            messageList.value.push({ ...lastCache })
        }
    }
}

const updateCachedMessage = (sessionId, aiMsgIndex, updater) => {
    const cache = sessionCaches.get(sessionId)
    const cacheMsg = cache?.messages[aiMsgIndex]
    const listMsg = currentSession.value?.sessionId === sessionId
        ? messageList.value[aiMsgIndex]
        : null

    // 浅拷贝缓存时可能与 messageList 共享同一对象，避免对同一引用执行两次 updater
    if (cacheMsg && listMsg && cacheMsg === listMsg) {
        updater(cacheMsg)
    } else {
        if (cacheMsg) updater(cacheMsg)
        if (listMsg) updater(listMsg)
    }
    if (cache) sessionCaches.set(sessionId, cache)
}

const stopSessionStream = (sessionId) => {
    const stream = activeStreams.get(sessionId)
    if (stream) {
        stream.close()
        activeStreams.delete(sessionId)
    }
    activeStreamSessions.value.delete(sessionId)
    activeStreamSessions.value = new Set(activeStreamSessions.value)
}

const currentSession = computed(() => {
    if (activeIndex.value === -1) return null
    return sessionList.value[activeIndex.value]
})

const selectedImagePreview = computed(() => {
    if (!selectedImage.value) return ''
    return URL.createObjectURL(selectedImage.value)
})

// 选择模型
const selectModel = (model) => {
    const session = currentSession.value
    if (session) {
        session.modelType = model
    }
    showModel.value = false
}

// 选择图片
const chooseImage = () => {
    fileInput.value?.click()
    showUpload.value = false
}

// 文件选择变化
const handleFileChange = (event) => {
    const file = event.target.files[0]
    if (file) {
        selectedImage.value = file
    }
    // 清空 input 以便重复选择同一文件
    event.target.value = ''
}

// 移除图片
const removeImage = () => {
    if (selectedImage.value) {
        URL.revokeObjectURL(selectedImagePreview.value)
        selectedImage.value = null
    }
}

// 判断 URL 是否为图片
const isImageUrl = (url) => {
    if (!url) return false
    if (url.startsWith('blob:')) return true
    if (url.startsWith('data:image/')) return true
    return /\.(jpg|jpeg|png|gif|webp|bmp)$/i.test(url)
}

// 图片预览
const previewImage = (url) => {
    previewImageUrl.value = url
    showImageViewer.value = true
}

// 打开帖子详情
const openDiscoverDetail = (post) => {
    selectedDiscoverItem.value = {
        id: post.id,
        src: post.src,
        title: post.title,
        avatar: post.avatar,
        nickname: post.nickname
    }
    showDiscoverDetail.value = true
}

// 滚动到底部
const scrollToBottom = () => {
    nextTick(() => {
        if (chatContainerRef.value) {
            chatContainerRef.value.scrollTop = chatContainerRef.value.scrollHeight
        }
    })
}

// 滚动到顶部加载更多
const handleScroll = () => {
    if (chatContainerRef.value && chatContainerRef.value.scrollTop < 50) {
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
    saveCurrentSessionCache()
    activeIndex.value = index
    const session = sessionList.value[index]
    if (!session) return

    const sessionId = session.sessionId

    if (sessionCaches.has(sessionId)) {
        restoreSessionCache(sessionId)
        scrollToBottom()
        if (!activeStreams.has(sessionId)) {
            await loadMessageList(sessionId)
            mergeCacheTailIfNewer(sessionId)
            saveCurrentSessionCache()
        }
        return
    }

    hasMore.value = true
    loadingMore.value = false
    oldestCursor.value = ''
    messageList.value = []

    await loadMessageList(sessionId)
    saveCurrentSessionCache()
    scrollToBottom()
}

// 加载消息历史（游标分页）
const loadMessageList = async (sessionId, cursor = '', direction = 'prev') => {
    try {
        if (!sessionId) return

        const data = await aiChat.getMessageHistory(sessionId, pageSize, cursor, direction)
        const list = data.list || []
        totalMessages.value = data.total || 0

        const messages = list.map(msg => ({
            messageId: msg.session_id + '_' + msg.created_at,
            content: msg.content,
            senderName: msg.send_name,
            sendTime: msg.created_at,
            isSelf: msg.send_id && !msg.send_id.startsWith('A'),
            type: msg.type,
            url: msg.url || '',
            sources: msg.sources ? JSON.parse(msg.sources) : [],
            posts: msg.posts ? JSON.parse(msg.posts) : []
        }))

        if (!cursor) {
            messageList.value = messages.reverse()
        } else if (direction === 'prev') {
            messageList.value = [...messages.reverse(), ...messageList.value]
        } else {
            messageList.value = [...messageList.value, ...messages]
        }

        if (list.length > 0) {
            const oldestMsg = list[list.length - 1]
            oldestCursor.value = oldestMsg.created_at
        }

        hasMore.value = messageList.value.length < totalMessages.value
    } catch (error) {
        console.error('获取消息历史失败:', error)
    }
}

// 加载更多消息
const loadMoreMessages = async () => {
    if (loadingMore.value || !hasMore.value || !currentSession.value || !oldestCursor.value) return
    loadingMore.value = true

    const oldScrollHeight = chatContainerRef.value?.scrollHeight || 0

    await loadMessageList(currentSession.value.sessionId, oldestCursor.value, 'prev')

    nextTick(() => {
        if (chatContainerRef.value) {
            const newScrollHeight = chatContainerRef.value.scrollHeight
            chatContainerRef.value.scrollTop = newScrollHeight - oldScrollHeight
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

    // 添加用户消息
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

    if (imageFile) {
        removeImage()
    }

    inputMessage.value = ''
    // 重置 textarea 高度
    if (textareaRef.value) {
        textareaRef.value.style.height = 'auto'
    }

    // 添加 AI 流式消息占位
    const aiMsgIndex = messageList.value.length
    messageList.value.push({
        messageId: 'ai_' + Date.now(),
        content: '',
        senderName: 'AI 助手',
        sendTime: '',
        isSelf: false,
        isStreaming: true,
        sources: [],
        posts: []
    })

    const sessionId = session.sessionId
    const timeStr = () => new Date().toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })

    saveCurrentSessionCache()
    activeStreamSessions.value.add(sessionId)
    activeStreamSessions.value = new Set(activeStreamSessions.value)
    scrollToBottom()

    const streamHandle = aiChat.sendMessage(
        sessionId,
        content,
        session.modelType,
        imageFile,
        useKnowledgeBase.value,
        useWebSearch.value,
        // onChunk
        (chunk) => {
            updateCachedMessage(sessionId, aiMsgIndex, (aiMsg) => {
                aiMsg.content += chunk
            })
            if (currentSession.value?.sessionId === sessionId) {
                scrollToBottom()
            }
        },
        // onSources
        (sources) => {
            updateCachedMessage(sessionId, aiMsgIndex, (aiMsg) => {
                aiMsg.sources = sources
            })
        },
        // onPosts
        (posts) => {
            updateCachedMessage(sessionId, aiMsgIndex, (aiMsg) => {
                aiMsg.posts = posts
            })
        },
        // onComplete
        () => {
            updateCachedMessage(sessionId, aiMsgIndex, (aiMsg) => {
                aiMsg.isStreaming = false
                aiMsg.sendTime = timeStr()
            })
            activeStreams.delete(sessionId)
            activeStreamSessions.value.delete(sessionId)
            activeStreamSessions.value = new Set(activeStreamSessions.value)
            const aiMsg = sessionCaches.get(sessionId)?.messages[aiMsgIndex]
            const s = sessionList.value.find(item => item.sessionId === sessionId)
            if (s && aiMsg) {
                s.lastMessage = aiMsg.content
            }
            saveCurrentSessionCache()
            if (currentSession.value?.sessionId === sessionId) {
                scrollToBottom()
            }
        },
        // onError
        (error) => {
            updateCachedMessage(sessionId, aiMsgIndex, (aiMsg) => {
                aiMsg.isStreaming = false
                if (!aiMsg.content) {
                    aiMsg.content = '抱歉，响应出现错误：' + error
                }
                aiMsg.sendTime = timeStr()
            })
            activeStreams.delete(sessionId)
            activeStreamSessions.value.delete(sessionId)
            activeStreamSessions.value = new Set(activeStreamSessions.value)
            saveCurrentSessionCache()
            if (currentSession.value?.sessionId === sessionId) {
                ElMessage.error('AI 响应失败')
            }
        }
    )
    activeStreams.set(sessionId, streamHandle)
}

// 创建新会话
const handleCreateSession = async () => {
    // 先保存当前会话的缓存（使用sessionId，不依赖activeIndex）
    const currentSessionId = currentSession.value?.sessionId
    if (currentSessionId) {
        sessionCaches.set(currentSessionId, {
            messages: cloneSessionMessages(messageList.value),
            oldestCursor: oldestCursor.value,
            hasMore: hasMore.value,
            totalMessages: totalMessages.value
        })
    }

    const result = await aiChat.createSession('deepseek')
    if (result) {
        await loadSessionList()
        const idx = sessionList.value.findIndex(s => s.sessionId === result.session_id)
        if (idx !== -1) {
            // 直接设置activeIndex，不调用selectSession避免重复保存
            activeIndex.value = idx
            // 清空消息列表，准备加载新会话
            messageList.value = []
            hasMore.value = true
            loadingMore.value = false
            oldestCursor.value = ''
            totalMessages.value = 0
        }
        ElMessage.success('创建会话成功')
    } else {
        ElMessage.error('创建会话失败')
    }
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
            stopSessionStream(item.sessionId)
            sessionCaches.delete(item.sessionId)
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
    document.addEventListener('click', closeMenus)
    await getCurrentUserInfo()
    await loadSessionList()
})

onUnmounted(() => {
    document.removeEventListener('click', closeMenus)
    for (const sessionId of activeStreams.keys()) {
        stopSessionStream(sessionId)
    }
})
</script>

<style scoped>
/* 会话栏 - 需要过渡动画 */
.session-sidebar {
    width: 260px;
    background: #fafafa;
    border-right: 1px solid #ececec;
    transition: width 0.3s ease;
    overflow: hidden;
}

.session-sidebar.collapsed {
    width: 0;
}

/* 收缩时立即隐藏文字内容，避免挤压变形 */
.session-sidebar.collapsed .session-header,
.session-sidebar.collapsed .divider,
.session-sidebar.collapsed .session-list {
    opacity: 0;
    transition: opacity 0.15s ease;
}

.session-sidebar:not(.collapsed) .session-header,
.session-sidebar:not(.collapsed) .divider,
.session-sidebar:not(.collapsed) .session-list {
    opacity: 1;
    transition: opacity 0.2s ease 0.1s;
}

.session-header {
    height: 72px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 18px;
}

.new-chat,
.collapse-btn {
    border: none;
    background: transparent;
    cursor: pointer;
    font-size: 14px;
    color: #333;
    display: flex;
    align-items: center;
    transition: color 0.15s ease;
}

.new-chat:hover,
.collapse-btn:hover {
    color: #0073ff;
}

.divider {
    height: 1px;
    background: #ececec;
    margin: 0 18px;
}

.session-list {
    padding: 20px 18px;
    overflow-y: auto;
    flex: 1;
}

.session-item {
    padding: 10px 0;
    cursor: pointer;
    border-bottom: 1px solid #f0f0f0;
    transition: all 0.15s ease;
}

.session-item:hover {
    background: #f0f0f0;
    margin: 0 -18px;
    padding: 10px 18px;
}

.session-item.active {
    color: #0073ff;
}

.delete-btn {
    opacity: 0;
    border: none;
    background: transparent;
    cursor: pointer;
    font-size: 14px;
    padding: 4px;
    display: flex;
    align-items: center;
    transition: opacity 0.15s ease;
}

.session-item:hover .delete-btn {
    opacity: 1;
}

/* 流式状态指示器 */
.streaming-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #0073ff;
}

.streaming-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #0073ff;
    animation: pulse 1.5s ease-in-out infinite;
    margin-right: 6px;
}

/* 头部按钮 */
.action-icon {
    width: 36px;
    height: 36px;
    border: none;
    background: transparent;
    cursor: pointer;
    border-radius: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #666;
    transition: background 0.15s ease;
}

.action-icon:hover {
    background: #ececec;
}

.header-btn {
    height: 32px;
    padding: 0 12px;
    border: none;
    border-radius: 16px;
    background: #f0f0f0;
    cursor: pointer;
    font-size: 13px;
    color: #666;
    display: flex;
    align-items: center;
    transition: background 0.15s ease;
}

.header-btn:hover {
    background: #e5e5e5;
}

/* 消息列表区域 */
.chat-container {
    flex: 1;
    overflow-y: auto;
    padding: 40px 0 220px;
}

/* AI 消息 */
.ai-message-wrapper {
    max-width: 80%;
    font-size: 16px;
    line-height: 1.6;
    color: #333;
}

/* 思考中动画 */
.thinking-animation {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
}

.thinking-dots {
    display: flex;
    gap: 4px;
}

.thinking-dots .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #0073ff;
    animation: thinking-bounce 1.4s ease-in-out infinite;
}

.thinking-dots .dot:nth-child(2) {
    animation-delay: 0.2s;
}

.thinking-dots .dot:nth-child(3) {
    animation-delay: 0.4s;
}

.thinking-text {
    font-size: 13px;
    color: #0073ff;
    animation: thinking-fade 1.4s ease-in-out infinite;
}

@keyframes thinking-bounce {
    0%, 80%, 100% {
        transform: scale(0.6);
        opacity: 0.4;
    }
    40% {
        transform: scale(1);
        opacity: 1;
    }
}

@keyframes thinking-fade {
    0%, 100% {
        opacity: 0.4;
    }
    50% {
        opacity: 1;
    }
}

/* 消息操作按钮 */
.message-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
    justify-content: flex-start;
}

.action-btn {
    width: 36px;
    height: 36px;
    border: none;
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #6b7280;
    transition: background 0.15s ease, color 0.15s ease;
}

.action-btn:hover {
    background: #e5e7eb;
    color: #374151;
}

.copy-success {
    color: #10b981 !important;
    animation: copy-pop 0.3s ease;
}

@keyframes copy-pop {
    0% { transform: scale(1); }
    50% { transform: scale(1.3); }
    100% { transform: scale(1); }
}

/* 用户消息 */
.user-message {
    max-width: 70%;
    padding: 14px 20px;
    background: white;
    color: #333;
    border-radius: 20px 20px 4px 20px;
    font-size: 16px;
    line-height: 1.6;
    box-shadow: 0 2px 8px rgba(0,0,0,.06);
}

/* 输入框区域 */
.input-wrapper {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 20px;
    display: flex;
    justify-content: center;
    pointer-events: none;
    z-index: 10;
}

.input-box {
    width: min(900px, calc(100% - 80px));
    pointer-events: auto;
    background: white;
    border-radius: 28px;
    padding: 20px;
    box-shadow: 0 6px 24px rgba(0,0,0,.06);
}

.remove-img-btn {
    position: absolute;
    top: -6px;
    right: -6px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: #ef4444;
    color: white;
    border: none;
    cursor: pointer;
    font-size: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
}

textarea {
    width: 100%;
    min-height: 50px;
    max-height: 155px;
    padding: 0;
    border: none;
    outline: none;
    resize: none;
    font-size: 16px;
    font-family: inherit;
    line-height: 1.6;
    overflow-y: auto;
}

.input-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 12px;
}

.left-actions {
    display: flex;
    align-items: center;
    gap: 12px;
}

.footer-btn {
    height: 38px;
    padding: 0 16px;
    border: none;
    border-radius: 20px;
    background: #f5f5f5;
    cursor: pointer;
    font-size: 14px;
    color: #666;
    display: flex;
    align-items: center;
    transition: background 0.15s ease;
}

.footer-btn:hover {
    background: #e5e5e5;
}

/* 下拉箭头旋转动画 */
.dropdown-arrow {
    transition: transform 0.25s ease;
    display: inline-block;
}

.dropdown-arrow.arrow-up {
    transform: rotate(180deg);
}

.dropdown {
    position: relative;
}

.dropdown-menu {
    position: absolute;
    bottom: 48px;
    left: 0;
    min-width: 180px;
    background: white;
    border-radius: 16px;
    overflow: hidden;
    box-shadow: 0 8px 24px rgba(0,0,0,.12);
    z-index: 1000;
    animation: dropdown-show 0.2s ease;
}

@keyframes dropdown-show {
    from {
        opacity: 0;
        transform: translateY(8px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.menu-item {
    padding: 14px 18px;
    cursor: pointer;
    font-size: 14px;
    color: #333;
    transition: background 0.15s ease;
}

.menu-item:hover {
    background: #f5f5f5;
}

.menu-item-disabled {
    opacity: 0.5;
    cursor: not-allowed;
    color: #9ca3af;
}

.menu-item-disabled:hover {
    background: transparent;
}

.menu-item.active {
    color: #0073ff;
    font-weight: 500;
}

.send-btn {
    width: 42px;
    height: 42px;
    border: none;
    border-radius: 50%;
    background: #0073ff;
    color: white;
    cursor: pointer;
    font-size: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.2s ease, transform 0.15s ease;
}

.send-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.send-btn:not(:disabled):hover {
    background: #0060d9;
    transform: scale(1.05);
}

.send-btn:not(:disabled):active {
    transform: scale(0.95);
}

/* 滚动条样式 */
@keyframes pulse {
    0%, 100% {
        opacity: 1;
    }
    50% {
        opacity: 0.5;
    }
}
</style>
