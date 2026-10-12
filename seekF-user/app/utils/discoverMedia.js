// 在浏览器读取媒体尺寸；读取失败或旧草稿缺少信息时使用未知尺寸。
export function readDiscoverMediaSize(source, type = 'image') {
  if (!source || typeof document === 'undefined') return Promise.resolve({ width: 0, height: 0 })
  return new Promise(resolve => {
    const element = type === 'video' ? document.createElement('video') : new Image()
    let timer
    const finish = (width = 0, height = 0) => {
      clearTimeout(timer)
      element.onload = null
      element.onloadedmetadata = null
      element.onerror = null
      if (type === 'video') { element.removeAttribute('src'); element.load() }
      resolve({ width, height })
    }
    if (type === 'video') {
      element.preload = 'metadata'
      element.onloadedmetadata = () => finish(element.videoWidth, element.videoHeight)
    } else element.onload = () => finish(element.naturalWidth, element.naturalHeight)
    element.onerror = () => finish()
    timer = setTimeout(() => finish(), 5000)
    element.src = source
  })
}
