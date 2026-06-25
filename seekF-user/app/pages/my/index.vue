<template>
  <div class="min-h-screen bg-white">
    <!-- 顶部用户信息区 -->
    <div class="max-w-5xl mx-auto pt-8 pb-6 px-6">
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
            <p class="text-gray-500 text-sm">账号: {{ userInfo.uuid || '加载中...' }}</p>
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
    <div class="max-w-5xl mx-auto">
      <el-tabs v-model="activeTab" class="w-full" @tab-change="handleTabChange">
        <!-- 收藏标签 -->
        <el-tab-pane label="收藏" name="collections">
          <!-- 收藏夹列表视图 -->
          <div v-if="!currentFolder" class="p-4">
            <div v-if="folders.length === 0 && !foldersLoading" class="py-20 flex flex-col items-center justify-center text-gray-400">
              <div class="w-20 h-20 rounded-full border border-gray-200 flex items-center justify-center mb-4 bg-gray-50">
                <Icon name="uil:bookmark" class="text-2xl" />
              </div>
              <p class="text-sm text-gray-500 mb-4">暂无收藏夹</p>
              <button class="px-4 py-2 text-white rounded-lg text-sm transition-colors" style="background-color: #60a5fa;" @click="showCreateFolder = true">
                创建收藏夹
              </button>
            </div>
            <div v-else>
              <div class="flex justify-between items-center mb-4">
                <span class="text-sm text-gray-500">共 {{ folders.length }} 个收藏夹</span>
                <button class="px-3 py-1.5 text-white rounded-lg text-xs transition-colors" style="background-color: #60a5fa;" @click="showCreateFolder = true">
                  新建
                </button>
              </div>
              <div class="grid grid-cols-3 gap-4" ref="folderGridRef">
                <div
                  v-for="folder in folders"
                  :key="folder.uuid"
                  class="folder-card fade-in bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer relative group"
                  @click="enterFolder(folder)"
                >
                  <div class="w-full h-40 bg-gray-100">
                    <img v-if="folder.cover_url" :src="folder.cover_url" class="w-full h-full object-cover" />
                    <div v-else class="w-full h-full flex items-center justify-center text-gray-300">
                      <Icon name="uil:folder" class="text-4xl" />
                    </div>
                  </div>
                  <div class="p-3">
                    <h3 class="text-sm font-medium truncate">{{ folder.name }}</h3>
                    <div class="flex items-center justify-between mt-1">
                      <span class="text-xs text-gray-400">{{ folder.post_count }} 篇</span>
                      <span v-if="folder.is_public" class="text-xs text-blue-400">公开</span>
                      <span v-else class="text-xs text-gray-400">私密</span>
                    </div>
                  </div>
                  <!-- 更多操作 -->
                  <div class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <el-dropdown trigger="click" @command="(cmd) => handleFolderCommand(cmd, folder)">
                      <button class="w-7 h-7 rounded-full bg-black/40 text-white flex items-center justify-center" @click.stop>
                        <Icon name="mdi:dots-vertical" class="text-sm" />
                      </button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="edit">编辑</el-dropdown-item>
                          <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="foldersLoading" class="py-20 flex justify-center items-center text-gray-500">
              <Icon name="uil:spinner" class="animate-spin text-xl mr-2" />
              <span>加载中...</span>
            </div>
          </div>

          <!-- 收藏夹内帖子列表视图 -->
          <div v-else class="p-4">
            <div class="flex items-center gap-3 mb-4">
              <button class="text-gray-500 hover:text-gray-700" @click="backToFolders">
                <Icon name="mdi:arrow-left" class="text-xl" />
              </button>
              <div>
                <h3 class="text-base font-medium">{{ currentFolder.name }}</h3>
                <span class="text-xs text-gray-400">{{ currentFolder.post_count }} 篇 · {{ currentFolder.is_public ? '公开' : '私密' }}</span>
              </div>
            </div>
            <div v-if="folderPosts.length === 0 && !folderPostsLoading" class="py-20 flex flex-col items-center justify-center text-gray-400">
              <p class="text-sm text-gray-500">收藏夹暂无内容</p>
            </div>
            <div v-else class="grid grid-cols-4 gap-4" ref="folderPostsGridRef">
              <div
                v-for="item in folderPosts"
                :key="item.uuid"
                class="liked-card fade-in bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer"
                @click="handleCollectedItemClick(item)"
              >
                <div class="w-full h-48 bg-gray-100">
                  <img v-if="item.first_url" :src="item.first_url" :alt="item.title" class="w-full h-full object-cover" />
                  <div v-else class="w-full h-full flex items-center justify-center text-gray-400">
                    <Icon name="uil:image" class="text-3xl" />
                  </div>
                </div>
                <div class="p-3">
                  <h3 class="text-sm font-medium line-clamp-2 mb-2">{{ item.title }}</h3>
                  <div class="flex items-center justify-between text-xs text-gray-500">
                    <div class="flex items-center gap-2">
                      <img v-if="item.avatar" :src="item.avatar" class="w-6 h-6 rounded-full object-cover" />
                      <div v-else class="w-6 h-6 rounded-full flex items-center justify-center text-white text-xs" :style="{ backgroundColor: getAvatarColor(item.uuid) }">
                        {{ item.title?.slice(0, 1) || '?' }}
                      </div>
                      <span>{{ item.nickname || '用户' }}</span>
                    </div>
                    <div class="flex items-center gap-1 text-orange-500">
                      <Icon name="mdi:bookmark" class="text-sm" />
                      <span>{{ item.collect_count }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="folderPostsLoading" class="py-8 flex justify-center items-center text-gray-500">
              <Icon name="uil:spinner" class="animate-spin text-xl mr-2" />
              <span>加载中...</span>
            </div>
            <div v-if="!folderPostsLoading && folderPostsNoMore && folderPosts.length > 0" class="py-8 text-center text-gray-400 text-sm">
              没有更多内容了
            </div>
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
            <div class="grid grid-cols-4 gap-4" ref="likedGridRef">
              <div
                v-for="item in likedPosts"
                :key="item.uuid"
                class="liked-card fade-in bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer"
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

    <!-- 帖子详情弹窗（点赞） -->
    <DiscoverDetail
      v-if="selectedLikedItem"
      :item="selectedLikedItem"
      @close="selectedLikedItem = null"
      @like-updated="handleLikedItemLikeUpdated"
    />

    <!-- 帖子详情弹窗（收藏） -->
    <DiscoverDetail
      v-if="selectedCollectedItem"
      :item="selectedCollectedItem"
      @close="selectedCollectedItem = null"
      @collect-updated="handleCollectedItemUpdated"
    />

    <!-- 创建/编辑收藏夹弹窗 -->
    <el-dialog v-model="showCreateFolder" :title="editingFolder ? '编辑收藏夹' : '新建收藏夹'" width="400px" center>
      <el-form :model="folderForm" label-width="80px" @submit.prevent>
        <el-form-item label="名称">
          <el-input v-model="folderForm.name" placeholder="请输入收藏夹名称" maxlength="50"></el-input>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="folderForm.description" type="textarea" :rows="2" placeholder="可选描述" maxlength="200"></el-input>
        </el-form-item>
        <el-form-item label="可见性">
          <el-switch v-model="folderForm.isPublic" active-text="公开" inactive-text="私密" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showCreateFolder = false">取消</el-button>
          <el-button type="primary" @click="handleSaveFolder" :loading="folderSaving">确认</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { useApi$ } from '~/composables/useApi'
import { ElMessage } from 'element-plus'

// 页面级 SEO
useSeoMeta({
  title: '我的',
  description: '查看和编辑个人信息，管理关注和粉丝。',
})
import { useAuthState } from '~/composables/useAuthState'
import { Plus } from '@element-plus/icons-vue'

const activeTab = ref('likes')
const editDialogVisible = ref(false)
const updating = ref(false)

// 网格容器 ref（用于动画）
const likedGridRef = ref(null)
const folderGridRef = ref(null)
const folderPostsGridRef = ref(null)

// 点赞帖子相关
const likedPosts = ref([])
const likedLoading = ref(false)
const likedNoMore = ref(false)
const likedPage = ref(1)
const likedPageSize = 12
const selectedLikedItem = ref(null)

// 收藏夹相关
const folders = ref([])
const foldersLoading = ref(false)
const currentFolder = ref(null)
const folderPosts = ref([])
const folderPostsLoading = ref(false)
const folderPostsNoMore = ref(false)
const folderPostsPage = ref(1)
const folderPostsPageSize = 12
const selectedCollectedItem = ref(null)
const showCreateFolder = ref(false)
const editingFolder = ref(null)
const folderSaving = ref(false)
const folderForm = reactive({
  name: '',
  description: '',
  isPublic: false,
})

// 头像颜色数组
const avatarColors = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#FFA07A', '#98D8C8',
  '#F7DC6F', '#BB8FCE', '#85C1E9', '#F8C471', '#82E0AA'
]

