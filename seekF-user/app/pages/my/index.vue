<template>
  <div class="min-h-screen bg-white">
    <!-- 顶部用户信息区 -->
    <div class="max-w-3xl mx-auto pt-8 pb-6 px-6">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-6">
          <!-- 头像 -->
          <div class="w-24 h-24 rounded-full border border-gray-100 flex items-center justify-center overflow-hidden bg-gray-50">
            <img v-if="userInfo.avatar" :src="userInfo.avatar" alt="头像" class="w-full h-full object-cover" />
            <Icon v-else name="uil:user" class="text-4xl text-gray-400" />
          </div>

          <!-- 用户信息 -->
          <div class="flex flex-col gap-2">
            <h1 class="text-xl font-semibold text-gray-900">{{ userInfo.nickname || '加载中...' }}</h1>
            <p class="text-gray-500 text-sm">小红书号: {{ userInfo.telephone || '加载中...' }}</p>
            <p class="text-gray-600 text-sm">{{ userInfo.signature || '还没有简介' }}</p>
            <div class="flex gap-6 text-sm text-gray-600 mt-1">
              <span><span class="font-medium">{{ userInfo.followCount || 0 }}</span> 关注</span>
              <span><span class="font-medium">{{ userInfo.followerCount || 0 }}</span> 粉丝</span>
              <span><span class="font-medium">{{ userInfo.likeCount || 0 }}</span> 获赞与收藏</span>
            </div>
          </div>
        </div>

        <!-- 编辑信息按钮 -->
        <button class="px-5 py-2 border border-red-500 text-red-500 rounded-md text-sm font-medium hover:bg-red-50 hover:text-red-600 transition-colors" @click="editInfo">
          编辑信息
        </button>
      </div>
    </div>

    <!-- 标签切换区 -->
    <div class="max-w-3xl mx-auto">
      <el-tabs v-model="activeTab" class="w-full">
        <!-- 笔记（发布）标签 -->
        <el-tab-pane label="笔记" name="notes">
          <div class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:smile" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">你还没有发布任何内容哦</p>
          </div>
        </el-tab-pane>

        <!-- 收藏标签 -->
        <el-tab-pane label="收藏" name="collections">
          <div class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:bookmark" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">暂无收藏内容</p>
          </div>
        </el-tab-pane>

        <!-- 点赞标签 -->
        <el-tab-pane label="点赞" name="likes">
          <div class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:heart" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">暂无点赞内容</p>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 编辑用户信息弹窗 -->
    <el-dialog v-model="editDialogVisible" title="编辑个人信息" width="400px" center>
      <el-form :model="editForm" label-width="80px" @submit.prevent>
        <el-form-item label="昵称">
          <el-input v-model="editForm.nickname" placeholder="请输入昵称"></el-input>
        </el-form-item>
        <el-form-item label="头像">
          <el-input v-model="editForm.avatar" placeholder="请输入头像链接"></el-input>
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="editForm.email" placeholder="请输入邮箱"></el-input>
        </el-form-item>
        <el-form-item label="生日">
          <el-date-picker
            v-model="editForm.birthday"
            type="date"
            placeholder="选择日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD">
          </el-date-picker>
        </el-form-item>
        <el-form-item label="个性签名">
          <el-input 
            v-model="editForm.signature" 
            type="textarea" 
            :rows="3"
            placeholder="请输入个性签名"
            maxlength="100"
            show-word-limit
          ></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="cancelEdit">取消</el-button>
          <el-button type="primary" @click="confirmEdit" :loading="updating">确认</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useApi$ } from '~/composables/useApi'
import { ElMessage } from 'element-plus'
import { useAuthState } from '~/composables/useAuthState'

const activeTab = ref('notes')
const editDialogVisible = ref(false)
const updating = ref(false)

const userInfo = reactive({
  uuid: '',
  nickname: '',
  telephone: '',
  avatar: '',
  email: '',  // 添加邮箱字段
  birthday: '',  // 添加生日字段
  signature: '',
  followCount: 0,
  followerCount: 0,
  likeCount: 0
})

// 编辑表单
const editForm = reactive({
  uuid: '',
  nickname: '',
  avatar: '',
  email: '',
  birthday: '',
  signature: ''
})

// 获取用户信息
const loadUserInfo = async () => {
  try {
    // 获取当前用户的UUID，通常从前端状态或JWT token中获得
    const authState = useAuthState()
    const user = authState.getUser()


    // 使用useApi$发送请求
    const data = await useApi$('/user/getUserinfo', {
      method: 'POST',
      body: {
        uuid: user.uuid
      }
    })

    if (data && data.code === 200) {
      // 将获取到的用户信息填充到响应式对象中
      Object.assign(userInfo, {
        uuid: data.data.uuid,
        nickname: data.data.nickname,
        telephone: data.data.telephone,
        avatar: data.data.avatar,
        email: data.data.email,  // 添加邮箱
        birthday: data.data.birthday,  // 添加生日
        signature: data.data.signature,
        // 这些计数可能需要单独的API获取，暂时设为默认值
        followCount: data.data.followCount || 0,
        followerCount: data.data.followerCount || 0,
        likeCount: data.data.likeCount || 0
      })
    } else {
      console.error('获取用户信息失败:', data?.message || '未知错误')
    }
  } catch (err) {
    console.error('获取用户信息时发生错误:', err)
  }
}

const editInfo = () => {
  // 将当前用户信息复制到编辑表单
  Object.assign(editForm, {
    uuid: userInfo.uuid,
    nickname: userInfo.nickname,
    avatar: userInfo.avatar,
    email: userInfo.email || '',
    birthday: userInfo.birthday || '',
    signature: userInfo.signature
  })
  
  editDialogVisible.value = true
}

// 取消编辑
const cancelEdit = () => {
  editDialogVisible.value = false
}

// 确认编辑
const confirmEdit = async () => {
  updating.value = true
  
  try {
    const updateData = { ...editForm }
    
    // 发送更新请求
    const data = await useApi$('/user/updateUserInfo', {
      method: 'POST',
      body: updateData
    })

    if (data && data.code === 200) {
      // 更新成功，刷新用户信息
      Object.assign(userInfo, {
        nickname: updateData.nickname,
        avatar: updateData.avatar,
        email: updateData.email,
        birthday: updateData.birthday,
        signature: updateData.signature
      })
      
      ElMessage.success('更新用户信息成功')
      editDialogVisible.value = false
    } else {
      console.error('更新用户信息失败:', data?.message || '未知错误')
      ElMessage.error(data?.message || '更新用户信息失败')
    }
  } catch (err) {
    console.error('更新用户信息时发生错误:', err)
    ElMessage.error('网络错误，请稍后重试')
  } finally {
    updating.value = false
  }
}

onMounted(() => {
  loadUserInfo()
})
</script>

<style scoped>
/* 自定义 Element Plus 标签样式，匹配小红书风格 */
:deep(.el-tabs__header) {
  margin: 0;
  border-bottom: 1px solid #e5e7eb;
}

:deep(.el-tabs__nav-wrap) {
  padding: 0;
}

:deep(.el-tabs__nav) {
  display: flex;
  border: none;
}

:deep(.el-tabs__item) {
  padding: 0 24px;
  height: 48px;
  line-height: 48px;
  color: #6b7280;
  font-size: 15px;
  border: none;
  position: relative;
}

:deep(.el-tabs__item.is-active) {
  color: #111827;
  font-weight: 500;
}

:deep(.el-tabs__item.is-active::after) {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 2px;
  background-color: #111827;
  transform: none;
}

:deep(.el-tabs__active-bar) {
  display: none;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>