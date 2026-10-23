// createChatInitialBottom 首次打开会话时跟随内容和视口尺寸到底部，用户开始操作后停止。
export function createChatInitialBottom({ wrap, content, root = wrap, isCurrent, onStop }) {
  let stopped = false
  let observer

  // dispose 清理当前会话的定位监听，防止迟到图片影响其他会话或历史阅读。
  const dispose = () => {
    if (stopped) return
    stopped = true
    observer?.disconnect()
    root.removeEventListener('wheel', onWheel)
    root.removeEventListener('pointerdown', dispose)
    root.removeEventListener('touchstart', dispose)
    root.removeEventListener('keydown', onKeydown)
    onStop?.()
  }

  // restore 图片或视口改变后直接到达当前会话的最新消息，保持首次定位无滚动动画。
  const restore = () => {
    if (stopped) return
    if (!isCurrent()) { dispose(); return }
    const top = Math.max(0, wrap.scrollHeight - wrap.clientHeight)
    if (Math.abs(wrap.scrollTop - top) > 0.5) wrap.scrollTo({ top, behavior: 'instant' })
  }

  // onWheel 上滚表示开始浏览历史，不能再被图片加载拉回底部。
  const onWheel = event => { if (event.deltaY < 0) dispose() }
  // onKeydown 支持键盘浏览消息时退出首次定位。
  const onKeydown = event => {
    if (['ArrowUp', 'ArrowDown', 'PageUp', 'PageDown', 'Home', 'End', ' '].includes(event.key)) dispose()
  }

  root.addEventListener('wheel', onWheel, { passive: true })
  root.addEventListener('pointerdown', dispose, { passive: true })
  root.addEventListener('touchstart', dispose, { passive: true })
  root.addEventListener('keydown', onKeydown)
  observer = new ResizeObserver(restore)
  observer.observe(content)
  observer.observe(wrap)
  restore()
  return { restore, dispose }
}
