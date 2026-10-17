<template>
  <div class="flex min-h-[calc(100vh-112px)] flex-col overflow-hidden rounded-2xl bg-white">
    <div v-if="editorLoading" class="flex min-h-[380px] items-center justify-center gap-2 text-sm text-gray-400" role="status"><Icon name="mdi:loading" class="animate-spin text-xl" />加载作品...</div>
    <!-- 内容区 -->
    <div v-show="!editorLoading" class="mx-auto w-full max-w-6xl flex-1 px-5 py-5 sm:px-8">
      <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
        <div class="flex items-center gap-2" aria-label="作品类型">
          <h1 v-if="isEditing" class="rounded-full bg-gray-100 px-5 py-2.5 text-sm font-medium text-gray-800">{{ mediaType === 'video' ? '视频编辑' : '图文编辑' }}</h1>
          <template v-else>
            <button type="button" class="rounded-full px-5 py-2.5 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500" :class="mediaType === 'video' ? 'bg-gray-100 text-gray-800' : 'text-gray-400 hover:text-gray-600'" :aria-pressed="mediaType === 'video'" :disabled="hasActiveUpload" @click="switchType('video')">上传视频</button>
            <button type="button" class="rounded-full px-5 py-2.5 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-500" :class="mediaType === 'image' ? 'bg-gray-100 text-gray-800' : 'text-gray-400 hover:text-gray-600'" :aria-pressed="mediaType === 'image'" :disabled="hasActiveUpload" @click="switchType('image')">上传图文</button>
          </template>
        </div>
        <div v-if="!isEditing && mediaList.length === 0" class="flex items-center gap-2"><button type="button" class="flex items-center gap-2 rounded-full bg-gray-100 px-4 py-2.5 text-sm text-gray-600 hover:bg-gray-200 disabled:opacity-50" :disabled="hasActiveUpload" @click="showDraftList"><Icon name="mdi:inbox-outline" />草稿箱<span>({{ draftCount }})</span></button></div>
      </div>
      <p v-if="!isEditing" class="mb-4 text-xs text-gray-400" role="status">{{ draftSaveStatus }}</p>
      <!-- 上传区域 -->
      <div class="mb-5">
        <div v-if="mediaList.length === 0" class="flex min-h-[380px] cursor-pointer flex-col items-center justify-center rounded-2xl bg-gray-50 px-4 py-14 sm:min-h-[55vh]" :class="dragging ? 'ring-2 ring-blue-400' : ''" role="button" tabindex="0" :aria-label="mediaType === 'video' ? '上传视频' : '上传图片'" @click="triggerUpload" @keydown.enter="triggerUpload" @keydown.space.prevent="triggerUpload" @dragover.prevent="dragging = true" @dragleave.prevent="dragging = false" @drop.prevent="handleDrop">
          <Icon name="mdi:cloud-upload" class="mb-4 text-8xl text-gray-200" />
          <p class="text-sm text-gray-600">{{ mediaType === 'video' ? '拖拽视频到此或点击上传' : '拖拽图片到此或点击上传' }}</p>
          <span class="mt-5 rounded-full bg-[#60a5fa] px-8 py-3 text-sm text-white">{{ mediaType === 'video' ? '上传视频' : '上传图片' }}</span>
          <span class="mt-4 text-xs text-gray-400">{{ mediaType === 'video' ? 'MP4 · 最大500MB · 支持暂停与断点续传' : 'JPG / PNG · 每张最大10MB · 最多9张 · 不限制像素尺寸' }}</span>
        </div>

        <!-- 视频模式：大预览 + 下方横条进度 -->
        <div v-else-if="mediaType === 'video'" class="space-y-2">
          <div
            v-for="(media, index) in mediaList"
            :key="media.url || media.fileName || index"
            class="video-upload-item"
          >
            <!-- 刷新后需重新选文件 -->
            <div
              v-if="media.uploadStatus === 'pending_reselect'"
              class="relative w-full aspect-video rounded-xl border-2 border-dashed border-[#60a5fa] overflow-hidden flex flex-col items-center justify-center cursor-pointer hover:bg-blue-50 transition-colors px-4"
              @click="triggerUpload"
            >
              <img
                v-if="media.posterUrl"
                :src="media.posterUrl"
                class="absolute inset-0 w-full h-full object-cover opacity-50"
                alt=""
              />
              <button
                class="absolute top-2 right-2 z-20 w-7 h-7 bg-black/10 rounded-full flex items-center justify-center hover:bg-black/20 transition-colors"
                @click.stop="removeMedia(index)"
              >
                <Icon name="mdi:close" class="text-gray-600 text-sm" />
              </button>
              <div class="relative z-10 flex flex-col items-center">
              <Icon name="mdi:video-plus-outline" class="text-4xl text-[#60a5fa] mb-2" />
              <span class="text-sm text-[#60a5fa] font-medium">点击重新选择视频以继续</span>
              <span v-if="media.fileName" class="text-xs text-gray-500 mt-1 truncate max-w-full">{{ media.fileName }}</span>
              <span v-if="media.uploadProgress > 0" class="text-xs text-gray-400 mt-0.5">已上传 {{ media.uploadProgress }}%</span>
              </div>
            </div>

            <template v-else>
              <div class="relative w-full aspect-video rounded-xl overflow-hidden bg-gray-200">
                <!-- 视频首帧封面 -->
                <img
                  v-if="media.posterUrl"
                  :src="media.posterUrl"
                  class="w-full h-full object-cover"
                  alt="视频封面"
                />
                <!-- 封面生成前的占位 -->
                <div
                  v-else
                  class="w-full h-full flex items-center justify-center bg-gray-100"
                >
                  <Icon name="uil:spinner" class="text-3xl text-gray-400 animate-spin" />
                </div>

                <!-- 上传中：半透明遮罩 + 加载图标 + 暂停按钮 -->
                <div
                  v-if="media.uploadStatus === 'uploading'"
                  class="absolute inset-0 bg-black/30 flex flex-col items-center justify-center"
                >
                  <Icon name="uil:spinner" class="text-5xl text-white animate-spin drop-shadow mb-4" />
                  <button
                    class="flex items-center gap-2 px-5 py-2.5 bg-white/90 rounded-full text-gray-800 font-medium text-sm hover:bg-white transition-all shadow-lg active:scale-95"
                    @click.stop="pauseVideoUpload(index)"
                  >
                    <Icon name="mdi:pause" class="text-lg" />
                    <span>暂停上传</span>
                  </button>
                </div>

                <!-- 暂停状态：显示继续和取消按钮 -->
                <div
                  v-else-if="media.uploadStatus === 'paused'"
                  class="absolute inset-0 bg-black/50 flex flex-col items-center justify-center backdrop-blur-sm"
                >
                  <div class="flex gap-4">
                    <button
                      class="flex items-center gap-2 px-6 py-3 bg-[#60a5fa] rounded-full text-white font-medium text-sm hover:bg-[#3b82f6] transition-all shadow-lg active:scale-95"
                      @click="resumeVideoUpload(index)"
                    >
                      <Icon name="mdi:play" class="text-lg" />
                      <span>继续上传</span>
                    </button>
                    <button
                      class="flex items-center gap-2 px-6 py-3 bg-white/20 rounded-full text-white font-medium text-sm hover:bg-white/30 transition-all active:scale-95"
                      @click="removeMedia(index)"
                    >
                      <Icon name="mdi:close" class="text-lg" />
                      <span>取消</span>
                    </button>
                  </div>
                  <span class="text-white text-sm mt-4 drop-shadow">已暂停 {{ media.uploadProgress }}%</span>
                </div>

                <!-- 上传成功：居中圆形已上传图标 -->
                <div
                  v-else-if="media.uploadStatus === 'done'"
                  class="absolute inset-0 flex items-center justify-center pointer-events-none"
                >
                  <div class="w-16 h-16 rounded-full bg-black/40 flex items-center justify-center backdrop-blur-sm">
                    <Icon name="mdi:check-circle" class="text-green-400 text-4xl" />
                  </div>
                </div>

                <!-- 失败：点击重试 -->
                <div
                  v-else-if="media.uploadStatus === 'error'"
                  class="absolute inset-0 bg-black/40 flex flex-col items-center justify-center cursor-pointer"
                  @click="retryVideoUpload(media, index)"
                >
                  <Icon name="mdi:refresh" class="text-4xl text-white mb-1 drop-shadow" />
                  <span class="text-white text-sm drop-shadow">上传失败，点击重试</span>
                </div>

                <button
                  v-if="media.uploadStatus !== 'uploading' && media.uploadStatus !== 'paused'"
                  class="absolute top-2 right-2 w-7 h-7 bg-black/50 rounded-full flex items-center justify-center hover:bg-black/70 transition-colors z-10"
                  @click="removeMedia(index)"
                >
                  <Icon name="mdi:close" class="text-white text-sm" />
                </button>
              </div>

              <!-- 横条进度（视频下方） -->
              <div v-if="media.uploadStatus !== 'done'" class="mt-2 px-1">
                <el-progress
                  :percentage="media.uploadProgress"
                  :stroke-width="8"
                  :show-text="false"
                  :color="media.uploadStatus === 'paused' ? '#f59e0b' : '#60a5fa'"
                />
                <div class="flex items-center justify-between mt-1.5 text-xs text-gray-500">
                  <span :class="media.uploadStatus === 'paused' ? 'text-amber-500' : ''">
                    {{ media.uploadStatusText || '准备上传...' }}
                  </span>
                  <span class="font-medium" :class="media.uploadStatus === 'paused' ? 'text-amber-500' : 'text-[#60a5fa]'">
                    {{ media.uploadProgress }}%
                  </span>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- 图片模式：缩略图列表 -->
        <div v-else class="flex gap-2 overflow-x-auto pb-2">
          <div
            v-for="(media, index) in mediaList"
            :key="index"
            class="relative flex-shrink-0 w-24 h-24 rounded-lg overflow-hidden bg-gray-100"
          >
            <img v-if="media.url" :src="media.url" class="w-full h-full object-cover" alt="待发布图片" />
            <button v-else type="button" class="flex h-full w-full flex-col items-center justify-center px-1 text-xs text-gray-400" @click="reselectImage(index)"><Icon name="mdi:image-plus-outline" class="text-xl" /><span class="line-clamp-2">{{ media.fileName || '重新选择图片' }}</span></button>
            <span v-if="index === 0" class="absolute left-1 top-1 rounded bg-black/40 px-1 text-[10px] text-white">封面</span>
            <div v-if="media.uploadStatus === 'uploading'" class="absolute inset-x-0 bottom-7 bg-black/50 px-1 text-center text-xs text-white" role="status">{{ media.uploadProgress }}%</div>
            <button v-if="media.uploadStatus === 'error' || media.uploadStatus === 'pending_reselect'" type="button" class="absolute inset-x-0 bottom-7 bg-black/50 py-1 text-xs text-white" @click="media.file ? retryImageUpload(media) : reselectImage(index)">{{ media.file ? '重试上传' : '重新选择' }}</button>
            <div class="absolute inset-x-0 bottom-0 flex items-center justify-between bg-black/50 p-1 text-white">
              <button type="button" aria-label="图片前移" :disabled="index === 0 || publishing" class="disabled:opacity-30" @click="moveImage(index, -1)"><Icon name="mdi:chevron-left" /></button>
              <button type="button" :disabled="publishing" class="text-[10px]" @click="setImageCover(index)">设为封面</button>
              <button type="button" aria-label="图片后移" :disabled="index === mediaList.length - 1 || publishing" class="disabled:opacity-30" @click="moveImage(index, 1)"><Icon name="mdi:chevron-right" /></button>
            </div>
            <button
              class="absolute top-1 right-1 w-5 h-5 bg-black/50 rounded-full flex items-center justify-center"
              @click="removeMedia(index)"
            >
              <Icon name="mdi:close" class="text-white text-xs" />
            </button>
          </div>
          <!-- 添加更多 -->
          <div
            v-if="mediaType === 'image'"
            class="flex-shrink-0 w-24 h-24 rounded-lg border-2 border-dashed border-gray-200 flex items-center justify-center cursor-pointer hover:border-[#60a5fa] transition-colors"
            @click="triggerUpload"
          >
            <Icon name="mdi:plus" class="text-2xl text-gray-300" />
          </div>
        </div>

        <input
          ref="fileInput"
          type="file"
          :accept="mediaType === 'video' ? 'video/mp4' : 'image/jpeg,image/png'"
          :multiple="mediaType === 'image'"
          class="hidden"
          @change="handleFileChange"
        />
      </div>

      <!-- 视频封面上传（仅视频模式显示） -->
      <div v-if="mediaType === 'video' && mediaList.length > 0" class="mb-5">
        <div class="text-sm text-gray-500 mb-2">视频封面（可选）</div>
        <div v-if="coverPreview" class="relative inline-block">
          <img :src="coverPreview" class="w-32 h-32 rounded-lg object-cover border border-gray-200" />
          <button
            class="absolute top-1 right-1 w-5 h-5 bg-black/50 rounded-full flex items-center justify-center"
            @click="removeCover"
          >
            <Icon name="mdi:close" class="text-white text-xs" />
          </button>
        </div>
        <div
          v-else
          class="w-32 h-32 rounded-lg border-2 border-dashed border-gray-200 flex flex-col items-center justify-center cursor-pointer hover:border-[#60a5fa] transition-colors"
          @click="coverInput?.click()"
        >
          <Icon name="mdi:image-plus-outline" class="text-2xl text-gray-300 mb-1" />
          <span class="text-xs text-gray-400">上传封面</span>
        </div>
        <el-button v-if="coverFile" class="ml-3" size="small" :loading="coverUploading" @click="retryCoverUpload">重试上传封面</el-button>
        <input
          ref="coverInput"
          type="file"
          accept="image/jpeg,image/png"
          class="hidden"
          @change="handleCoverChange"
        />
      </div>

      <div v-if="mediaList.length > 0">
      <!-- 标题 -->
      <div class="mb-4">
        <DiscoverCreationSuggestions v-if="mediaType === 'image'" :titles="suggestionTitles" :suggested-tags="suggestionTags" :ready="suggestionsReady" :open="suggestionsOpen" :loading="suggestionsLoading" :error="suggestionsError" @toggle="toggleSuggestions" @close="suggestionsOpen = false" @generate="generateSuggestions" @select-title="acceptSuggestedTitle" />
        <el-input
          v-model="title"
          placeholder="填写标题会有更多赞哦~"
          maxlength="100"
          @input="value => title = limitDiscoverText(value, 50)"
          size="large"
        ><template #suffix><span class="text-xs text-[var(--el-text-color-secondary)]">{{ Array.from(title).length }}/50</span></template></el-input>
      </div>

      <!-- 正文 -->
      <div class="mb-5">
        <el-input
          v-model="content"
          type="textarea"
          placeholder="添加正文..."
          :rows="6"
          maxlength="10000"
          @input="value => content = limitDiscoverText(value, 5000)"
          resize="none"
        />
        <p class="mt-1 text-right text-xs text-gray-400">{{ Array.from(content).length }}/5000</p>
      </div>

      <div class="mb-5 flex items-center gap-3 text-sm text-gray-500"><span>谁可以看</span><el-radio-group v-model="visibility" :disabled="publishing"><el-radio-button :value="0">公开</el-radio-button><el-radio-button :value="1">仅自己可见</el-radio-button></el-radio-group></div>
      <div class="mb-5 space-y-2">
        <div class="flex flex-wrap items-center gap-3 text-sm text-gray-500"><span>观看方式</span><el-radio-group v-model="supportOnly" :disabled="publishing || isEditing"><el-radio-button :value="0">免费</el-radio-button><el-radio-button :value="1" :disabled="supportLoading || !supportSettings?.activated">助力专属</el-radio-button></el-radio-group></div>
        <p class="text-xs text-gray-400">发布后不能切换免费或助力属性。<NuxtLink v-if="!isEditing" to="/discover/support" class="text-blue-500">{{ supportSettings?.activated ? '管理助力价格' : '开通创作者助力' }}</NuxtLink></p>
        <p v-if="mediaType === 'image' && supportOnly === 1" class="text-xs text-gray-400">首张图片作为公开封面，正文及其余图片仅助力用户可见。</p>
        <p v-if="supportError" class="text-xs text-gray-500" role="alert">{{ supportError }}<button type="button" class="ml-2 text-blue-500" @click="loadSupportSettings">重试</button></p>
      </div>
      <!-- 标签 -->
      <DiscoverCreationSuggestions v-if="mediaType === 'image'" mode="tags" :suggested-tags="suggestionTags" :selected-tags="tags" :ready="suggestionsReady" @select-tag="acceptSuggestedTag" />
      <div class="flex flex-wrap items-center gap-2">
        <el-tag
          v-for="(tag, index) in tags"
          :key="tag"
          closable
          :disable-transitions="false"
          @close="removeTag(index)"
          size="large"
        >
          <button type="button" @click="editTag(index)">#{{ tag }}</button>
        </el-tag>
        <el-input
          v-if="tagInputVisible"
          ref="tagInputRef"
          v-model="tagInputValue"
          class="!w-[120px]"
          maxlength="40"
          @input="value => tagInputValue = limitDiscoverText(value, 20)"
          size="default"
          @keyup.enter="handleTagConfirm"
          @blur="handleTagConfirm"
        />
        <el-button v-else size="default" @click="showTagInput">
          + 新标签
        </el-button>
      </div>
      </div>
    </div>

    <!-- 底部按钮 -->
    <div v-if="!editorLoading && (isEditing || mediaList.length > 0)" class="sticky bottom-0 bg-white border-t border-gray-100 px-4 py-3">
      <div class="max-w-4xl mx-auto flex flex-wrap items-center justify-center gap-3">
        <el-button v-if="isEditing" round :disabled="hasActiveUpload || savingDraft" @click="returnToManagement">返回</el-button>
        <el-button v-if="!isEditing" round :disabled="hasActiveUpload" :loading="savingDraft" @click="persistDraft">保存草稿</el-button>
        <el-button
          type="primary"
          round
          :disabled="!canPublish || savingDraft"
          :loading="publishing"
          :style="{ backgroundColor: '#60a5fa', borderColor: '#60a5fa' }"
          @click="handlePublish"
        >
          发布
        </el-button>
      </div>
    </div>
    <el-dialog v-model="showDrafts" title="本地草稿" width="min(560px, calc(100vw - 24px))" append-to-body>
      <p class="mb-3 text-xs text-gray-400">草稿保存在当前浏览器，未上传的文件恢复后需要重新选择；未被作品引用的云端文件保留7天。</p>
      <p v-if="!draftList.length" class="py-8 text-center text-gray-400">暂无草稿</p>
      <div v-for="draft in draftList" :key="draft.id" class="flex items-center justify-between gap-3 border-b border-gray-100 py-3">
        <div class="min-w-0"><p class="truncate text-sm text-gray-700">{{ draft.title || '未命名草稿' }}</p><p class="mt-1 text-xs text-gray-400">{{ new Date(draft.updatedAt).toLocaleString() }} · {{ draft.postUuid ? '编辑草稿' : '发布草稿' }}</p></div>
        <div class="flex shrink-0 gap-2"><el-button size="small" @click="openDraft(draft)">继续编辑</el-button><el-button size="small" @click="deleteDraft(draft.id)">删除</el-button></div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
