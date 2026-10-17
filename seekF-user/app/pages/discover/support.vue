<template>
  <div class="mx-auto max-w-[1180px] space-y-6 pb-8 sm:pt-2">
    <header class="flex flex-wrap items-center justify-between gap-3 px-1">
      <div>
        <h1 class="text-xl font-semibold tracking-tight text-gray-800">创作助力</h1>
        <p class="mt-1.5 text-sm text-gray-400">连接喜欢你的人，让创作获得更多支持</p>
      </div>
      <span v-if="!loading && !error" class="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium" :class="settings?.activated ? 'bg-emerald-50 text-emerald-600' : 'bg-white text-gray-500 ring-1 ring-gray-200'">
        <span class="h-1.5 w-1.5 rounded-full" :class="settings?.activated ? 'bg-emerald-500' : 'bg-gray-400'" />
        {{ settings?.activated ? '已开通助力' : '尚未开通' }}
      </span>
    </header>

    <section class="relative isolate overflow-hidden rounded-3xl border border-blue-100/70 bg-gradient-to-br from-[#edf5ff] via-[#f5f9ff] to-white px-6 py-8 sm:px-9 lg:flex lg:min-h-[280px] lg:items-center lg:py-10">
      <div class="relative z-10 lg:w-[58%]">
        <p class="mb-4 flex items-center gap-2 text-xs font-medium tracking-[0.16em] text-blue-500"><Icon name="mdi:hand-heart-outline" class="text-base" /> 为热爱，添一份支持</p>
        <h2 class="text-[26px] font-semibold leading-snug tracking-tight text-slate-800 sm:text-[32px]">让好内容，<br class="sm:hidden" />收获更多同行者</h2>
        <p class="mt-4 max-w-md text-sm leading-7 text-slate-500">为支持你的用户带来专属图文与视频，<br class="hidden sm:block" />把每一份认可，变成持续创作的动力。</p>
        <div class="mt-6 flex flex-wrap gap-x-5 gap-y-2 text-xs text-slate-500">
          <span class="inline-flex items-center gap-1.5"><Icon name="lucide:circle-check" class="text-sm text-blue-400" /> 专属图文与视频</span>
          <span class="inline-flex items-center gap-1.5"><Icon name="lucide:circle-check" class="text-sm text-blue-400" /> 30天观看权益</span>
        </div>
      </div>
      <img src="/images/creator-support-hero.png" alt="" width="1536" height="1024" fetchpriority="low" class="mx-auto -mb-6 mt-1 w-full max-w-[340px] mix-blend-multiply sm:max-w-[380px] lg:absolute lg:-right-2 lg:bottom-0 lg:mb-0 lg:mt-0 lg:w-[44%] lg:max-w-[440px]" />
    </section>

    <section v-if="loading" class="flex min-h-[300px] items-center justify-center gap-2 rounded-2xl border border-gray-100 bg-white text-sm text-gray-400" role="status" aria-live="polite"><Icon name="svg-spinners:180-ring" class="text-lg text-blue-400" /> 正在加载助力设置…</section>
    <section v-else-if="error" class="flex min-h-[300px] flex-col items-center justify-center gap-4 rounded-2xl border border-gray-100 bg-white px-5 text-center" role="alert">
      <span class="flex h-12 w-12 items-center justify-center rounded-2xl bg-blue-50 text-blue-400"><Icon name="lucide:cloud-off" class="text-2xl" /></span>
      <p class="text-sm text-gray-500">{{ error }}</p>
      <el-button plain @click="loadSettings">重新加载</el-button>
    </section>
    <div v-else class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
      <form class="rounded-2xl border border-gray-100 bg-white p-6 sm:p-8" @submit.prevent="saveSettings">
        <div class="flex items-start gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-blue-500"><Icon name="lucide:sliders-horizontal" class="text-xl" /></span>
          <div><h2 class="font-semibold text-gray-800">助力设置</h2><p class="mt-1 text-xs leading-5 text-gray-400">为你的专属内容，设定一份合适的心意</p></div>
        </div>

        <div class="mt-8">
          <label for="support-price" class="block text-sm font-medium text-gray-700">30天助力价格</label>
          <div class="mt-3 flex flex-wrap items-center gap-3">
            <el-input-number id="support-price" v-model="priceYuan" :min="5" :max="100" :step="1" :precision="0" :disabled="saving" size="large" class="!w-[220px] max-w-full [&_.el-input__inner]:!text-xl [&_.el-input__inner]:!font-semibold" aria-label="30天助力价格，5至100整数元" />
            <span class="shrink-0 text-sm text-gray-500">元 / 30天</span>
          </div>
          <p class="mt-3 text-xs leading-5 text-gray-400">支持5～100元，仅限整数金额</p>
        </div>

        <div class="mt-7 flex items-start gap-2 rounded-xl bg-slate-50 px-4 py-3 text-xs leading-6 text-slate-500"><Icon name="lucide:info" class="mt-1 shrink-0 text-sm text-slate-400" /><p>{{ settings?.activated ? '改价仅影响新订单，已支付订单的价格和权益保持不变。' : '开通后即可在发布图文或视频时选择「助力专属」，让支持者观看你的专属内容。' }}</p></div>
        <div class="mt-7 flex flex-wrap items-center gap-4 border-t border-gray-100 pt-6">
          <el-button type="primary" native-type="submit" size="large" class="!min-w-[140px] !rounded-xl !font-medium" :loading="saving" :disabled="saving || !sameUser()">{{ settings?.activated ? '保存价格' : '开通助力' }}</el-button>
          <NuxtLink v-if="settings?.activated" to="/discover/create" class="inline-flex items-center gap-1 text-sm text-blue-500 hover:text-blue-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-blue-400">去发布作品<Icon name="lucide:arrow-up-right" class="text-base" /></NuxtLink>
          <span v-else class="text-xs text-gray-400">用专属内容，回应每一份支持</span>
        </div>
      </form>

      <aside class="space-y-3">
        <p class="px-1 text-xs font-medium text-gray-400">支持者将获得</p>
        <div class="relative overflow-hidden rounded-2xl border border-blue-100 bg-gradient-to-br from-white to-blue-50 p-6">
          <div class="flex items-center justify-between gap-3"><span class="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500 text-white shadow-sm shadow-blue-100"><Icon name="mdi:hand-heart-outline" class="text-[22px]" /></span><span class="rounded-full bg-blue-100/70 px-2.5 py-1 text-[11px] font-medium text-blue-500">创作者专属</span></div>
          <h2 class="mt-5 text-base font-semibold text-slate-700">你的助力方案</h2>
          <div class="mt-3 flex items-baseline gap-1 text-blue-500" aria-live="polite" aria-atomic="true"><span class="text-xl font-medium">¥</span><span class="text-[46px] font-semibold leading-tight tracking-tight">{{ previewPrice }}</span><span class="ml-1 text-xs font-normal text-slate-400">/ 30天</span></div>
          <div class="my-5 h-px bg-blue-100" />
          <ul class="space-y-3 text-[13px] leading-6 text-slate-600">
            <li class="flex items-start gap-2"><Icon name="lucide:check" class="mt-1 shrink-0 text-base text-blue-400" /> 观看你的全部助力专属图文与视频</li>
            <li class="flex items-start gap-2"><Icon name="lucide:check" class="mt-1 shrink-0 text-base text-blue-400" /> 有效期内新发布的助力作品也可看</li>
            <li class="flex items-start gap-2"><Icon name="lucide:check" class="mt-1 shrink-0 text-base text-blue-400" /> 一次助力，享有30天观看权益</li>
          </ul>
          <p class="mt-6 text-[11px] leading-5 text-slate-400">方案预览 · 以保存后的价格为准</p>
        </div>
      </aside>
    </div>

    <section class="rounded-2xl border border-gray-100 bg-white p-6 sm:p-7">
      <h2 class="flex items-center gap-2 text-sm font-medium text-gray-700"><Icon name="lucide:book-open" class="text-base text-gray-400" /> 开通前，了解这些</h2>
      <div class="mt-5 grid gap-5 md:grid-cols-3 md:gap-7">
        <div v-for="(note, index) in supportNotes" :key="note.title" class="flex items-start gap-3">
          <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-slate-50 text-[11px] font-medium text-slate-400">0{{ index + 1 }}</span>
          <div><h3 class="text-[13px] font-medium leading-6 text-gray-600">{{ note.title }}</h3><p class="mt-1 text-xs leading-6 text-gray-400">{{ note.content }}</p></div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
