<template>
  <span
    v-if="cert && cert.title"
    class="verified-badge"
    :class="[sizeClass, { 'is-dot': variant === 'dot' }]"
    :style="{ color: toneColor }"
    :title="tip"
  >
    <!--
      dot 形态的对勾用内联 SVG：bootstrap-icons 是字体字形，在不同字号下
      基线会偏（圆里看起来「掉下去」），SVG 才能做到视觉绝对居中。
    -->
    <svg
      v-if="variant === 'dot'"
      class="verified-dot-check"
      viewBox="0 0 16 16"
      aria-hidden="true"
    >
      <path
        d="M13.1 4.5 6.6 11 2.9 7.3"
        fill="none"
        stroke="currentColor"
        stroke-width="2.6"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <i v-else class="bi bi-patch-check-fill" aria-hidden="true" />
    <span v-if="showText" class="verified-text">{{ cert.title }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { certColor } from '@/utils/cert'

/**
 * 认证标识（昵称旁的金色 / 蓝色 / 绿色小盾牌）
 *
 * 数据来自 GET user-certification/one（后端 model.CertSummary），只有「已通过且未过期」
 * 才会返回，所以这里不需要再判断状态。
 *
 * 用法：
 *   <VerifiedBadge :cert="cert" />                     // 昵称旁，仅图标 + hover 提示
 *   <VerifiedBadge :cert="cert" show-text size="sm" />  // 带认证名称
 *
 * 颜色由类型配置决定（utils/cert 的 certColor）：后台给「认证类型」配了就按配置，
 * 没配则内置兜底——职业=金 / 兴趣=蓝 / 机构=绿。
 * 简介下的认证行读同一个函数，保证同一认证在页面上只有一个颜色。
 */
const props = defineProps({
  // 后端 CertSummary：{ cert_type, type_text, title, description, handle_time }（不含认证领域）
  cert: { type: Object, default: null },
  // 是否在图标后显示认证名称
  showText: { type: Boolean, default: false },
  // sm / md（昵称行用 md，列表里用 sm）
  size: { type: String, default: 'md' },
  /**
   * 展示形态：
   *   icon → 盾牌图标（默认，用于文字旁边）
   *   dot  → 圆形实心 + 白色对勾，用于头像右下角的角标
   */
  variant: { type: String, default: 'icon' }
})

const toneColor = computed(() => certColor(props.cert || {}))

const sizeClass = computed(() => (props.size === 'sm' ? 'is-sm' : 'is-md'))

const tip = computed(() => {
  const cert = props.cert || {}
  const type = cert.type_text || '认证'
  // 认证领域（category）不在公开摘要里返回，hover 只给类型 + 名称 + 后台写的说明
  const desc = cert.description ? `\n${cert.description}` : ''
  return `${type} · ${cert.title || ''}${desc}`
})
</script>

<style scoped>
.verified-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex-shrink: 0;
  font-weight: 600;
  line-height: 1;
  cursor: help;
}
.verified-badge.is-md {
  font-size: 15px;
}
.verified-badge.is-sm {
  font-size: 12px;
}

/*
 * 头像右下角的角标形态：圆形实心 + 白色对勾
 *
 * 颜色仍取认证类型色（父级内联 style 的 color），背景用 currentColor，
 * 图标单独刷成白色；1.5px 白描边让角标在深色 / 花哨头像上也立得住。
 */
.verified-badge.is-dot {
  width: 1em;
  height: 1em;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 0 1.5px #fff, 0 1px 3px rgba(0, 0, 0, 0.22);
  /* 圆内只放一个对勾，用 flex 居中 + SVG 保证水平/垂直都正 */
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.verified-dot-check {
  display: block;
  flex-shrink: 0;
  width: 56%;
  height: 56%;
  color: #fff;
}
.verified-text {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 999px;
  color: inherit;
  background: color-mix(in srgb, currentColor 12%, transparent);
  border: 1px solid color-mix(in srgb, currentColor 35%, transparent);
  white-space: nowrap;
}
</style>
