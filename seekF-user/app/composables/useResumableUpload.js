const CHUNK_SIZE = 5 * 1024 * 1024
const STORAGE_PREFIX = 'seekf_video_upload_'

/**
 * 暂停上传错误类型
 * 用于区分暂停和取消/错误
 */
export class UploadPausedError extends Error {
  constructor() {
    super('上传已暂停')
    this.name = 'UploadPausedError'
  }
}

/**
 * 生成文件指纹，用于断点续传识别同一文件
 */
function getFileFingerprint(file) {
  return `${file.name}_${file.size}_${file.lastModified}`
}

function getStorageKey(fingerprint) {
  return STORAGE_PREFIX + fingerprint
}

function loadUploadState(fingerprint) {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(getStorageKey(fingerprint))
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function saveUploadState(fingerprint, state) {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(getStorageKey(fingerprint), JSON.stringify(state))
}

function clearUploadState(fingerprint) {
  if (typeof localStorage === 'undefined') return
  localStorage.removeItem(getStorageKey(fingerprint))
}

function calcTotalParts(fileSize, partSize) {
  return Math.ceil(fileSize / partSize)
}

function calcProgress(uploadedParts, fileSize) {
  const uploadedBytes = uploadedParts.reduce((sum, p) => sum + (p.size || 0), 0)
  return Math.min(100, Math.round((uploadedBytes / fileSize) * 100))
}

function fillPartSizes(parts, fileSize, partSize) {
  const totalParts = calcTotalParts(fileSize, partSize)
  return parts.map((p) => {
    if (p.size) return p
    const isLast = p.partNumber === totalParts
    const size = isLast
      ? fileSize - (totalParts - 1) * partSize
      : partSize
    return { ...p, size }
  })
}

/**
 * 视频断点续传上传
 * @param {File} file - 视频文件
 * @param {string} fileType - 业务类型，如 discover_video / message_video
 * @param {object} callbacks
 * @param {function(number): void} callbacks.onProgress - 进度 0-100
 * @param {function(string): void} [callbacks.onStatus] - 状态文案
 * @param {AbortSignal} [callbacks.signal] - 取消信号
 * @param {{paused: boolean}} [callbacks.pauseSignal] - 暂停信号对象，设置 paused=true 暂停上传
 * @returns {Promise<{url: string, objectKey: string}>}
 */
export async function uploadVideoResumable(file, fileType, callbacks = {}) {
  const { onProgress, onStatus, signal, pauseSignal } = callbacks
  const config = useRuntimeConfig()
  const apiBase = (config.public.apiBase || 'http://localhost:8080/').replace(/\/$/, '')

  const fingerprint = getFileFingerprint(file)
  let state = loadUploadState(fingerprint)
  let partSize = CHUNK_SIZE

  const notifyProgress = (parts) => {
    onProgress?.(calcProgress(parts, file.size))
  }

  // 尝试恢复已有上传任务
  if (state?.uploadId && state?.objectKey) {
    onStatus?.('正在恢复上传...')
    onProgress?.(1)
    try {
      const statusRes = await $fetch(`${apiBase}/user/file/upload/status`, {
        method: 'POST',
        credentials: 'include',
        body: {
          upload_id: state.uploadId,
          object_key: state.objectKey,
        },
      })
      if (statusRes.code === 200 && statusRes.data?.parts) {
        partSize = state.partSize || CHUNK_SIZE
        state.parts = fillPartSizes(
          statusRes.data.parts.map((p) => ({
            partNumber: p.partNumber,
            etag: p.etag,
            size: p.size || 0,
          })),
          file.size,
          partSize,
        )
        saveUploadState(fingerprint, state)
        notifyProgress(state.parts)
      }
    } catch {
      state = null
    }
  }

  // 初始化新任务
  if (!state?.uploadId) {
    onStatus?.('正在初始化上传...')
    onProgress?.(1)
    const initRes = await $fetch(`${apiBase}/user/file/upload/init`, {
      method: 'POST',
      credentials: 'include',
      body: {
        file_name: file.name,
        file_type: fileType,
        file_size: file.size,
        content_type: file.type || 'video/mp4',
      },
      signal,
    })
    if (initRes.code !== 200 || !initRes.data?.uploadId) {
      throw new Error(initRes.message || '初始化上传失败')
    }
    partSize = initRes.data.partSize || CHUNK_SIZE
    state = {
      fingerprint,
      uploadId: initRes.data.uploadId,
      objectKey: initRes.data.objectKey,
      url: initRes.data.url,
      parts: [],
      fileName: file.name,
      fileType,
      fileSize: file.size,
      partSize,
    }
    saveUploadState(fingerprint, state)
  } else {
    partSize = state.partSize || CHUNK_SIZE
  }

  const totalParts = calcTotalParts(file.size, partSize)
  const uploadedPartNumbers = new Set((state.parts || []).map((p) => p.partNumber))

  // 上传缺失分片
  for (let partNumber = 1; partNumber <= totalParts; partNumber++) {
    // 检查取消信号
    if (signal?.aborted) {
      throw new DOMException('上传已取消', 'AbortError')
    }
    // 检查暂停信号
    if (pauseSignal?.paused) {
      onStatus?.('上传已暂停')
      throw new UploadPausedError()
    }
    if (uploadedPartNumbers.has(partNumber)) {
      continue
    }

    const start = (partNumber - 1) * partSize
    const end = Math.min(start + partSize, file.size)
    const chunk = file.slice(start, end)
    const chunkSize = end - start

    onStatus?.(`正在上传 ${partNumber}/${totalParts} 分片...`)

    // 分片开始前先更新进度（避免长时间停在 0%）
    const startPercent = Math.round(((partNumber - 1) / totalParts) * 100)
    onProgress?.(Math.max(startPercent, 1))

    const formData = new FormData()
    formData.append('file', chunk, `${file.name}.part${partNumber}`)
    formData.append('uploadId', state.uploadId)
    formData.append('objectKey', state.objectKey)
    formData.append('partNumber', String(partNumber))

    const chunkRes = await $fetch(`${apiBase}/user/file/upload/chunk`, {
      method: 'POST',
      credentials: 'include',
      body: formData,
      signal,
    })

    if (chunkRes.code !== 200 || !chunkRes.data?.etag) {
      throw new Error(chunkRes.message || `分片 ${partNumber} 上传失败`)
    }

    state.parts.push({
      partNumber: chunkRes.data.partNumber,
      etag: chunkRes.data.etag,
      size: chunkSize,
    })
    uploadedPartNumbers.add(partNumber)
    saveUploadState(fingerprint, state)
    notifyProgress(state.parts)
  }

  // 合并分片
  onStatus?.('正在合并文件...')
  onProgress?.(99)

  const completeRes = await $fetch(`${apiBase}/user/file/upload/complete`, {
    method: 'POST',
    credentials: 'include',
    body: {
      upload_id: state.uploadId,
      object_key: state.objectKey,
      parts: state.parts.map((p) => ({
        part_number: p.partNumber,
        etag: p.etag,
      })),
    },
    signal,
  })

  if (completeRes.code !== 200 || !completeRes.data?.url) {
    throw new Error(completeRes.message || '合并分片失败')
  }

  clearUploadState(fingerprint)
  onProgress?.(100)
  onStatus?.('上传完成')

  return {
    url: completeRes.data.url,
    objectKey: completeRes.data.objectKey,
  }
}

/**
 * 取消并清理上传任务
 */
export async function abortVideoUpload(fingerprint) {
  const state = loadUploadState(fingerprint)
  clearUploadState(fingerprint)

  if (!state?.uploadId || !state?.objectKey) {
    return
  }

  const config = useRuntimeConfig()
  const apiBase = (config.public.apiBase || 'http://localhost:8080/').replace(/\/$/, '')
  try {
    await $fetch(`${apiBase}/user/file/upload/abort`, {
      method: 'POST',
      credentials: 'include',
      body: {
        upload_id: state.uploadId,
        object_key: state.objectKey,
      },
    })
  } catch {
    // 忽略取消失败
  }
}

/**
 * 恢复暂停的视频上传
 * 如果有未完成的任务则继续，否则开始新任务
 * @param {File} file - 视频文件
 * @param {string} fileType - 业务类型
 * @param {object} callbacks - 回调函数
 * @returns {Promise<{url: string, objectKey: string}>}
 */
export async function resumeUpload(file, fileType, callbacks = {}) {
  return uploadVideoResumable(file, fileType, callbacks)
}

export async function deleteOssFile({ objectKey, url } = {}) {
  const config = useRuntimeConfig()
  const apiBase = (config.public.apiBase || 'http://localhost:8080/').replace(/\/$/, '')

  const res = await $fetch(`${apiBase}/user/file/delete`, {
    method: 'POST',
    credentials: 'include',
    body: {
      object_key: objectKey || '',
      url: url || '',
    },
  })

  if (res.code !== 200) {
    throw new Error(res.message || '删除文件失败')
  }
}

export { getFileFingerprint, loadUploadState, CHUNK_SIZE }
