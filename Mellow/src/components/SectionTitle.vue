<template>
  <section class="section-card">
    <header class="section-header">
      <h2 class="section-title">
        <span class="title-bar"></span>
        <slot name="title">{{ title }}</slot>
      </h2>
      <div class="section-extra">
        <slot name="extra" />
      </div>
    </header>
    <div class="section-body">
      <slot />
    </div>
  </section>
</template>

<script setup>
defineProps({ title: String })
</script>

<style scoped>
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
/* H3 规范：900 衬线 20px */
.section-title {
  display: flex;
  align-items: center;
  font-family: var(--font-serif);
  font-size: 20px;
  font-weight: 900;
  color: var(--text);
  letter-spacing: 0.02em;
}
/* 版块标记统一用朱砂（与导航选中态、站点配置面板保持一致） */
.title-bar {
  display: inline-block;
  width: 4px;
  height: 20px;
  background: var(--primary);
  margin-right: 10px;
  border-radius: 2px;
}
/* Meta：13px / 500 */
.section-extra {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
</style>