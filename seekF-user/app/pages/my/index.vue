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
            <p class="text-gray-500 text-sm">账号: {{ userInfo.telephone || '加载中...' }}</p>
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
      <el-tabs v-model="activeTab" class="w-full" @tab-change="handleTabChange">
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
          <div v-if="likedPosts.length === 0" class="py-20 flex flex-col items-center justify-center text-gray-400">
            <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
              <Icon name="uil:heart" class="text-2xl" />
            </div>
            <p class="text-sm text-gray-500">暂无点赞内容</p>
          </div>
          <div v-else class="p-4">
            <div class="grid grid-cols-4 gap-4">
              <div
                v-for="item in likedPosts"
                :key="item.uuid"
                class="liked-card bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer"
                @click="handleLikedItemClick(item)"
              >
                <!-- 图片 -->
                <div class="w-full h-48 bg-gray-100">
                  <img
                    v-if="item.first_url"
                    :src="item.first_url"
                    :alt="item.title"
                    class="w-full h-full object-cover"
                  />
                  <div v-else class="w-full h-full flex items-center justify-center text-gray-400">
                    <Icon name="uil:image" class="text-3xl" />
                  </div>
                </div>
                <!-- 内容 -->
                <div class="p-3">
                  <h3 class="text-sm font-medium line-clamp-2 mb-2">{{ item.title }}</h3>
                  <div class="flex items-center justify-between text-xs text-gray-500">
                    <div class="flex items-center gap-2">
                      <img
                        v-if="item.avatar"
                        :src="item.avatar"
                        class="w-6 h-6 rounded-full object-cover"
                      />
                      <div
                        v-else
                        class="w-6 h-6 rounded-full flex items-center justify-center text-white text-xs"
                        :style="{ backgroundColor: getAvatarColor(item.uuid) }"
                      >
                        {{ item.title?.slice(0, 1) || '?' }}
                      </div>
                      <span>{{ item.nickname || '用户' }}</span>
                    </div>
                    <div class="flex items-center gap-1 text-red-500">
                      <Icon name="mdi:heart" class="text-sm" />
                      <span>{{ item.like_count }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <!-- 加载更多 -->
            <div v-if="likedLoading" class="py-8 flex justify-center items-center text-gray-500">
              <Icon name="uil:spinner" class="animate-spin text-xl mr-2" />
              <span>加载中...</span>
            </div>
            <div v-if="!likedLoading && likedNoMore && likedPosts.length > 0" class="py-8 text-center text-gray-400 text-sm">
              没有更多内容了
            </div>
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
          <el-upload
            class="avatar-uploader"
            :action="useRuntimeConfig().public.apiBase+'user/file/upload'"
            :data="{ fileType: 'user_avatar' }"
            :show-file-list="false"
            :on-success="handleAvatarSuccess"
            :before-upload="beforeAvatarUpload"
            :with-credentials="true"
          >
            <img v-if="editForm.avatar" :src="editForm.avatar" class="avatar" />
            <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
          </el-upload>
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

    <!-- 帖子详情弹窗 -->
    <DiscoverDetail
      v-if="selectedLikedItem"
      :item="selectedLikedItem"
      @close="selectedLikedItem = null"
      @like-updated="handleLikedItemLikeUpdated"
    />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useApi$ } from '~/composables/useApi'
import { ElMessage } from 'element-plus'
import { useAuthState } from '~/composables/useAuthState'
import { Plus } from '@element-plus/icons-vue'

const activeTab = ref('likes')
const editDialogVisible = ref(false)
const updating = ref(false)

// 点赞帖子相关
const likedPosts = ref([])
const likedLoading = ref(false)
const likedNoMore = ref(false)
const likedPage = ref(1)
const likedPageSize = 12
const selectedLikedItem = ref(null)

// 头像颜色数组
const avatarColors = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA'
]

// 根据ID生成固定的头像颜色
const getAvatarColor = (id) => {
  const str = String(id ?? '')
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = (hash * 31 + str.charCodeAt(i)) >>> 0
  }
  return avatarColors[hash % avatarColors.length]
}

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
    // 获取个人信息
    const data = await useApi$('/user/userinfo/getMyInfo', {
      method: 'POST'
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

// 加载点赞帖子列表
const loadLikedPosts = async () => {
  if (likedLoading.value || likedNoMore.value) return
  likedLoading.value = true
  try {
    const res = await useApi$('/user/discover/liked-list', {
      method: 'POST',
      body: { page: likedPage.value, page_size: likedPageSize }
    })
    if (res.code === 200 && res.data) {
      const list = res.data.list || []
      if (list.length < likedPageSize) {
        likedNoMore.value = true
      }
      likedPosts.value = [...likedPosts.value, ...list]
      likedPage.value++
    }
  } catch (e) {
    console.error('加载点赞列表失败:', e)
  } finally {
    likedLoading.value = false
  }
}

// 点击点赞帖子卡片
const handleLikedItemClick = (item) => {
  selectedLikedItem.value = {
    ...item,
    id: item.uuid,
    src: item.first_url,
    type: item.media_type === 1 ? 'video' : 'image',
  }
}

// 处理点赞状态更新（从详情弹窗返回）
const handleLikedItemLikeUpdated = ({ isLiked, likeCount }) => {
  if (!isLiked) {
    // 取消点赞，从列表中移除
    likedPosts.value = likedPosts.value.filter(p => p.uuid !== selectedLikedItem.value?.uuid)
  } else if (selectedLikedItem.value) {
    // 更新点赞数
    const idx = likedPosts.value.findIndex(p => p.uuid === selectedLikedItem.value.uuid)
    if (idx !== -1) {
      likedPosts.value[idx] = { ...likedPosts.value[idx], like_count: likeCount }
      likedPosts.value = [...likedPosts.value]
    }
  }
}

// 标签切换处理
const handleTabChange = (tab) => {
  if (tab === 'likes' && likedPosts.value.length === 0) {
    loadLikedPosts()
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

// 处理头像上传成功
const handleAvatarSuccess = (response, uploadFile) => {
  if (response && response.code === 200) {
    // 确保 editForm.avatar 是一个字符串
    if (typeof response.data === 'object' && response.data.url) {
      editForm.avatar = response.data.url
    } else {
      editForm.avatar = response.data
    }
    ElMessage.success('头像上传成功')
  } else {
    ElMessage.error(response?.message || '头像上传失败')
  }
}

// 上传前的校验
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

// 确认编辑
const confirmEdit = async () => {
  updating.value = true
  
  try {
    const updateData = { ...editForm }
    
    // 发送更新请求
    const data = await useApi$('/user/userinfo/updateUserInfo', {
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
  } finally {
    updating.value = false
  }
}

onMounted(() => {
  loadUserInfo()
  loadLikedPosts()
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

/* 头像上传样式 */
.avatar-uploader .avatar {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-uploader-icon {
  width: 100px;
  height: 100px;
  line-height: 100px;
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

.el-upload {
  display: block;
}
</style>