definePageMeta({
  layout: 'creator',
  keepalive: false,
  // 编辑目标或草稿变化时使用新实例，确保每次跳转都执行初始化。
  key: route => [route.path, route.query.edit || '', route.query.draft || ''].join(':'),
  pageTransition: { name: 'creator-page', mode: 'out-in' },
})
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { discoverMediaURL } from '~/utils/discoverAccess'
import { readDiscoverMediaSize } from '~/utils/discoverMedia'
import {
  uploadVideoResumable,
  abortVideoUpload,
  getFileFingerprint,
  loadUploadState,
  UploadPausedError,
} from '~/composables/useResumableUpload'

// 页面级 SEO
useSeoMeta({
  title: '发布',
  description: '发布图片或视频动态，分享您的生活。',
})
import {
  loadDiscoverDraft,
  saveDiscoverDraft,
  clearDiscoverDraft,
  buildDiscoverDraft,
  applyDiscoverDraft,
  emptyDraft,
  listDiscoverDrafts,
  setActiveDiscoverDraft,
} from '~/composables/useDiscoverDraft'
import {
  extractVideoPoster,
  posterUrlToDataUrl,
  revokePosterUrl,
} from '~/composables/useVideoPoster'

const route = useRoute()
const editorAuth = useAuthState()
const editorUserId = editorAuth.getUser()?.uuid || editorAuth.getUser()?.Uuid || ''
let editorDisposed = false
const editingId = ref('')
const isEditing = computed(() => !!route.query.edit || !!editingId.value)
const editorLoading = ref(true)
const editingVersion = ref(0)
const visibility = ref(0)
const supportOnly = ref(0)
const supportSettings = ref(null)
const supportLoading = ref(false)
const supportError = ref('')
const sameEditor = () => !editorDisposed && editorUserId === (editorAuth.getUser()?.uuid || editorAuth.getUser()?.Uuid || '')
const loadSupportSettings = async () => {
  supportLoading.value = true; supportError.value = ''
  try {
    const response = await useApi$('/user/support/creator/info', { body: {}, retry: 0 })
    if (!sameEditor()) return
    if (response.code !== 200 || !response.data) throw new Error(response.message || '获取助力设置失败')
    supportSettings.value = response.data
  } catch (error) { if (sameEditor()) supportError.value = error?.data?.message || error.message || '获取助力设置失败' }
  finally { if (sameEditor()) supportLoading.value = false }
}
const draftId = ref('')
const draftSaveStatus = ref('尚未保存')
const draftList = ref([])
const draftCount = ref(0)
const dragging = ref(false)
const handleDrop = event => { dragging.value = false; if (!hasActiveUpload.value) void handleFileChange({ target: { files: event.dataTransfer.files, value: '' } }) }
const showDrafts = ref(false)
const selectedImageIndex = ref(null)
const editingTagIndex = ref(null)
const requestKey = ref('')
const publishPayload = ref('')
const hasActiveUpload = computed(() => publishing.value || coverUploading.value || mediaList.value.some(media => media.uploadStatus === 'uploading'))
const fileInput = ref(null)
const coverInput = ref(null)
const mediaType = ref('image')
const mediaList = ref([])
const title = ref('')
const content = ref('')
const publishing = ref(false)
const savingDraft = ref(false)
// 类型切换仅保留内存中的媒体，不写入草稿箱。
let mediaByType = { image: null, video: null }

