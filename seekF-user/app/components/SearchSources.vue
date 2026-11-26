<template>
  <div v-if="sources.length" class="search-sources" @keydown.esc.stop="collapse">
    <button
      ref="trigger"
      type="button"
      @click="expanded = !expanded"
      class="search-sources-trigger"
      :class="{ active: expanded }"
      :aria-expanded="expanded"
      :aria-controls="listId"
    >
      <Icon :name="knowledge ? 'uil:file-alt' : 'uil:search'" class="text-base" />
      <span>{{ knowledge ? '参考文档' : `已搜索 ${sources.length} 个来源` }}</span>
      <Icon name="uil:angle-down" class="source-chevron text-base" :class="{ expanded }" />
    </button>
    <Transition name="source-dropdown">
    <div v-if="expanded" class="source-dropdown-motion">
    <div class="source-dropdown-clip">
    <section :id="listId" class="source-dropdown" :aria-label="knowledge ? '知识库参考文档' : '网页搜索来源'">
      <header class="source-heading"><span>{{ knowledge ? '知识库参考文档' : '网页来源' }}</span><span class="source-count">{{ sources.length }} {{ knowledge ? '份文档' : '个结果' }}</span></header>
      <ol class="source-list">
      <li v-for="(source, idx) in sources" :key="knowledge ? source.doc_uuid : `${source.url}-${idx}`">
      <button
        v-if="knowledge"
        type="button"
        class="source-link w-full text-left disabled:cursor-wait"
        :disabled="previewLoading"
        @click="openDocument(source)"
      >
        <span class="source-number">{{ String(idx + 1).padStart(2, '0') }}</span>
        <span class="source-copy">
          <span class="source-title">{{ source.title || '知识库文档' }}</span>
          <span v-if="source.matches[0].snippet" class="line-clamp-2 text-xs leading-relaxed text-slate-500">{{ source.matches[0].snippet }}</span>
        </span>
        <Icon name="uil:angle-right" class="source-external" />
      </button>
      <a
        v-else
        :href="source.url"
        target="_blank"
        rel="noopener noreferrer"
        class="source-link"
      >
        <span class="source-number">
          {{ String(idx + 1).padStart(2, '0') }}
        </span>
        <span class="source-copy">
          <span class="source-title">{{ source.title || source.url }}</span>
          <span class="source-domain"><Icon name="uil:globe" />{{ getDomain(source.url) }}</span>
        </span>
        <Icon name="uil:external-link-alt" class="source-external" />
      </a>
      </li>
      </ol>
    </section>
    </div>
    </div>
    </Transition>
    <el-dialog v-if="knowledge" v-model="previewVisible" :title="previewDocument?.title || '知识库文档'" width="min(700px, 92vw)" append-to-body @opened="showFirstHighlight">
      <div class="max-h-[60vh] overflow-y-auto">
        <details v-if="previewDocument" class="mb-4 rounded-lg border border-slate-200 px-3 py-2 text-sm">
          <summary class="cursor-pointer text-slate-700">命中详情（{{ previewDocument.matches.length }} 个分块）</summary>
          <ul class="mt-2 space-y-2">
            <li v-for="match in previewDocument.matches" :key="match.chunk_number">
              <p class="text-xs text-slate-500">
                第 {{ match.chunk_number }} 个分块 · {{ Number.isFinite(match.rerank_score) ? `置信度 ${(match.rerank_score * 100).toFixed(2)}%` : '暂无精排分数' }}
              </p>
              <p v-if="match.snippet" class="mt-1 break-words text-slate-700">{{ match.snippet }}</p>
            </li>
          </ul>
        </details>
        <pre ref="previewBody" class="whitespace-pre-wrap break-words text-sm text-slate-700"><template v-for="(segment, index) in previewSegments" :key="index"><mark v-if="segment.highlight" class="rounded-sm bg-amber-100 text-inherit">{{ segment.text }}</mark><span v-else>{{ segment.text }}</span></template></pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, ref, useId } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  sources: { type: Array, default: () => [] },
  knowledge: Boolean
})
const sources = computed(() => {
  if (!props.knowledge) return props.sources.filter(source => !source.doc_uuid)

  // 同一文档的多个命中分块合并为一条参考文档。
  const documents = new Map()
  for (const match of props.sources) {
    if (!match.doc_uuid) continue
    const document = documents.get(match.doc_uuid)
    if (document) document.matches.push(match)
    else documents.set(match.doc_uuid, { doc_uuid: match.doc_uuid, title: match.title, matches: [match] })
  }
  return [...documents.values()]
})
const knowledgeAPI = props.knowledge ? useKnowledge() : null

