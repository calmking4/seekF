<template>
  <div class="flex h-screen bg-gray-100">
    <!-- 左侧侧边栏 -->
    <aside class="w-80 bg-white border-r flex flex-col pr-3"> <!-- 这里添加 pr-3 -->
      <!-- 顶部搜索栏 - 参考样式 -->
      <SearchBar />

      <!-- 通知入口 - 箭头替换为 Nuxt Icon -->
      <div class="border-t">
        <div
          class="flex items-center justify-between px-3 py-2 hover:bg-gray-50 cursor-pointer"
        >
          <span>好友通知</span>
          <Icon name="uil:angle-right" class="text-gray-400" />
        </div>
        <div
          class="flex items-center justify-between px-3 py-2 hover:bg-gray-50 cursor-pointer"
        >
          <span>群通知</span>
          <Icon name="uil:angle-right" class="text-gray-400" />
        </div>
      </div>

      <!-- 好友/群聊切换 -->
      <el-tabs v-model="activeTab" class="flex-1 overflow-y-auto">
        <!-- 好友列表 -->
        <el-tab-pane label="好友" name="friend">
          <div class="py-1">
            <div
              v-for="group in friendGroups"
              :key="group.name"
              class="border-b last:border-b-0"
            >
              <div
                class="flex items-center justify-between px-3 py-1.5 hover:bg-gray-50 cursor-pointer"
                @click="group.expanded = !group.expanded"
              >
                <div class="flex items-center gap-2">
                  <!-- 箭头替换为 Nuxt Icon 并添加旋转效果 -->
                  <Icon 
                    name="uil:angle-right" 
                    class="text-gray-400 transition-transform duration-200"
                    :class="{ 'rotate-90': group.expanded }"
                  />
                  <span>{{ group.name }}</span>
                </div>
                <span class="text-xs text-gray-400">
                  {{ group.online }}/{{ group.total }}
                </span>
              </div>
              <transition name="el-collapse-transition">
                <div v-if="group.expanded" class="bg-gray-50">
                  <div
                    v-for="friend in group.friends"
                    :key="friend.id"
                    class="flex items-center gap-3 px-6 py-2 hover:bg-gray-100 cursor-pointer"
                  >
                    <el-avatar :size="32" :src="friend.avatar" />
                    <span class="text-sm">{{ friend.name }}</span>
                  </div>
                </div>
              </transition>
            </div>
          </div>
        </el-tab-pane>

        <!-- 群聊列表 -->
        <el-tab-pane label="群聊" name="group">
          <div class="py-1">
            <div
              v-for="group in groupCategories"
              :key="group.name"
              class="border-b last:border-b-0"
            >
              <div
                class="flex items-center justify-between px-3 py-1.5 hover:bg-gray-50 cursor-pointer"
                @click="group.expanded = !group.expanded"
              >
                <div class="flex items-center gap-2">
                  <!-- 箭头替换为 Nuxt Icon 并添加旋转效果 -->
                  <Icon 
                    name="uil:angle-right" 
                    class="text-gray-400 transition-transform duration-200"
                    :class="{ 'rotate-90': group.expanded }"
                  />
                  <span>{{ group.name }}</span>
                </div>
                <span class="text-xs text-gray-400">{{ group.count }}</span>
              </div>
              <transition name="el-collapse-transition">
                <div v-if="group.expanded" class="bg-gray-50">
                  <div
                    v-for="item in group.list"
                    :key="item.id"
                    class="flex items-center gap-3 px-6 py-2 hover:bg-gray-100 cursor-pointer"
                  >
                    <el-avatar :size="32" :src="item.avatar" />
                    <span class="text-sm">{{ item.name }}</span>
                  </div>
                </div>
              </transition>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </aside>

    <!-- 右侧主内容区 -->
    <main class="flex-1 flex flex-col">
      <!-- 顶部操作栏 -->
      <header class="h-10 border-b flex items-center justify-end px-3 gap-2">
        <el-button size="small" type="primary" link icon="Monitor" />
        <el-button size="small" type="primary" link icon="Minus" />
        <el-button size="small" type="primary" link icon="FullScreen" />
        <el-button size="small" type="primary" link icon="Close" />
      </header>

      <!-- 内容区 -->
      <div class="flex-1 flex flex-col items-center justify-center text-gray-400">
        <Icon name="uil:comment-alt" class="text-6xl mb-3" />
      </div>
    </main>
  </div>