// 封面相关
const coverFile = ref(null)
const coverUploading = ref(false)
const coverPreview = ref('')
const coverUrl = ref('')

// 标签相关
const tags = ref([])
const tagInputValue = ref('')
const tagInputVisible = ref(false)
const tagInputRef = ref()

const {
  titles: suggestionTitles, suggestedTags: suggestionTags, ready: suggestionsReady,
  open: suggestionsOpen, loading: suggestionsLoading, error: suggestionsError,
  generate: generateSuggestions, toggle: toggleSuggestions,
} = useDiscoverCreationSuggestions(() => ({
  mediaType: mediaType.value, media: mediaList.value,
  title: title.value, content: content.value, tags: tags.value,
  editorKey: [editingId.value, draftId.value].join(':'),
  userId: (editorAuth.getUser()?.uuid || editorAuth.getUser()?.Uuid || '') === editorUserId ? editorUserId : '',
  disabled: editorLoading.value || publishing.value || savingDraft.value,
}))
// AI只提供候选，用户明确点击后才改变当前输入，沿用手动草稿保存流程。
const acceptSuggestedTitle = value => {
  if (!suggestionsReady.value) return
  title.value = value
  suggestionsOpen.value = false
}
const acceptSuggestedTag = value => {
  if (!suggestionsReady.value || tags.value.includes(value) || tags.value.length >= 10) return
  tags.value.push(value)
}

