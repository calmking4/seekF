// splitChatSearchText 只拆分文字用于高亮，不把消息或关键词解释为HTML。
export function splitChatSearchText(text, keyword) {
  text = String(text || '')
  keyword = String(keyword || '').trim()
  if (!keyword) return [{ text, match: false }]
  const escaped = keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return text.split(new RegExp(`(${escaped})`, 'gi')).filter(Boolean).map(part => ({ text: part, match: part.toLocaleLowerCase() === keyword.toLocaleLowerCase() }))
}

