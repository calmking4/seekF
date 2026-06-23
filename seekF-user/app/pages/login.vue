<template>
  <div class="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-background p-4">
    <InspirauiParticlesBg
      class="absolute inset-0 z-0"
      :quantity="1000"
      :ease="100"
      color="60a5fa"
      :staticity="10"
      refresh
    />

    <div class="relative z-10 w-full max-w-[60rem] bg-white/90 rounded-2xl shadow-2xl overflow-hidden backdrop-blur-md border border-white/20">
      <div class="flex flex-col lg:flex-row">
        <div
          class="w-full h-72 lg:h-auto lg:w-[29rem] bg-white bg-cover bg-center lg:bg-[position:38%_center]"
          :style="{ backgroundImage: `url(${loginBg})` }"
          aria-hidden="true"
        ></div>

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
              :disabled="loading"
            >
              <span v-if="loading">登录中...</span>
              <span v-else>登录</span>
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

          <!-- 第三方登录 -->
          <div class="mt-8">
            <div class="relative mb-4">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-gray-200"></div>
              </div>
              <div class="relative flex justify-center text-xs">
                <span class="bg-white px-3 text-gray-500">其他登录方式</span>
              </div>
            </div>
            <div class="flex gap-3">
              <button
                type="button"
                class="flex-1 h-12 flex items-center justify-center gap-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors"
                @click="loginWithGithub"
              >
                <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
                </svg>
                GitHub
              </button>
              <button
                type="button"
                class="flex-1 h-12 flex items-center justify-center gap-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors"
                @click="loginWithGitee"
              >
                <svg class="w-5 h-5" viewBox="0 0 1024 1024" fill="currentColor" aria-hidden="true">
                  <path d="M512 1024q-104 0-199-40-92-39-163-110T40 711Q0 616 0 512t40-199Q79 221 150 150T313 40q95-40 199-40t199 40q92 39 163 110t110 163q40 95 40 199t-40 199q-39 92-110 163T711 984q-95 40-199 40z m259-569H480q-10 0-17.5 7.5T455 480v64q0 10 7.5 17.5T480 569h177q11 0 18.5 7.5T683 594v13q0 31-22.5 53.5T607 683H367q-11 0-18.5-7.5T341 657V417q0-31 22.5-53.5T417 341h354q11 0 18-7t7-18v-63q0-11-7-18t-18-7H417q-38 0-72.5 14T283 283q-27 27-41 61.5T228 417v354q0 11 7 18t18 7h373q46 0 85.5-22.5t62-62Q796 672 796 626V480q0-10-7-17.5t-18-7.5z"/>
                </svg>
                Gitee
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import loginBg from '~/assets/images/background.png'

definePageMeta({
  layout: 'auth'
})

const loginType = ref('password');
const loginForm = ref({
  username: '',
  password: '',
  phone: '',
  code: '',
  remember: true
});
const codeCountdown = ref(0);
const loading = ref(false);

// 公共输入框样式
const inputClass = "w-full h-12 px-4 border border-gray-300 rounded-lg transition-all duration-200 focus:border-[#60a5fa] focus:ring-2 focus:ring-[#60a5fa]/20 outline-none text-sm bg-white text-gray-900 placeholder:text-gray-400";

const getVerifyCode = async () => {
  if (!/^1[3-9]\d{9}$/.test(loginForm.value.phone)) {
    ElMessage('请输入正确的手机号');
    return;
  }
  
  try {
    // 调用发送验证码接口
    const res = await useApi$('/user/sendVerifyCode', {
      method: 'POST',
      body: {
        telephone: loginForm.value.phone
      }
    });
    
    if (res && res.code === 200) {
      ElMessage.success('验证码发送成功');
      // 开始倒计时
      codeCountdown.value = 60;
      const timer = setInterval(() => {
        codeCountdown.value--;
        if (codeCountdown.value <= 0) clearInterval(timer);
      }, 1000);
    } else {
      ElMessage.error(res?.message || '验证码发送失败');
    }
  } catch (err) {
    console.error('发送验证码错误:', err);
    ElMessage.error(err?.data?.message || err?.message || '发送验证码请求失败');
  }
};

const handleLogin = async () => {
  if (loginType.value === 'password') {
    if (!loginForm.value.username || !loginForm.value.password) {
      ElMessage('请输入账号和密码');
      return;
    }

    loading.value = true;

    try {
      // 使用 useApi$ 发送登录请求（$fetch：返回响应对象，失败会 throw）
      const res = await useApi$('/user/login', {
        method: 'POST',
        body: {
          telephone: loginForm.value.username,
          password: loginForm.value.password
        }
      });

      if (res && res.code === 200) {
        // 处理成功响应，获取用户信息和token
        const { user } = res.data;

        // 存储用户信息和token
        const authState = useAuthState();
        authState.setUser(user);

        ElMessage.success('登录成功');

        // 跳转到首页或其他页面
        navigateTo('/');
      } else {
        ElMessage.error(res?.message || '登录失败');
      }
    } catch (err) {
      console.error('登录错误:', err);
      ElMessage.error(err?.data?.message || err?.message || '登录请求失败');
    } finally {
      loading.value = false;
    }
  } else {
    if (!loginForm.value.phone || !loginForm.value.code) {
      ElMessage('请输入手机号和验证码');
      return;
    }

    loading.value = true;

    try {
      // 使用 useApi$ 发送验证码登录请求
      const res = await useApi$('/user/loginByCode', {
        method: 'POST',
        body: {
          telephone: loginForm.value.phone,
          code: loginForm.value.code
        }
      });

      if (res && res.code === 200) {
        // 处理成功响应，获取用户信息和token
        const { user } = res.data;

        // 存储用户信息和token
        const authState = useAuthState();
        authState.setUser(user);

        ElMessage.success('登录成功');

        // 跳转到首页或其他页面
        navigateTo('/');
      } else {
        ElMessage.error(res?.message || '登录失败');
      }
    } catch (err) {
      console.error('登录错误:', err);
      ElMessage.error(err?.data?.message || err?.message || '登录请求失败');
    } finally {
      loading.value = false;
    }
  }
};

const goBack = () => window.history.back();

const loginWithGithub = () => {
  const config = useRuntimeConfig()
  window.location.href = `${config.public.apiBase}user/github/login`
};

const loginWithGitee = () => {
  const config = useRuntimeConfig()
  window.location.href = `${config.public.apiBase}user/gitee/login`
};
</script>


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