// 视频上传 AbortController
const uploadAbortMap = ref(new Map())
// 视频上传暂停控制
const pauseControlMap = ref(new Map())

const canPublish = computed(() => {
  if (!sameEditor()) return false
  if (supportOnly.value === 1 && !isEditing.value && (supportLoading.value || !supportSettings.value?.activated)) return false
  if (mediaList.value.length === 0 || !title.value.trim() || Array.from(title.value.trim()).length > 50 || Array.from(content.value).length > 5000 || coverFile.value) return false
  if (mediaType.value === 'video') {
    return mediaList.value.every(m => m.uploadStatus === 'done' && m.uploadedUrl)
  }
  return mediaList.value.length <= 9 && mediaList.value.every(m => m.uploadStatus === 'done' && m.uploadedUrl)
})

// 保存成功后重置工作台并释放本地预览，已保存的草稿仍保留在草稿箱。
const resetPublishWorkspace = () => {
  const media = [...mediaList.value, ...(mediaByType.image || []), ...(mediaByType.video || [])]
  const localURLs = new Set([...media.flatMap(item => [item.url, item.posterUrl]), coverPreview.value].filter(url => typeof url === 'string' && url.startsWith('blob:')))
  localURLs.forEach(url => URL.revokeObjectURL(url))
  mediaList.value = []; mediaByType = { image: null, video: null }
  title.value = ''; content.value = ''; tags.value = []; visibility.value = 0
  supportOnly.value = 0
  coverFile.value = null; coverUrl.value = ''; coverPreview.value = ''
  draftId.value = ''; draftSaveStatus.value = '尚未保存'
  requestKey.value = ''; publishPayload.value = ''
  selectedImageIndex.value = null; editingTagIndex.value = null
  tagInputValue.value = ''; tagInputVisible.value = false; showDrafts.value = false
  if (fileInput.value) fileInput.value.value = ''
  if (coverInput.value) coverInput.value.value = ''
}

