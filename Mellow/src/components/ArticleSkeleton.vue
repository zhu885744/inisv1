<template>
  <div class="skeleton-article" :class="{ 'has-cover': cover }">
    <div v-if="cover" class="skeleton-cover skeleton" />
    <div class="skeleton-main">
      <div class="skeleton-title skeleton" />
      <div class="skeleton-meta skeleton" />
      <div class="skeleton-line skeleton" />
      <div class="skeleton-line short skeleton" />
    </div>
  </div>
</template>

<script setup>
defineProps({
  cover: { type: Boolean, default: true }
})
</script>

<style scoped>
.skeleton-article {
  display: flex;
  gap: 16px;
  padding: 20px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.skeleton-article:last-child {
  border-bottom: none;
}
.skeleton-cover {
  width: 160px;
  height: 108px;
  flex-shrink: 0;
  border-radius: var(--radius);
}
.skeleton-main {
  flex: 1;
  min-width: 0;
  padding-top: 4px;
}
.skeleton-title {
  width: 55%;
  height: 20px;
  margin-bottom: 14px;
}
.skeleton-meta {
  width: 40%;
  height: 12px;
  margin-bottom: 14px;
}
.skeleton-line {
  width: 100%;
  height: 12px;
  margin-bottom: 10px;
}
.skeleton-line.short {
  width: 70%;
}

@media (max-width: 480px) {
  .skeleton-article.has-cover {
    flex-direction: column;
  }
  .skeleton-article.has-cover .skeleton-cover {
    width: 100%;
    height: 160px;
  }
}
</style>
