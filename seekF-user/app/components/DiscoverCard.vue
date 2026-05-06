<template>
  <Teleport to="body">
    <Transition name="discover">
      <div v-if="show" class="discover-overlay" @click.self="handleClose">
      <div class="note-card">
        <!-- 关闭按钮 -->
        <button class="close-btn" @click="handleClose">&times;</button>

        <!-- 左侧图片区 -->
        <div class="image-container">
          <img
            v-if="item?.src"
            :src="item.src"
            :alt="item.title"
            class="cover-image"
          />
          <div v-else class="image-content">
            <div class="text-bg">
              <span class="text-line">记录一次</span>
              <span class="text-line">完整的</span>
              <span class="text-line highlight">释放</span>
            </div>
          </div>
        </div>

        <!-- 右侧内容区 -->
        <div class="content-container">
          <!-- 作者信息（固定顶部） -->
          <div class="author-header">
            <div class="author-info">
              <el-avatar size="40" src="https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" />
              <span class="author-name">{{ item?.title || '嵯峨之诗' }}</span>
            </div>
            <el-button type="danger" size="small" class="follow-btn">关注</el-button>
          </div>

          <!-- 可滚动的中间区域 -->
          <div class="scroll-area">
            <!-- 帖子内容 -->
            <div class="post-content">
              <h3 class="post-title">{{ item?.title || '记录一次完整的释放1' }}</h3>
              <p class="post-note">圈外勿入，谢谢理解 🙏</p>
              <p class="post-text">
                今天带朋友做了一次释放，她一直在哭，几乎崩溃，但释放完之后她觉得无比的安心。她同意我把这次神奇的释放过程完整的呈现给大家，希望对有同样感受的宝宝们有所启发。
              </p>
              <p class="post-text">
                补充一点，释放不需要心智，不需要过度思考。因为这是我第一次带她释放，然后她头脑转得飞快hhh。但当她不再过度思考，只是顺着情绪的水流，释放就不再卡住了。
              </p>
              <div class="post-tags">
                <span class="tag">#释放法</span>
                <span class="tag">#显化sp</span>
              </div>
              <p class="post-time">编辑于 2025-07-27</p>
            </div>

            <!-- 评论区 -->
            <div class="comment-section">
              <p class="comment-title">共 110 条评论</p>
              <div class="comment-item" v-for="(c, index) in comments" :key="index">
                <div class="comment-header">
                  <el-avatar size="32" src="https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" />
                  <div class="comment-info">
                    <span class="comment-author">{{ c.author }}</span>
                    <p class="comment-time">{{ c.time }} {{ c.location }}</p>
                  </div>
                </div>
                <p class="comment-text">{{ c.content }}</p>
                <div class="comment-actions">
                  <span><i class="el-icon-star-off" /> {{ c.like }}</span>
                  <span class="reply-btn">回复</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 底部互动栏（固定底部） -->
          <div class="interaction-bar">
            <div class="input-area">
              <span class="input-placeholder">说点什么...</span>
            </div>
            <div class="action-icons">
              <span class="action-item">
                <Icon name="mdi:heart-outline" />
                <span class="action-count">{{ item?.likeCount ?? 919 }}</span>
              </span>
              <span class="action-item">
                <Icon name="mdi:star-outline" />
                <span class="action-count">348</span>
              </span>
              <span class="action-item">
                <Icon name="mdi:chat-outline" />
                <span class="action-count">110</span>
              </span>
              <span class="action-item">
                <Icon name="ri:share-circle-fill" />
              </span>
            </div>
          </div>
        </div>
      </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  item: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close'])

const show = ref(false)

onMounted(() => {
  requestAnimationFrame(() => {
    show.value = true
  })
})

const handleClose = () => {
  show.value = false
  setTimeout(() => emit('close'), 300)
}

// 评论数据
const comments = ref([
  {
    author: '小鱼大王',
    time: '03-24',
    location: '河南',
    content: '小鱼大王的',
    like: 3
  },
  {
    author: '乏了',
    time: '7小时前',
    location: '重庆',
    content: 'fa',
    like: 1
  }
])
</script>

