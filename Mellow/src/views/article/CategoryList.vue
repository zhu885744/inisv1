<template>
  <div>
    <SectionTitle title="分类">
      <template #extra>
        <span class="text-muted">共 {{ allCategories.length }} 个分类</span>
      </template>
    </SectionTitle>
    <div v-if="catLoading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>
    <div v-else class="category-grid">
      <router-link
        v-for="c in allCategories"
        :key="c.id"
        :to="`/category/${c.key || c.id}`"
        class="category-card"
      >
        <img v-if="c.avatar" :src="c.avatar" class="category-cover" @error="onImgError" loading="lazy" />
        <div v-else class="category-cover category-cover-empty">{{ c.name?.[0] || '·' }}</div>
        <div class="category-name">{{ c.name }}</div>
      </router-link>
      <EmptyState v-if="!allCategories.length" text="暂无分类" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import { call } from '@/api/request'

const allCategories = ref([])
const catLoading = ref(false)

function onImgError(e) {
  e.target.style.display = 'none'
}

// 获取全部分类（/api/article-group/all）
async function loadAllCategories() {
  catLoading.value = true
  try {
    const res = await call('article-group', 'all', { method: 'GET' })
    allCategories.value = res?.data?.data || []
  } catch {
    allCategories.value = []
  } finally {
    catLoading.value = false
  }
}

onMounted(loadAllCategories)
</script>

<style scoped>
.is-loading {
  padding: 64px;
  text-align: center;
  color: var(--text-muted);
}
.category-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
  padding: 8px 0 24px;
}
.category-card {
  display: block;
  padding: 14px;
  border-radius: var(--radius);
  background: var(--bg-card);
  border: 1px solid var(--border);
  text-align: center;
  transition: all 0.2s;
}
.category-card:hover {
  border-color: var(--primary);
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
}
.category-cover {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  margin: 0 auto 10px;
  background: var(--bg-muted);
}
.category-cover-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 26px;
  font-family: var(--font-serif);
  color: var(--primary);
  background: var(--accent-soft);
}
.category-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}
</style>
