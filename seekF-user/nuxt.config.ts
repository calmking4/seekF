export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: [
    '@nuxtjs/tailwindcss',
    'motion-v/nuxt',
    '@element-plus/nuxt',
    '@nuxt/icon',
  ],
  runtimeConfig:{
    public:{
      apiBase:"http://127.0.0.1:8080/"
    }
  },
})