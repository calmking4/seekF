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

    <div class="relative z-10 w-full max-w-4xl bg-white/90 rounded-2xl shadow-2xl overflow-hidden backdrop-blur-md border border-white/20">
      <div class="flex flex-col lg:flex-row">
        <!-- 左侧：注册说明/装饰 -->
        <div class="w-full lg:w-96 bg-gradient-to-br from-gray-50 to-gray-100 flex flex-col items-center justify-center px-8 py-12 lg:py-16">
          <h3 class="text-xl font-semibold text-gray-800 mb-8">欢迎注册</h3>
          <div class="w-48 h-48 bg-white rounded-xl shadow-lg flex items-center justify-center mb-4 border border-gray-200">
            <Icon name="uil:user-plus" class="text-6xl text-[#60a5fa]" />
          </div>
          <p class="text-sm text-gray-500 text-center">
            注册后即可享受完整功能<br/>
            安全便捷，一键登录
          </p>
        </div>

        <!-- 右侧：注册表单 -->
        <div class="flex-1 px-8 py-10 lg:px-12 lg:py-16">
          <button 
            class="absolute top-4 right-4 w-8 h-8 flex items-center justify-center text-gray-400 hover:text-gray-600 transition-colors text-2xl leading-none"
            @click="goBack"
          >×</button>
          
          <!-- 表单标题 -->
          <h3 class="text-2xl font-semibold text-gray-800 mb-8">账号注册</h3>

          <!-- 注册表单 -->
          <form @submit.prevent="handleRegister" class="space-y-5">
            <!-- 用户名 -->
            <input 
              type="text" 
              v-model="registerForm.nickname" 
              placeholder="请设置用户名"
              :class="inputClass"
            />
            
            <!-- 手机号 -->
            <input 
              type="tel" 
              v-model="registerForm.telephone" 
              placeholder="请输入手机号"
              :class="inputClass"
            />
            
            <!-- 验证码 -->
            <div class="flex gap-3">
              <input 
                type="text" 
                v-model="registerForm.code" 
                placeholder="请输入验证码"
                :class="inputClass"
              />
              <button 
                type="button"
                class="w-32 h-12 bg-gray-100 text-gray-700 rounded-lg text-sm font-medium transition-all duration-200 hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="!registerForm.telephone || codeCountdown > 0"
                @click="getVerifyCode"
              >
                {{ codeCountdown > 0 ? `${codeCountdown}s` : '获取验证码' }}
              </button>
            </div>
            
            <!-- 密码 -->
            <input 
              type="password" 
              v-model="registerForm.password" 
              placeholder="请设置密码（6-16位）"
              :class="inputClass"
            />
            
            <!-- 确认密码 -->
            <input 
              type="password" 
              v-model="registerForm.confirmPassword" 
              placeholder="请确认密码"
              :class="inputClass"
            />
            
            <!-- 协议勾选 -->
            <div class="flex items-center gap-2 text-xs text-gray-600">
              <input 
                type="checkbox" 
                v-model="registerForm.agree" 
                class="agree-checkbox w-4 h-4" 
              />
              <span>
                我已阅读并同意
                <a href="#" class="text-[#60a5fa] hover:underline">《用户服务协议》</a>
                和
                <a href="#" class="text-[#60a5fa] hover:underline">《隐私政策》</a>
              </span>
            </div>
            
            <!-- 注册按钮 -->
            <button 
              type="submit"
              class="w-full h-12 bg-gradient-to-r from-[#60a5fa] to-[#3b82f6] text-white rounded-lg text-base font-semibold shadow-lg hover:shadow-xl transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
              :disabled="!registerForm.agree || loading"
            >
              <span v-if="!loading">注册账号</span>
              <span v-else>注册中...</span>
            </button>
          </form>

          <!-- 登录入口 -->
          <div class="mt-6 text-center text-sm text-gray-600">
            已有账号？
            <NuxtLink to="/login" class="text-[#60a5fa] hover:underline font-medium ml-1">去登录</NuxtLink>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>