/** 点击保存草稿时持久化当前内容，成功后回到上传工作台。 */
const persistDraft = async () => {
  if (editorLoading.value || savingDraft.value || hasActiveUpload.value || editorDisposed || (editorAuth.getUser()?.uuid || editorAuth.getUser()?.Uuid || '') !== editorUserId) return
  const patch = buildDiscoverDraft({ mediaType: mediaType.value, title: title.value, content: content.value, tags: tags.value, coverUrl: coverUrl.value, mediaList: mediaList.value, visibility: visibility.value, supportOnly: supportOnly.value, requestKey: requestKey.value, publishPayload: publishPayload.value })
  const previous = (draftId.value ? loadDiscoverDraft(draftId.value) : null) || emptyDraft()
  const saved = { ...previous, ...patch, id: draftId.value || previous.id, postUuid: editingId.value, version: editingVersion.value }
  const otherType = mediaType.value === 'image' ? 'video' : 'image'
  const otherMedia = mediaByType[otherType]
  const other = otherMedia ? buildDiscoverDraft({ mediaType: otherType, title: '', content: '', tags: [], coverUrl: coverUrl.value, mediaList: otherMedia }) : previous
  if (mediaType.value === 'image') saved.video = other.video
  else saved.images = other.images
  savingDraft.value = true
  try {
    if (saved.video?.posterUrl?.startsWith('blob:')) saved.video.posterUrl = await posterUrlToDataUrl(saved.video.posterUrl)
    const result = saveDiscoverDraft(saved)
    if (!result) { draftCount.value = listDiscoverDrafts().length; draftSaveStatus.value = '尚未填写内容'; return }
    draftCount.value = listDiscoverDrafts().length
    ElMessage.success('草稿已保存')
    resetPublishWorkspace()
    await navigateTo('/discover/create', { replace: true })
  } catch (error) { draftSaveStatus.value = '保存失败：' + (error.message || '本地存储不可用') }
  finally { savingDraft.value = false }
}

/** 补全视频首帧封面（草稿恢复或 OSS 地址） */
const ensureVideoPoster = async (media, index) => {
  if (media.posterUrl) return
  const source = media.file || media.uploadedUrl
  if (!source) return
  try {
    const posterUrl = await extractVideoPoster(source)
    if (mediaList.value[index] === media) updateMediaUpload(index, { posterUrl })
  } catch (e) {
    console.warn('提取视频封面失败:', e)
  }
}

/** 从草稿恢复当前类型的媒体列表 */
const restoreMediaFromDraft = (type) => {
  const draft = draftId.value ? loadDiscoverDraft(draftId.value) : null
  if (!draft) {
    mediaList.value = []
    return
  }
  const applied = applyDiscoverDraft({ ...draft, mediaType: type })
  mediaList.value = applied?.mediaList || []
  if (type === 'video') {
    coverUrl.value = draft.coverUrl || ''
    coverPreview.value = draft.coverUrl || ''
  }
}

const restoreDraftOnMount = () => {
  const draft = loadDiscoverDraft(draftId.value)
  if (!draft) return

  mediaByType = { image: null, video: null }
  draftId.value = draft.id
  editingId.value = draft.postUuid || ''
  editingVersion.value = draft.version || 0
  visibility.value = draft.visibility || 0
  supportOnly.value = draft.supportOnly === 1 ? 1 : 0
  requestKey.value = draft.requestKey || ''
  publishPayload.value = draft.publishPayload || ''
  mediaType.value = draft.mediaType || 'image'
  title.value = draft.title || ''
  content.value = draft.content || ''
  tags.value = draft.tags || []
  coverUrl.value = draft.coverUrl || ''
  coverPreview.value = draft.coverUrl || ''
  restoreMediaFromDraft(draft.mediaType || 'image')

  // 恢复视频首帧封面
  if (draft.mediaType === 'video' && mediaList.value[0]) {
    ensureVideoPoster(mediaList.value[0], 0)
  }

  const video = draft.video
  if (video?.needsReselect || video?.uploadStatus === 'pending_reselect') {
    ElMessage.info('已恢复草稿，请重新选择同一视频文件以继续上传')
  } else if (draft.mediaType === 'video' && video?.uploadStatus === 'done') {
    ElMessage.success('已恢复未发布的视频草稿')
  } else if (draft.title || draft.content) {
    ElMessage.info('已恢复未发布的草稿')
  }
}

const switchType = async (type) => {
  if (hasActiveUpload.value) { ElMessage.info('请先暂停或完成上传'); return }
  if (mediaType.value === type) return
  mediaByType[mediaType.value] = mediaList.value
  mediaType.value = type
  if (mediaByType[type]) mediaList.value = mediaByType[type]
  else restoreMediaFromDraft(type)
}

