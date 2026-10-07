<template>
  <div class="min-h-screen bg-white">
    <div class="mx-auto max-w-5xl px-6 pt-6 pb-10">
      <button type="button" class="mb-6 flex items-center gap-2 text-sm text-gray-500 hover:text-blue-500" @click="navigateTo('/discover')"><Icon name="mdi:arrow-left" class="text-lg" />返回发现</button>
      <div v-if="loading && !profile" class="py-12" aria-label="用户主页加载中"><el-skeleton :rows="4" animated /></div>
      <div v-else-if="!profile" class="py-20 text-center">
        <Icon name="uil:user-exclamation" class="mb-3 text-4xl text-gray-300" />
        <p class="mb-4 text-sm text-gray-500">{{ profileError || '暂时无法获取用户信息' }}</p>
        <el-button @click="loadUser">重新加载</el-button>
      </div>
      <template v-else>
        <div class="flex flex-wrap items-center justify-between gap-6 pb-7">
          <div class="flex min-w-0 items-center gap-4 sm:gap-6">
            <div class="flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-full border border-gray-100 bg-gray-50">
              <img v-if="profile.avatar" :src="profile.avatar" :alt="`${profile.nickname}的头像`" class="h-full w-full object-cover" />
              <Icon v-else name="uil:user" class="text-4xl text-gray-400" />
            </div>
            <div class="flex min-w-0 flex-col gap-2">
              <h1 class="break-words text-xl font-semibold text-gray-900">{{ profile.nickname || '用户' }}</h1>
              <p class="break-all text-sm text-gray-500">账号：{{ profile.uuid }}</p>
              <p class="break-words text-sm text-gray-600">{{ profile.signature || '还没有简介' }}</p>
              <div class="mt-1 flex flex-wrap gap-x-6 gap-y-2 text-sm text-gray-600">
                <button type="button" class="hover:text-blue-500" @click="navigateTo(`/user/${userId}/following`)"><span class="font-medium">{{ profile.following_count || 0 }}</span> 关注</button>
                <button type="button" class="hover:text-blue-500" @click="navigateTo(`/user/${userId}/followers`)"><span class="font-medium">{{ profile.follower_count || 0 }}</span> 粉丝</button>
                <span><span class="font-medium">{{ profile.total_likes || 0 }}</span> 获赞</span>
              </div>
            </div>
          </div>
          <div v-if="!isSelf" class="flex items-center gap-3">
            <el-button :type="profile.is_followed ? 'default' : 'primary'" :loading="followLoading" round @click="handleToggleFollow">{{ profile.is_followed ? '已关注' : '关注' }}</el-button>
            <el-button round @click="navigateTo(`/chat?userId=${encodeURIComponent(userId)}`)"><Icon name="uil:comment-alt" class="mr-1" />发消息</el-button>
          </div>
          <el-button v-else round @click="navigateTo('/my')">我的主页</el-button>
        </div>
        <ProfileTabs v-model="activeTab" :tabs="[{ name: 'posts', label: `帖子 ${profile.post_count || 0}` }, { name: 'folders', label: '公开收藏夹' }]">
          <template #posts>
            <div class="py-4">
              <div v-if="posts.length === 0 && !loading" class="empty-state"><Icon name="uil:image" class="mb-4 text-4xl text-gray-300" /><p>{{ postsError || '暂无帖子' }}</p><el-button v-if="postsError" class="mt-4" @click="fetchProfile">重试</el-button></div>
              <ProfilePostGrid v-else :posts="posts" @open="openPost" @like="toggleLike" />
              <div v-if="posts.length" class="pt-6 text-center"><el-button v-if="!postsNoMore || postsError" :loading="loading" @click="fetchProfile">{{ postsError ? '重试加载' : '加载更多帖子' }}</el-button><p v-else class="text-xs text-gray-400">没有更多内容了</p></div>
            </div>
          </template>
          <template #folders>
            <div class="py-4">
              <template v-if="!currentFolder">
                <p class="mb-4 text-sm text-gray-500">共 {{ folders.length }} 个公开收藏夹</p>
                <div v-if="foldersLoading" class="empty-state"><Icon name="uil:spinner" class="mb-3 animate-spin text-2xl" />加载中...</div>
                <div v-else-if="folders.length === 0" class="empty-state"><Icon name="uil:folder" class="mb-4 text-4xl text-gray-300" /><p>{{ foldersError || '暂无公开收藏夹' }}</p><el-button v-if="foldersError" class="mt-4" @click="fetchFolders">重试</el-button></div>
                <div v-else class="grid grid-cols-2 gap-4 md:grid-cols-3">
                  <button v-for="folder in folders" :key="folder.uuid" type="button" class="overflow-hidden rounded-xl border border-gray-100 bg-white text-left shadow-sm transition-shadow hover:shadow-md" @click="enterFolder(folder)">
                    <div class="h-40 bg-gray-100"><img v-if="folder.cover_url" :src="folder.cover_url" :alt="folder.name" class="h-full w-full object-cover" /><div v-else class="flex h-full items-center justify-center text-gray-300"><Icon name="uil:folder" class="text-4xl" /></div></div>
                    <div class="p-3"><h3 class="truncate text-sm font-medium text-gray-800">{{ folder.name }}</h3><p v-if="folder.description" class="mt-1 truncate text-xs text-gray-500">{{ folder.description }}</p><div class="mt-2 flex justify-between text-xs"><span class="text-gray-400">{{ folder.post_count || 0 }} 篇</span><span class="text-blue-400">公开</span></div></div>
                  </button>
                </div>
              </template>
              <template v-else>
                <div class="mb-5 flex items-center gap-3"><button type="button" aria-label="返回公开收藏夹" class="rounded-lg p-2 text-gray-500 hover:bg-gray-50 hover:text-blue-500" @click="backToFolders"><Icon name="mdi:arrow-left" class="block text-xl" /></button><div><h3 class="text-base font-medium text-gray-800">{{ currentFolder.name }}</h3><p class="mt-1 text-xs text-gray-400">{{ currentFolder.post_count || 0 }} 篇 · 公开</p></div></div>
                <div v-if="folderPostsLoading && folderPosts.length === 0" class="empty-state"><Icon name="uil:spinner" class="mb-3 animate-spin text-2xl" />加载中...</div>
                <div v-else-if="folderPosts.length === 0" class="empty-state"><Icon name="uil:bookmark" class="mb-4 text-4xl text-gray-300" /><p>{{ folderPostsError || '收藏夹暂无内容' }}</p><el-button v-if="folderPostsError" class="mt-4" @click="loadFolderPosts">重试</el-button></div>
                <ProfilePostGrid v-else :posts="folderPosts" @open="openPost" @like="toggleLike" />
                <div v-if="folderPosts.length" class="pt-6 text-center"><el-button v-if="!folderPostsNoMore || folderPostsError" :loading="folderPostsLoading" @click="loadFolderPosts">{{ folderPostsError ? '重试加载' : '加载更多帖子' }}</el-button><p v-else class="text-xs text-gray-400">没有更多内容了</p></div>
              </template>
            </div>
          </template>
        </ProfileTabs>
      </template>
    </div>
    <DiscoverDetail v-if="selectedPost" :key="selectedPost.id" :item="selectedPost" @close="selectedPost = null" @like-updated="handleLikeUpdated" @collect-updated="handleCollectUpdated" @follow-updated="handleFollowUpdated" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'

