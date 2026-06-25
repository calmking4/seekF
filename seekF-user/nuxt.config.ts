export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: [
    '@nuxtjs/tailwindcss',
    'motion-v/nuxt',
    '@element-plus/nuxt',
    '@nuxt/icon',
  ],
  css: [
    '~/assets/css/scrollbar.css'
  ],
  runtimeConfig:{
    public:{
      apiBase:"http://localhost:8080/",
      wsBase:"ws://localhost:8080/",
    }
  },
  // SEO 全局配置
  app: {
    head: {
      titleTemplate: '%s - seekF',
      // 默认标题
      title: '寻找乐趣',
      htmlAttrs: {
        lang: 'zh-CN',
      },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        // 默认描述
        { name: 'description', content: 'seekF 是一个集成了 AI 能力的即时通讯平台，支持智能对话、知识库管理、社交发现等功能。' },
        // Open Graph（分享到社交媒体时显示）
        { property: 'og:title', content: 'seekF - AI 智能社交平台' },
        { property: 'og:description', content: '集成了 AI 能力的即时通讯平台，支持智能对话、知识库管理、社交发现。' },
        { property: 'og:type', content: 'website' },
        { property: 'og:locale', content: 'zh_CN' },
        // 主题色（移动端浏览器状态栏）
        { name: 'theme-color', content: '#409eff' },
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
      ],
    },
  },
})