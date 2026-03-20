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
              <el-button type="primary" size="small">添加</el-button>
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
              <el-button v-else type="primary" size="small">申请加入</el-button>
            </div>
          </el-card>
        </div>
      </el-tab-pane>
    </el-tabs>
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
        userResults.value = data.data || []
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