const expanded = ref(false)
const trigger = ref(null)
const previewLoading = ref(false)
const previewVisible = ref(false)
const previewDocument = ref(null)
const previewSegments = ref([])
const previewBody = ref(null)
const showFirstHighlight = () => {
  previewBody.value?.querySelector('mark')?.scrollIntoView({ block: 'nearest' })
}
const openDocument = async (source) => {
  if (previewLoading.value) return
  previewLoading.value = true
  try {
    const chunkNumbers = [...new Set(source.matches.map(match => match.chunk_number).filter(number => Number.isInteger(number) && number > 0))]
    const preview = await knowledgeAPI.getDocumentPreview(source.doc_uuid, chunkNumbers)
    if (!preview) {
      ElMessage.error('无法读取参考文档，文档可能已被删除')
      return
    }
    previewDocument.value = source
    previewSegments.value = preview.segments?.length ? preview.segments : [{ text: preview.content }]
    previewVisible.value = true
  } finally {
    previewLoading.value = false
  }
}
const listId = `search-sources-${useId()}`
const collapse = () => {
  expanded.value = false
  trigger.value?.focus({ preventScroll: true })
}
const getDomain = (url) => {
  try { return new URL(url).hostname.replace(/^www\./, '') }
  catch { return url || '网页来源' }
}
</script>

<style scoped>
.search-sources { max-width: 100%; }
.source-chevron { flex-shrink: 0; transition: transform .4s cubic-bezier(.22, 1, .36, 1); }
.source-chevron.expanded { transform: rotate(180deg); }
.source-dropdown-motion { display: grid; grid-template-rows: 1fr; width: 720px; max-width: 100%; margin-bottom: 18px; }
.source-dropdown-clip { min-height: 0; overflow: hidden; }
.source-dropdown { width: 100%; overflow: hidden; border: 1px solid #e5e9ef; border-radius: 14px; background: #fff; box-shadow: 0 6px 24px #24344d06; }
.source-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 13px 16px; border-bottom: 1px solid #f0f2f5; color: #4b5563; font-size: 12px; font-weight: 500; }
.source-count { color: #9ca3af; font-size: 11px; font-weight: 400; }
.source-list { max-height: 340px; margin: 0; padding: 6px; list-style: none; overflow-y: auto; overscroll-behavior: contain; }
.source-list li + li { border-top: 1px solid #f3f4f6; }
.source-link { display: flex; align-items: center; gap: 12px; padding: 12px 10px; border-radius: 9px; color: inherit; text-decoration: none; transition: background .18s; }
.source-link:hover { background: #f5f8fc; }
.source-link:focus-visible { outline: 2px solid #0073ff; outline-offset: -2px; }
.source-number { display: grid; place-items: center; width: 28px; height: 28px; flex-shrink: 0; border-radius: 8px; background: #f3f6fa; color: #8596ad; font-size: 11px; font-weight: 500; font-variant-numeric: tabular-nums; }
.source-copy { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 5px; }
.source-title { display: -webkit-box; overflow: hidden; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow-wrap: anywhere; color: #374151; font-size: 13px; line-height: 1.5; transition: color .18s; }
.source-link:hover .source-title { color: #0073ff; }
.source-domain { display: flex; align-items: center; gap: 5px; color: #9ca3af; font-size: 11px; overflow-wrap: anywhere; }
.source-domain .iconify { flex-shrink: 0; }
.source-external { flex-shrink: 0; color: #b1bac7; font-size: 14px; }
.source-link:hover .source-external { color: #649bde; }
.source-dropdown-enter-active { transition: grid-template-rows .36s cubic-bezier(.22, 1, .36, 1), opacity .3s ease, margin-bottom .36s ease; }
.source-dropdown-leave-active { transition: grid-template-rows .44s cubic-bezier(.4, 0, .2, 1), opacity .38s ease, margin-bottom .44s cubic-bezier(.4, 0, .2, 1); }
.source-dropdown-enter-from, .source-dropdown-leave-to { grid-template-rows: 0fr; opacity: 0; margin-bottom: 0; }
@media (prefers-reduced-motion: reduce) {
  .source-dropdown-enter-active, .source-dropdown-leave-active, .source-chevron { transition: none; }
}
</style>
