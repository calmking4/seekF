// 常用表情按完整字符保存，保留肤色、变体选择符及组合序列。
export const emojiCategories = [
  { id: 'faces', label: '表情', emojis: ['😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣', '😊', '🙂', '🙃', '😉', '😍', '🥰', '😘', '😋', '😎', '🤩', '🥳', '🤔', '🤭', '🤫', '😴', '🥱', '😭', '🥹', '😤', '😡', '😱', '😳', '😔', '🥺', '🤗', '🫡', '🤦', '🤷', '😇', '🤯', '😷', '🤒'] },
  { id: 'gestures', label: '手势', emojis: ['👍', '👎', '👏', '🙌', '🤝', '🙏', '💪', '👌', '✌️', '🤞', '🤟', '🤘', '👋', '🤚', '🖐️', '✋', '👊', '✊', '🤛', '🤜', '👈', '👉', '👆', '👇', '☝️', '🫶', '🫰', '✍️', '💅', '🤳', '🧑‍💻', '👨‍💻'] },
  { id: 'hearts', label: '心情', emojis: ['❤️', '🧡', '💛', '💚', '💙', '💜', '🖤', '🤍', '🤎', '💔', '❤️‍🔥', '💕', '💞', '💓', '💗', '💖', '💝', '💘', '💯', '💢', '💥', '💫', '💦', '💤', '✨', '⭐', '🌟', '🔥', '🎉', '🎊', '🎈', '🎁'] },
  { id: 'life', label: '日常', emojis: ['🐶', '🐱', '🐼', '🐰', '🦊', '🐻', '🐷', '🐸', '🌸', '🌹', '🌻', '🌷', '🍀', '🌴', '🌈', '☀️', '🌙', '☁️', '🌧️', '❄️', '🍎', '🍊', '🍉', '🍓', '🍰', '🎂', '🍔', '🍕', '☕', '🍵', '🍺', '🥂', '🎵', '🎮', '🏀', '⚽', '🚗', '✈️', '🏠', '💻'] },
]

const supportedEmojis = new Set(emojiCategories.flatMap(category => category.emojis))

// normalizeRecentEmojis 只保留面板支持的最近24个表情，并去重和过滤异常缓存。
export function normalizeRecentEmojis(value) {
  return Array.isArray(value) ? [...new Set(value.filter(emoji => supportedEmojis.has(emoji)))].slice(0, 24) : []
}

// insertEmojiAtSelection 按文本框的UTF-16选区插入完整表情，返回恢复光标的位置。
export function insertEmojiAtSelection(text, emoji, start = text.length, end = start) {
  start = Math.max(0, Math.min(text.length, start))
  end = Math.max(start, Math.min(text.length, end))
  return { text: text.slice(0, start) + emoji + text.slice(end), caret: start + emoji.length }
}
