<template>
    <div ref="containerRef" class="markdown-body" v-html="renderedHtml" @click="handleCopyClick"></div>
</template>

<script setup>
import { marked } from 'marked'
import hljs from 'highlight.js/lib/core'
// 按需引入常用语言，减小包体积
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import java from 'highlight.js/lib/languages/java'
import html from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import sql from 'highlight.js/lib/languages/sql'
import bash from 'highlight.js/lib/languages/bash'
import json from 'highlight.js/lib/languages/json'
import rust from 'highlight.js/lib/languages/rust'
import cpp from 'highlight.js/lib/languages/cpp'
import markdown from 'highlight.js/lib/languages/markdown'
import yaml from 'highlight.js/lib/languages/yaml'
import php from 'highlight.js/lib/languages/php'

// 注册语言
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('python', python)
hljs.registerLanguage('py', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('java', java)
hljs.registerLanguage('html', html)
hljs.registerLanguage('xml', html)
hljs.registerLanguage('css', css)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('json', json)
hljs.registerLanguage('rust', rust)
hljs.registerLanguage('cpp', cpp)
hljs.registerLanguage('c', cpp)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('md', markdown)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('yml', yaml)
hljs.registerLanguage('php', php)

const props = defineProps({
    content: {
        type: String,
        default: ''
    },
    isStreaming: {
        type: Boolean,
        default: false
    }
})

// SVG图标常量
const ICON_COPY = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>'
const ICON_CHECK = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>'
const ICON_CHEVRON_UP = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"/></svg>'
const ICON_CHEVRON_DOWN = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>'

// 自定义renderer，处理代码块高亮
const renderer = {
    code({ text, lang }) {
        let highlighted
        if (lang && hljs.getLanguage(lang)) {
            highlighted = hljs.highlight(text, { language: lang }).value
        } else {
            highlighted = hljs.highlightAuto(text).value
        }
        return '<pre class="code-block"><div class="code-header"><span class="code-lang">' + (lang || '') + '</span><div class="code-actions"><button class="toggle-btn icon-btn" data-toggle="true" title="收起">' + ICON_CHEVRON_UP + '</button><button class="copy-btn icon-btn" data-copy="true" title="复制">' + ICON_COPY + '</button></div></div><code class="hljs' + (lang ? ' language-' : '') + '">' + highlighted + '</code></pre>'
    }
}

// 事件委托处理复制和折叠
function handleCopyClick(e) {
    // 复制按钮
    const copyBtn = e.target.closest('.copy-btn')
    if (copyBtn) {
        const pre = copyBtn.closest('pre')
        const code = pre?.querySelector('code')
        if (!code) return
        navigator.clipboard.writeText(code.textContent).then(() => {
            copyBtn.innerHTML = ICON_CHECK
            copyBtn.classList.add('copied')
            copyBtn.title = '已复制'
            setTimeout(() => {
                copyBtn.innerHTML = ICON_COPY
                copyBtn.classList.remove('copied')
                copyBtn.title = '复制'
            }, 1500)
        })
        return
    }
    // 折叠按钮
    const toggleBtn = e.target.closest('.toggle-btn')
    if (toggleBtn) {
        const pre = toggleBtn.closest('pre')
        if (!pre) return
        pre.classList.toggle('collapsed')
        const isCollapsed = pre.classList.contains('collapsed')
        toggleBtn.innerHTML = isCollapsed ? ICON_CHEVRON_DOWN : ICON_CHEVRON_UP
        toggleBtn.title = isCollapsed ? '展开' : '收起'
    }
}

marked.use({ renderer })

// 配置marked
marked.setOptions({
    gfm: true,      // GitHub Flavored Markdown
    breaks: true,    // 单个换行转<br>
})

// 轻量XSS过滤
function sanitize(html) {
    return html
        .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
        .replace(/<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi, '')
        .replace(/<object\b[^<]*(?:(?!<\/object>)<[^<]*)*<\/object>/gi, '')
        .replace(/<embed\b[^<]*(?:(?!<\/embed>)<[^<]*)*<\/embed>/gi, '')
        .replace(/\son\w+\s*=\s*["'][^"']*["']/gi, '')
}

const renderedHtml = computed(() => {
    if (!props.content) return ''
    const rawHtml = marked.parse(props.content)
    return sanitize(rawHtml)
})
</script>

<style>
/* markdown-body样式，非scoped以作用于v-html内容 */
.markdown-body {
    font-size: 0.875rem;
    line-height: 1.6;
    word-break: break-word;
}

.markdown-body h1 {
    font-size: 1.25rem;
    font-weight: 600;
    margin-top: 0.75em;
    margin-bottom: 0.5em;
}

.markdown-body h2 {
    font-size: 1.125rem;
    font-weight: 600;
    margin-top: 0.75em;
    margin-bottom: 0.5em;
}

.markdown-body h3 {
    font-size: 1rem;
    font-weight: 600;
    margin-top: 0.75em;
    margin-bottom: 0.5em;
}

.markdown-body h4,
.markdown-body h5,
.markdown-body h6 {
    font-size: 0.875rem;
    font-weight: 600;
    margin-top: 0.5em;
    margin-bottom: 0.25em;
}

.markdown-body p {
    margin-bottom: 0.5em;
}

.markdown-body p:last-child {
    margin-bottom: 0;
}

.markdown-body ul,
.markdown-body ol {
    padding-left: 1.5em;
    margin-bottom: 0.5em;
}

.markdown-body li {
    margin-bottom: 0.25em;
}

.markdown-body li > ul,
.markdown-body li > ol {
    margin-bottom: 0;
}

/* 行内代码 */
.markdown-body code:not(.hljs) {
    background: #f3f4f6;
    color: #e11d48;
    padding: 0.125em 0.375em;
    border-radius: 0.25rem;
    font-size: 0.85em;
    font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
}

/* 代码块 */
.markdown-body pre.code-block {
    background: #f5f5f5;
    border-radius: 0.5rem;
    overflow-x: auto;
    margin-bottom: 0.75rem;
    position: relative;
    border: 1px solid #e5e7eb;
}

.markdown-body pre.code-block .code-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.375rem 0.75rem;
    background: #ebebeb;
    border-bottom: 1px solid #e5e7eb;
    border-radius: 0.5rem 0.5rem 0 0;
}