const triggerUpload = () => {
  selectedImageIndex.value = null
  fileInput.value?.click()
}

const updateMediaUpload = (index, patch) => {
  if (editorDisposed) return
  const item = mediaList.value[index]
  if (!item) return
  Object.assign(item, patch)
}

const startVideoUpload = async (media, index) => {
  const update = patch => { if (mediaList.value[index] === media) updateMediaUpload(index, patch) }
  if (!media.file) {
    update({
      uploadStatus: 'pending_reselect',
      uploadStatusText: '请选择视频文件',
    })
    return
  }

  const controller = new AbortController()
  uploadAbortMap.value.set(index, controller)

  // 暂停控制对象
  const pauseControl = { paused: false }
  pauseControlMap.value.set(index, pauseControl)

  update({
    uploadStatus: 'uploading',
    uploadProgress: media.uploadProgress || 0,
    uploadStatusText: '准备上传...',
    fingerprint: getFileFingerprint(media.file),
  })

  try {
    const result = await uploadVideoResumable(media.file, 'discover_video', {
      onProgress: (percent) => {
        update({ uploadProgress: percent })
      },
      onStatus: (text) => {
        update({ uploadStatusText: text })
      },
      signal: controller.signal,
      pauseSignal: pauseControl,
    })
    update({
      uploadedUrl: result.url,
      objectKey: result.objectKey,
      uploadStatus: 'done',
      uploadProgress: 100,
      uploadStatusText: '上传完成',
    })
    ElMessage.success('视频上传成功')
  } catch (e) {
    if (e instanceof UploadPausedError) {
      // 暂停状态，保留当前进度
      update({
        uploadStatus: 'paused',
        uploadStatusText: '上传已暂停',
      })
    } else if (e.name === 'AbortError') {
      update({
        uploadStatus: 'idle',
        uploadProgress: 0,
        uploadStatusText: '',
      })
    } else {
      update({
        uploadStatus: 'error',
        uploadStatusText: '上传失败，点击重试',
      })
      console.error('视频上传失败:', e)
      ElMessage.error(e.message || '视频上传失败，请重试')
    }
  } finally {
    uploadAbortMap.value.delete(index)
    pauseControlMap.value.delete(index)
  }
}

const retryVideoUpload = (media, index) => {
  if (media.uploadStatus === 'error' && media.file) {
    updateMediaUpload(index, { uploadStatus: 'idle', uploadProgress: 0 })
    startVideoUpload(media, index)
  } else {
    triggerUpload()
  }
}

const cancelVideoUpload = (media) => {
  const index = mediaList.value.indexOf(media)
  const controller = uploadAbortMap.value.get(index)
  controller?.abort()
  if (media.file) {
    abortVideoUpload(getFileFingerprint(media.file))
  }
}

/**
 * 暂停视频上传
 */
const pauseVideoUpload = (index) => {
  const control = pauseControlMap.value.get(index)
  if (control) {
    control.paused = true
  }
}

/**
 * 继续视频上传
 */
const resumeVideoUpload = (index) => {
  const media = mediaList.value[index]
  if (!media?.file) return

  // 重置暂停状态
  const control = pauseControlMap.value.get(index)
  if (control) {
    control.paused = false
  }

  // 继续上传
  startVideoUpload(media, index)
}

const handleFileChange = async (e) => {
  const files = Array.from(e.target.files)
  for (const file of files) {
    if (!validateSelectedFile(file, mediaType.value)) continue
    if (mediaType.value === 'image' && selectedImageIndex.value == null && mediaList.value.length >= 9) { ElMessage.warning('图片最多9张'); break }
    const url = URL.createObjectURL(file)
    const type = file.type.startsWith('video') ? 'video' : 'image'
    const size = await readDiscoverMediaSize(url, type)
    if (mediaType.value === 'video') {
      const fingerprint = getFileFingerprint(file)
      const existing = mediaList.value[0]
      if (existing?.uploadStatus === 'pending_reselect' && existing.fingerprint && existing.fingerprint !== fingerprint) { URL.revokeObjectURL(url); ElMessage.warning('请选择与草稿相同的视频文件'); continue }
      const uploadState = loadUploadState(fingerprint)

      let initialProgress = 0
      if (uploadState?.parts?.length && uploadState.fileSize === file.size) {
        const partSize = uploadState.partSize || 5 * 1024 * 1024
        const uploadedBytes = uploadState.parts.reduce((s, p) => s + (p.size || partSize), 0)
        initialProgress = Math.min(99, Math.round((uploadedBytes / file.size) * 100))
      }

      if (existing?.url?.startsWith('blob:')) {
        URL.revokeObjectURL(existing.url)
      }
      if (existing?.posterUrl) {
        revokePosterUrl(existing.posterUrl)
      }

      // 截取视频首帧作为封面
      let posterUrl = ''
      try {
        posterUrl = await extractVideoPoster(file)
      } catch (err) {
        console.warn('提取视频封面失败:', err)
      }

      mediaList.value = [{
        url,
        ...size,
        type: 'video',
        file,
        fileName: file.name,
        fileSize: file.size,
        lastModified: file.lastModified,
        fingerprint,
        posterUrl,
        uploadStatus: 'idle',
        uploadProgress: initialProgress,
        uploadStatusText: initialProgress > 0 ? '检测到未完成的上传，即将续传...' : '',
        uploadedUrl: '',
      }]
      nextTick(() => startVideoUpload(mediaList.value[0], 0))
    } else {
      const media = { url, ...size, type: 'image', file, fileName: file.name, uploadStatus: 'idle', uploadProgress: 0, uploadedUrl: '' }
      if (selectedImageIndex.value != null) { const old = mediaList.value[selectedImageIndex.value]; if (old?.url?.startsWith('blob:')) URL.revokeObjectURL(old.url); mediaList.value.splice(selectedImageIndex.value, 1, media); selectedImageIndex.value = null }
      else mediaList.value.push(media)
      await retryImageUpload(mediaList.value.find(item => item.file === file))
    }
  }
  e.target.value = ''
}