const route = useRoute()
const auth = useAuthState()
const userId = computed(() => String(route.params.id || ''))
const profile = ref(null)
const isSelf = computed(() => userId.value === auth.getUser()?.uuid)
const activeTab = ref('posts')
const posts = ref([])
const loading = ref(true)
const profileError = ref('')
const postsError = ref('')
const postsPage = ref(1)
const postsNoMore = ref(false)
const folders = ref([])
const foldersLoading = ref(false)
const foldersError = ref('')
const currentFolder = ref(null)
const folderPosts = ref([])
const folderPostsLoading = ref(false)
const folderPostsError = ref('')
const folderPostsPage = ref(1)
const folderPostsNoMore = ref(false)
const followLoading = ref(false)
const selectedPost = ref(null)
const pendingLikes = new Set()
const pageSize = 12
let userVersion = 0
let folderVersion = 0

useSeoMeta({
  title: () => profile.value?.nickname ? `${profile.value.nickname} 的主页` : '用户主页',
  description: () => profile.value?.signature || '查看用户的帖子和公开收藏夹',
})

// 核对路由版本，避免快速切换用户后旧响应覆盖新主页。
const fetchProfile = async () => {
  if (loading.value || postsNoMore.value || !userId.value) return
  const version = userVersion
  const page = postsPage.value
  loading.value = true
  postsError.value = ''
  try {
    const res = await useApi$('/user/discover/profile', { body: { user_id: userId.value, page, page_size: pageSize } })
    if (version !== userVersion) return
    if (res.code !== 200 || !res.data) throw new Error(res.message || '获取用户主页失败')
    const data = res.data
    if (page === 1) profile.value = data
    const list = data.posts || []
    const existing = new Set(posts.value.map(post => post.uuid))
    posts.value.push(...list.filter(post => !existing.has(post.uuid)))
    postsNoMore.value = list.length < pageSize || posts.value.length >= (data.total ?? data.post_count ?? 0)
    postsPage.value = page + 1
  } catch (error) {
    if (version !== userVersion) return
    console.error('获取用户主页失败:', error)
    if (!profile.value) profileError.value = error.message || '获取用户主页失败，请稍后重试'
    else postsError.value = '加载帖子失败，请稍后重试'
  } finally {
    if (version === userVersion) loading.value = false
  }
}

