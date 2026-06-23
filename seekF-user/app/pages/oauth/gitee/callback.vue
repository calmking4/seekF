<template>
  <div class="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-background p-4">
    <div class="w-full max-w-lg">
      <!-- 卡片 -->
      <div class="bg-white rounded-2xl shadow-xl p-8 text-center transform transition-all duration-500"
           :class="status === 'success' ? 'scale-100' : 'scale-100'">

        <!-- Gitee 图标 -->
        <div class="mb-6">
          <div class="w-20 h-20 mx-auto rounded-full flex items-center justify-center"
               :class="status === 'error' ? 'bg-red-500' : 'bg-red-500'">
            <svg v-if="status !== 'error'" class="w-10 h-10 text-white" viewBox="0 0 1024 1024" fill="currentColor">
              <path d="M512 1024q-104 0-199-40-92-39-163-110T40 711Q0 616 0 512t40-199Q79 221 150 150T313 40q95-40 199-40t199 40q92 39 163 110t110 163q40 95 40 199t-40 199q-39 92-110 163T711 984q-95 40-199 40z m259-569H480q-10 0-17.5 7.5T455 480v64q0 10 7.5 17.5T480 569h177q11 0 18.5 7.5T683 594v13q0 31-22.5 53.5T607 683H367q-11 0-18.5-7.5T341 657V417q0-31 22.5-53.5T417 341h354q11 0 18-7t7-18v-63q0-11-7-18t-18-7H417q-38 0-72.5 14T283 283q-27 27-41 61.5T228 417v354q0 11 7 18t18 7h373q46 0 85.5-22.5t62-62Q796 672 796 626V480q0-10-7-17.5t-18-7.5z"/>
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
definePageMeta({
  layout: 'auth'
})

const status = ref('loading')
const title = ref('正在登录')
const message = ref('正在验证 Gitee 账号信息...')

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
    console.error('Gitee 回调处理错误:', err)
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
