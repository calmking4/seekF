<template>
  <div class="flex h-screen bg-gray-100">
    <!-- 左侧侧边栏 -->
    <aside class="bg-white border-r flex flex-col pr-3 relative shrink-0" :style="{ width: sidebarWidth + 'px' }">
      <!-- 拖动条 -->
      <div class="sidebar-resize-handle" @mousedown="startSidebarResize"></div>
      <!-- 顶部搜索栏 - 参考样式 -->
      <SearchBar />

      <!-- 好友/群聊切换 -->
      <el-tabs v-model="activeTab" class="flex-1 overflow-y-auto [&_.el-tabs\_\_nav-scroll]:flex [&_.el-tabs\_\_nav-scroll]:justify-center [&_.el-tabs\_\_nav]:!float-none [&_.el-tabs\_\_nav]:gap-10 [&_.el-tabs\_\_item]:!px-2" @tab-click="handleTabClick">
        <!-- 好友列表 -->
        <el-tab-pane label="好友" name="friend">
          <div class="py-1">
            <div v-if="friends.length === 0" class="p-8 text-center text-gray-400">
              暂无好友
            </div>
            <div v-else class="space-y-1">
              <div
                v-for="friend in friends"
                :key="friend.id"
                class="flex items-center gap-3 px-3 py-2 hover:bg-gray-100 cursor-pointer"
                :class="{ 'bg-blue-50 text-blue-600': currentView === 'userProfile' && selectedContact?.id === friend.id }"
                @click="selectFriend(friend)"
              >
                <el-avatar :size="32" :src="friend.avatar" />
                <span class="text-sm">{{ friend.name }}</span>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 群聊列表 -->
        <el-tab-pane label="群聊" name="group">
          <div class="py-1">
            <div
              v-for="(group, index) in groupCategories"
              :key="group.name"
              class="border-b last:border-b-0"
            >
              <button
                type="button"
                :aria-expanded="!!group.expanded"
                :aria-controls="`contact-group-${index}`"
                class="flex w-full items-center justify-between px-3 py-2 text-left transition-colors hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400"
                @click="group.expanded = !group.expanded"
              >
                <div class="flex items-center gap-2">
                  <!-- 箭头替换为 Nuxt Icon 并添加旋转效果 -->
                  <Icon 
                    name="uil:angle-right" 
                    class="text-gray-400 transition-transform duration-[360ms] ease-[cubic-bezier(.22,1,.36,1)] motion-reduce:transition-none"
                    :class="{ 'rotate-90': group.expanded }"
                  />
                  <span>{{ group.name }}</span>
                </div>
                <span class="text-xs text-gray-400">{{ group.count }}</span>
              </button>
              <!-- 保留列表节点，通过网格行高和透明度实现可中断的展开与收起。 -->
              <div
                :id="`contact-group-${index}`"
                :aria-hidden="!group.expanded"
                :inert="!group.expanded"
                class="grid transition-[grid-template-rows,opacity] duration-[360ms] ease-[cubic-bezier(.22,1,.36,1)] motion-reduce:transition-none"
                :class="group.expanded ? 'grid-rows-[1fr] opacity-100' : 'pointer-events-none grid-rows-[0fr] opacity-0'"
              >
                <div class="min-h-0 overflow-hidden bg-gray-50">
                  <div
                    v-for="item in group.list"
                    :key="item.group_id || item.id"
                    class="flex items-center gap-3 px-6 py-2 hover:bg-gray-100 cursor-pointer"
                    @click="currentView = 'chat'; selectGroup(item)"
                  >
                    <el-avatar :size="32" :src="item.avatar" />
                    <span class="text-sm">{{ item.group_name || item.name }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="flex-1 min-w-0 flex flex-col">
      <!-- 顶部操作栏 -->
      <header class="h-10 border-b flex items-center justify-between px-3 gap-2">
        <div class="flex items-center gap-2">
          <h1 class="text-sm font-medium">{{ getCurrentViewTitle() }}</h1>
        </div>
        <div class="flex items-center gap-2">
          <el-button size="small" type="primary" link icon="Monitor" />
          <el-button size="small" type="primary" link icon="Minus" />
          <el-button size="small" type="primary" link icon="FullScreen" />
          <el-button size="small" type="primary" link icon="Close" />
        </div>
      </header>

      <!-- 内容区 -->
      <div class="flex-1 min-h-0 overflow-y-auto">
        <!-- 聊天视图 -->
        <div v-if="currentView === 'chat'" class="flex-1 flex flex-col items-center justify-center text-gray-400">
          <Icon name="uil:comment-alt" class="text-6xl mb-3" />
          <p v-if="selectedContact">与 {{ selectedContact.name }} 聊天</p>
          <p v-else>选择一个好友或群聊开始聊天</p>
        </div>
        
        <UserProfile
          v-if="currentView === 'userProfile' && selectedContact"
          :user-id="selectedContact.id"
          embedded
        />

        <!-- 群组信息视图 -->
        <div v-if="currentView === 'groupInfo'" class="flex-1 p-6 overflow-auto">
          <div v-if="groupInfo" class="bg-white rounded-lg shadow-sm p-6">
            <!-- 基本信息 -->
            <div class="mb-8">
              <div class="flex items-center gap-6">
                <el-avatar :size="100" :src="groupInfo.avatar" class="flex-shrink-0 border-4 border-gray-100" />
                <div class="flex-1">
                  <h2 class="text-2xl font-bold mb-2">{{ groupInfo.name }}</h2>
                  <div class="flex items-center gap-3 text-sm text-gray-500">
                    <span>{{ groupInfo.member_cnt }} 人</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 详细信息 -->
            <div class="space-y-4">
              <div class="bg-gray-50 rounded-lg p-4">
                <h3 class="text-lg font-medium mb-3">基本信息</h3>
                <div class="grid grid-cols-2 gap-4">
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">群组ID</span>
                    <span class="font-medium">{{ groupInfo.uuid }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">群组名称</span>
                    <span class="font-medium">{{ groupInfo.name }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">群主</span>
                    <span class="font-medium">{{ groupInfo.owner_id }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">加群方式</span>
                    <span class="font-medium">{{ groupInfo.add_mode === 0 ? '直接加入' : groupInfo.add_mode === 1 ? '需要验证' : '禁止加入' }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">成员数量</span>
                    <span class="font-medium">{{ groupInfo.member_cnt }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">群组状态</span>
                    <span class="font-medium">{{ groupInfo.status === 0 ? '正常' : '已解散' }}</span>
                  </div>
                </div>
              </div>
              
              <div class="bg-gray-50 rounded-lg p-4">
                <h3 class="text-lg font-medium mb-3">群组资料</h3>
                <div class="flex flex-col">
                  <span class="text-sm text-gray-500 mb-1">群公告</span>
                  <span class="font-medium">{{ groupInfo.notice || '暂无公告' }}</span>
                </div>
              </div>
              
            </div>
            
            <!-- 底部按钮 -->
            <div class="flex gap-4 mt-8">
              <el-button type="default" class="flex-1">群成员管理</el-button>
              <el-button type="primary" class="flex-1" @click="openSession(groupInfo.uuid)">发消息</el-button>
            </div>
          </div>
          <div v-else class="flex-1 flex flex-col items-center justify-center text-gray-400">
            <Icon name="uil:users-alt" class="text-6xl mb-3" />
            <p>加载群组信息失败</p>
          </div>
        </div>
        
        <!-- 默认视图 -->
        <div v-if="currentView === 'default'" class="flex-1 flex flex-col items-center justify-center text-gray-400">
          <Icon name="uil:comment-alt" class="text-6xl mb-3" />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useApi$ } from '~/composables/useApi'
import { ElMessage } from 'element-plus'

// 页面级 SEO
useSeoMeta({
  title: '联系人',
  description: '管理好友和群组，查看用户主页和群组信息。',
})

const activeTab = ref('friend')
const currentView = ref('default')
const selectedContact = ref(null)

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

// 好友列表数据
const friends = ref([])

// 群聊分类数据
const groupCategories = ref([])

// 获取当前视图标题
const getCurrentViewTitle = () => {
  switch (currentView.value) {
    case 'chat':
      return selectedContact.value ? selectedContact.value.name : '聊天'
    case 'userProfile':
      return selectedContact.value ? `${selectedContact.value.name} 的主页` : '用户主页'
    case 'groupInfo':
      return selectedContact.value ? selectedContact.value.name : '群组信息'
    default:
      return '联系人'
  }
}

// 选择好友
const selectFriend = (friend) => {
  selectedContact.value = friend
  currentView.value = 'userProfile'
}

// 群组信息数据
const groupInfo = ref(null)

// 选择群聊
const selectGroup = async (group) => {
  selectedContact.value = {
    id: group.group_id || group.id,
    name: group.group_name || group.name,
    avatar: group.avatar
  }
  currentView.value = 'groupInfo'
  
  try {
    const data = await useApi$('/user/group/getGroupInfo', {
      method: 'POST',
      body: {
        group_id: group.group_id || group.id
      }
    })
    
    if (data && data.code === 200) {
      groupInfo.value = data.data
    } else {
      ElMessage.error(data?.message || '获取群组信息失败')
      groupInfo.value = null
    }
  } catch (error) {
    console.error('获取群组信息失败:', error)
    groupInfo.value = null
  }
}

// 加载好友列表
const loadFriends = async () => {
  try {
    const data = await useApi$('/user/contact/getUserList', {
      method: 'POST'
    })
    
    if (data && data.code === 200) {
      const friendList = data.data || []
      
      // 直接存储好友列表，不使用分组
      friends.value = friendList.map((friend) => ({
        id: friend.user_id,
        name: friend.user_name,
        avatar: friend.avatar
      }))
    } else {
      ElMessage.error(data?.message || '获取好友列表失败')
      // 如果获取失败，显示空的好友列表
      friends.value = []
    }
  } catch (error) {
    console.error('获取好友列表失败:', error)
    // 如果网络错误，显示空的好友列表
    friends.value = []
  }
}

// 处理标签页点击
const handleTabClick = (tab) => {
  currentView.value = 'default'
  selectedContact.value = null
  groupInfo.value = null
  
  // 当点击好友标签页时，加载好友列表
  if (tab?.props?.name === 'friend') {
    loadFriends()
  }
  // 当点击群聊标签页时，加载群聊列表
  if (tab?.props?.name === 'group') {
    loadMyGroup()
    loadMyJoinedGroup()
  }
}

// 打开会话
const openSession = async (receiveId) => {
  try {
    const data = await useApi$('/user/session/openSession', {
      method: 'POST',
      body: {
        receive_id: receiveId
      }
    })

    if (data && data.code === 200) {
      ElMessage.success('会话已打开')
      // 获取会话ID
      const sessionId = data.data
      // 跳转到聊天页面，并传递会话ID和接收者ID
      await navigateTo({
        path: '/chat',
        query: {
          session_id: sessionId,
          receive_id: receiveId
        }
      })
    } else {
      ElMessage.error(data?.message || '打开会话失败')
    }
  } catch (error) {
    console.error('打开会话失败:', error)
    ElMessage.error('打开会话失败')
  }
}

// 获取我创建的群聊
const loadMyGroup = async () => {
  try {
    const data = await useApi$('/user/group/loadMyGroup', {
      method: 'POST'
    })
    
    if (data && data.code === 200) {
      const groups = data.data || []
      // 更新群聊分类数据，只使用真实数据
      if (groupCategories.value.length === 0) {
        groupCategories.value.push({
          name: '我创建的群聊',
          count: groups.length,
          expanded: false,
          list: groups
        })
      } else {
        groupCategories.value[0].list = groups
        groupCategories.value[0].count = groups.length
      }
    } else {
      ElMessage.error(data?.message || '获取我创建的群聊失败')
      // 如果获取失败，显示空的群聊列表
      if (groupCategories.value.length === 0) {
        groupCategories.value.push({
          name: '我创建的群聊',
          count: 0,
          expanded: false,
          list: []
        })
      } else {
        groupCategories.value[0].list = []
        groupCategories.value[0].count = 0
      }
    }
  } catch (error) {
    console.error('获取我创建的群聊失败:', error)
    // 如果网络错误，显示空的群聊列表
    if (groupCategories.value.length === 0) {
      groupCategories.value.push({
        name: '我创建的群聊',
        count: 0,
        expanded: false,
        list: []
      })
    } else {
      groupCategories.value[0].list = []
      groupCategories.value[0].count = 0
    }
  }
}

// 获取我加入的群聊
const loadMyJoinedGroup = async () => {
  try {
    const data = await useApi$('/user/group/loadMyJoinedGroup', {
      method: 'POST'
    })
    
    if (data && data.code === 200) {
      const groups = data.data || []
      // 更新群聊分类数据，只使用真实数据
      if (groupCategories.value.length < 2) {
        groupCategories.value.push({
          name: '我加入的群聊',
          count: groups.length,
          expanded: false,
          list: groups
        })
      } else {
        groupCategories.value[1].list = groups
        groupCategories.value[1].count = groups.length
      }
    } else {
      ElMessage.error(data?.message || '获取我加入的群聊失败')
      // 如果获取失败，显示空的群聊列表
      if (groupCategories.value.length < 2) {
        groupCategories.value.push({
          name: '我加入的群聊',
          count: 0,
          expanded: false,
          list: []
        })
      } else {
        groupCategories.value[1].list = []
        groupCategories.value[1].count = 0
      }
    }
  } catch (error) {
    console.error('获取我加入的群聊失败:', error)
    // 如果网络错误，显示空的群聊列表
    if (groupCategories.value.length < 2) {
      groupCategories.value.push({
        name: '我加入的群聊',
        count: 0,
        expanded: false,
        list: []
      })
    } else {
      groupCategories.value[1].list = []
      groupCategories.value[1].count = 0
    }
  }
}

onMounted(() => {
  loadFriends()
  loadMyGroup()
  loadMyJoinedGroup()
})
</script>

<style scoped>
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
