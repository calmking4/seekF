<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="handleUpdateVisible"
    title="全网搜索"
    width="800px"
    @close="handleClose"
  >
    <div class="mb-4">
      <el-input
        v-model="keyword"
        placeholder="输入搜索关键词"
        class="w-full"
        @keyup.enter="search"
      >
        <template #append>
          <el-button type="primary" @click="search">搜索</el-button>
        </template>
      </el-input>
    </div>
    <el-tabs v-model="activeTab">
      <el-tab-pane label="用户" name="user">
        <div class="min-h-64 flex items-center justify-center" v-if="!userResults.length">
          <div class="text-center">
            <div class="w-20 h-20 mx-auto mb-4 rounded-full border-2 border-gray-200 flex items-center justify-center">
              <el-icon class="w-10 h-10 text-gray-300"><User /></el-icon>
            </div>
            <p class="text-gray-500">输入关键词搜索用户</p>
          </div>
        </div>
        <div v-else class="max-h-80 overflow-y-auto">
          <el-card v-for="user in userResults" :key="user.user_id || user.id" class="mb-2">
            <div class="flex items-center gap-3">
              <el-avatar :size="40" :src="user.avatar" />
              <div class="flex-1">
                <div class="text-sm font-medium">{{ user.nickname || user.name }}</div>
                <div class="text-xs text-gray-500">{{ user.phone || user.email }}</div>
              </div>
              <el-button v-if="user.is_friend" type="info" size="small" disabled>已在好友</el-button>
              <el-button v-else-if="user.is_applied" type="warning" size="small" disabled>已申请</el-button>
              <el-button v-else type="primary" size="small" @click="showApplyDialog(user, 'user')">添加</el-button>
            </div>
          </el-card>
        </div>
      </el-tab-pane>
      <el-tab-pane label="群聊" name="group">
        <div class="min-h-64 flex items-center justify-center" v-if="!groupResults.length">
          <div class="text-center">
            <div class="w-20 h-20 mx-auto mb-4 rounded-full border-2 border-gray-200 flex items-center justify-center">
              <el-icon class="w-10 h-10 text-gray-300"><ChatLineSquare /></el-icon>
            </div>
            <p class="text-gray-500">输入关键词搜索群聊</p>
          </div>
        </div>
        <div v-else class="max-h-80 overflow-y-auto">
          <el-card v-for="group in groupResults" :key="group.group_id || group.id" class="mb-2">
            <div class="flex items-center gap-3">
              <el-avatar :size="40" :src="group.avatar" />
              <div class="flex-1">
                <div class="text-sm font-medium">{{ group.group_name || group.name }}</div>
                <div class="text-xs text-gray-500">群聊</div>
              </div>
              <el-button v-if="group.is_in_group" type="info" size="small" disabled>已在群中</el-button>
              <el-button v-else-if="group.is_applied" type="warning" size="small" disabled>已申请</el-button>
              <el-button v-else-if="group.add_mode === 0" type="primary" size="small" @click="joinGroupDirectly(group)">直接加入</el-button>
              <el-button v-else type="primary" size="small" @click="showApplyDialog(group, 'group')">申请加入</el-button>
            </div>
          </el-card>
        </div>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
  
  <!-- 申请加入对话框 -->
  <el-dialog
    v-model="applyDialogVisible"
    :title="applyForm.contactType === 'group' ? '申请加入群聊' : '申请添加好友'"
    width="500px"
  >
    <el-form :model="applyForm" label-width="80px">
      <el-form-item :label="applyForm.contactType === 'group' ? '群聊名称' : '用户名称'">
        <el-input v-model="applyForm.contactName" disabled />
      </el-form-item>
      <el-form-item label="申请理由">
        <el-input
          v-model="applyForm.message"
          type="textarea"
          :rows="4"
          placeholder="请输入申请理由"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="applyDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitApply">提交申请</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue'
import { ElMessage } from 'element-plus'
import { useApi$ } from '~/composables/useApi'
import { User, ChatLineSquare } from '@element-plus/icons-vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:visible', 'close'])

const activeTab = ref('user')
const keyword = ref('')
const userResults = ref([])
const groupResults = ref([])
const applyDialogVisible = ref(false)
const applyForm = ref({
  contactId: '',
  contactName: '',
  contactType: '', // 'user' 或 'group'
  message: ''
})

const handleClose = () => {
  emit('update:visible', false)
  emit('close')
  // 重置搜索状态
  keyword.value = ''
  userResults.value = []
  groupResults.value = []
}

