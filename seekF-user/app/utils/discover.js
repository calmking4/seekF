// 将不同入口的帖子统一为详情和卡片使用的数据结构。
export function normalizeDiscoverPost(post) {
  return {
    ...post,
    id: post.uuid || post.id,
    src: post.first_url ?? post.src ?? '',
    coverUrl: post.cover_url ?? post.coverUrl ?? '',
    type: post.media_type === 1 || post.type === 'video' ? 'video' : 'image',
    title: post.title || '未命名动态',
    nickname: post.nickname || '匿名用户',
    width: Number(post.width) || 0,
    height: Number(post.height) || 0,
    likeCount: Number(post.like_count ?? post.likeCount) || 0,
    isLiked: Boolean(post.is_liked ?? post.isLiked),
  }
}

// 仅有效助力权益显示已助力，作者自身的观看权限不代表购买过权益。
export function discoverSupportLabel(post) {
  const expiresAt = Date.parse(post.support_expires_at || '')
  return post.can_play === true && expiresAt > Date.now() ? '已助力' : '助力专属'
}

// 未知尺寸使用稳定比例，长图限制在 3:4，视频默认 16:9。
export function discoverAspectRatio(post) {
  const ratio = post.width > 0 && post.height > 0 ? post.width / post.height : (post.type === 'video' ? 16 / 9 : 4 / 3)
  return Math.min(16 / 9, Math.max(3 / 4, ratio))
}

// 合并分页数据时按唯一标识去重，已有内容保持顺序。
export function mergeDiscoverPage(existing, incoming) {
  const ids = new Set(existing.map(item => item.id || item.uuid))
  return [...existing, ...incoming.filter(item => {
    const id = item.id || item.uuid
    if (!id || ids.has(id)) return false
    ids.add(id)
    return true
  })]
}

// 登录回跳仅接受站内帖子详情路径，拒绝外部地址与其他任意路径。
export function discoverReturnPath(value) {
  if (typeof value === 'string' && /^\/support(?:\/result)?(?:\?[^#]*)?$/.test(value)) return value
  return typeof value === 'string' && /^\/discover\/post\/[A-Za-z0-9_-]+$/.test(value) ? value : '/'
}

// 首图按媒体区高度等比展开；视口宽度不足时仅收窄左栏，不裁切图片。
export function discoverMediaWidth(size, mediaHeight, availableWidth) {
  if (!size || size.width <= 0 || size.height <= 0 || mediaHeight <= 0 || availableWidth <= 0) return null
  return Math.min(availableWidth, mediaHeight * size.width / size.height)
}
