// createChatMessageReflow 逐帧收起退场消息的实际占位，让周围消息连续补位。
export function createChatMessageReflow() {
  const jobs = new Map()

  // restore 恢复行原来的内联样式，避免覆盖正常消息布局。
  const restore = job => {
    if (job.originalStyle === null) job.node.removeAttribute('style')
    else job.node.setAttribute('style', job.originalStyle)
  }

  // stop 清理当前占位动画的帧、兜底计时器及用户操作监听。
  const stop = job => {
    cancelAnimationFrame(job.frame)
    clearTimeout(job.timer)
    for (const type of ['wheel', 'pointerdown', 'touchstart', 'keydown']) job.wrap?.removeEventListener(type, job.stopFollowing)
    jobs.delete(job.node)
  }

  // measureRecall 按实际字体与工具类测量撤回提示的最终占位，不写死提示高度。
  const measureRecall = (node, scale) => {
    const probe = document.createElement('div')
    probe.className = 'mb-6 flex justify-center'
    Object.assign(probe.style, { position: 'absolute', visibility: 'hidden', pointerEvents: 'none', width: `${node.offsetWidth}px` })
    const label = document.createElement('span')
    label.className = 'rounded-full bg-gray-200 px-6 py-1 text-xs text-gray-400'
    label.textContent = '你撤回了一条消息'
    probe.appendChild(label)
    node.after(probe)
    const style = getComputedStyle(probe)
    const size = { height: probe.getBoundingClientRect().height / scale, marginTop: parseFloat(style.marginTop) || 0, marginBottom: parseFloat(style.marginBottom) || 0 }
    probe.remove()
    return size
  }

  // collapse 保留原行节点，在250毫秒内持续改变高度和间距，结束后才删除或替换。
  const collapse = ({ node, wrap, getAction, onComplete }) => {
    if (!node?.isConnected) { onComplete(() => {}); return }
    const viewport = wrap?.getBoundingClientRect()
    const scale = viewport?.height && wrap.clientHeight ? viewport.height / wrap.clientHeight : 1
    const style = getComputedStyle(node)
    const job = {
      node, wrap, originalStyle: node.getAttribute('style'), frame: null, timer: null, started: null,
      action: getAction(), following: !!wrap && wrap.scrollHeight - wrap.clientHeight - wrap.scrollTop < 6,
      from: { height: node.getBoundingClientRect().height / scale, marginTop: parseFloat(style.marginTop) || 0, marginBottom: parseFloat(style.marginBottom) || 0 }
    }
    const deleted = { height: 0, marginTop: 0, marginBottom: 0 }
    const recalled = job.action === 'delete' ? deleted : measureRecall(node, scale)
    job.target = job.action === 'delete' ? deleted : recalled
    job.current = { ...job.from }
    job.stopFollowing = () => { job.following = false }
    jobs.set(node, job)
    for (const type of ['wheel', 'pointerdown', 'touchstart', 'keydown']) wrap?.addEventListener(type, job.stopFollowing, { passive: true })
    Object.assign(node.style, { overflow: 'hidden', opacity: '0', animation: 'none', transform: 'none', minHeight: '0', boxSizing: 'border-box', pointerEvents: 'none' })

    // apply 每帧改变真实布局占位；停在底部时持续保持底部，上方消息随高度收缩向下移动。
    const apply = progress => {
      for (const field of ['height', 'marginTop', 'marginBottom']) {
        job.current[field] = job.from[field] + (job.target[field] - job.from[field]) * progress
        node.style[field] = `${job.current[field]}px`
      }
      if (job.following) wrap.scrollTop = Math.max(0, wrap.scrollHeight - wrap.clientHeight)
    }
    // complete 在最终布局尺寸上提交消息状态，下一次DOM更新后由页面恢复行样式。
    const complete = () => {
      if (!jobs.has(node)) return
      if (!node.isConnected) { stop(job); onComplete(() => restore(job)); return }
      job.target = getAction() === 'delete' ? deleted : recalled
      apply(1)
      stop(job)
      onComplete(() => restore(job))
    }
    // step 用连续布局变化驱动补位，避免依赖重排后的位置差恰好被捕获。
    const step = time => {
      if (!jobs.has(node)) return
      if (!node.isConnected) { complete(); return }
      if (getAction() !== job.action) {
        job.action = getAction()
        job.from = { ...job.current }
        job.target = job.action === 'delete' ? deleted : recalled
        job.started = time
      }
      if (job.started === null) job.started = time
      const elapsed = Math.min(1, (time - job.started) / 250)
      apply(elapsed * elapsed * (3 - 2 * elapsed))
      if (elapsed >= 1) complete()
      else job.frame = requestAnimationFrame(step)
    }
    apply(0)
    job.frame = requestAnimationFrame(step)
    job.timer = setTimeout(complete, 1600)
  }

  // clear 会话切换和卸载时立即取消动画并释放DOM样式及监听。
  const clear = () => {
    for (const job of jobs.values()) { stop(job); restore(job) }
  }

  return { collapse, clear, isActive: () => jobs.size > 0 }
}
