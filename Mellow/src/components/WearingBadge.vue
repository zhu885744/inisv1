<template>
  <span
    v-if="badge && badge.id"
    class="wear-badge"
    :class="[`is-${rarity}`, { 'is-text': showName }]"
    :style="style"
    :title="titleText"
  >
    <img v-if="badge.image" :src="badge.image" :alt="badge.name" loading="lazy" />
    <i v-else :class="badge.icon || 'bi bi-award'" />
    <span v-if="showName" class="wear-name">{{ badge.name }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'

/**
 * 佩戴中的勋章（昵称旁的那一枚）
 *
 * 数据来自 users.json.wearing_badge（佩戴时后端把名称 / 图标 / 图片 / 配色 / 稀有度一起冗余写入），
 * 所以评论、文章、动态列表渲染作者时不需要再额外请求勋章接口。
 *
 * 与 AchievementBadge 的分工：
 *   - AchievementBadge 是「勋章墙 / 列表」里的完整勋章单元（可置灰、带条件文案）；
 *   - 这里是「昵称旁」的紧凑标识，只表现一枚，稀有度决定描边与光效。
 */
const props = defineProps({
  badge: { type: Object, default: null },
  // 是否显示勋章名（默认只显示图标，避免昵称行太吵）
  showName: { type: Boolean, default: false }
})

const RARITY_NAME = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说'
}

const rarity = computed(() => (RARITY_NAME[props.badge?.rarity] ? props.badge.rarity : 'common'))

function tint(value, alpha) {
  return /^#[0-9a-f]{6}$/i.test(value || '') ? `${value}${alpha}` : 'var(--border)'
}

const style = computed(() => {
  const color = props.badge?.color || 'var(--warning)'
  return {
    color,
    borderColor: tint(color, '59'),
    background: tint(color, '1f')
  }
})

const titleText = computed(() => {
  const badge = props.badge || {}
  if (!badge.name) return ''
  const level = RARITY_NAME[rarity.value]
  const desc = badge.description ? ` · ${badge.description}` : ''
  return `${badge.name}（${level}）${desc}`
})
</script>

<style scoped>
.wear-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex-shrink: 0;
  height: 20px;
  min-width: 20px;
  padding: 0 2px;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 12px;
  line-height: 1;
  overflow: hidden;
  vertical-align: middle;
}
.wear-badge.is-text {
  height: 22px;
  padding: 0 8px 0 3px;
  font-size: 11px;
}
.wear-badge img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 999px;
}
.wear-badge .bi {
  font-size: 11px;
  line-height: 1;
}
.wear-name {
  max-width: 84px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 稀有度：描边 + 光效（不影响主题色填充，只在外面加一圈） */
.wear-badge.is-rare {
  box-shadow: 0 0 0 1px rgba(79, 157, 255, 0.75);
}
.wear-badge.is-epic {
  box-shadow: 0 0 0 1px rgba(160, 107, 255, 0.85), 0 0 6px rgba(160, 107, 255, 0.35);
}
.wear-badge.is-legendary {
  box-shadow: 0 0 0 1px #f0b429, 0 0 9px rgba(240, 180, 41, 0.55);
  animation: wear-breathe 2.4s ease-in-out infinite;
}

@keyframes wear-breathe {
  0%,
  100% {
    box-shadow: 0 0 0 1px #f0b429, 0 0 6px rgba(240, 180, 41, 0.35);
  }
  50% {
    box-shadow: 0 0 0 1px #ffd166, 0 0 12px rgba(240, 180, 41, 0.75);
  }
}

@media (prefers-reduced-motion: reduce) {
  .wear-badge.is-legendary {
    animation: none;
  }
}
</style>
