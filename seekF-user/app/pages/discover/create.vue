<template>
  <div class="min-h-screen bg-gray-50 flex flex-col">
    <!-- 内容区 -->
    <div class="flex-1 max-w-4xl w-full mx-auto mt-4 mb-4 bg-white rounded-xl shadow-sm px-5 py-5 overflow-y-auto">
      <!-- 类型选择 -->
      <div class="flex gap-3 mb-5">
        <div
          class="flex-1 py-3 rounded-xl border-2 text-center cursor-pointer transition-colors"
          :class="mediaType === 'image' ? 'border-[#60a5fa] bg-blue-50 text-[#60a5fa]' : 'border-gray-200 text-gray-400'"
          @click="switchType('image')"
        >
          <Icon name="mdi:image-outline" class="text-2xl mb-1" />
          <div class="text-sm font-medium">图片</div>
        </div>
        <div
          class="flex-1 py-3 rounded-xl border-2 text-center cursor-pointer transition-colors"
          :class="mediaType === 'video' ? 'border-[#60a5fa] bg-blue-50 text-[#60a5fa]' : 'border-gray-200 text-gray-400'"
          @click="switchType('video')"
        >
          <Icon name="mdi:video-outline" class="text-2xl mb-1" />
          <div class="text-sm font-medium">视频</div>
        </div>
      </div>

      <!-- 上传区域 -->
      <div class="mb-5">
        <div
          v-if="mediaList.length === 0"
          class="border-2 border-dashed border-gray-200 rounded-xl flex flex-col items-center justify-center py-14 cursor-pointer hover:border-[#60a5fa] transition-colors"
          @click="triggerUpload"
        >
          <Icon :name="mediaType === 'video' ? 'mdi:video-plus-outline' : 'mdi:image-plus-outline'" class="text-4xl text-gray-300 mb-2" />
          <span class="text-sm text-gray-400">{{ mediaType === 'video' ? '上传视频' : '上传图片' }}</span>
          <span class="text-xs text-gray-300 mt-1">{{ mediaType === 'video' ? '支持 mp4，支持断点续传' : '支持 jpg、png' }}</span>
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
            <img :src="media.url" class="w-full h-full object-cover" />
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
          :accept="mediaType === 'video' ? 'video/mp4' : 'image/*'"
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
        <input
          ref="coverInput"
          type="file"
          accept="image/*"
          class="hidden"
          @change="handleCoverChange"
        />
      </div>

      <!-- 标题 -->
      <div class="mb-4">
        <el-input
          v-model="title"
          placeholder="填写标题会有更多赞哦~"
          maxlength="50"
          show-word-limit
          size="large"
        />
      </div>

      <!-- 正文 -->
      <div class="mb-5">
        <el-input
          v-model="content"
          type="textarea"
          placeholder="添加正文..."
          :rows="6"
          resize="none"
        />
      </div>

      <!-- 标签 -->
      <div class="flex flex-wrap items-center gap-2">
        <el-tag
          v-for="(tag, index) in tags"
          :key="tag"
          closable
          :disable-transitions="false"
          @close="removeTag(index)"
          size="large"
        >
          #{{ tag }}
        </el-tag>
        <el-input
          v-if="tagInputVisible"
          ref="tagInputRef"
          v-model="tagInputValue"
          style="width: 120px"
          size="default"
          @keyup.enter="handleTagConfirm"
          @blur="handleTagConfirm"
        />
        <el-button v-else size="default" @click="showTagInput">
          + 新标签
        </el-button>
      </div>
    </div>

    <!-- 底部按钮 -->
    <div class="sticky bottom-0 bg-white border-t border-gray-100 px-4 py-3">
      <div class="max-w-4xl mx-auto flex items-center justify-between">
        <el-button round @click="navigateTo('/discover')" class="">返回</el-button>
        <el-button
          type="primary"
          round
          :disabled="!canPublish"
          :loading="publishing"
          :style="{ backgroundColor: '#60a5fa', borderColor: '#60a5fa' }"
          @click="handlePublish"
        >
          发布
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import {
  uploadVideoResumable,
  abortVideoUpload,
  deleteOssFile,
  getFileFingerprint,
  loadUploadState,
  UploadPausedError,
  resumeUpload,
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
} from '~/composables/useDiscoverDraft'
import {
  extractVideoPoster,
  posterUrlToDataUrl,
  revokePosterUrl,
} from '~/composables/useVideoPoster'

