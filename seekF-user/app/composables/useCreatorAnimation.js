// 数据中心数值和曲线共用帧动画，后一次更新接续当前画面，卸载时停止。
export function useCreatorAnimation(duration = 480) {
  let frame = null
  let mounted = false
  let media
  let finish
  const cancel = () => {
    if (frame !== null) cancelAnimationFrame(frame)
    frame = null
    finish = null
  }
  const animate = update => {
    cancel()
    if (!mounted || media?.matches) { update(1); return }
    finish = () => update(1)
    let start
    update(0)
    const tick = time => {
      start ??= time
      const progress = Math.min(1, Math.max(0, (time - start) / duration))
      update(1 - (1 - progress) ** 3)
      if (progress < 1) frame = requestAnimationFrame(tick)
      else { frame = null; finish = null }
    }
    frame = requestAnimationFrame(tick)
  }
  const onMotionChange = () => {
    if (!media.matches) return
    const complete = finish
    cancel()
    complete?.()
  }
  onMounted(() => {
    mounted = true
    media = window.matchMedia?.('(prefers-reduced-motion: reduce)')
    media?.addEventListener('change', onMotionChange)
  })
  onUnmounted(() => { mounted = false; cancel(); media?.removeEventListener('change', onMotionChange) })
  return { animate, cancel }
}