const removeMedia = async (index) => {
  const media = mediaList.value[index]
  if (!media) return

  if (media.type === 'video') {
    if (media.uploadStatus === 'uploading') {
      cancelVideoUpload(media)
    } else if (media.uploadStatus === 'paused') {
      // 暂停状态取消，需要清理 OSS 分片
      if (media.file) {
        await abortVideoUpload(getFileFingerprint(media.file))
      } else if (media.fingerprint) {
        await abortVideoUpload(media.fingerprint)
      }
    } else if (media.uploadStatus === 'done') {
      // 本地移除不直接删除云资源，保留其他草稿引用，过期资源由后端检查后清理。
    } else if (media.file) {
      await abortVideoUpload(getFileFingerprint(media.file))
    } else if (media.fingerprint) {
      await abortVideoUpload(media.fingerprint)
    }
  }

  if (media.url?.startsWith('blob:')) {
    URL.revokeObjectURL(media.url)
  }
  if (media.posterUrl) {
    revokePosterUrl(media.posterUrl)
  }
  mediaList.value.splice(index, 1)
}

// 封面上传失败时保留本地文件，可以独立重试。
const retryCoverUpload = async () => {
  if (!coverFile.value || coverUploading.value) return
  coverUploading.value = true
  const file = coverFile.value
  try {
    const url = await uploadImage(file, 'discover_image')
    if (coverFile.value !== file) return
    if (coverPreview.value?.startsWith('blob:')) URL.revokeObjectURL(coverPreview.value)
    coverUrl.value = url; coverPreview.value = url; coverFile.value = null;
  } catch (error) { ElMessage.error(error.message || '封面上传失败，请重试') }
  finally { coverUploading.value = false }
}
const handleCoverChange = async event => {
  const file = event.target.files[0]
  if (!file || !validateSelectedFile(file, 'image')) return
  if (coverPreview.value?.startsWith('blob:')) URL.revokeObjectURL(coverPreview.value)
  coverFile.value = file; coverUrl.value = ''; coverPreview.value = URL.createObjectURL(file); event.target.value = ''
  await retryCoverUpload()
}

const removeCover = async () => {
  if (coverPreview.value?.startsWith('blob:')) {
    URL.revokeObjectURL(coverPreview.value)
  }
  coverFile.value = null
  coverPreview.value = ''
  coverUrl.value = ''
}

// 标签操作
const showTagInput = () => {
  tagInputVisible.value = true
  nextTick(() => {
    tagInputRef.value?.input?.focus()
  })
}

