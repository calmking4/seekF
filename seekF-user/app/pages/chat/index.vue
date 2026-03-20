<template>
  <div class="flex h-full bg-gray-100">
    <!-- 左侧：搜索栏 + 联系人/会话列表 -->
    <aside class="w-80 bg-white border-r border-gray-200 h-full flex flex-col flex-shrink-0 pr-3">
      <!-- 顶部搜索栏 -->
      <SearchBar />

      <!-- 联系人/会话列表 -->
      <div class="flex-1 overflow-y-auto">
        <div
          v-for="(item, index) in chatList"
          :key="index"
          class="flex items-center gap-3 px-3 py-3 hover:bg-gray-50 cursor-pointer transition-colors border-b border-gray-100"
          :class="{ 'bg-gray-100': activeIndex === index }"
          @click="activeIndex = index"
        >
          <!-- 头像 -->
          <div class="w-12 h-12 rounded-full bg-blue-100 flex items-center justify-center text-blue-500 flex-shrink-0">
            {{ item.avatarText }}
          </div>
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
          <div class="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-blue-500">
            {{ currentChat.avatarText }}
          </div>
          <h3 class="font-medium flex-1">{{ currentChat.name }}</h3>
          <div class="flex gap-4 text-gray-500">
            <button><Icon name="uil:search" /></button>
            <button><Icon name="uil:ellipsis-h" /></button>
          </div>
        </div>

        <!-- 聊天内容区 -->
        <div class="flex-1 p-6 overflow-y-auto space-y-4">
          <!-- 对方消息 -->
          <div class="flex items-start gap-3">
            <div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center text-gray-500 flex-shrink-0">
              {{ currentChat.avatarText }}
            </div>
            <div class="bg-white rounded-lg px-4 py-2 max-w-[60%] shadow-sm">
              <p class="text-sm">{{ currentChat.lastMsg }}</p>
              <p class="text-xs text-gray-400 text-right mt-1">09:30</p>
            </div>
          </div>

          <!-- 我方消息 -->
          <div class="flex items-start gap-3 justify-end">
            <div class="bg-[#D9FDD3] rounded-lg px-4 py-2 max-w-[60%]">
              <p class="text-sm">收到，链接已添加到邮件中。</p>
              <p class="text-xs text-gray-400 text-right mt-1">09:32</p>
            </div>
            <div class="w-8 h-8 rounded-full bg-gray-300 flex items-center justify-center text-gray-600 flex-shrink-0">
              我
            </div>
          </div>

          <!-- 系统消息 -->
          <div class="flex justify-center">
            <span class="text-xs bg-gray-300 text-white px-3 py-1 rounded-full">昨天 10:17</span>
          </div>

          <!-- 更多消息示例 -->
          <div class="flex items-start gap-3">
            <div class="w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center text-gray-500 flex-shrink-0">
              三
            </div>
            <div class="bg-white rounded-lg px-4 py-2 max-w-[60%] shadow-sm">
              <p class="text-sm text-blue-500" @click="window.open('https://www.brics-ofsmd.com/uploads/2026/Abstracts.docx')">
                https://www.brics-ofsmd.com/uploads/2026/Abstracts.docx
              </p>
              <p class="text-xs text-gray-400 text-right mt-1">10:17</p>
            </div>
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
              placeholder="请输入消息..."
              class="flex-1 border border-gray-300 rounded-lg p-2 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm resize-none"
              rows="2"
            ></textarea>
            <button class="bg-blue-500 text-white px-4 py-2 rounded-lg hover:bg-blue-600 transition-colors">
              发送
            </button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>

// 模拟聊天列表数据（参考你截图中的内容）
const chatList = ref([
  {
    name: '沉默的福尔摩斯',
    avatarText: '沉',
    lastMsg: '如何在邮件中附件件?您看是您教我一下还是您自行插入一下',
    time: '昨天',
    unread: 0
  },
  {
    name: '三叔',
    avatarText: '三',
    lastMsg: '选中要加链接的文字，加上链接就可以',
    time: '10:17',
    unread: 2
  },
  {
    name: 'TB系统开发',
    avatarText: 'T',
    lastMsg: '已形成',
    time: '01/06',
    unread: 0
  },
  {
    name: '公众号',
    avatarText: '公',
    lastMsg: '黑马程序员-首发...',
    time: '15:02',
    unread: 0
  }
])

// 当前选中的会话索引
const activeIndex = ref(-1)

// 当前选中的聊天对象
const currentChat = computed(() => {
  if (activeIndex.value === -1) return {}
  return chatList.value[activeIndex.value]
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