// 发现页媒体始终通过后端鉴权读取，原始OSS地址只作为资源标识。
export function discoverMediaURL(value, apiBase) {
 if (!value || typeof value !== 'string' || value.startsWith('blob:') || value.startsWith('data:')) return value
 try {
  let source = value
  let parsed = new URL(source, 'http://resource.local/')
  // 将历史草稿中的旧媒体代理地址重新绑定到当前入口，避免继续绕过 Nginx。
  if (parsed.pathname.endsWith('/user/file/view')) {
   source = parsed.searchParams.get('url')
   if (!source) return value
   parsed = new URL(source, 'http://resource.local/')
  }
  if (!/^\/discover\/(images|videos)\//.test(parsed.pathname)) return value
  return String(apiBase || '').replace(/\/$/, '') + '/user/file/view?url=' + encodeURIComponent(source)
 } catch { return value }
}

export function mapDiscoverMediaResponse(data, apiBase) {
 if (!data || typeof data !== 'object') return data
 for (const [key, value] of Object.entries(data)) {
  if (['url', 'first_url', 'cover_url', 'post_cover'].includes(key) && typeof value === 'string') data[key] = discoverMediaURL(value, apiBase)
  else if (key === 'urls' && Array.isArray(value)) data[key] = value.map(item => discoverMediaURL(item, apiBase))
  else if (value && typeof value === 'object') mapDiscoverMediaResponse(value, apiBase)
 }
 return data
}
