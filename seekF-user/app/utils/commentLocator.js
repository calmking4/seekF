// locateDiscoverComment 沿现有分页查找通知评论，保留已加载内容并在关闭或请求失败时停止。
export async function locateDiscoverComment({ target, getComments, hasMore, loadMore, loadReplies, isCurrent }) {
  const rootId = target.parent_id || target.uuid
  let root
  while (isCurrent()) {
    root = getComments().find(comment => comment.uuid === rootId)
    if (root || !hasMore()) break
    const previousPage = getComments().length
    await loadMore()
    if (!isCurrent()) return null
    if (getComments().length <= previousPage && hasMore()) throw new Error('加载评论未取得进展，请重试')
  }
  if (!isCurrent() || !root) return null
  if (!target.parent_id) return root
  root.expanded = true
  while (isCurrent()) {
    const reply = root.replies.find(comment => comment.uuid === target.uuid)
    if (reply) return reply
    if (!root.replyHasMore) return null
    const previousPage = root.replies.length
    await loadReplies(root)
    if (!isCurrent()) return null
    if (root.replies.length <= previousPage && root.replyHasMore) throw new Error('加载回复未取得进展，请重试')
  }
  return null
}
