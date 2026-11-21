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
    <div v-if="items.length" class="tag-cloud">
      <RouterLink
        v-for="item in items"
        :key="item.id"
        :to="`${basePath}/${item.id}`"
        class="tag tag-primary"
      >{{ item.name }}</RouterLink>
    </div>
    <p v-else class="tag-empty">{{ emptyText }}</p>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'

/** 侧边栏「标签云」模块：分类（/category/:id）与话题（/topic/:id）共用 */
defineProps({
  title: { type: String, default: '分类导航' },
  icon: { type: String, default: 'bi-folder2' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  items: { type: Array, default: () => [] },
  /** 详情页前缀：分类 /category，话题 /topic */
  basePath: { type: String, default: '/category' },
  emptyText: { type: String, default: '暂无内容' }
})
</script>

<style scoped>
.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tag-cloud .tag {
  font-size: 11px;
  padding: 3px 8px;
}
.tag-empty {
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px 4px;
}
</style>
