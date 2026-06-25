<template>
  <div class="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-background p-4">
    <div class="w-full max-w-lg">
      <!-- 卡片 -->
      <div class="bg-white rounded-2xl shadow-xl p-8 text-center transform transition-all duration-500"
           :class="status === 'success' ? 'scale-100' : 'scale-100'">

        <!-- GitHub 图标 -->
        <div class="mb-6">
          <div class="w-20 h-20 mx-auto rounded-full bg-gray-900 flex items-center justify-center"
               :class="status === 'error' ? 'bg-red-500' : ''">
            <svg v-if="status !== 'error'" class="w-10 h-10 text-white" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
            </svg>
            <svg v-else class="w-10 h-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </div>
        </div>

        <!-- 加载动画 -->
        <div v-if="status === 'loading'" class="mb-6">
          <div class="flex justify-center gap-1.5">
            <div class="w-3 h-3 rounded-full bg-[#60a5fa] animate-bounce" style="animation-delay: 0ms"></div>
            <div class="w-3 h-3 rounded-full bg-[#60a5fa] animate-bounce" style="animation-delay: 150ms"></div>
            <div class="w-3 h-3 rounded-full bg-[#60a5fa] animate-bounce" style="animation-delay: 300ms"></div>
          </div>
        </div>

        <!-- 成功动画 -->
        <div v-if="status === 'success'" class="mb-6">
          <div class="w-16 h-16 mx-auto rounded-full bg-green-100 flex items-center justify-center animate-scale-in">
            <svg class="w-8 h-8 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
          </div>
        </div>

        <!-- 状态文字 -->
        <h2 class="text-xl font-semibold text-gray-800 mb-2">
          {{ title }}
        </h2>
        <p class="text-gray-500 text-sm">
          {{ message }}
        </p>

        <!-- 失败时的返回按钮 -->
        <div v-if="status === 'error'" class="mt-6">
          <button
            @click="navigateTo('/login')"
            class="px-6 py-2.5 bg-[#60a5fa] text-white rounded-lg text-sm font-medium hover:bg-[#3b82f6] transition-colors"
          >
            返回登录
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// 页面级 SEO
useSeoMeta({
  title: 'GitHub 登录',
  description: '正在通过 GitHub 账号登录 seekF。',
})

definePageMeta({
  layout: 'auth'
})

const status = ref('loading')
const title = ref('正在登录')
const message = ref('正在验证 GitHub 账号信息...')

onMounted(() => {
  try {
    const route = useRoute()
    const userParam = route.query.user

    if (userParam) {
      const user = JSON.parse(userParam)
      const authState = useAuthState()
      authState.setUser(user)

      status.value = 'success'
      title.value = '登录成功'
      message.value = `欢迎回来，${user.nickname || '用户'}`

      setTimeout(() => navigateTo('/'), 1500)
    } else {
      status.value = 'error'
      title.value = '登录失败'
      message.value = '未获取到用户信息，请重试'
    }
  } catch (err) {
    console.error('GitHub 回调处理错误:', err)
    status.value = 'error'
    title.value = '登录失败'
    message.value = err.message || '处理登录信息时出错'
  }
})
</script>

<style scoped>
@keyframes scale-in {
  0% {
    transform: scale(0);
    opacity: 0;
  }
  50% {
    transform: scale(1.2);
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.animate-scale-in {
  animation: scale-in 0.5s ease-out forwards;
}
</style>
