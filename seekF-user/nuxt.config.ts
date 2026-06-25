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
})