<style scoped>
.discover-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.note-card {
  position: relative;
  display: flex;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  height: 80vh;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.close-btn {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 10;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  font-size: 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  transition: background 0.2s;
}

.close-btn:hover {
  background: rgba(0, 0, 0, 0.6);
}

.image-container {
  flex: 1;
  min-width: 0;
  background: #000;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cover-image {
  display: block;
  max-width: 100%;
  height: 100%;
  object-fit: contain;
}

.image-content {
  background: #fff;
  border: 18px solid #fde6e9;
  border-radius: 12px;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.text-bg {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.text-line {
  font-size: 72px;
  font-weight: bold;
  line-height: 1.2;
}

.highlight {
  background: #cce0f5;
  padding: 0 10px;
}

.page-indicator {
  position: absolute;
  top: 10px;
  right: 10px;
  background: rgba(0, 0, 0, 0.3);
  color: #fff;
  border-radius: 10px;
  padding: 2px 8px;
  font-size: 12px;
}

.arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  font-size: 24px;
  color: #999;
  cursor: pointer;
}

.left-arrow {
  left: 10px;
}

.right-arrow {
  right: 10px;
}

.content-container {
  flex-shrink: 0;
  width: 480px;
  display: flex;
  flex-direction: column;
}

.scroll-area {
  flex: 1;
  overflow-y: auto;
  padding: 0 20px;
  scrollbar-width: none;
}

.scroll-area::-webkit-scrollbar {
  display: none;
}

.author-header {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.author-info {
  display: flex;
  align-items: center;
}

.author-name {
  margin-left: 10px;
  font-size: 16px;
  font-weight: 500;
}

.follow-btn {
  border-radius: 20px;
  padding: 0 15px;
}

.post-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 10px 0;
}

.post-note {
  color: #666;
  font-size: 14px;
  margin: 0 0 15px 0;
}

.post-text {
  font-size: 14px;
  line-height: 1.6;
  color: #333;
  margin: 0 0 15px 0;
}

.post-tags {
  margin: 10px 0;
}

.tag {
  color: #3a86ff;
  font-size: 14px;
  margin-right: 10px;
}

.post-time {
  font-size: 12px;
  color: #999;
  margin: 15px 0;
}

.post-content {
  padding-top: 16px;
}

.comment-section {
  margin-top: 20px;
  padding-bottom: 16px;
}

.comment-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 15px;
}

.comment-item {
  margin-bottom: 20px;
}

.comment-header {
  display: flex;
  align-items: center;
}

.comment-info {
  margin-left: 10px;
}

.comment-author {
  font-size: 14px;
  font-weight: 500;
}

.comment-time {
  font-size: 12px;
  color: #999;
  margin: 2px 0 0 0;
}

.comment-text {
  font-size: 14px;
  margin: 8px 0 5px 42px;
}

.comment-actions {
  font-size: 12px;
  color: #999;
  margin-left: 42px;
}

.reply-btn {
  margin-left: 15px;
  cursor: pointer;
}

.interaction-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-top: 1px solid #eee;
}

.input-area {
  display: flex;
  align-items: center;
  background: #f5f5f5;
  border-radius: 20px;
  padding: 8px 15px;
  flex: 1;
  margin-right: 10px;
}

.input-placeholder {
  margin-left: 8px;
  color: #999;
  font-size: 14px;
}

.action-icons {
  display: flex;
  gap: 16px;
  font-size: 14px;
  color: #666;
}

.action-item {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  font-size: 20px;
  transition: color 0.2s;
}

.action-item:hover {
  color: #ff2442;
}

.action-count {
  font-size: 13px;
}

/* 小红书风格弹窗动画 */
.discover-enter-active {
  transition: opacity 0.35s ease;
}

.discover-enter-active .note-card {
  transition: transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1),
              opacity 0.25s ease;
}

.discover-leave-active {
  transition: opacity 0.25s ease;
}

.discover-leave-active .note-card {
  transition: transform 0.25s cubic-bezier(0.4, 0, 1, 1),
              opacity 0.2s ease;
}

.discover-enter-from {
  opacity: 0;
}

.discover-enter-from .note-card {
  transform: scale(0.85);
  opacity: 0;
}

.discover-leave-to {
  opacity: 0;
}

.discover-leave-to .note-card {
  transform: scale(0.9);
  opacity: 0;
}
</style>