const DRAFT_KEY = 'seekf_discover_create_draft'

function emptyDraft() {
  return {
    mediaType: 'image',
    title: '',
    content: '',
    tags: [],
    coverUrl: '',
    video: null,
    images: [],
  }
}

export function loadDiscoverDraft() {
  if (typeof sessionStorage === 'undefined') return null
  try {
    const raw = sessionStorage.getItem(DRAFT_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function saveDiscoverDraft(draft) {
  if (typeof sessionStorage === 'undefined') return
  sessionStorage.setItem(DRAFT_KEY, JSON.stringify(draft))
}

export function clearDiscoverDraft() {
  if (typeof sessionStorage === 'undefined') return
  sessionStorage.removeItem(DRAFT_KEY)
}

/**
 * 从页面状态构建草稿
 */
export function buildDiscoverDraft({
  mediaType,
  title,
  content,
  tags,
  coverUrl,
  mediaList,
}) {
  const draft = {
    mediaType,
    title,
    content,
    tags: [...tags],
    coverUrl: coverUrl || '',
    video: null,
    images: [],
  }

  if (mediaType === 'video') {
    const media = mediaList[0]
    if (media) {
      draft.video = {
        uploadStatus: media.uploadStatus,
        uploadedUrl: media.uploadedUrl || '',
        objectKey: media.objectKey || '',
        posterUrl: media.posterUrl || '',
        previewUrl: media.posterUrl || (media.uploadStatus === 'done'
          ? (media.uploadedUrl || media.url)
          : (media.url?.startsWith('blob:') ? '' : media.url)),
        fileName: media.file?.name || media.fileName || '',
        fileSize: media.file?.size || media.fileSize || 0,
        lastModified: media.file?.lastModified || media.lastModified || 0,
        uploadProgress: media.uploadProgress || 0,
        uploadStatusText: media.uploadStatusText || '',
        fingerprint: media.fingerprint || '',
        needsReselect: !media.file && media.uploadStatus !== 'done',
      }
    }
  } else {
    draft.images = mediaList
      .filter(m => m.type === 'image')
      .map(m => ({
        previewUrl: m.url?.startsWith('blob:') ? '' : m.url,
        uploadedUrl: m.uploadedUrl || '',
        fileName: m.file?.name || '',
      }))
      .filter(m => m.previewUrl || m.uploadedUrl)
  }

  return draft
}

/**
 * 将草稿还原为页面初始状态
 */
export function applyDiscoverDraft(draft) {
  if (!draft) return null

  const result = {
    mediaType: draft.mediaType || 'image',
    title: draft.title || '',
    content: draft.content || '',
    tags: draft.tags || [],
    coverUrl: draft.coverUrl || '',
    coverPreview: draft.coverUrl || '',
    mediaList: [],
  }

  if (draft.mediaType === 'video' && draft.video) {
    const v = draft.video
    // 暂停状态刷新后清除视频，不需要恢复
    if (v.uploadStatus === 'paused') {
      result.mediaList = []
      return result
    }
    const preview = v.previewUrl || v.uploadedUrl || ''
    const needsReselect = v.needsReselect
    result.mediaList = [{
      type: 'video',
      url: preview,
      file: null,
      fileName: v.fileName,
      fileSize: v.fileSize,
      lastModified: v.lastModified,
      fingerprint: v.fingerprint,
      uploadStatus: needsReselect ? 'pending_reselect' : v.uploadStatus,
      uploadProgress: v.uploadProgress || (v.uploadStatus === 'done' ? 100 : 0),
      uploadStatusText: needsReselect
        ? '请选择视频文件'
        : (v.uploadStatusText || ''),
      uploadedUrl: v.uploadedUrl || '',
      objectKey: v.objectKey || '',
      posterUrl: v.posterUrl || '',
    }]
  } else if (draft.images?.length) {
    result.mediaList = draft.images.map(img => ({
      type: 'image',
      url: img.previewUrl || img.uploadedUrl,
      uploadedUrl: img.uploadedUrl || '',
      file: null,
    }))
  }

  return result
}

export { emptyDraft, DRAFT_KEY }