const fileInput = ref(null)
const coverInput = ref(null)
const mediaType = ref('image')
const mediaList = ref([])
const title = ref('')
const content = ref('')
const publishing = ref(false)
const skipPersist = ref(true)

// 封面相关
const coverFile = ref(null)
const coverPreview = ref('')
const coverUrl = ref('')

// 标签相关
const tags = ref([])
const tagInputValue = ref('')
const tagInputVisible = ref(false)
const tagInputRef = ref()

// 视频上传 AbortController
const uploadAbortMap = ref(new Map())
// 视频上传暂停控制
const pauseControlMap = ref(new Map())

const canPublish = computed(() => {
  if (mediaList.value.length === 0 || !title.value.trim()) return false
  if (mediaType.value === 'video') {
    return mediaList.value.every(m => m.uploadStatus === 'done' && m.uploadedUrl)
  }
  return true
})

/** 持久化草稿（保留图片/视频各自的数据） */
const persistDraft = async () => {
  const existing = loadDiscoverDraft() || emptyDraft()
  const patch = buildDiscoverDraft({
    mediaType: mediaType.value,
    title: title.value,
    content: content.value,
    tags: tags.value,
    coverUrl: coverUrl.value,
    mediaList: mediaList.value,
  })
  existing.mediaType = mediaType.value
  existing.title = title.value
  existing.content = content.value
  existing.tags = [...tags.value]
  existing.coverUrl = coverUrl.value
  if (mediaType.value === 'video') {
    existing.video = patch.video
    if (existing.video?.posterUrl?.startsWith('blob:')) {
      try {
        existing.video.posterUrl = await posterUrlToDataUrl(existing.video.posterUrl)
      } catch {
        // 转换失败则保留 blob，至少同会话可用
      }
    }
  } else {
    existing.images = patch.images
  }
  saveDiscoverDraft(existing)
}

/** 补全视频首帧封面（草稿恢复或 OSS 地址） */
const ensureVideoPoster = async (media, index) => {
  if (media.posterUrl) return
  const source = media.file || media.uploadedUrl
  if (!source) return
  try {
    const posterUrl = await extractVideoPoster(source)
    updateMediaUpload(index, { posterUrl })
  } catch (e) {
    console.warn('提取视频封面失败:', e)
  }
}