// 卡片入场动画
let cardObserver = null
const observeNewItems = () => {
  nextTick(() => {
    if (!cardObserver) {
      cardObserver = new IntersectionObserver((entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('visible')
            cardObserver.unobserve(entry.target)
          }
        })
      }, { threshold: 0.1 })
    }
    document.querySelectorAll('.fade-in:not(.visible)').forEach((el) => {
      cardObserver.observe(el)
    })
  })
}

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
      observeNewItems()
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
  if (tab === 'collections') {
    currentFolder.value = null
    folderPosts.value = []
    loadFolders()
  }
}

// ========== 收藏夹 ==========

const loadFolders = async () => {
  foldersLoading.value = true
  try {
    const res = await useApi$('/user/discover/folder/list', { method: 'POST' })
    if (res.code === 200) {
      folders.value = res.data?.list || []
      observeNewItems()
    }
  } catch (e) {
    console.error('加载收藏夹失败:', e)
  } finally {
    foldersLoading.value = false
  }
}

const backToFolders = () => {
  currentFolder.value = null
  folderPosts.value = []
  folderPostsPage.value = 1
  folderPostsNoMore.value = false
  loadFolders()
}

const enterFolder = (folder) => {
  currentFolder.value = folder
  folderPosts.value = []
  folderPostsPage.value = 1
  folderPostsNoMore.value = false
  loadFolderPosts()
}