.markdown-body pre.code-block .code-actions {
    display: flex;
    align-items: center;
    gap: 0.375rem;
}

.markdown-body pre.code-block .code-lang {
    font-size: 0.7rem;
    color: #6b7280;
    text-transform: uppercase;
}

.markdown-body pre.code-block code {
    display: block;
    background: transparent;
    padding: 0.75rem;
    font-size: 0.8em;
    color: #1f2937;
    font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
}

/* 代码块图标按钮 */
.markdown-body .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 24px;
    background: transparent;
    color: #9ca3af;
    border: none;
    border-radius: 0.25rem;
    cursor: pointer;
    transition: background 0.2s, color 0.2s;
}

.markdown-body .icon-btn:hover {
    background: rgba(0, 0, 0, 0.08);
    color: #4b5563;
}

/* 复制成功动画 */
.markdown-body .icon-btn.copied {
    color: #10b981;
    animation: copy-pop 0.3s ease;
}

@keyframes copy-pop {
    0% { transform: scale(1); }
    50% { transform: scale(1.3); }
    100% { transform: scale(1); }
}

/* 代码块折叠状态 */
.markdown-body pre.code-block.collapsed code {
    display: none;
}

.markdown-body pre.code-block.collapsed {
    border-radius: 0.5rem;
    margin-bottom: 0.75rem;
}

.markdown-body pre.code-block.collapsed .code-header {
    border-radius: 0.5rem;
    border-bottom: 1px solid #e5e7eb;
}

/* highlight.js浅色主题token颜色 */
.hljs-keyword,
.hljs-selector-tag,
.hljs-built_in,
.hljs-name { color: #7c3aed; }
.hljs-string,
.hljs-attr { color: #059669; }
.hljs-number,
.hljs-literal { color: #d97706; }
.hljs-comment,
.hljs-doctag { color: #9ca3af; font-style: italic; }
.hljs-function .hljs-title,
.hljs-title.function_ { color: #2563eb; }
.hljs-type,
.hljs-class .hljs-title { color: #0891b2; }
.hljs-variable,
.hljs-template-variable { color: #dc2626; }
.hljs-params { color: #6b7280; }
.hljs-meta { color: #7c3aed; }
.hljs-tag { color: #dc2626; }
.hljs-attribute { color: #d97706; }
.hljs-symbol,
.hljs-bullet { color: #059669; }
.hljs-regexp { color: #dc2626; }
.hljs-link { color: #2563eb; text-decoration: underline; }
.hljs-deletion { color: #dc2626; }
.hljs-addition { color: #059669; }
.hljs-emphasis { font-style: italic; }
.hljs-strong { font-weight: bold; }

/* 引用块 */
.markdown-body blockquote {
    border-left: 3px solid #d1d5db;
    padding-left: 1em;
    color: #6b7280;
    margin-bottom: 0.5em;
}

/* 表格 */
.markdown-body table {
    border-collapse: collapse;
    width: 100%;
    margin-bottom: 0.75em;
}

.markdown-body th,
.markdown-body td {
    border: 1px solid #e5e7eb;
    padding: 0.375rem 0.75rem;
    text-align: left;
}

.markdown-body th {
    background: #f9fafb;
    font-weight: 600;
}

/* 链接 */
.markdown-body a {
    color: #3b82f6;
    text-decoration: underline;
}

/* 分割线 */
.markdown-body hr {
    border: none;
    border-top: 1px solid #e5e7eb;
    margin: 0.75em 0;
}

/* 加粗 */
.markdown-body strong {
    font-weight: 600;
}

/* 图片 */
.markdown-body img {
    max-width: 100%;
    border-radius: 0.375rem;
}

/* 任务列表 */
.markdown-body input[type="checkbox"] {
    margin-right: 0.375em;
}
</style>
