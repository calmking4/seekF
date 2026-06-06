/**
 * 从视频文件或 URL 截取首帧作为封面
 * @param {File|Blob|string} source - 本地文件或视频 URL
 * @returns {Promise<string>} 封面 blob URL
 */
export function extractVideoPoster(source) {
  return new Promise((resolve, reject) => {
    if (typeof document === 'undefined') {
      reject(new Error('仅支持浏览器环境'))
      return
    }

    const video = document.createElement('video')
    video.muted = true
    video.playsInline = true
    video.preload = 'auto'

    const isLocal = source instanceof File || source instanceof Blob
    let objectUrl = ''

    if (isLocal) {
      objectUrl = URL.createObjectURL(source)
      video.src = objectUrl
    } else {
      video.crossOrigin = 'anonymous'
      video.src = source
    }

    const cleanup = () => {
      if (objectUrl) URL.revokeObjectURL(objectUrl)
      video.src = ''
      video.load()
    }

    const capture = () => {
      const w = video.videoWidth
      const h = video.videoHeight
      if (!w || !h) {
        cleanup()
        reject(new Error('无法读取视频尺寸'))
        return
      }

      try {
        const canvas = document.createElement('canvas')
        const scale = Math.min(1, 960 / w)
        canvas.width = Math.round(w * scale)
        canvas.height = Math.round(h * scale)
        canvas.getContext('2d').drawImage(video, 0, 0, canvas.width, canvas.height)
        canvas.toBlob((blob) => {
          cleanup()
          if (blob) {
            resolve(URL.createObjectURL(blob))
          } else {
            reject(new Error('生成封面失败'))
          }
        }, 'image/jpeg', 0.82)
      } catch (e) {
        cleanup()
        reject(e)
      }
    }

    video.addEventListener('loadeddata', () => {
      const t = video.duration && Number.isFinite(video.duration)
        ? Math.min(0.1, video.duration / 2)
        : 0.1
      video.currentTime = t
    })

    video.addEventListener('seeked', capture, { once: true })

    video.addEventListener('error', () => {
      cleanup()
      reject(new Error('视频加载失败'))
    }, { once: true })
  })
}

/** blob/data URL 互转，供草稿持久化 */
export async function posterUrlToDataUrl(posterUrl) {
  if (!posterUrl || posterUrl.startsWith('data:')) return posterUrl
  if (!posterUrl.startsWith('blob:')) return posterUrl
  const res = await fetch(posterUrl)
  const blob = await res.blob()
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = reject
    reader.readAsDataURL(blob)
  })
}

export function revokePosterUrl(posterUrl) {
  if (posterUrl?.startsWith('blob:')) {
    URL.revokeObjectURL(posterUrl)
  }
}
