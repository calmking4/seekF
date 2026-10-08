<template>
    <aside class="question-outline" aria-label="本次对话的问题导航">
        <nav class="outline-list" aria-label="历史问题">
            <button
                v-if="hasMore"
                class="outline-item load-earlier"
                :disabled="loading"
                :aria-label="loading ? '正在加载更早的问题' : '加载更早的问题'"
                @click="$emit('load-more')"
            >
                <span class="question-text">{{ loading ? '加载中...' : '加载更早的问题' }}</span>
                <span class="question-marker earlier-marker" aria-hidden="true">···</span>
            </button>
            <button
                v-for="question in questions"
                :key="question.messageId"
                class="outline-item"
                :class="{ active: activeId === question.messageId }"
                :aria-current="activeId === question.messageId ? 'location' : undefined"
                :aria-label="questionLabel(question)"
                @click="$emit('select', question.messageId)"
            >
                <span class="question-text">{{ questionLabel(question) }}</span>
                <span class="question-marker" aria-hidden="true"><span class="question-dot"></span></span>
            </button>
        </nav>
    </aside>
</template>

<script setup>
defineProps({
    questions: { type: Array, default: () => [] },
    activeId: { type: String, default: '' },
    hasMore: Boolean,
    loading: Boolean
})
defineEmits(['select', 'load-more'])

const questionLabel = question =>
    !question.content || question.content === '图片' ? '图片提问' : question.content
</script>

<style scoped>
.question-outline {
    position: absolute;
    top: 45%;
    right: 12px;
    transform: translateY(-50%);
    width: 32px;
    padding: 6px 0;
    border: 1px solid transparent;
    border-radius: 14px;
    z-index: 20;
    transition: width .18s ease, background .18s ease, box-shadow .18s ease;
}
.question-outline:hover,
.question-outline:has(:focus-visible) {
    width: min(300px, calc(100% - 24px));
    background: white;
    border-color: #ececec;
    box-shadow: 0 6px 24px rgba(0, 0, 0, .08);
}
.outline-list {
    max-height: min(420px, 40vh);
    overflow-y: auto;
    overflow-x: hidden;
    scrollbar-width: none;
    overscroll-behavior: contain;
}
.outline-list::-webkit-scrollbar { display: none; }
.outline-item {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    width: 100%;
    min-height: 28px;
    padding: 0;
    border: none;
    background: transparent;
    color: #6b7280;
    cursor: pointer;
    text-align: left;
}
.question-marker {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 30px;
    min-height: 28px;
}
.question-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: #c4c7cc;
    transition: background .15s ease, transform .15s ease;
}
.outline-item.active .question-dot { background: #6b7280; transform: scale(1.4); }
.outline-item:hover .question-dot,
.outline-item:focus-visible .question-dot { background: #0073ff; transform: scale(1.4); }
.question-text {
    display: none;
    flex: 1;
    min-width: 0;
    padding: 4px 0 4px 16px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    font-size: 13px;
    line-height: 20px;
}
.question-outline:hover .question-text,
.question-outline:has(:focus-visible) .question-text { display: block; }
.outline-item:hover,
.outline-item:focus-visible { background: #f5f7fa; color: #374151; }
.outline-item.active .question-text { color: #0073ff; }
.outline-item:focus-visible { outline: 2px solid #0073ff; outline-offset: -2px; }
.earlier-marker { font-size: 14px; color: #9ca3af; }
.load-earlier:disabled { cursor: wait; opacity: .5; }
@media (prefers-reduced-motion: reduce) {
    .question-outline, .question-dot { transition: none; }
}
</style>
