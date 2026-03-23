<template>
  <div class="flex h-screen bg-gray-100">
    <!-- 左侧侧边栏 -->
    <aside class="w-80 bg-white border-r flex flex-col pr-3"> <!-- 这里添加 pr-3 -->
      <!-- 顶部搜索栏 - 参考样式 -->
      <SearchBar />

      <!-- 通知入口 - 箭头替换为 Nuxt Icon -->
      <div class="border-t">
        <div
          class="flex items-center justify-between px-3 py-2 hover:bg-gray-50 cursor-pointer"
          @click="currentView = 'friendNotification'"
        >
          <span>好友通知</span>
          <Icon name="uil:angle-right" class="text-gray-400" />
        </div>
        <div
          class="flex items-center justify-between px-3 py-2 hover:bg-gray-50 cursor-pointer"
          @click="currentView = 'groupNotification'"
        >
          <span>群通知</span>
          <Icon name="uil:angle-right" class="text-gray-400" />
        </div>
      </div>

      <!-- 好友/群聊切换 -->
      <el-tabs v-model="activeTab" class="flex-1 overflow-y-auto" @tab-click="handleTabClick">
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
              v-for="group in groupCategories"
              :key="group.name"
              class="border-b last:border-b-0"
            >
              <div
                class="flex items-center justify-between px-3 py-1.5 hover:bg-gray-50 cursor-pointer"
                @click="group.expanded = !group.expanded"
              >
                <div class="flex items-center gap-2">
                  <!-- 箭头替换为 Nuxt Icon 并添加旋转效果 -->
                  <Icon 
                    name="uil:angle-right" 
                    class="text-gray-400 transition-transform duration-200"
                    :class="{ 'rotate-90': group.expanded }"
                  />
                  <span>{{ group.name }}</span>
                </div>
                <span class="text-xs text-gray-400">{{ group.count }}</span>
              </div>
              <transition name="el-collapse-transition">
                <div v-if="group.expanded" class="bg-gray-50">
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
              </transition>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="flex-1 flex flex-col">
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
      <div class="flex-1 overflow-y-auto">
        <!-- 好友通知视图 -->
        <div v-if="currentView === 'friendNotification'" class="p-4">
          
          <!-- 别人申请加我好友 -->
          <div v-if="friendRequests.length > 0" class="mb-4">
            <h3 class="text-sm font-medium mb-2">收到的请求</h3>
            <div class="space-y-2">
              <div v-for="req in friendRequests" :key="req.contact_id" class="flex items-start gap-3 px-4 py-3 bg-white rounded-lg hover:bg-gray-50">
                <el-avatar :size="44" :src="req.contact_avatar" class="flex-shrink-0" />
                <div class="flex-1 min-w-0">
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <div class="flex items-center min-w-0 gap-2">
                        <div class="text-sm font-medium text-gray-900 truncate min-w-0">
                          {{ req.status === 1 ? (req.is_received ? '已同意加好友：' + req.contact_name : '对方已同意加好友：' + req.contact_name) : (req.is_received ? req.contact_name + ' 请求加为好友' : '请求加为好友：' + req.contact_name) }}
                        </div>
                        <span class="text-sm text-gray-500 font-normal whitespace-nowrap">
                          {{ formatApplyTime(req.apply_time) }}
                        </span>
                      </div>
                    </div>
                    <div v-if="req.status !== 0" class="text-xs text-gray-500 font-medium whitespace-nowrap">
                      {{ getApplyStatusText(req.status) }}
                    </div>
                  </div>
                  <div class="text-xs text-gray-500 mt-1 whitespace-nowrap">
                    {{ req.message }}
                  </div>
                </div>
                <div v-if="req.status === 0" class="flex gap-2 justify-center items-center">
                  <el-button type="primary" size="small" @click="passFriendRequest(req.contact_id)">同意</el-button>
                  <el-button size="small" @click="refuseFriendRequest(req.contact_id)">拒绝</el-button>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 我申请别人的状态 -->
          <div v-if="myFriendApplies.length > 0" class="mb-4">
            <h3 class="text-sm font-medium mb-2">发出的请求</h3>
            <div class="space-y-3">
              <div
                v-for="apply in myFriendApplies"
                :key="apply.contact_id"
                class="flex items-start gap-3 px-4 py-3 bg-white rounded-lg hover:bg-gray-50"
              >
                <el-avatar :size="44" :src="apply.contact_avatar" class="flex-shrink-0" />

                <div class="flex-1 min-w-0">
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <div class="flex items-center min-w-0 gap-2">
                        <div class="text-sm font-medium text-gray-900 truncate min-w-0">
                          {{ getApplyMainText(apply) }}
                        </div>
                        <span class="text-sm text-gray-500 font-normal whitespace-nowrap">
                          {{ formatApplyTime(apply.apply_time) }}
                        </span>
                      </div>
                    </div>

                    <div class="text-xs text-gray-500 font-medium whitespace-nowrap">
                      {{ getApplyStatusText(apply.status) }}
                    </div>
                  </div>

                  <div v-if="apply.status === 0" class="text-xs text-gray-500 mt-1 whitespace-nowrap">
                    申请理由：{{ getApplyRemark(apply.message) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-if="friendRequests.length === 0 && myFriendApplies.length === 0" class="p-8 text-center text-gray-500">
            暂无好友通知
          </div>
        </div>
        
        <!-- 群通知视图 -->
        <div v-if="currentView === 'groupNotification'" class="p-4">
          
          <!-- 别人申请加入我的群 -->
          <div v-if="groupRequests.length > 0" class="mb-4">
            <h3 class="text-sm font-medium mb-2">收到的请求</h3>
            <div class="space-y-2">
              <div v-for="req in groupRequests" :key="req.contact_id + req.group_id" class="flex items-start gap-3 px-4 py-3 bg-white rounded-lg hover:bg-gray-50">
                <el-avatar :size="44" :src="req.contact_avatar" class="flex-shrink-0" />
                <div class="flex-1 min-w-0">
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <div class="flex items-center min-w-0 gap-2">
                        <div class="text-sm font-medium text-gray-900 truncate min-w-0">
                          {{ req.status === 1 ? (req.is_received ? '已同意加好友：' + req.contact_name : '对方已同意加好友：' + req.contact_name) : (req.is_received ? req.contact_name + ' 请求加为好友' : '请求加为好友：' + req.contact_name) }}
                        </div>
                        <span class="text-sm text-gray-500 font-normal whitespace-nowrap">
                          {{ formatApplyTime(req.apply_time) }}
                        </span>
                      </div>
                    </div>
                    <div v-if="req.status !== 0" class="text-xs text-gray-500 font-medium whitespace-nowrap">
                      {{ getApplyStatusText(req.status) }}
                    </div>
                  </div>
                  <div class="text-xs text-gray-500 mt-1 whitespace-nowrap">
                    申请加入群聊：{{ req.group_name }}
                  </div>
                  <div class="text-xs text-gray-500 mt-1 whitespace-nowrap">
                    {{ req.message }}
                  </div>
                </div>
                <div v-if="req.status === 0" class="flex gap-2 justify-center items-center">
                  <el-button type="primary" size="small" @click="passGroupRequest(req.contact_id, req.group_id)">同意</el-button>
                  <el-button size="small" @click="refuseGroupRequest(req.contact_id, req.group_id)">拒绝</el-button>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 我申请加入别人的群的状态 -->
          <div v-if="myGroupApplies.length > 0" class="mb-4">
            <h3 class="text-sm font-medium mb-2">发出的请求</h3>
            <div class="space-y-3">
              <div
                v-for="apply in myGroupApplies"
                :key="apply.contact_id"
                class="flex items-start gap-3 px-4 py-3 bg-white rounded-lg hover:bg-gray-50"
              >
                <el-avatar :size="44" :src="apply.contact_avatar" class="flex-shrink-0" />

                <div class="flex-1 min-w-0">
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <div class="flex items-center min-w-0 gap-2">
                        <div class="text-sm font-medium text-gray-900 truncate min-w-0">
                          {{ getApplyMainText(apply) }}
                        </div>
                        <span class="text-sm text-gray-500 font-normal whitespace-nowrap">
                          {{ formatApplyTime(apply.apply_time) }}
                        </span>
                      </div>
                    </div>

                    <div class="text-xs text-gray-500 font-medium whitespace-nowrap">
                      {{ getApplyStatusText(apply.status) }}
                    </div>
                  </div>

                  <div v-if="apply.status === 0" class="text-xs text-gray-500 mt-1 whitespace-nowrap">
                    申请理由：{{ getApplyRemark(apply.message) }}
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-if="groupRequests.length === 0 && myGroupApplies.length === 0" class="p-8 text-center text-gray-500">
            暂无群通知
          </div>
        </div>
        
        <!-- 聊天视图 -->
        <div v-if="currentView === 'chat'" class="flex-1 flex flex-col items-center justify-center text-gray-400">
          <Icon name="uil:comment-alt" class="text-6xl mb-3" />
          <p v-if="selectedContact">与 {{ selectedContact.name }} 聊天</p>
          <p v-else>选择一个好友或群聊开始聊天</p>
        </div>
        
        <!-- 用户信息视图 -->
        <div v-if="currentView === 'userInfo'" class="flex-1 p-6 overflow-auto">
          <div v-if="userInfo" class="bg-white rounded-lg shadow-sm p-6">
            <!-- 基本信息 -->
            <div class="mb-8">
              <div class="flex items-center gap-6">
                <el-avatar :size="100" :src="userInfo.avatar" class="flex-shrink-0 border-4 border-gray-100" />
                <div class="flex-1">
                  <h2 class="text-2xl font-bold mb-2">{{ userInfo.nickname }}</h2>
                  <div class="flex items-center gap-3 text-sm text-gray-500">
                    <span class="flex items-center gap-1">
                      <span class="inline-block w-2 h-2 bg-green-500 rounded-full"></span>
                      在线
                    </span>
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
                    <span class="text-sm text-gray-500 mb-1">用户ID</span>
                    <span class="font-medium">{{ userInfo.uuid }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">昵称</span>
                    <span class="font-medium">{{ userInfo.nickname }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">手机号</span>
                    <span class="font-medium">{{ userInfo.telephone }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">邮箱</span>
                    <span class="font-medium">{{ userInfo.email }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">性别</span>
                    <span class="font-medium">{{ userInfo.gender === 0 ? '男' : '女' }}</span>
                  </div>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-500 mb-1">生日</span>
                    <span class="font-medium">{{ userInfo.birthday }}</span>
                  </div>
                </div>
              </div>
              
              <div class="bg-gray-50 rounded-lg p-4">
                <h3 class="text-lg font-medium mb-3">个人资料</h3>
                <div class="flex flex-col">
                  <span class="text-sm text-gray-500 mb-1">签名</span>
                  <span class="font-medium">{{ userInfo.signature || '暂无签名' }}</span>
                </div>
                <div class="flex flex-col mt-4">
                  <span class="text-sm text-gray-500 mb-1">注册时间</span>
                  <span class="font-medium">{{ userInfo.created_at }}</span>
                </div>
              </div>
              

            </div>
            
            <!-- 底部按钮 -->
            <div class="flex gap-4 mt-8">
              <el-button type="default" class="flex-1">编辑资料</el-button>
              <el-button type="primary" class="flex-1">发消息</el-button>
            </div>
          </div>
          <div v-else class="flex-1 flex flex-col items-center justify-center text-gray-400">
            <Icon name="uil:user" class="text-6xl mb-3" />
            <p>加载用户信息失败</p>
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

const activeTab = ref('friend')
const currentView = ref('default') // 'default', 'friendNotification', 'groupNotification', 'chat'
const selectedContact = ref(null)

// 好友列表数据
const friends = ref([])

// 群聊分类数据
const groupCategories = ref([])

// 好友请求数据
const friendRequests = ref([])

// 我申请的好友状态
const myFriendApplies = ref([])

// 群聊请求数据
const groupRequests = ref([])

// 我申请的群聊状态
const myGroupApplies = ref([])

// 用户信息数据
const userInfo = ref(null)

// 好友分组
const friendGroup = ref('my_friends')

// 获取申请状态文本
const getApplyStatusText = (status) => {
  switch (status) {
    case 0:
      return '等待验证认证'
    case 1:
      return '已同意'
    case 2:
      return '已拒绝'
    case 3:
      return '已拉黑'
    default:
      return '未知状态'
  }
}

// 格式化申请时间：后端返回 `YYYY-MM-DD HH:mm:ss`，前端展示 `YYYY/MM/DD`
const formatApplyTime = (applyTime) => {
  if (!applyTime) return ''
  const datePart = String(applyTime).split(' ')[0]
  return datePart ? datePart.replace(/-/g, '/') : ''
}

// 主文案：匹配截图样式（pending 显示“正在验证你的邀请”，其他显示“请求加好友/加入群聊”）
const getApplyMainText = (apply) => {
  const name = apply?.contact_name || ''
  const type = apply?.contact_type
  const isGroup = type === 'group'
  const isReceived = apply?.is_received || false

  if (apply?.status === 0) {
    if (isReceived) {
      return isGroup ? `${name} 正在验证加入群聊` : `${name} 正在验证你的邀请`
    } else {
      return isGroup ? `正在验证加入群聊：${name}` : `正在验证你的邀请：${name}`
    }
  } else if (apply?.status === 1) {
    if (isReceived) {
      return isGroup ? `已同意加入群聊：${name}` : `已同意加好友：${name}`
    } else {
      return isGroup ? `对方已同意加入群聊：${name}` : `对方已同意加好友：${name}`
    }
  } else if (apply?.status === 2) {
    if (isReceived) {
      return isGroup ? `已拒绝加入群聊：${name}` : `已拒绝加好友：${name}`
    } else {
      return isGroup ? `对方已拒绝加入群聊：${name}` : `对方已拒绝加好友：${name}`
    }
  } else if (apply?.status === 3) {
    if (isReceived) {
      return isGroup ? `已拉黑加入群聊：${name}` : `已拉黑加好友：${name}`
    } else {
      return isGroup ? `对方已拉黑加入群聊：${name}` : `对方已拉黑加好友：${name}`
    }
  }

  if (isReceived) {
    return isGroup ? `${name} 请求加入群聊` : `${name} 请求加好友`
  } else {
    return isGroup ? `请求加入群聊：${name}` : `请求加好友：${name}`
  }
}

// 留言：后端 message 可能是 `申请理由：xxx`，这里去掉前缀
const getApplyRemark = (message) => {
  const msg = message ? String(message) : ''
  return msg.replace(/^申请理由：/, '') || '无'
}

// 获取当前视图标题
const getCurrentViewTitle = () => {
  switch (currentView.value) {
    case 'friendNotification':
      return '好友通知'
    case 'groupNotification':
      return '群通知'
    case 'chat':
      return selectedContact.value ? selectedContact.value.name : '聊天'
    case 'userInfo':
      return selectedContact.value ? selectedContact.value.name : '用户信息'
    default:
      return '联系人'
  }
}

// 选择好友
const selectFriend = async (friend) => {
  selectedContact.value = friend
  currentView.value = 'userInfo'
  
  try {
    const data = await useApi$('/user/userinfo/getUserinfo', {
      method: 'POST',
      body: {
        uuid: friend.id
      }
    })
    
    if (data && data.code === 200) {
      userInfo.value = data.data
    } else {
      ElMessage.error(data?.message || '获取用户信息失败')
      userInfo.value = null
    }
  } catch (error) {
    console.error('获取用户信息失败:', error)
    userInfo.value = null
  }
}

// 选择群聊
const selectGroup = (group) => {
  selectedContact.value = {
    id: group.group_id || group.id,
    name: group.group_name || group.name,
    avatar: group.avatar
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
  
  // 当点击好友标签页时，加载好友列表
  if (tab?.props?.name === 'friend') {
    loadFriends()
  }
}

// 同意好友申请
const passFriendRequest = async (contactId) => {
  try {
    const data = await useApi$('/user/contact/passContactApply', {
      method: 'POST',
      body: {
        contact_id: contactId
      }
    })
    
    if (data && data.code === 200) {
      ElMessage.success('同意好友申请成功')
      // 重新加载通知数据
      await loadAllNotifications()
    } else {
      ElMessage.error(data?.message || '同意好友申请失败')
    }
  } catch (error) {
    console.error('同意好友申请失败:', error)
  }
}

// 拒绝好友申请
const refuseFriendRequest = async (contactId) => {
  try {
    const data = await useApi$('/user/contact/refuseContactApply', {
      method: 'POST',
      body: {
        contact_id: contactId
      }
    })
    
    if (data && data.code === 200) {
      ElMessage.success('拒绝好友申请成功')
      // 重新加载通知数据
      await loadAllNotifications()
    } else {
      ElMessage.error(data?.message || '拒绝好友申请失败')
    }
  } catch (error) {
    console.error('拒绝好友申请失败:', error)
  }
}

// 同意群聊申请
const passGroupRequest = async (contactId, groupId) => {
  try {
    const data = await useApi$('/user/contact/passContactApply', {
      method: 'POST',
      body: {
        group_id: groupId,
        contact_id: contactId
      }
    })
    
    if (data && data.code === 200) {
      ElMessage.success('同意群聊申请成功')
      // 重新加载通知数据
      await loadAllNotifications()
    } else {
      ElMessage.error(data?.message || '同意群聊申请失败')
    }
  } catch (error) {
    console.error('同意群聊申请失败:', error)
  }
}

// 拒绝群聊申请
const refuseGroupRequest = async (contactId, groupId) => {
  try {
    const data = await useApi$('/user/contact/refuseContactApply', {
      method: 'POST',
      body: {
        group_id: groupId,
        contact_id: contactId
      }
    })
    
    if (data && data.code === 200) {
      ElMessage.success('拒绝群聊申请成功')
      // 重新加载通知数据
      await loadAllNotifications()
    } else {
      ElMessage.error(data?.message || '拒绝群聊申请失败')
    }
  } catch (error) {
    console.error('拒绝群聊申请失败:', error)
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

// 获取所有通知数据
const loadAllNotifications = async () => {
  try {
    const data = await useApi$('/user/contact/getMyApplyList', {
      method: 'POST'
    })
    
    if (data && data.code === 200) {
      const allNotifications = data.data || []
      
      // 分离不同类型的通知
      friendRequests.value = allNotifications.filter(item => item.contact_type === 'user' && item.is_received)
      myFriendApplies.value = allNotifications.filter(item => item.contact_type === 'user' && !item.is_received)
      
      // 处理群聊申请
      // 对于收到的群聊申请，需要获取群聊信息
      const receivedGroupApplies = allNotifications.filter(item => item.contact_type === 'group' && item.is_received)
      const myGroupAppliesList = allNotifications.filter(item => item.contact_type === 'group' && !item.is_received)
      
      // 获取我创建的群聊，用于匹配群聊申请
      const myGroupsData = await useApi$('/user/group/loadMyGroup', {
        method: 'POST'
      })
      
      if (myGroupsData && myGroupsData.code === 200) {
        const myGroups = myGroupsData.data || []
        const groupMap = new Map()
        
        // 构建群聊ID到群聊信息的映射
        myGroups.forEach(group => {
          groupMap.set(group.group_id, group)
        })
        
        // 为收到的群聊申请添加群聊信息
        groupRequests.value = receivedGroupApplies.map(apply => {
          const group = groupMap.get(apply.contact_id)
          return {
            ...apply,
            group_id: apply.contact_id,
            group_name: group ? (group.group_name || group.name) : '未知群聊'
          }
        })
      } else {
        groupRequests.value = []
      }
      
      myGroupApplies.value = myGroupAppliesList
    } else {
      ElMessage.error(data?.message || '获取通知数据失败')
    }
  } catch (error) {
    console.error('获取通知数据失败:', error)
  }
}

onMounted(() => {
  loadFriends()
  loadMyGroup()
  loadMyJoinedGroup()
  loadAllNotifications()
})
</script>

<style scoped>
/* 自定义滚动条样式 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 3px;
}
::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

/* 旋转样式 */
.rotate-90 {
  transform: rotate(90deg);
}

/* 过渡效果 */
.transition-transform {
  transition-property: transform;
}

.duration-200 {
  transition-duration: 200ms;
}
</style>