definePageMeta({ layout: 'creator', keepalive: false, pageTransition: { name: 'creator-page', mode: 'out-in' } })
useSeoMeta({ title: '创作助力' })
const auth = useAuthState()
const ownerId = auth.getUser()?.uuid || auth.getUser()?.Uuid || ''
const settings = ref(null)
const priceYuan = ref(10)
const previewPrice = computed(() => Number.isInteger(priceYuan.value) && priceYuan.value >= 5 && priceYuan.value <= 100 ? priceYuan.value : '—')
const supportNotes = [
  { title: '发布时选择观看方式', content: '开通后，图文和视频均可设置为助力专属，让支持者观看你的专属作品。' },
  { title: '作品属性发布后固定', content: '作品发布后，不能在免费与助力专属之间切换，请在发布前确认。' },
  { title: '按30天计算权益', content: '每次助力提供连续30天观看权益，到期后由用户自主购买，不自动续费。' },
]
const loading = ref(true)
const saving = ref(false)
const error = ref('')
let disposed = false
const sameUser = () => !!ownerId && !disposed && ownerId === (auth.getUser()?.uuid || auth.getUser()?.Uuid || '')
const loadSettings = async () => {
  loading.value = true; error.value = ''
  try {
    const response = await useApi$('/user/support/creator/info', { body: {}, retry: 0 })
    if (!sameUser()) return
    if (response.code !== 200 || !response.data) throw new Error(response.message || '获取助力设置失败')
    settings.value = response.data; priceYuan.value = response.data.price_yuan
  } catch (err) { if (sameUser()) error.value = err?.data?.message || err.message || '获取助力设置失败' }
  finally { if (sameUser()) loading.value = false }
}
const saveSettings = async () => {
  if (saving.value || !sameUser()) return
  if (!Number.isInteger(priceYuan.value) || priceYuan.value < 5 || priceYuan.value > 100) { ElMessage.warning('助力价格必须为5至100整数元'); return }
  saving.value = true
  try {
    const response = await useApi$('/user/support/creator/save', { body: { price_yuan: priceYuan.value }, retry: 0 })
    if (!sameUser()) return
    if (response.code !== 200 || !response.data) throw new Error(response.message || '保存助力设置失败')
    settings.value = response.data; priceYuan.value = response.data.price_yuan
    ElMessage.success(response.message || '助力设置已保存')
  } catch (err) { if (sameUser()) ElMessage.error(err?.data?.message || err.message || '保存助力设置失败') }
  finally { if (sameUser()) saving.value = false }
}
onMounted(loadSettings)
onUnmounted(() => { disposed = true })
</script>
