const DRAFT_KEY = 'seekf_discover_drafts_v2'

// 用户身份参与存储键，禁止跨账号恢复草稿。
const draftUser = () => {
 const user = useAuthState().getUser()
 return user?.uuid || user?.Uuid || ''
}
const draftStorageKey = () => draftUser() ? DRAFT_KEY + ':' + draftUser() : ''
const activeKey = () => draftStorageKey() + ':active'
const newId = () => globalThis.crypto?.randomUUID?.() || Date.now().toString(36) + Math.random().toString(36).slice(2)

export function emptyDraft() {
 return { id: newId(), mediaType: 'image', title: '', content: '', tags: [], coverUrl: '', visibility: 0, requestKey: '', publishPayload: '', video: null, images: [] }
}

// 空白工作台不算草稿，兼容过滤旧版本自动生成的空记录。
export function hasDiscoverDraftContent(draft) {
 if (!draft || typeof draft !== 'object') return false
 const mediaHasContent = media => !!(media && (media.uploadedUrl || media.fileName || media.fingerprint))
 return !!(String(draft.title || '').trim() || String(draft.content || '').trim() || draft.coverUrl || draft.postUuid || (Array.isArray(draft.tags) && draft.tags.some(tag => String(tag).trim())) || (Array.isArray(draft.images) && draft.images.some(mediaHasContent)) || mediaHasContent(draft.video))
}

export function listDiscoverDrafts() {
 if (typeof localStorage === 'undefined' || !draftStorageKey()) return []
 try { const saved = JSON.parse(localStorage.getItem(draftStorageKey()) || '[]'); return Array.isArray(saved) ? saved.filter(hasDiscoverDraftContent) : [] } catch { return [] }
}

export function setActiveDiscoverDraft(id) {
 if (typeof localStorage !== 'undefined' && draftStorageKey()) localStorage.setItem(activeKey(), id)
}

export function loadDiscoverDraft(id) {
 if (typeof localStorage === 'undefined' || !draftStorageKey()) return null
 const selected = id || localStorage.getItem(activeKey())
 return listDiscoverDrafts().find(draft => draft.id === selected) || null
}

export function saveDiscoverDraft(draft) {
 if (typeof localStorage === 'undefined' || !draftStorageKey()) throw new Error('请登录后保存草稿')
 if (!hasDiscoverDraftContent(draft)) {
  // 清空已有草稿时移除自己的空记录，避免恢复出清空前的旧内容。
  if (draft?.id && listDiscoverDrafts().some(item => item.id === draft.id)) clearDiscoverDraft(draft.id)
  return null
 }
 const saved = { ...draft, id: draft.id || newId(), updatedAt: Date.now() }
 const list = listDiscoverDrafts().filter(item => item.id !== saved.id)
 localStorage.setItem(draftStorageKey(), JSON.stringify([saved, ...list]))
 setActiveDiscoverDraft(saved.id)
 return saved
}

export function clearDiscoverDraft(id) {
 if (typeof localStorage === 'undefined' || !draftStorageKey()) return
 const selected = id || localStorage.getItem(activeKey())
 localStorage.setItem(draftStorageKey(), JSON.stringify(listDiscoverDrafts().filter(item => item.id !== selected)))
 if (localStorage.getItem(activeKey()) === selected) localStorage.removeItem(activeKey())
}

// 草稿只保存可恢复的远程地址和文件元信息，blob地址无法跨页面恢复。
export function buildDiscoverDraft({ mediaType, title, content, tags, coverUrl, mediaList, visibility = 0, requestKey = '', publishPayload = '' }) {
 const safeURL = value => typeof value === 'string' && !value.startsWith('blob:') ? value : ''
 const serialize = media => ({
  width: media.width || 0, height: media.height || 0,
  previewUrl: safeURL(media.posterUrl || media.uploadedUrl),
  uploadedUrl: media.uploadedUrl || '', objectKey: media.objectKey || '',
  fileName: media.file?.name || media.fileName || '', fileSize: media.file?.size || media.fileSize || 0,
  lastModified: media.file?.lastModified || media.lastModified || 0,
  uploadProgress: media.uploadProgress || 0, fingerprint: media.fingerprint || '',
  uploadStatus: media.uploadedUrl ? 'done' : 'pending_reselect',
  needsReselect: !media.uploadedUrl, posterUrl: safeURL(media.posterUrl),
 })
 return { mediaType, title, content, tags: [...tags], visibility, requestKey, publishPayload, coverUrl: safeURL(coverUrl),
  video: mediaType === 'video' && mediaList[0] ? serialize(mediaList[0]) : null,
  images: mediaType === 'image' ? mediaList.map(serialize) : [],
 }
}

export function applyDiscoverDraft(draft) {
 if (!draft) return null
 const rows = draft.mediaType === 'video' ? (draft.video ? [draft.video] : []) : (draft.images || [])
 return { mediaType: draft.mediaType || 'image', title: draft.title || '', content: draft.content || '', tags: draft.tags || [],
  visibility: draft.visibility || 0, coverUrl: draft.coverUrl || '', coverPreview: draft.coverUrl || '',
  mediaList: rows.map(media => ({ ...media, type: draft.mediaType === 'video' ? 'video' : 'image', file: null,
   url: media.previewUrl || media.uploadedUrl || '', uploadStatus: media.uploadedUrl ? 'done' : 'pending_reselect',
   uploadStatusText: media.uploadedUrl ? '上传完成' : '请重新选择文件', uploadProgress: media.uploadedUrl ? 100 : (media.uploadProgress || 0),
  })),
 }
}

export { DRAFT_KEY }
