<template>
  <Transition name="slide-up">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 bg-gray-900 flex flex-col"
    >
      <!-- 视频区域 -->
      <div class="flex-1 flex gap-4 p-4">
        <!-- 本地视频 -->
        <div class="flex-1 bg-gray-800 rounded-xl overflow-hidden relative">
          <video
            ref="localVideoRef"
            autoplay
            playsinline
            muted
            class="w-full h-full object-cover"
          ></video>
          <div class="absolute bottom-4 left-4 bg-black/50 px-3 py-1 rounded-full text-white text-sm">
            我
          </div>
          <div
            v-if="isCameraOff"
            class="absolute inset-0 bg-gray-700 flex items-center justify-center"
          >
            <Icon name="fluent:video-off-24-filled" class="text-6xl text-gray-400" />
          </div>
        </div>

        <!-- 远程视频 -->
        <div class="flex-1 bg-gray-800 rounded-xl overflow-hidden relative">
          <video
            ref="remoteVideoRef"
            autoplay
            playsinline
            class="w-full h-full object-cover"
          ></video>
          <div class="absolute bottom-4 left-4 bg-black/50 px-3 py-1 rounded-full text-white text-sm">
            {{ remoteName }}
          </div>
          <div
            v-if="!remoteStream"
            class="absolute inset-0 bg-gray-700 flex flex-col items-center justify-center"
          >
            <el-avatar :size="80" :src="remoteAvatar" class="mb-4">
              {{ remoteName ? remoteName.charAt(0) : '?' }}
            </el-avatar>
            <p class="text-white text-lg">{{ remoteName }}</p>
            <p class="text-gray-400 text-sm mt-2">等待对方接入...</p>
          </div>
        </div>
      </div>

      <!-- 底部控制栏 -->
      <div class="bg-gray-800/80 backdrop-blur-sm py-6">
        <!-- 通话计时 -->
        <div class="text-center text-white text-lg mb-4">
          {{ formatDuration }}
        </div>

        <!-- 控制按钮 -->
        <div class="flex justify-center items-center gap-6">
          <!-- 静音按钮 -->
          <button
            class="w-14 h-14 rounded-full flex items-center justify-center transition-colors"
            :class="isMuted ? 'bg-red-500 text-white' : 'bg-gray-600 text-white hover:bg-gray-500'"
            @click="onToggleMute"
          >
            <Icon
              :name="isMuted ? 'fluent:mic-off-24-filled' : 'fluent:mic-24-filled'"
              class="text-2xl"
            />
          </button>

          <!-- 挂断按钮 -->
          <button
            class="w-16 h-16 rounded-full bg-red-500 hover:bg-red-600 text-white flex items-center justify-center transition-colors"
            @click="onEndCall"
          >
            <Icon name="fluent:call-end-24-filled" class="text-2xl" />
          </button>

          <!-- 摄像头按钮 -->
          <button
            class="w-14 h-14 rounded-full flex items-center justify-center transition-colors"
            :class="isCameraOff ? 'bg-red-500 text-white' : 'bg-gray-600 text-white hover:bg-gray-500'"
            @click="onToggleCamera"
          >
            <Icon
              :name="isCameraOff ? 'fluent:video-off-24-filled' : 'fluent:video-24-filled'"
              class="text-2xl"
            />
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  localStream: {
    type: Object,
    default: null
  },
  remoteStream: {
    type: Object,
    default: null
  },
  remoteName: {
    type: String,
    default: ''
  },
  remoteAvatar: {
    type: String,
    default: ''
  },
  formatDuration: {
    type: String,
    default: '00:00'
  },
  isMuted: {
    type: Boolean,
    default: false
  },
  isCameraOff: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['end-call', 'toggle-mute', 'toggle-camera'])

const localVideoRef = ref(null)
const remoteVideoRef = ref(null)

// 监听本地流变化
watch(() => props.localStream, (stream) => {
  if (localVideoRef.value && stream) {
    localVideoRef.value.srcObject = stream
  }
}, { immediate: true })

// 监听远程流变化
watch(() => props.remoteStream, (stream) => {
  if (remoteVideoRef.value && stream) {
    remoteVideoRef.value.srcObject = stream
  }
}, { immediate: true })

// 组件挂载后设置流
onMounted(() => {
  if (props.localStream && localVideoRef.value) {
    localVideoRef.value.srcObject = props.localStream
  }
  if (props.remoteStream && remoteVideoRef.value) {
    remoteVideoRef.value.srcObject = props.remoteStream
  }
})

const onEndCall = () => {
  emit('end-call')
}

const onToggleMute = () => {
  emit('toggle-mute')
}

const onToggleCamera = () => {
  emit('toggle-camera')
}
</script>

<style scoped>
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