</template>

<script setup>
const activeTab = ref('friend')

// 好友分组数据
const friendGroups = ref([
  {
    name: '我的设备',
    online: 1,
    total: 1,
    expanded: false,
    friends: [
      { id: 1, name: '我的iPhone', avatar: 'https://picsum.photos/32/32?random=1' }
    ]
  },
  {
    name: '机器人',
    online: 1,
    total: 1,
    expanded: false,
    friends: [
      { id: 2, name: 'QQ小冰', avatar: 'https://picsum.photos/32/32?random=2' }
    ]
  },
  {
    name: '特别关心',
    online: 0,
    total: 0,
    expanded: false,
    friends: []
  },
  {
    name: '我的好友',
    online: 91,
    total: 136,
    expanded: false,
    friends: Array.from({ length: 10 }, (_, i) => ({
      id: i + 3,
      name: `好友${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 3}`
    }))
  },
  {
    name: '朋友',
    online: 5,
    total: 9,
    expanded: false,
    friends: Array.from({ length: 5 }, (_, i) => ({
      id: i + 13,
      name: `朋友${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 13}`
    }))
  },
  {
    name: '家人',
    online: 3,
    total: 6,
    expanded: false,
    friends: Array.from({ length: 3 }, (_, i) => ({
      id: i + 18,
      name: `家人${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 18}`
    }))
  },
  {
    name: '同学',
    online: 10,
    total: 20,
    expanded: false,
    friends: Array.from({ length: 10 }, (_, i) => ({
      id: i + 21,
      name: `同学${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 21}`
    }))
  },
  {
    name: '六年级同学',
    online: 8,
    total: 12,
    expanded: false,
    friends: Array.from({ length: 8 }, (_, i) => ({
      id: i + 31,
      name: `六年级同学${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 31}`
    }))
  },
  {
    name: '漫友',
    online: 7,
    total: 16,
    expanded: false,
    friends: Array.from({ length: 7 }, (_, i) => ({
      id: i + 39,
      name: `漫友${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 39}`
    }))
  },
  {
    name: '未知分组',
    online: 0,
    total: 0,
    expanded: false,
    friends: []
  },
  {
    name: '陌生人（注意）',
    online: 0,
    total: 1,
    expanded: false,
    friends: []
  }
])

// 群聊分类数据
const groupCategories = ref([
  {
    name: '置顶群聊',
    count: 0,
    expanded: false,
    list: []
  },
  {
    name: '未命名的群聊',
    count: 1,
    expanded: false,
    list: [
      { id: 101, name: '未命名群聊1', avatar: 'https://picsum.photos/32/32?random=101' }
    ]
  },
  {
    name: '我创建的群聊',
    count: 2,
    expanded: false,
    list: [
      { id: 102, name: '我的项目组', avatar: 'https://picsum.photos/32/32?random=102' },
      { id: 103, name: '家庭群', avatar: 'https://picsum.photos/32/32?random=103' }
    ]
  },
  {
    name: '我管理的群聊',
    count: 4,
    expanded: false,
    list: [
      { id: 104, name: '班级群', avatar: 'https://picsum.photos/32/32?random=104' },
      { id: 105, name: '兴趣小组', avatar: 'https://picsum.photos/32/32?random=105' },
      { id: 106, name: '工作群', avatar: 'https://picsum.photos/32/32?random=106' },
      { id: 107, name: '技术交流群', avatar: 'https://picsum.photos/32/32?random=107' }
    ]
  },
  {
    name: '我加入的群聊',
    count: 45,
    expanded: false,
    list: Array.from({ length: 10 }, (_, i) => ({
      id: i + 108,
      name: `加入的群聊${i + 1}`,
      avatar: `https://picsum.photos/32/32?random=${i + 108}`
    }))
  }
])
</script>

<style scoped>
/* 自定义滚动条样式 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 3px;
}
::-webkit-scrollbar-thumb:hover {
  background: #9ca3af;
}

/* 旋转样式 */
.rotate-90 {
  transform: rotate(90deg);
}

/* 过渡效果 */
.transition-transform {
  transition-property: transform;
}

.duration-200 {
  transition-duration: 200ms;
}
</style>