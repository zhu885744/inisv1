<template>
  <template v-for="(part, i) in parts" :key="i">
    <img
      v-if="part.emoji"
      :src="part.url"
      class="inline-emoji"
      alt="emoji"
      :style="{ width: size + 'px', height: size + 'px' }"
      loading="lazy"
      decoding="async"
    />
    <template v-else>{{ part.text }}</template>
  </template>
</template>

<script setup>
import { computed } from 'vue'
import { getFullUrl } from '@/utils/runtimeConfig'

/**
 * 安全地把纯文本里的 [emoji:url] 渲染成行内表情图片
 *
 * 与 @/utils/emoji 的 renderEmoji 视觉一致（同为 .inline-emoji 样式），
 * 但用「元素渲染」代替 v-html：标题 / 摘要这类未经后端净化的文本
 * 也能显示表情，而不会引入 XSS 风险。
 */
const props = defineProps({
  text: { type: [String, Number], default: '' },
  // 表情尺寸（px），与正文行高匹配
  size: { type: Number, default: 22 }
})

const EMOJI_RE = /\[emoji:\s*([^\]\s]+)\s*\]/g

const parts = computed(() => {
  const raw = String(props.text ?? '')
  if (!raw) return []

  const list = []
  let last = 0
  let match

  EMOJI_RE.lastIndex = 0
  while ((match = EMOJI_RE.exec(raw)) !== null) {
    if (match.index > last) {
      list.push({ text: raw.slice(last, match.index) })
    }
    list.push({ emoji: true, url: getFullUrl(match[1]) })
    last = match.index + match[0].length
  }
  if (last < raw.length) {
    list.push({ text: raw.slice(last) })
  }
  return list
})
</script>