definePageMeta({
  layout: 'auth'
})

// 注册表单数据
const registerForm = ref({
  nickname: '',
  telephone: '',
  code: '', // 验证码字段保留
  password: '',
  confirmPassword: '',
  agree: false
});

// 验证码倒计时
const codeCountdown = ref(0);

// 加载状态
const loading = ref(false);

// 公共输入框样式（和登录页保持一致）
const inputClass = "w-full h-12 px-4 border border-gray-300 rounded-lg transition-all duration-200 focus:border-[#60a5fa] focus:ring-2 focus:ring-[#60a5fa]/20 outline-none text-sm bg-white text-gray-900 placeholder:text-gray-400";

// 获取验证码
const getVerifyCode = () => {
  // 手机号验证
  if (!/^1[3-9]\d{9}$/.test(registerForm.value.telephone)) {
    ElMessage.error('请输入正确的手机号');
    return;
  }
  
  // 暂时模拟验证码发送
  ElMessage.success('验证码已发送（暂为模拟）');
  
  // 启动倒计时
  codeCountdown.value = 60;
  const timer = setInterval(() => {
    codeCountdown.value--;
    if (codeCountdown.value <= 0) clearInterval(timer);
  }, 1000);
  
  console.log('发送注册验证码到：', registerForm.value.telephone);
};

// 处理注册
const handleRegister = async () => {
  // 表单验证
  if (!registerForm.value.nickname) {
    ElMessage.error('请设置用户名');
    return;
  }
  
  if (!/^1[3-9]\d{9}$/.test(registerForm.value.telephone)) {
    ElMessage.error('请输入正确的手机号');
    return;
  }
  
  // 暂时忽略验证码验证，因为后端还未实现
  
  if (registerForm.value.password.length < 6 || registerForm.value.password.length > 16) {
    ElMessage.error('密码长度必须在6-16位之间');
    return;
  }
  
  if (registerForm.value.password !== registerForm.value.confirmPassword) {
    ElMessage.error('两次输入的密码不一致');
    return;
  }
  
  if (!registerForm.value.agree) {
    ElMessage.error('请阅读并同意用户协议和隐私政策');
    return;
  }
  
  // 设置加载状态
  loading.value = true;
  
  try {
    // 使用 useApi 发送注册请求到后端
    const { data, pending, error } = await useApi('/user/register', {
      method: 'POST',
      body: {
        nickname: registerForm.value.nickname,
        telephone: registerForm.value.telephone,
        password: registerForm.value.password
      }
    });

    if (error.value) {
      ElMessage.error(error.value.data?.message || '注册失败');
      return;
    }

    // 注册成功
    ElMessage.success(data?.message || '注册成功！');
    
    // 清空表单
    registerForm.value = {
      nickname: '',
      telephone: '',
      code: '',
      password: '',
      confirmPassword: '',
      agree: false
    };
    
    // 跳转到登录页面
    setTimeout(() => {
      navigateTo('/login');
    }, 1500);

  } catch (err) {
    console.error('注册错误:', err);
    let errorMessage = '注册失败';
    if (err.data && err.data.message) {
      errorMessage = err.data.message;
    } else if (typeof err === 'string') {
      errorMessage = err;
    }
    ElMessage.error(errorMessage);
  } finally {
    loading.value = false;
  }
};

// 返回上一页
const goBack = () => window.history.back();
</script>

<style scoped>
/* 协议勾选框样式（和登录页记住密码样式保持一致） */
.agree-checkbox {
  appearance: none;
  width: 1rem;
  height: 1rem;
  border: 2px solid #d1d5db;
  border-radius: 0.25rem;
  cursor: pointer;
  position: relative;
  transition: all 0.2s;
}

.agree-checkbox:checked {
  background-color: #60a5fa;
  border-color: #60a5fa;
}

.agree-checkbox:checked::after {
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

/* 禁用状态的按钮样式 */
:deep(.bg-gradient-to-r:disabled) {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none !important;
  box-shadow: none !important;
}
</style>