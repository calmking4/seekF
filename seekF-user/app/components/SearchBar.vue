<template>
  <div class="p-3 border-b border-gray-100 flex items-center gap-3">
    <div class="flex-1 relative">
      <input
        type="text"
        placeholder="搜索"
        class="w-full pl-10 pr-4 py-2 rounded-lg bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-400 text-sm"
      >
      <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">🔍</span>
    </div>
    <button @click="showCreateMenu = !showCreateMenu" class="text-gray-500 hover:text-blue-500 transition-colors">
      <Icon name="uil:plus" class="w-5 h-5" />
    </button>
    <!-- 创建菜单 -->
    <div v-if="showCreateMenu" class="absolute top-14 right-3 bg-white rounded-lg shadow-lg border border-gray-200 py-2 z-10">
      <div class="px-4 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2" @click="showCreateGroupForm = true; showCreateMenu = false">
        <Icon name="uil:users-alt" class="w-4 h-4" />
        <span>创建群聊</span>
      </div>
      <div class="px-4 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2" @click="showSearchModal = true; showCreateMenu = false">
        <Icon name="uil:user-plus" class="w-4 h-4" />
        <span>加好友/群聊</span>
      </div>
    </div>
    
    <!-- 搜索模态框 -->
    <SearchModal v-model:visible="showSearchModal" @close="showCreateMenu = false" />
    
    <!-- 创建群聊表单 -->
    <div v-if="showCreateGroupForm" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-medium">创建群聊</h3>
          <button @click="showCreateGroupForm = false" class="text-gray-500 hover:text-gray-700">
            <Icon name="uil:times" class="w-5 h-5" />
          </button>
        </div>
        <form @submit.prevent="createGroup">
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">群聊名称 <span class="text-red-500">*</span></label>
            <input
              type="text"
              v-model="createGroupForm.name"
              placeholder="请输入群聊名称"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-400"
              required
            >
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">群公告</label>
            <textarea
              v-model="createGroupForm.notice"
              placeholder="请输入群公告"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-400 resize-none"
              rows="3"
            ></textarea>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">加群方式 <span class="text-red-500">*</span></label>
            <el-radio-group v-model="createGroupForm.addMode" required>
              <el-radio label="0">直接加入</el-radio>
              <el-radio label="1">群主审核</el-radio>
            </el-radio-group>
          </div>
          <div class="mb-6">
            <label class="block text-sm font-medium text-gray-700 mb-1">群头像</label>
            <div class="flex items-center gap-3">
              <el-upload
                class="avatar-uploader"
                :action="useRuntimeConfig().public.apiBase+'user/file/upload'"
                :data="{ fileType: 'group_avatar' }"
                :show-file-list="false"
                :on-success="handleAvatarSuccess"
                :before-upload="beforeAvatarUpload"
                :with-credentials="true"
              >
                <img v-if="createGroupForm.avatar" :src="createGroupForm.avatar" class="avatar" />
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
              </el-upload>
              <div class="flex-1">
                <p class="text-xs text-gray-500">支持 JPG、PNG 格式，建议尺寸 200x200</p>
              </div>
            </div>
          </div>
          <div class="flex gap-3">
            <button type="button" @click="showCreateGroupForm = false" class="flex-1 py-2 border border-gray-300 rounded-lg hover:bg-gray-50">
              取消
            </button>
            <button type="submit" class="flex-1 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">
              创建
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useApi$ } from '~/composables/useApi'
import { Plus } from '@element-plus/icons-vue'
import { useRuntimeConfig } from 'nuxt/app'
import SearchModal from './SearchModal.vue'

const showCreateMenu = ref(false)
const showCreateGroupForm = ref(false)
const showSearchModal = ref(false)
const createGroupForm = ref({
  name: '',
  notice: '',
  addMode: '',
  avatar: ''
})

const handleAvatarSuccess = (response, uploadFile) => {
  if (response && response.code === 200) {
    if (typeof response.data === 'object' && response.data.url) {
      createGroupForm.value.avatar = response.data.url
    } else {
      createGroupForm.value.avatar = response.data
    }
    ElMessage.success('头像上传成功')
  } else {
    ElMessage.error(response?.message || '头像上传失败')
  }
}

const beforeAvatarUpload = (file) => {
  const isJPG = file.type === 'image/jpeg' || file.type === 'image/png' || file.type === 'image/gif'
  const isLt2M = file.size / 1024 / 1024 < 2

  if (!isJPG) {
    ElMessage.error('只能上传 JPG、PNG 或 GIF 格式的图片')
  }
  if (!isLt2M) {
    ElMessage.error('上传图片大小不能超过 2MB')
  }
  return isJPG && isLt2M
}

const createGroup = async () => {
  try {
    const data = await useApi$('/user/group/createGroup', {
      method: 'POST',
      body: createGroupForm.value
    })
    
    if (data && data.code === 200) {
      ElMessage.success('创建群聊成功')
      showCreateGroupForm.value = false
      createGroupForm.value = {
        name: '',
        notice: '',
        addMode: '',
        avatar: ''
      }
    } else {
      ElMessage.error(data?.message || '创建群聊失败')
    }
  } catch (error) {
    console.error('创建群聊失败:', error)
    ElMessage.error('网络错误，请稍后重试')
  }
}
</script>

<style scoped>
/* 定位样式 */
.p-3 {
  position: relative;
}

.avatar-uploader .avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-uploader-icon {
  width: 80px;
  height: 80px;
  line-height: 80px;
  border: 1px dashed #d9d9d9;
  border-radius: 50%;
  font-size: 24px;
  color: #999;
  background-color: #fafafa;
  transition: all 0.3s;
}

.avatar-uploader:hover .avatar-uploader-icon {
  border-color: #409eff;
}

.avatar-uploader {
  display: block;
}
</style>