const handleTagConfirm = () => {
  const val = tagInputValue.value.trim().replace(/^#/, '')
  if (val && Array.from(val).length <= 20 && !tags.value.some((tag, index) => tag === val && index !== editingTagIndex.value)) {
    if (editingTagIndex.value != null) tags.value.splice(editingTagIndex.value, 1, val)
    else if (tags.value.length < 10) tags.value.push(val)
    else ElMessage.warning('标签最多10个')
  } else if (val) ElMessage.warning('标签不能重复且最多20字')
  editingTagIndex.value = null
  tagInputVisible.value = false
  tagInputValue.value = ''
}

const removeTag = (index) => {
  tags.value.splice(index, 1)
}

const limitDiscoverText = (value, max) => Array.from(value || '').slice(0, max).join('')

const validateSelectedFile = (file, type) => {
  const valid = type === 'video' ? file.type === 'video/mp4' && /\.mp4$/i.test(file.name) && file.size <= 500 * 1024 * 1024 : ((file.type === 'image/jpeg' && /\.jpe?g$/i.test(file.name)) || (file.type === 'image/png' && /\.png$/i.test(file.name))) && file.size <= 10 * 1024 * 1024
  if (!valid || file.size <= 0) { ElMessage.warning(type === 'video' ? '请选择不超过500MB的MP4视频' : '请选择不超过10MB的JPG或PNG图片'); return false }
  return true
}

const uploadImage = (file, fileType, onProgress) => new Promise((resolve, reject) => {
  const config = useRuntimeConfig()
  const xhr = new XMLHttpRequest()
  xhr.open('POST', (config.public.apiBase || 'http://localhost:8080/').replace(/\/$/, '') + '/user/file/upload')
  xhr.withCredentials = true
  xhr.upload.onprogress = event => { if (event.lengthComputable) onProgress?.(Math.round(event.loaded / event.total * 100)) }
  xhr.onerror = () => reject(new Error('图片上传失败，请重试'))
  xhr.onload = () => {
    try { const response = JSON.parse(xhr.responseText); if (xhr.status < 200 || xhr.status >= 300 || response.code !== 200 || !response.data?.url) throw new Error(response.message || '图片上传失败'); resolve(discoverMediaURL(response.data.url, config.public.apiBase)) } catch (error) { reject(error) }
  }
  const form = new FormData(); form.append('file', file); form.append('fileType', fileType); xhr.send(form)
})

const retryImageUpload = async media => {
  if (!media.file || media.uploadStatus === 'uploading') return
  media.uploadStatus = 'uploading'; media.uploadProgress = 0
  try { media.uploadedUrl = await uploadImage(media.file, 'discover_image', progress => { media.uploadProgress = progress }); media.uploadStatus = 'done'; media.uploadProgress = 100 }
  catch (error) { media.uploadStatus = 'error'; ElMessage.error(error.message || '图片上传失败') }
}
const reselectImage = index => { selectedImageIndex.value = index; fileInput.value?.click() }
const moveImage = (index, direction) => { const target = index + direction; if (target < 0 || target >= mediaList.value.length) return; const [media] = mediaList.value.splice(index, 1); mediaList.value.splice(target, 0, media); }
const setImageCover = index => { const [media] = mediaList.value.splice(index, 1); mediaList.value.unshift(media); }
const editTag = index => { editingTagIndex.value = index; tagInputValue.value = tags.value[index]; showTagInput() }
const showDraftList = () => { draftList.value = listDiscoverDrafts(); draftCount.value = draftList.value.length; showDrafts.value = true }
const openDraft = async draft => {
  if (hasActiveUpload.value || savingDraft.value) { ElMessage.info('请先完成或暂停上传及保存'); return }
  setActiveDiscoverDraft(draft.id); draftId.value = draft.id; restoreDraftOnMount(); await refreshDraftResources(); showDrafts.value = false
  await nextTick(); await navigateTo('/discover/create?draft=' + encodeURIComponent(draftId.value), { replace: true })
}
const deleteDraft = async id => { clearDiscoverDraft(id); draftList.value = listDiscoverDrafts(); draftCount.value = draftList.value.length; if (draftId.value === id) { draftId.value = ''; draftSaveStatus.value = '草稿已删除，当前输入仍保留'; } }

// 返回作品管理，草稿仅通过保存按钮写入。
const returnToManagement = async () => {
  if (hasActiveUpload.value) return
  await navigateTo('/discover/manage')
}

const handlePublish = async () => {
  if (publishing.value || savingDraft.value || !canPublish.value) return
  publishing.value = true
  try {
    const urls = mediaList.value.map(media => media.uploadedUrl)
    const body = { title: title.value.trim(), content: content.value.trim(), media_type: mediaType.value === 'video' ? 1 : 0, tags: [...tags.value], urls, media_sizes: mediaList.value.map(media => ({ width: media.width || 0, height: media.height || 0 })), cover_url: mediaType.value === 'image' ? urls[0] : coverUrl.value, visibility: visibility.value, support_only: supportOnly.value }
    const payload = JSON.stringify(body)
    if (!requestKey.value || publishPayload.value !== payload) { requestKey.value = crypto.randomUUID(); publishPayload.value = payload }
    const response = await useApi$(editingId.value ? '/user/discover/update' : '/user/discover/create', { body: { ...body, request_key: requestKey.value, ...(editingId.value ? { post_uuid: editingId.value } : {}), version: editingVersion.value }, retry: 0 })
    if (response.code !== 200) throw new Error(response.message || '保存失败')
    if (draftId.value) clearDiscoverDraft(draftId.value)
    ElMessage.success(editingId.value ? '作品已更新' : '发布成功')
    await navigateTo('/discover/manage')
  } catch (error) { ElMessage.error(error?.data?.message || error.message || '保存失败，输入已保留') }
  finally { publishing.value = false }
}

const refreshDraftResources = async () => {
 const urls = [...new Set([...mediaList.value.map(media => media.uploadedUrl).filter(Boolean), ...(coverUrl.value ? [coverUrl.value] : [])])]
 if (!urls.length) return
 try {
  const response = await useApi$('/user/file/assets/status', { body: { urls }, retry: 0 })
  if (editorDisposed) return
  if (response.code !== 200) return
  let missing = false
  for (const media of mediaList.value) if (media.uploadedUrl && response.data?.[media.uploadedUrl] === false) { media.uploadedUrl = ''; media.url = ''; media.posterUrl = ''; media.uploadStatus = 'pending_reselect'; media.uploadStatusText = '云端文件已过期，请重新选择'; media.uploadProgress = 0; missing = true }
  if (coverUrl.value && response.data?.[coverUrl.value] === false) { coverUrl.value = ''; coverPreview.value = ''; missing = true }
  if (missing) ElMessage.warning('草稿中的部分云端文件已过期，请重新选择文件')
 } catch (error) { console.warn('检查草稿资源失败:', error) }
}

const loadEditingPost = async id => {
  const response = await useApi$('/user/discover/detail', { body: { uuid: id }, retry: 0 })
  if (editorDisposed) return
  if (response.code !== 200 || !response.data) throw new Error(response.message || '作品无法编辑')
  const post = response.data
  if (post.user_id !== (editorAuth.getUser()?.uuid || editorAuth.getUser()?.Uuid)) throw new Error('只能编辑自己的作品')
  editingId.value = id; editingVersion.value = post.version; visibility.value = post.visibility || 0
  supportOnly.value = post.support_only === 1 ? 1 : 0
  draftId.value = emptyDraft().id; title.value = post.title; content.value = post.content || ''; tags.value = post.tags || []; mediaType.value = post.media_type === 1 ? 'video' : 'image'; coverUrl.value = post.cover_url || ''; coverPreview.value = coverUrl.value
  // 先展示服务端返回的作品，尺寸与视频首帧后台补全，避免等待媒体加载导致空白。
  mediaList.value = (post.urls || []).map(url => ({ url, uploadedUrl: url, file: null, type: mediaType.value, uploadStatus: 'done', uploadProgress: 100, width: 0, height: 0 }))
  mediaList.value.forEach((media, index) => {
    void readDiscoverMediaSize(media.url, media.type).then(size => {
      if (!editorDisposed && mediaList.value[index] === media) Object.assign(media, size)
    })
  })
  if (mediaType.value === 'video' && mediaList.value[0]) void ensureVideoPoster(mediaList.value[0], 0)
}

// 刷新/关闭页面时提示未保存的视频上传
const beforeUnloadHandler = (e) => {
  const hasPausedVideo = mediaList.value.some(m => m.uploadStatus === 'paused')
  const hasUploadingVideo = mediaList.value.some(m => m.uploadStatus === 'uploading')
  if (hasPausedVideo || hasUploadingVideo) {
    e.preventDefault()
    e.returnValue = '视频上传未完成，确定要离开吗？'
  }
}

onMounted(async () => {
  void loadSupportSettings()
  draftCount.value = listDiscoverDrafts().length
  try {
    if (route.query.draft) { draftId.value = String(route.query.draft); restoreDraftOnMount() }
    else if (route.query.edit) await loadEditingPost(String(route.query.edit))
  }
  catch (error) {
    if (!editorDisposed) {
      ElMessage.error(error?.data?.message || error.message || '加载作品失败')
      await navigateTo('/discover/manage')
    }
    return
  }
  if (editorDisposed) return
  editorLoading.value = false
  await refreshDraftResources()
  if (editorDisposed) return
  window.addEventListener('beforeunload', beforeUnloadHandler)
})

onUnmounted(() => {
  editorDisposed = true
  for (const controller of uploadAbortMap.value.values()) controller.abort()
  window.removeEventListener('beforeunload', beforeUnloadHandler)
})

</script>

<style scoped>
:deep(.el-button.is-round) {
    padding: 18px 40px;
}
</style>
