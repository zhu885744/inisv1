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
    <div class="stat-row">
      <span>文章</span>
      <strong>{{ stats.article }}</strong>
    </div>
    <div class="stat-row">
      <span>动态</span>
      <strong>{{ stats.moment }}</strong>
    </div>
    <div class="stat-row">
      <span>话题</span>
      <strong>{{ tagCount }}</strong>
    </div>
    <div class="stat-row">
      <span>友链</span>
      <strong>{{ stats.link }}</strong>
    </div>
    <div class="stat-row">
      <span>已运行</span>
      <strong>{{ runtimeDays }} 天</strong>
    </div>
    <div class="stat-row">
      <span>建站时间</span>
      <strong>{{ siteDateText }}</strong>
    </div>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'

/** 侧边栏「站点信息」模块：纯展示，数据由 SidebarRight 统一拉取 */
defineProps({
  title: { type: String, default: '站点信息' },
  icon: { type: String, default: 'bi-bar-chart' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  stats: { type: Object, default: () => ({ article: 0, moment: 0, link: 0 }) },
  tagCount: { type: [Number, String], default: 0 },
  runtimeDays: { type: [Number, String], default: 0 },
  siteDateText: { type: String, default: '' }
})
</script>

<style scoped>
.stat-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  font-size: 13px;
  color: var(--text-soft);
}
.stat-row + .stat-row {
  border-top: 1px dashed var(--border-soft);
}
.stat-row strong {
  color: var(--primary-deep);
  font-weight: 600;
}
</style>
