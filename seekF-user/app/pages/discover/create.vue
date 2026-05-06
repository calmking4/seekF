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
          <span class="text-xs text-gray-300 mt-1">{{ mediaType === 'video' ? '支持 mp4' : '支持 jpg、png' }}</span>
        </div>

        <!-- 已上传的媒体列表 -->
        <div v-else class="flex gap-2 overflow-x-auto pb-2">
          <div
            v-for="(media, index) in mediaList"
            :key="index"
            class="relative flex-shrink-0 w-24 h-24 rounded-lg overflow-hidden bg-gray-100"
          >
            <img v-if="media.type === 'image'" :src="media.url" class="w-full h-full object-cover" />
            <video v-else :src="media.url" class="w-full h-full object-cover" muted />
            <button
              class="absolute top-1 right-1 w-5 h-5 bg-black/50 rounded-full flex items-center justify-center"
              @click="removeMedia(index)"
            >
              <Icon name="mdi:close" class="text-white text-xs" />
            </button>
            <div v-if="media.type === 'video'" class="absolute bottom-1 left-1">
              <Icon name="mdi:play-circle" class="text-white text-lg drop-shadow" />
            </div>
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
import { ref, computed, nextTick } from 'vue'

const fileInput = ref(null)
const mediaType = ref('image')
const mediaList = ref([])
const title = ref('')
const content = ref('')

// 标签相关
const tags = ref([])
const tagInputValue = ref('')
const tagInputVisible = ref(false)
const tagInputRef = ref()

const canPublish = computed(() => {
  return mediaList.value.length > 0 && title.value.trim().length > 0
})

const switchType = (type) => {
  if (mediaType.value !== type) {
    mediaType.value = type
    mediaList.value = []
  }
}

const triggerUpload = () => {
  fileInput.value?.click()
}

const handleFileChange = (e) => {
  const files = Array.from(e.target.files)
  files.forEach(file => {
    const url = URL.createObjectURL(file)
    const type = file.type.startsWith('video') ? 'video' : 'image'
    if (mediaType.value === 'video') {
      mediaList.value = [{ url, type: 'video', file }]
    } else {
      mediaList.value.push({ url, type: 'image', file })
    }
  })
  e.target.value = ''
}

const removeMedia = (index) => {
  URL.revokeObjectURL(mediaList.value[index].url)
  mediaList.value.splice(index, 1)
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

const handlePublish = () => {
  // TODO: 调用发布接口
  console.log('发布:', {
    type: mediaType.value,
    title: title.value,
    content: content.value,
    tags: tags.value,
    media: mediaList.value.map(m => ({ type: m.type, file: m.file }))
  })
}
</script>

<style scoped>
:deep(.el-button.is-round) { 
    padding: 18px 40px;
}
</style>