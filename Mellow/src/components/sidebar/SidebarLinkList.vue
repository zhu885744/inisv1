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
    <ul v-if="items.length" class="link-list">
      <li v-for="(item, index) in items" :key="item.id || index" class="link-item">
        <RouterLink v-if="linkHref(item) && !isExternal(item)" :to="linkHref(item)" class="link-row">
          <i v-if="item.icon" :class="item.icon" class="link-icon" aria-hidden="true" />
          <span class="link-name">{{ item.name }}</span>
          <i class="bi bi-chevron-right link-arrow" aria-hidden="true" />
        </RouterLink>
        <a
          v-else-if="linkHref(item)"
          :href="linkHref(item)"
          class="link-row"
          target="_blank"
          rel="noopener noreferrer"
        >
          <i v-if="item.icon" :class="item.icon" class="link-icon" aria-hidden="true" />
          <span class="link-name">{{ item.name }}</span>
          <i class="bi bi-box-arrow-up-right link-arrow" aria-hidden="true" />
        </a>
        <!-- 地址协议不合法（如 javascript:）：只显示名称，不生成可点击链接 -->
        <span v-else class="link-row">
          <i v-if="item.icon" :class="item.icon" class="link-icon" aria-hidden="true" />
          <span class="link-name">{{ item.name }}</span>
        </span>
      </li>
    </ul>
    <p v-else class="link-empty">{{ emptyText }}</p>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'
import { safeUrl } from '@/utils/url'

/** 侧边栏「自定义链接」模块：图标 + 名称 + 链接，站内走路由、站外新窗口 */
defineProps({
  title: { type: String, default: '友情链接' },
  icon: { type: String, default: 'bi-link-45deg' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  items: { type: Array, default: () => [] },
  emptyText: { type: String, default: '暂无链接' }
})

function linkHref(item) {
  return safeUrl(item?.url)
}

function isExternal(item) {
  return /^([a-z][a-z0-9+.-]*:|\/\/)/i.test(linkHref(item))
}
</script>

<style scoped>
.link-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.link-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 4px;
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-soft);
  transition: background 0.15s, color 0.15s;
}
.link-row:hover {
  background: var(--bg-muted);
  color: var(--primary);
}
.link-icon {
  flex-shrink: 0;
  width: 16px;
  font-size: 13px;
  color: var(--primary);
}
.link-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.link-arrow {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--text-light);
}
.link-empty {
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px 4px;
}
</style>