const fetchFolders = async () => {
  if (foldersLoading.value || !userId.value) return
  const version = userVersion
  foldersLoading.value = true
  foldersError.value = ''
  try {
    const res = await useApi$('/user/discover/folder/list', { body: { user_id: userId.value } })
    if (version !== userVersion) return
    if (res.code !== 200) throw new Error(res.message || '获取公开收藏夹失败')
    folders.value = (res.data?.list || []).filter(folder => folder.is_public)
  } catch (error) {
    if (version !== userVersion) return
    console.error('获取公开收藏夹失败:', error)
    foldersError.value = '获取公开收藏夹失败，请稍后重试'
  } finally {
    if (version === userVersion) foldersLoading.value = false
  }
}

const backToFolders = () => {
  folderVersion++
  currentFolder.value = null
  folderPosts.value = []
  folderPostsLoading.value = false
  folderPostsError.value = ''
  folderPostsPage.value = 1
  folderPostsNoMore.value = false
}

const loadUser = async () => {
  userVersion++
  profile.value = null
  profileError.value = ''
  postsError.value = ''
  posts.value = []
  postsPage.value = 1
  postsNoMore.value = false
  folders.value = []
  loading.value = false
  foldersLoading.value = false
  followLoading.value = false
  selectedPost.value = null
  activeTab.value = 'posts'
  backToFolders()
  await Promise.all([fetchProfile(), fetchFolders()])
}

const loadFolderPosts = async () => {
  if (!currentFolder.value || folderPostsLoading.value || folderPostsNoMore.value) return
  const version = folderVersion
  const page = folderPostsPage.value
  folderPostsLoading.value = true
  folderPostsError.value = ''
  try {
    const res = await useApi$('/user/discover/collected-list', { body: { folder_uuid: currentFolder.value.uuid, page, page_size: pageSize } })
    if (version !== folderVersion) return
    if (res.code !== 200) throw new Error(res.message || '获取收藏夹帖子失败')
    const list = res.data?.list || []
    const existing = new Set(folderPosts.value.map(post => post.uuid))
    folderPosts.value.push(...list.filter(post => !existing.has(post.uuid)))
    folderPostsNoMore.value = list.length < pageSize || folderPosts.value.length >= (res.data?.total || 0)
    folderPostsPage.value = page + 1
  } catch (error) {
    if (version !== folderVersion) return
    console.error('获取收藏夹帖子失败:', error)
    folderPostsError.value = '获取收藏夹帖子失败，请稍后重试'
  } finally {
    if (version === folderVersion) folderPostsLoading.value = false
  }
}

