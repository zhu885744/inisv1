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
    <ul class="archive-list">
      <li>
        <RouterLink to="/archives" class="archive-item">
          <span>所有文章</span>
          <span class="text-muted">{{ total }}</span>
        </RouterLink>
      </li>
    </ul>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'

/** 侧边栏「文章归档」模块 */
defineProps({
  title: { type: String, default: '文章归档' },
  icon: { type: String, default: 'bi-archive' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  total: { type: [Number, String], default: 0 }
})
</script>

<style scoped>
.archive-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.archive-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 4px;
  font-size: 13px;
  color: var(--text-soft);
}
.archive-item:hover {
  color: var(--primary);
}
</style>
