<template>
  <div class="flex min-h-screen bg-white">
    <!-- 左侧侧边栏 -->
    <aside class="fixed top-0 left-0 bottom-0 w-[200px] bg-white border-r border-gray-100 p-0 flex flex-col">
      <!-- 侧边栏头部 Logo -->
      <div class="px-5 py-4 mb-4">
        <div class="text-xl font-bold text-[#60a5fa]">seekF</div>
      </div>

      <!-- 侧边栏导航 -->
      <nav class="flex-1 px-2">
        <NuxtLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-3 py-3 rounded-lg mb-1 transition-colors hover:bg-gray-50"
          :class="{ 'bg-blue-50 text-[#0073ff]': route.path === item.path }"
        >
          <Icon :name="item.icon" class="text-lg" />
          <span class="font-medium">{{ item.label }}</span>
        </NuxtLink>
      </nav>

      <!-- 侧边栏底部 -->
      <div class="px-2 pb-4">
        <button
          @click="logout"
          class="flex items-center gap-3 w-full px-3 py-3 rounded-lg border border-gray-200 hover:bg-gray-50 transition-colors"
        >
          <Icon name="uil:signout" class="text-lg" />
          <span class="font-medium">退出登录</span>
        </button>
      </div>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="ml-[200px] flex-1 p-6">
      <slot />
    </main>
  </div>
</template>

<script setup>

// 引入路由对象，用于判断当前激活的导航项
const route = useRoute()

// 侧边栏导航数据
const navItems = [
  { path: '/chat', label: '消息', icon: 'uil:comment-alt' },
  { path: '/contact', label: '联系人', icon: 'uil:users-alt' },
  { path: '/discover', label: '发现', icon: 'uil:plus-square' },
  { path: '/aichat', label: 'AIChat', icon: 'uil:robot' },
  { path: '/my', label: '我', icon: 'uil:user' }
]

const logout = async () => {
  try {
    // 向后端发送退出登录请求
    const res = await useApi$('/user/logout', {
      method: 'POST'
    });
    
    if (res && res.code === 200) {
      ElMessage.success(res.message || '退出登录成功');
    } else {
      console.error('登出请求失败:', res);
    }
  } catch (err) {
    console.error('登出请求异常:', err);
    ElMessage.error(err?.data?.message || err?.message || '退出登录失败');
    // 即使后端请求失败，也要清除本地信息
  } finally {
    // 清除用户信息和token
    const authState = useAuthState();
    authState.clear();
    
    // 跳转到登录页
    navigateTo('/login');
  }
}
</script>