const enterFolder = (folder) => {
  backToFolders()
  currentFolder.value = folder
  return loadFolderPosts()
}

const handleToggleFollow = async () => {
  if (followLoading.value || isSelf.value || !profile.value) return
  const version = userVersion
  const wasFollowing = !!profile.value.is_followed
  followLoading.value = true
  try {
    const res = await useApi$('/user/follow/toggle', { body: { follow_user_id: userId.value } })
    if (version !== userVersion) return
    if (res.code !== 200) throw new Error(res.message || '关注操作失败')
    const following = !!res.data.is_followed
    profile.value.is_followed = following
    profile.value.follower_count = Math.max(0, (profile.value.follower_count || 0) + Number(following) - Number(wasFollowing))
    ElMessage.success(following ? '关注成功' : '已取消关注')
  } catch (error) {
    if (version !== userVersion) return
    console.error('关注操作失败:', error)
    ElMessage.error('关注操作失败，请稍后重试')
  } finally {
    if (version === userVersion) followLoading.value = false
  }
}

const openPost = (post) => {
  selectedPost.value = { id: post.uuid, user_id: post.user_id, src: post.first_url, type: post.media_type === 1 ? 'video' : 'image', title: post.title, avatar: post.avatar, nickname: post.nickname }
}

const handleFollowUpdated = ({ userId: author, isFollowing }) => {
  if (author !== userId.value || !profile.value) return
  const wasFollowing = !!profile.value.is_followed
  profile.value.is_followed = isFollowing
  profile.value.follower_count = Math.max(0, (profile.value.follower_count || 0) + Number(isFollowing) - Number(wasFollowing))
}

const handleLikeUpdated = ({ postId, likeCount, isLiked }) => {
  const ownPost = posts.value.find(post => post.uuid === postId)
  if (ownPost && profile.value) profile.value.total_likes = Math.max(0, (profile.value.total_likes || 0) + likeCount - (ownPost.like_count || 0))
  for (const list of [posts.value, folderPosts.value]) {
    const post = list.find(item => item.uuid === postId)
    if (post) Object.assign(post, { like_count: likeCount, is_liked: isLiked })
  }
}

const handleCollectUpdated = ({ postId, collectCount, isCollected }) => {
  for (const list of [posts.value, folderPosts.value]) {
    const post = list.find(item => item.uuid === postId)
    if (post) Object.assign(post, { collect_count: collectCount, is_collected: isCollected })
  }
}

const toggleLike = async (post) => {
  if (pendingLikes.has(post.uuid)) return
  const version = userVersion
  pendingLikes.add(post.uuid)
  try {
    const res = await useApi$('/user/discover/like', { body: { target_uuid: post.uuid } })
    if (version !== userVersion) return
    if (res.code !== 200) throw new Error(res.message || '点赞失败')
    handleLikeUpdated({ postId: post.uuid, likeCount: res.data.like_count, isLiked: res.data.is_liked })
  } catch (error) {
    if (version !== userVersion) return
    console.error('点赞失败:', error)
    ElMessage.error('点赞失败，请稍后重试')
  } finally {
    pendingLikes.delete(post.uuid)
  }
}

watch(userId, loadUser)
onMounted(loadUser)
onUnmounted(() => { userVersion++; folderVersion++ })
</script>

<style scoped>
.empty-state {
  display: flex;
  min-height: 260px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #6b7280;
  font-size: 14px;
}
</style>
