<template>
  <span
    v-if="hasContent"
    class="title-badge"
    :class="[`title-badge-${size}`, { 'is-animated': animated, 'is-image': hasImage }]"
    :style="badgeStyle"
  >
    <!-- 图片头衔：纯图展示 —— 有图片时只渲染图片，不叠加任何文字 -->
    <img
      v-if="hasImage"
      class="title-badge-img"
      :src="payload.image"
      :style="imageStyle"
      :alt="label || '头衔'"
      loading="lazy"
    />
    <template v-else>
      <i v-if="payload.icon" :class="payload.icon" class="title-badge-icon" />
      <span v-if="label" class="title-badge-text">{{ label }}</span>
    </template>
  </span>
</template>

<script setup>
import { computed } from 'vue'

/**
 * TitleBadge - 统一的头衔/称号渲染组件
 *
 * 数据来源：用户 json.title_decoration = { id, name, payload }
 * 兼容旧数据：没有 payload 时退化为纯文本展示，颜色沿用旧的文本映射。
 *
 * payload 约定（由后台装扮管理器可视化编辑）：
 *   文字头衔：text + color / bg / border / glow / icon / animated
 *   图片头衔：image + scale（scale 控制图片高度倍数，1 = 该尺寸下的默认高度）
 *
 * 两种形态互斥：有 image 就只显示图片（payload 里若还残留 text 也不渲染），
 * 没有 image 时才按文字头衔处理（text 为空则用调用方传入的 text）。
 */
const props = defineProps({
  payload: { type: Object, default: () => ({}) },
  text: { type: String, default: '' },
  size: { type: String, default: 'md' } // sm | md | lg
})

// 图片头衔在各尺寸下的基准高度（px），再乘 payload.scale
const IMAGE_HEIGHT = { sm: 16, md: 18, lg: 22 }

const label = computed(() => props.text || props.payload?.text || '')
const animated = computed(() => props.payload?.animated === true || props.payload?.animated === 1)

const image = computed(() => {
  const value = props.payload?.image
  return typeof value === 'string' ? value.trim() : ''
})
const hasImage = computed(() => image.value !== '')

// 只要「有图片 / 有文字 / 有图标」就渲染：纯图片头衔没有文字也必须显示
const hasContent = computed(() => Boolean(hasImage.value || label.value || props.payload?.icon))

const badgeStyle = computed(() => {
  const p = props.payload || {}
  const style = {}

  if (p.color) style.color = p.color
  // 图片头衔：默认继承所在行文字色、无色底（图片自己是主角），后台显式配了才用
  else if (hasImage.value) style.color = 'inherit'

  if (p.bg) style.background = p.bg
  else if (hasImage.value) style.background = 'transparent'

  if (p.glow) style.boxShadow = String(p.glow).includes('px') ? p.glow : `0 0 10px ${p.glow}`

  // border 支持两种写法：完整 CSS 值，或仅颜色（自动补成 1px solid）
  if (p.border) {
    style.border = String(p.border).includes(' ') ? p.border : `1px solid ${p.border}`
  }

  return style
})

// 图片高度：按 size 取基准并乘 scale（限制在 0.5 ~ 3，避免后台填出错值把页面撑坏）
const imageStyle = computed(() => {
  const base = IMAGE_HEIGHT[props.size] || IMAGE_HEIGHT.md
  const raw = Number(props.payload?.scale)
  const scale = Number.isFinite(raw) && raw > 0 ? Math.min(3, Math.max(0.5, raw)) : 1
  return { height: `${(base * scale).toFixed(2)}px` }
})
</script>

<style scoped>
.title-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: 4px;
  font-weight: 600;
  white-space: nowrap;
  vertical-align: middle;
  /*
   * 注意：这里**不要**写 line-height。
   * 旧的 .title-tag / .author-title 都不设行高（继承所在行的行高），
   * 写死成 1.4 会让标签比默认头衔矮一截，看起来就是「内边距特别小」。
   * 同理不加默认边框：旧标签只有背景色，多 1px 边框也会让视觉高度不同。
   */
  /* 无 payload 时的兜底配色：用语义变量，深色模式下自动跟随主题 */
  color: var(--warning);
  background: var(--gold-wash);
}

/* 尺寸与旧标签逐项对齐，避免同一位置「装扮头衔」和「默认头衔」看起来不一样大：
   sm → 列表（.title-tag：11px / 1px 6px）
   md → 作者主页（.author-title：12px / 2px 8px）
   lg → 装扮商城 / 我的装扮 的大号预览 */
.title-badge-sm {
  padding: 1px 6px;
  font-size: 11px;
}

.title-badge-md {
  padding: 2px 8px;
  font-size: 12px;
}

.title-badge-lg {
  padding: 3px 12px;
  font-size: 14px;
}

/* 图片头衔：去掉默认的金色底与内边距，图片多大就占多大 */
.title-badge.is-image {
  padding: 0;
  background: transparent;
}

.title-badge-img {
  display: block;
  width: auto;
  max-width: 100%;
  object-fit: contain;
}

.title-badge-icon {
  font-size: 0.9em;
  line-height: 1;
}

.title-badge.is-animated {
  animation: title-badge-shine 2.4s ease-in-out infinite;
}

@keyframes title-badge-shine {
  0%,
  100% {
    filter: brightness(1);
  }
  50% {
    filter: brightness(1.28);
  }
}
</style>
