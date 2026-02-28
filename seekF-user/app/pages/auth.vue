<script setup lang="ts">
import { ref } from "vue";

const loginType = ref('password');
const loginForm = ref({
  username: '',
  password: '',
  phone: '',
  code: '',
  remember: true
});
const codeCountdown = ref(0);

// 公共输入框样式
const inputClass = "w-full h-12 px-4 border border-gray-300 rounded-lg transition-all duration-200 focus:border-[#60a5fa] focus:ring-2 focus:ring-[#60a5fa]/20 outline-none text-sm bg-white text-gray-900 placeholder:text-gray-400";

const getVerifyCode = () => {
  if (!/^1[3-9]\d{9}$/.test(loginForm.value.phone)) {
    alert('请输入正确的手机号');
    return;
  }
  codeCountdown.value = 60;
  const timer = setInterval(() => {
    codeCountdown.value--;
    if (codeCountdown.value <= 0) clearInterval(timer);
  }, 1000);
  console.log('发送验证码到：', loginForm.value.phone);
};

const handleLogin = () => {
  if (loginType.value === 'password') {
    if (!loginForm.value.username || !loginForm.value.password) {
      alert('请输入账号和密码');
      return;
    }
  } else {
    if (!loginForm.value.phone || !loginForm.value.code) {
      alert('请输入手机号和验证码');
      return;
    }
  }
  console.log('登录参数：', loginForm.value);
  alert('登录成功（仅演示）');
};

const goBack = () => window.history.back();
</script>

<template>
  <div class="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-background p-4">
    <InspirauiParticlesBg
      class="absolute inset-0 z-0"
      :quantity="100"
      :ease="100"
      color="#000"
      :staticity="10"
      refresh
    />

    <div class="relative z-10 w-full max-w-4xl bg-white/90 rounded-2xl shadow-2xl overflow-hidden backdrop-blur-md border border-white/20">
      <div class="flex flex-col lg:flex-row">
        <!-- 左侧：扫码登录 -->
        <div class="w-full lg:w-96 bg-gradient-to-br from-gray-50 to-gray-100 flex flex-col items-center justify-center px-8 py-12 lg:py-16">
          <h3 class="text-xl font-semibold text-gray-800 mb-8">扫码登录</h3>
          <div class="w-48 h-48 bg-white rounded-xl shadow-lg flex items-center justify-center mb-4 border border-gray-200">
            <div class="w-44 h-44 bg-gray-100 rounded-lg"></div>
          </div>
          <p class="text-sm text-gray-500">请使用微信扫码登录账号</p>
        </div>

        <!-- 右侧：登录表单 -->
        <div class="flex-1 px-8 py-10 lg:px-12 lg:py-16">
          <button 
            class="absolute top-4 right-4 w-8 h-8 flex items-center justify-center text-gray-400 hover:text-gray-600 transition-colors text-2xl leading-none"
            @click="goBack"
          >×</button>
          
          <!-- 标签切换 -->
          <div class="flex mb-8 border-b border-gray-200">
            <button 
              v-for="(item, key) in { password: '账号密码登录', code: '手机号验证码登录' }"
              :key="key"
              class="flex-1 pb-3 text-sm font-medium text-gray-600 transition-colors relative"
              :class="loginType === key ? 'text-[#60a5fa]' : 'hover:text-gray-900'"
              @click="loginType = key"
            >
              {{ item }}
              <span 
                v-if="loginType === key"
                class="absolute bottom-0 left-0 right-0 h-0.5 bg-[#60a5fa] rounded-full"
              ></span>
            </button>
          </div>

          <!-- 账号密码登录 -->
          <form v-if="loginType === 'password'" @submit.prevent="handleLogin" class="space-y-5">
            <input 
              type="text" 
              v-model="loginForm.username" 
              placeholder="请输入账号/手机号"
              :class="inputClass"
            />
            <input 
              type="password" 
              v-model="loginForm.password" 
              placeholder="请输入密码"
              :class="inputClass"
            />
            <div class="flex justify-between items-center text-xs">
              <label class="flex items-center gap-2 text-gray-600 cursor-pointer">
                <input type="checkbox" v-model="loginForm.remember" class="remember-checkbox w-4 h-4" />
                记住密码
              </label>
              <a href="#" class="text-gray-500 hover:text-[#60a5fa] transition-colors">忘记密码？</a>
            </div>
            <button 
              type="submit"
              class="w-full h-12 bg-gradient-to-r from-[#60a5fa] to-[#3b82f6] text-white rounded-lg text-base font-semibold shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
            >
              登录
            </button>
          </form>

          <!-- 手机号验证码登录 -->
          <form v-else @submit.prevent="handleLogin" class="space-y-5">
            <input 
              type="tel" 
              v-model="loginForm.phone" 
              placeholder="请输入手机号"
              :class="inputClass"
            />
            <div class="flex gap-3">
              <input 
                type="text" 
                v-model="loginForm.code" 
                placeholder="请输入验证码"
                :class="inputClass"
              />
              <button 
                type="button"
                class="w-32 h-12 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium transition-all duration-200 hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="!loginForm.phone || codeCountdown > 0"
                @click="getVerifyCode"
              >
                {{ codeCountdown > 0 ? `${codeCountdown}s` : '获取验证码' }}
              </button>
            </div>
            <button 
              type="submit"
              class="w-full h-12 bg-gradient-to-r from-[#60a5fa] to-[#3b82f6] text-white rounded-lg text-base font-semibold shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
            >
              登录
            </button>
          </form>

          <!-- 注册入口 -->
          <div class="mt-6 text-center text-sm text-gray-600">
            还没有账号？
            <a href="/register" class="text-[#60a5fa] hover:underline font-medium ml-1">去注册</a>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<style scoped>
.remember-checkbox {
  appearance: none;
  width: 1rem;
  height: 1rem;
  border: 2px solid #d1d5db;
  border-radius: 0.25rem;
  cursor: pointer;
  position: relative;
  transition: all 0.2s;
}

.remember-checkbox:checked {
  background-color: #60a5fa;
  border-color: #60a5fa;
}

.remember-checkbox:checked::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -60%) rotate(45deg);
  width: 4px;
  height: 8px;
  border: solid white;
  border-width: 0 2px 2px 0;
}
</style>
