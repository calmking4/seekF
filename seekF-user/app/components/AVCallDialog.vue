<template>
  <Transition name="fade">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    >
      <div class="bg-white rounded-2xl p-8 shadow-2xl min-w-[320px] text-center">
        <!-- 来电者头像 -->
        <div class="mb-4">
          <el-avatar :size="80" :src="callerInfo.avatar" class="ring-4 ring-green-200 animate-pulse">
            {{ callerInfo.name ? callerInfo.name.charAt(0) : '?' }}
          </el-avatar>
        </div>

        <!-- 来电者姓名 -->
        <h3 class="text-xl font-semibold mb-2">{{ callerInfo.name }}</h3>

        <!-- 来电提示 -->
        <p class="text-gray-500 mb-6">{{ callerInfo.name }} 邀请你进行视频通话...</p>

        <!-- 按钮组 -->
        <div class="flex justify-center gap-8">
          <!-- 拒绝按钮 -->
          <button
            class="w-16 h-16 rounded-full bg-red-500 hover:bg-red-600 text-white flex items-center justify-center transition-colors shadow-lg"
            @click="onReject"
          >
            <Icon name="fluent:call-end-24-filled" class="text-2xl" />
          </button>

          <!-- 接听按钮 -->
          <button
            class="w-16 h-16 rounded-full bg-green-500 hover:bg-green-600 text-white flex items-center justify-center transition-colors shadow-lg animate-pulse"
            @click="onAccept"
          >
            <Icon name="fluent:call-24-filled" class="text-2xl" />
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
  callerInfo: {
    type: Object,
    default: () => ({
      id: '',
      name: '',
      avatar: ''
    })
  }
})

const emit = defineEmits(['accept', 'reject'])

const onAccept = () => {
  emit('accept')
}

const onReject = () => {
  emit('reject')
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
