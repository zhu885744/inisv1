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
    <ul v-if="items.length" class="rank-list">
      <li v-for="(item, index) in items" :key="item.id" class="rank-item">
        <RouterLink :to="`/archives/${item.id}`" class="rank-link">
          <span class="rank-num" :class="`rank-${index + 1}`">{{ index + 1 }}</span>
          <span class="rank-title">{{ item.title }}</span>
        </RouterLink>
      </li>
    </ul>
    <p v-else class="rank-empty">{{ emptyText }}</p>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'

/** 侧边栏「文章列表」模块（默认取随机文章） */
defineProps({
  title: { type: String, default: '随机文章' },
  icon: { type: String, default: 'bi-shuffle' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  items: { type: Array, default: () => [] },
  emptyText: { type: String, default: '暂无文章' }
})
</script>

<style scoped>
.rank-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.rank-link {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-soft);
  line-height: 1.5;
}
.rank-link:hover {
  color: var(--primary);
}
.rank-num {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  background: var(--text-muted);
}
.rank-num.rank-1 {
  background: #d9544d;
}
.rank-num.rank-2 {
  background: #d4a148;
}
.rank-num.rank-3 {
  background: #4a90e2;
}
.rank-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rank-empty {
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px 4px;
}
</style>
