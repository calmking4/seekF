// captureChatHistoryAnchor 记录首条可见消息及其相对滚动区顶部的位置。
export function captureChatHistoryAnchor(wrap, preferredElement = null) {
  if (!wrap) return null
  const bounds = wrap.getBoundingClientRect()
  const top = bounds.top
  const scale = bounds.height && wrap.clientHeight ? bounds.height / wrap.clientHeight : 1
  const element = preferredElement && wrap.contains(preferredElement) ? preferredElement : [...wrap.querySelectorAll('[data-message-id]')].find(node => node.getBoundingClientRect().bottom > top)
  if (!element) return null
  const offset = (element.getBoundingClientRect().top - top) / scale
  return { element, offset, contentTop: wrap.scrollTop + offset }
}

// hasChatHistoryShifted 区分内容重排与用户滚动，防止尺寸变化产生的滚动事件覆盖阅读锚点。
export function hasChatHistoryShifted(wrap, anchor) {
  if (!wrap || !anchor || !wrap.contains(anchor.element)) return false
  const bounds = wrap.getBoundingClientRect()
  const scale = bounds.height && wrap.clientHeight ? bounds.height / wrap.clientHeight : 1
  const contentTop = wrap.scrollTop + (anchor.element.getBoundingClientRect().top - bounds.top) / scale
  return Math.abs(contentTop - anchor.contentTop) > 0.5
}

// restoreChatHistoryAnchor 按原消息位置补偿前插及图片加载造成的位移，不计入底部新增消息高度。
export function restoreChatHistoryAnchor(wrap, anchor) {
  if (!wrap || !anchor || !wrap.contains(anchor.element)) return
  const bounds = wrap.getBoundingClientRect()
  const scale = bounds.height && wrap.clientHeight ? bounds.height / wrap.clientHeight : 1
  const contentTop = wrap.scrollTop + (anchor.element.getBoundingClientRect().top - bounds.top) / scale
  const target = Math.max(0, contentTop - anchor.offset)
  if (Math.abs(wrap.scrollTop - target) > 0.5) wrap.scrollTo({ top: target, behavior: 'instant' })
  anchor.contentTop = contentTop
}
