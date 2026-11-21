<template>
  <component
    :is="clickable ? 'button' : 'div'"
    class="achievement-badge"
    :class="[`achievement-badge-${size}`, { 'is-held': held, 'is-locked': !held, 'is-clickable': clickable }]"
    :type="clickable ? 'button' : undefined"
    :title="titleText"
  >
    <span class="achievement-badge-icon" :style="iconStyle">
      <img v-if="badge.image" :src="badge.image" :alt="badge.name" loading="lazy" />
      <i v-else :class="badge.icon || 'bi bi-award'" />
    </span>
    <span class="achievement-badge-name">{{ badge.name || '勋章' }}</span>
    <span v-if="showCondition && badge.condition_name" class="achievement-badge-cond">
      {{ conditionText }}
    </span>
  </component>
</template>

<script setup>
import { computed } from 'vue'

/**
 * 勋章展示单元
 *
 * 与 TitleBadge 的区别：TitleBadge 是「头衔」（用户自己设置的称号，可带图片/发光），
 * 这里是「成就勋章」（靠任务条件获得，未获得时置灰）。视觉口径沿用同一套颜色变量。
 */
const props = defineProps({
  badge: { type: Object, default: () => ({}) },
  size: { type: String, default: 'md' }, // sm | md
  held: { type: Boolean, default: true },
  showCondition: { type: Boolean, default: false },
  clickable: { type: Boolean, default: false }
})

const color = computed(() => props.badge.color || 'var(--warning)')

// 主题色是 #rrggbb 时直接用 8 位十六进制加透明通道（兼容性比 color-mix 好），
// 否则退回中性边框，不让颜色配置错误把组件样式带崩
function tint(value, alpha) {
  return /^#[0-9a-f]{6}$/i.test(value) ? `${value}${alpha}` : 'var(--border)'
}

const iconStyle = computed(() => {
  if (!props.held) return {}
  return {
    color: color.value,
    borderColor: tint(color.value, '59'),
    background: tint(color.value, '1f')
  }
})

const conditionText = computed(() => {
  const badge = props.badge
  const unit = badge.condition_unit || ''
  if (badge.condition === 'manual') return badge.condition_name
  return `${badge.condition_name} ${badge.target ?? badge.condition_value ?? ''}${unit}`
})

const titleText = computed(() => {
  const badge = props.badge
  if (!badge.name) return ''
  const desc = badge.description ? ` · ${badge.description}` : ''
  return `${badge.name}${desc}`
})
</script>

<style scoped>
.achievement-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px 4px 4px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  font-size: 12px;
  color: var(--text);
  max-width: 100%;
}
.achievement-badge.is-locked {
  opacity: 0.55;
  filter: grayscale(0.6);
}
.achievement-badge.is-clickable {
  cursor: pointer;
  transition: border-color var(--dur) var(--ease), transform var(--dur) var(--ease);
}
.achievement-badge.is-clickable:hover {
  border-color: var(--accent-soft);
  transform: translateY(-1px);
}
.achievement-badge-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--bg-muted);
  font-size: 12px;
  overflow: hidden;
}
.achievement-badge-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.achievement-badge-name {
  font-weight: 600;
  white-space: nowrap;
}
.achievement-badge-cond {
  color: var(--text-muted);
  font-size: 11px;
  white-space: nowrap;
}
.achievement-badge-sm {
  padding: 2px 8px 2px 2px;
  font-size: 11px;
}
.achievement-badge-sm .achievement-badge-icon {
  width: 18px;
  height: 18px;
  font-size: 10px;
}
</style>
