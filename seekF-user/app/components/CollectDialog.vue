<template>
  <Teleport to="body">
    <Transition name="collect-fade">
      <div v-if="visible" class="collect-overlay" @click.self="$emit('close')">
        <div class="collect-dialog">
          <div class="dialog-header">
            <h3>收藏到</h3>
            <button class="close-btn" @click="$emit('close')">
              <Icon name="mdi:close" />
            </button>
          </div>

          <!-- 收藏夹列表 -->
          <div class="folder-list">
            <div
              v-for="folder in folders"
              :key="folder.uuid"
              class="folder-item"
              @click="handleCollect(folder)"
            >
              <div class="folder-cover">
                <img v-if="folder.cover_url" :src="folder.cover_url" alt="" />
                <div v-else class="folder-cover-empty">
                  <Icon name="uil:folder" />
                </div>
              </div>
              <div class="folder-info">
                <span class="folder-name">{{ folder.name }}</span>
                <span class="folder-count">{{ folder.post_count }} 篇</span>
              </div>
              <Icon name="mdi:chevron-right" class="folder-arrow" />
            </div>
            <div v-if="folders.length === 0 && !loading" class="empty-tip">
              暂无收藏夹，请先创建
            </div>
            <div v-if="loading" class="loading-tip">
              <Icon name="uil:spinner" class="animate-spin" />
              加载中...
            </div>
          </div>

          <!-- 新建收藏夹 -->
          <div class="create-section">
            <div v-if="!showCreate" class="create-btn" @click="showCreate = true">
              <Icon name="mdi:plus" />
              <span>新建收藏夹</span>
            </div>
            <div v-else class="create-form">
              <input
                v-model="newFolderName"
                class="create-input"
                placeholder="收藏夹名称"
                maxlength="50"
                @keyup.enter="handleCreateFolder"
              />
              <div class="create-options">
                <label class="public-option">
                  <input type="checkbox" v-model="newFolderPublic" />
                  <span>公开</span>
                </label>
                <div class="create-actions">
                  <button class="cancel-btn" @click="showCreate = false">取消</button>
                  <button class="confirm-btn" @click="handleCreateFolder" :disabled="!newFolderName.trim()">
                    创建
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useApi$ } from '~/composables/useApi'
import { ElMessage } from 'element-plus'

const props = defineProps({
  visible: { type: Boolean, default: false },
  postUuid: { type: String, default: '' },
})

const emit = defineEmits(['close', 'collected'])

const folders = ref([])
const loading = ref(false)
const showCreate = ref(false)
const newFolderName = ref('')
const newFolderPublic = ref(false)

const loadFolders = async () => {
  loading.value = true
  try {
    const res = await useApi$('/user/discover/folder/list', { method: 'POST' })
    if (res.code === 200) {
      folders.value = res.data?.list || []
    }
  } catch (e) {
    console.error('加载收藏夹失败:', e)
  } finally {
    loading.value = false
  }
}

const handleCollect = async (folder) => {
  try {
    const res = await useApi$('/user/discover/collect', {
      method: 'POST',
      body: { post_uuid: props.postUuid, folder_uuid: folder.uuid },
    })
    if (res.code === 200) {
      ElMessage.success('收藏成功')
      emit('collected', { collectCount: res.data.collect_count, folderUuid: folder.uuid })
      emit('close')
    } else {
      ElMessage.error(res.message || '收藏失败')
    }
  } catch (e) {
    console.error('收藏失败:', e)
  }
}

const handleCreateFolder = async () => {
  if (!newFolderName.value.trim()) return
  try {
    const res = await useApi$('/user/discover/folder/create', {
      method: 'POST',
      body: {
        name: newFolderName.value.trim(),
        description: '',
        is_public: newFolderPublic.value ? 1 : 0,
      },
    })
    if (res.code === 200) {
      ElMessage.success('创建成功')
      newFolderName.value = ''
      newFolderPublic.value = false
      showCreate.value = false
      await loadFolders()
    } else {
      ElMessage.error(res.message || '创建失败')
    }
  } catch (e) {
    console.error('创建收藏夹失败:', e)
  }
}

watch(() => props.visible, (val) => {
  if (val) {
    loadFolders()
    showCreate.value = false
    newFolderName.value = ''
    newFolderPublic.value = false
  }
})
</script>

<style scoped>
.collect-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.collect-dialog {
  background: #fff;
  border-radius: 16px;
  width: 380px;
  max-height: 500px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.dialog-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 20px;
  color: #999;
  padding: 4px;
}

.folder-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.folder-item {
  display: flex;
  align-items: center;
  padding: 10px 20px;
  cursor: pointer;
  transition: background 0.2s;
}

.folder-item:hover {
  background: #f9f9f9;
}

.folder-cover {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  margin-right: 12px;
}

.folder-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.folder-cover-empty {
  width: 100%;
  height: 100%;
  background: #f5f5f5;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  color: #bbb;
}

.folder-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.folder-name {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.folder-count {
  font-size: 12px;
  color: #999;
  margin-top: 2px;
}

.folder-arrow {
  color: #ccc;
  font-size: 18px;
}

.empty-tip,
.loading-tip {
  text-align: center;
  padding: 30px 0;
  color: #999;
  font-size: 14px;
}

.loading-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.create-section {
  border-top: 1px solid #f0f0f0;
  padding: 12px 20px;
}

.create-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #60a5fa;
  font-size: 14px;
  font-weight: 500;
}

.create-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.create-input {
  width: 100%;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.create-input:focus {
  border-color: #60a5fa;
}

.create-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.public-option {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #666;
  cursor: pointer;
}

.create-actions {
  display: flex;
  gap: 8px;
}

.cancel-btn,
.confirm-btn {
  border: none;
  border-radius: 6px;
  padding: 6px 16px;
  font-size: 13px;
  cursor: pointer;
}

.cancel-btn {
  background: #f5f5f5;
  color: #666;
}

.confirm-btn {
  background: #60a5fa;
  color: #fff;
}

.confirm-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 动画 */
.collect-fade-enter-active,
.collect-fade-leave-active {
  transition: opacity 0.2s ease;
}

.collect-fade-enter-active .collect-dialog,
.collect-fade-leave-active .collect-dialog {
  transition: transform 0.2s ease;
}

.collect-fade-enter-from,
.collect-fade-leave-to {
  opacity: 0;
}

.collect-fade-enter-from .collect-dialog,
.collect-fade-leave-to .collect-dialog {
  transform: translateY(20px);
}
</style>