const loadFolderPosts = async () => {
  if (folderPostsLoading.value || folderPostsNoMore.value || !currentFolder.value) return
  folderPostsLoading.value = true
  try {
    const res = await useApi$('/user/discover/collected-list', {
      method: 'POST',
      body: { folder_uuid: currentFolder.value.uuid, page: folderPostsPage.value, page_size: folderPostsPageSize },
    })
    if (res.code === 200 && res.data) {
      const list = res.data.list || []
      if (list.length < folderPostsPageSize) {
        folderPostsNoMore.value = true
      }
      folderPosts.value = [...folderPosts.value, ...list]
      folderPostsPage.value++
      observeNewItems()
    }
  } catch (e) {
    console.error('加载收藏夹帖子失败:', e)
  } finally {
    folderPostsLoading.value = false
  }
}

const handleCollectedItemClick = (item) => {
  selectedCollectedItem.value = {
    ...item,
    id: item.uuid,
    src: item.first_url,
    type: item.media_type === 1 ? 'video' : 'image',
  }
}

const handleCollectedItemUpdated = ({ isCollected, collectCount }) => {
  if (!isCollected) {
    // 取消收藏，从列表中移除
    folderPosts.value = folderPosts.value.filter(p => p.uuid !== selectedCollectedItem.value?.uuid)
    // 更新收藏夹帖子数
    if (currentFolder.value) {
      currentFolder.value = { ...currentFolder.value, post_count: Math.max(0, (currentFolder.value.post_count || 1) - 1) }
    }
  } else if (selectedCollectedItem.value) {
    const idx = folderPosts.value.findIndex(p => p.uuid === selectedCollectedItem.value.uuid)
    if (idx !== -1) {
      folderPosts.value[idx] = { ...folderPosts.value[idx], collect_count: collectCount }
      folderPosts.value = [...folderPosts.value]
    }
  }
}

const handleFolderCommand = (cmd, folder) => {
  if (cmd === 'edit') {
    editingFolder.value = folder
    folderForm.name = folder.name
    folderForm.description = folder.description || ''
    folderForm.isPublic = folder.is_public
    showCreateFolder.value = true
  } else if (cmd === 'delete') {
    deleteFolder(folder)
  }
}

const deleteFolder = async (folder) => {
  try {
    const res = await useApi$('/user/discover/folder/delete', {
      method: 'POST',
      body: { uuid: folder.uuid },
    })
    if (res.code === 200) {
      ElMessage.success('删除成功')
      folders.value = folders.value.filter(f => f.uuid !== folder.uuid)
    } else {
      ElMessage.error(res.message || '删除失败')
    }
  } catch (e) {
    console.error('删除收藏夹失败:', e)
  }
}

const handleSaveFolder = async () => {
  if (!folderForm.name.trim()) {
    ElMessage.error('请输入收藏夹名称')
    return
  }
  folderSaving.value = true
  try {
    if (editingFolder.value) {
      // 编辑
      const res = await useApi$('/user/discover/folder/update', {
        method: 'POST',
        body: {
          uuid: editingFolder.value.uuid,
          name: folderForm.name.trim(),
          description: folderForm.description.trim(),
          is_public: folderForm.isPublic ? 1 : 0,
        },
      })
      if (res.code === 200) {
        ElMessage.success('更新成功')
        await loadFolders()
      } else {
        ElMessage.error(res.message || '更新失败')
      }
    } else {
      // 新建
      const res = await useApi$('/user/discover/folder/create', {
        method: 'POST',
        body: {
          name: folderForm.name.trim(),
          description: folderForm.description.trim(),
          is_public: folderForm.isPublic ? 1 : 0,
        },
      })
      if (res.code === 200) {
        ElMessage.success('创建成功')
        await loadFolders()
      } else {
        ElMessage.error(res.message || '创建失败')
      }
    }
    showCreateFolder.value = false
    editingFolder.value = null
    folderForm.name = ''
    folderForm.description = ''
    folderForm.isPublic = false
  } catch (e) {
    console.error('保存收藏夹失败:', e)
  } finally {
    folderSaving.value = false
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

onUnmounted(() => {
  cardObserver?.disconnect()
})
</script>

<style scoped>
.fade-in {
  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}

.fade-in.visible {
  opacity: 1;
  transform: translateY(0);
}

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