/** 从草稿恢复当前类型的媒体列表 */
const restoreMediaFromDraft = (type) => {
  const draft = loadDiscoverDraft()
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
  const draft = loadDiscoverDraft()
  if (!draft) return

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

const switchType = (type) => {
  if (mediaType.value === type) return
  void persistDraft()
  mediaType.value = type
  restoreMediaFromDraft(type)
}

const triggerUpload = () => {
  fileInput.value?.click()
}

const updateMediaUpload = (index, patch) => {
  const item = mediaList.value[index]
  if (!item) return
  Object.assign(item, patch)
  void persistDraft()
}

const startVideoUpload = async (media, index) => {
  if (!media.file) {
    updateMediaUpload(index, {
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

  updateMediaUpload(index, {
    uploadStatus: 'uploading',
    uploadProgress: media.uploadProgress || 0,
    uploadStatusText: '准备上传...',
    fingerprint: getFileFingerprint(media.file),
  })

  try {
    const result = await uploadVideoResumable(media.file, 'discover_video', {
      onProgress: (percent) => {
        updateMediaUpload(index, { uploadProgress: percent })
      },
      onStatus: (text) => {
        updateMediaUpload(index, { uploadStatusText: text })
      },
      signal: controller.signal,
      pauseSignal: pauseControl,
    })
    updateMediaUpload(index, {
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
      updateMediaUpload(index, {
        uploadStatus: 'paused',
        uploadStatusText: '上传已暂停',
      })
    } else if (e.name === 'AbortError') {
      updateMediaUpload(index, {
        uploadStatus: 'idle',
        uploadProgress: 0,
        uploadStatusText: '',
      })
    } else {
      updateMediaUpload(index, {
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
    const url = URL.createObjectURL(file)
    const type = file.type.startsWith('video') ? 'video' : 'image'
    if (mediaType.value === 'video') {
      const fingerprint = getFileFingerprint(file)
      const existing = mediaList.value[0]
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
      mediaList.value.push({ url, type: 'image', file })
      persistDraft()
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
      try {
        await deleteOssFile({
          objectKey: media.objectKey,
          url: media.uploadedUrl,
        })
      } catch (e) {
        console.error('删除视频失败:', e)
        ElMessage.warning('云端视频删除失败，已从列表移除')
      }
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
  persistDraft()
}

// 封面：选择后立即上传 OSS，刷新后可恢复
const handleCoverChange = async (e) => {
  const file = e.target.files[0]
  if (!file) return
  if (coverPreview.value?.startsWith('blob:')) {
    URL.revokeObjectURL(coverPreview.value)
  }
  coverFile.value = file
  coverPreview.value = URL.createObjectURL(file)
  e.target.value = ''

  try {
    const url = await uploadImage(file, 'discover_image')
    coverUrl.value = url
    coverPreview.value = url
    coverFile.value = null
    persistDraft()
  } catch (err) {
    ElMessage.error('封面上传失败')
    console.error(err)
  }
}

const removeCover = async () => {
  if (coverUrl.value) {
    try {
      await deleteOssFile({ url: coverUrl.value })
    } catch (e) {
      console.error('删除封面失败:', e)
    }
  }
  if (coverPreview.value?.startsWith('blob:')) {
    URL.revokeObjectURL(coverPreview.value)
  }
  coverFile.value = null
  coverPreview.value = ''
  coverUrl.value = ''
  persistDraft()
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
  if (val && !tags.value.includes(val) && tags.value.length < 10) {
    tags.value.push(val)
  }
  tagInputVisible.value = false
  tagInputValue.value = ''
}

const removeTag = (index) => {
  tags.value.splice(index, 1)
}

const uploadImage = async (file, fileType) => {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('fileType', fileType)
  const res = await useApi$('/user/file/upload', {
    body: formData,
    method: 'POST',
  })
  if (res.code === 200 && res.data?.url) {
    return res.data.url
  }
  throw new Error(res.message || '文件上传失败')
}

const handlePublish = async () => {
  publishing.value = true
  try {
    const urls = []
    for (const media of mediaList.value) {
      if (media.type === 'video') {
        if (media.uploadStatus === 'uploading') {
          ElMessage.warning('视频正在上传中，请稍候')
          return
        }
        if (media.uploadStatus === 'paused') {
          ElMessage.warning('视频上传已暂停，请继续上传或取消后再发布')
          return
        }
        if (media.uploadStatus === 'error' || media.uploadStatus === 'pending_reselect') {
          ElMessage.warning('请先完成视频上传后再发布')
          return
        }
        urls.push(media.uploadedUrl)
      } else {
        if (media.uploadedUrl) {
          urls.push(media.uploadedUrl)
        } else if (media.file) {
          urls.push(await uploadImage(media.file, 'discover_image'))
        }
      }
    }

    const postRes = await useApi$('/user/discover/create', {
      body: {
        title: title.value,
        content: content.value,
        media_type: mediaType.value === 'video' ? 1 : 0,
        tags: tags.value,
        urls,
        cover_url: coverUrl.value,
      },
    })
    if (postRes.code === 200) {
      clearDiscoverDraft()
      ElMessage.success('发布成功')
      navigateTo('/discover')
    } else {
      ElMessage.error(postRes.message || '发布失败')
    }
  } catch (e) {
    console.error('发布失败:', e)
    ElMessage.error(e.message || '发布失败')
  } finally {
    publishing.value = false
  }
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

onMounted(() => {
  restoreDraftOnMount()
  skipPersist.value = false
  window.addEventListener('beforeunload', beforeUnloadHandler)
})

onUnmounted(() => {
  window.removeEventListener('beforeunload', beforeUnloadHandler)
})

watch([title, content, tags, mediaType], () => {
  if (!skipPersist.value) {
    void persistDraft()
  }
}, { deep: true })
</script>

<style scoped>
:deep(.el-button.is-round) {
    padding: 18px 40px;
}
</style>
