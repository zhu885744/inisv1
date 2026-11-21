<template>
  <SidebarCard
    :title="title"
    :icon="icon"
    :more-text="moreText"
    :more-url="moreUrl"
    :collapsible="collapsible"
    :default-collapsed="defaultCollapsed"
    :store-key="storeKey"
  >
    <ul v-if="items.length" class="comment-mini-list">
      <li v-for="item in items" :key="item.id" class="comment-mini-item">
        <RouterLink :to="item._link" class="comment-mini-link">
          <img
            v-if="item._avatar"
            :src="item._thumb"
            class="comment-mini-avatar"
            :alt="item._nickname"
            loading="lazy"
            decoding="async"
            @error="onAvatarError"
          />
          <span v-else class="comment-mini-dot" :style="{ background: item._color }">
            {{ (item._nickname || '匿').charAt(0).toUpperCase() }}
          </span>
          <span class="comment-mini-body">
            <span class="comment-mini-text">
              <strong>{{ item._nickname }}</strong>：<span v-html="item._summary"></span>
            </span>
            <span class="comment-mini-date">{{ item._date }}</span>
          </span>
        </RouterLink>
      </li>
    </ul>
    <p v-else class="comment-mini-empty">{{ emptyText }}</p>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'

/**
 * 侧边栏「最新评论」模块
 *
 * items 由 SidebarRight 统一加工（昵称 / 头像 / 摘要 / 跳转地址），本组件只负责展示，
 * 因此后台换数据源（例如改成「热门评论」）时无需改这里。
 */
defineProps({
  title: { type: String, default: '最新评论' },
  icon: { type: String, default: 'bi-chat-dots' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  items: { type: Array, default: () => [] },
  emptyText: { type: String, default: '暂无评论' }
})

function onAvatarError(e) {
  e.target.style.display = 'none'
}
</script>

<style scoped>
.comment-mini-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.comment-mini-link {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: var(--text-soft);
}
.comment-mini-link:hover {
  color: var(--primary);
}
.comment-mini-dot {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  text-shadow: 0 1px 1px rgba(0, 0, 0, 0.2);
}
.comment-mini-body {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.comment-mini-text {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.comment-mini-text strong {
  color: var(--text);
  font-weight: 600;
}
.comment-mini-text :deep(img.inline-emoji) {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  display: inline-block;
  object-fit: contain;
  margin: 0 1px;
}
.comment-mini-avatar {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  object-fit: cover;
}
.comment-mini-date {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.comment-mini-empty {
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px 4px;
}
</style>
