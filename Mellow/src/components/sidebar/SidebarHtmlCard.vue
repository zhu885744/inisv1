<template>
  <SidebarCard
    :title="title"
    :icon="icon"
    :collapsible="collapsible"
    :default-collapsed="defaultCollapsed"
    :store-key="storeKey"
    :flat="!title && !icon && !collapsible"
  >
    <!-- 内容由管理员在后台「网站设置 → 个性侧边栏」填写，与原侧边栏一致按 HTML 渲染 -->
    <div v-if="content" class="sidebar-html" v-html="content" />
    <p v-else class="sidebar-html-empty">该模块还没有填写内容</p>
  </SidebarCard>
</template>

<script setup>
import SidebarCard from './SidebarCard.vue'

/** 侧边栏「自定义内容」模块：管理员填什么就渲染什么 */
defineProps({
  title: { type: String, default: '' },
  icon: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' },
  content: { type: String, default: '' }
})
</script>

<style scoped>
.sidebar-html {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  word-break: break-word;
}
.sidebar-html :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: var(--radius-sm);
}
.sidebar-html :deep(a) {
  color: var(--primary);
}
.sidebar-html :deep(p) {
  margin: 0 0 8px;
}
.sidebar-html :deep(p:last-child) {
  margin-bottom: 0;
}
.sidebar-html :deep(ul),
.sidebar-html :deep(ol) {
  margin: 0 0 8px;
  padding-left: 18px;
}
.sidebar-html-empty {
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px 4px;
}
</style>