const handleUpdateVisible = (value) => {
  emit('update:visible', value)
  if (!value) {
    // 重置搜索状态
    keyword.value = ''
    userResults.value = []
    groupResults.value = []
  }
}

const search = async () => {
  if (!keyword.value.trim()) {
    ElMessage.warning('请输入搜索关键词')
    return
  }
  
  if (activeTab.value === 'user') {
    // 搜索用户
    try {
      const data = await useApi$('/user/contact/searchUsers', {
        method: 'POST',
        body: {
          keyword: keyword.value
        }
      })
      
      if (data && data.code === 200) {
        const users = data.data || []
        
        // 获取我的好友列表
        const contactData = await useApi$('/user/contact/getUserList', {
          method: 'POST'
        })
        
        const friendList = contactData && contactData.code === 200 ? contactData.data || [] : []
        const friendIds = new Set(friendList.map(f => f.id))
        
        // 获取我的申请列表
        const applyData = await useApi$('/user/contact/getMyApplyList', {
          method: 'POST'
        })
        
        const applyList = applyData && applyData.code === 200 ? applyData.data || [] : []
        const appliedIds = new Set(applyList.filter(a => !a.is_received).map(a => a.contact_id))
        
        // 为用户添加状态
        userResults.value = users.map(user => {
          const userId = user.user_id || user.id
          return {
            ...user,
            is_friend: friendIds.has(userId),
            is_applied: appliedIds.has(userId)
          }
        })
      } else {
        ElMessage.error(data?.message || '搜索用户失败')
      }
    } catch (error) {
      console.error('搜索用户失败:', error)
      ElMessage.error('网络错误，请稍后重试')
    }
  } else if (activeTab.value === 'group') {
    // 搜索群聊
    try {
      const data = await useApi$('/user/group/searchGroups', {
        method: 'POST',
        body: {
          keyword: keyword.value
        }
      })
      
      if (data && data.code === 200) {
        groupResults.value = data.data || []
      } else {
        ElMessage.error(data?.message || '搜索群聊失败')
      }
    } catch (error) {
      console.error('搜索群聊失败:', error)
      ElMessage.error('网络错误，请稍后重试')
    }
  }
}

const joinGroupDirectly = async (group) => {
  try {
    const enterData = await useApi$('/user/group/enterGroupDirectly', {
      method: 'POST',
      body: {
        group_id: group.group_id
      }
    })
    
    if (enterData && enterData.code === 200) {
      ElMessage.success('加入群聊成功')
      // 更新群聊状态
      group.is_in_group = true
    } else {
      ElMessage.error(enterData?.message || '加入群聊失败')
    }
  } catch (error) {
    console.error('加入群聊失败:', error)
    ElMessage.error('网络错误，请稍后重试')
  }
}

const showApplyDialog = (contact, type) => {
  // 兼容不同后端返回字段：用户用 user_id，群用 group_id
  const contactId = type === 'user'
    ? (contact.user_id || contact.id)
    : (contact.group_id || contact.id)

  applyForm.value = {
    contactId,
    contactName: contact.nickname || contact.group_name || contact.name,
    contactType: type, // 'user' 或 'group'
    message: ''
  }
  applyDialogVisible.value = true
}

const submitApply = async () => {
  try {
    // 后端允许 message 为空；为空时使用默认文案以匹配列表展示效果
    if (!applyForm.value.contactId) {
      ElMessage.error('无法获取联系人ID，请稍后重试')
      return
    }
    const submitMessage = applyForm.value.message.trim() || '已发送验证消息'

    const applyData = await useApi$('/user/contact/applyContact', {
      method: 'POST',
      body: {
        contact_id: applyForm.value.contactId,
        message: submitMessage
      }
    })
    
    if (applyData && applyData.code === 200) {
      if (applyForm.value.contactType === 'group') {
        ElMessage.success('申请已发送，等待群主审核')
        // 更新群聊状态
        const group = groupResults.value.find(g => g.group_id === applyForm.value.contactId)
        if (group) {
          group.is_applied = true
        }
      } else {
        ElMessage.success('好友申请已发送')
      }
      applyDialogVisible.value = false
    } else {
      ElMessage.error(applyData?.message || '申请失败')
    }
  } catch (error) {
    console.error('申请失败:', error)
    ElMessage.error('网络错误，请稍后重试')
  }
}
</script>

<style scoped>
/* 自定义样式 */
.el-dialog__body {
  padding: 20px;
}

.el-card {
  border-radius: 8px;
  overflow: hidden;
}
</style>
