<template>
  <article class="docs-projects">
    <h1 class="docs-title">文档</h1>
    <p class="docs-lead">选择一个文档项目开始阅读，每个项目都是一套独立的内容。</p>

    <div v-if="projects.length" class="docs-grid">
      <RouterLink
        v-for="item in projects"
        :key="item.id"
        :to="item.path"
        class="docs-card"
      >
        <span class="docs-card-icon" aria-hidden="true">
          <i :class="iconOf(item)" />
        </span>

        <span class="docs-card-body">
          <span class="docs-card-title">{{ item.name }}</span>
          <span v-if="item.description" class="docs-card-desc">{{ item.description }}</span>
          <span class="docs-card-meta">
            共 {{ item.docs_count || 0 }} 篇文档
            <template v-if="!item.comment_enabled"> · 评论已关闭</template>
          </span>
        </span>

        <i class="bi bi-arrow-right docs-card-arrow" aria-hidden="true" />
      </RouterLink>
    </div>

    <p v-else-if="!loading" class="docs-empty">
      还没有可访问的文档项目，请先在后台「文档库」中创建并启用。
    </p>
  </article>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listDocsProjects } from '@/api/docs'

const projects = ref([])
const loading = ref(true)

// 后台自定义图标为空时给个兜底，避免出现空白方块
const DEFAULT_ICON = 'bi bi-journal-bookmark'

function iconOf(item) {
  const icon = String(item?.icon || '').trim()
  return icon ? (icon.startsWith('bi') ? icon : `bi ${icon}`) : DEFAULT_ICON
}

async function load() {
  loading.value = true
  try {
    const res = await listDocsProjects()
    projects.value = res?.data || []
  } catch {
    projects.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.docs-title {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 28px;
  font-weight: 700;
  color: var(--text);
  letter-spacing: 1px;
}
.docs-lead {
  margin: 12px 0 28px;
  font-size: 14px;
  line-height: 1.9;
  color: var(--text-muted);
}
.docs-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.docs-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: var(--bg-alt);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  text-decoration: none;
  transition: border-color 0.2s, transform 0.2s;
}
.docs-card:hover {
  border-color: var(--primary-soft);
  transform: translateY(-2px);
}
.docs-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  font-size: 17px;
  color: var(--primary);
  background: var(--accent-wash);
  border-radius: var(--radius-sm);
}
.docs-card-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1;
}
.docs-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.docs-card-desc {
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--text-light);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.docs-card-meta {
  margin-top: 2px;
  font-size: 11.5px;
  color: var(--text-light);
}
.docs-card-arrow {
  flex-shrink: 0;
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-light);
  transition: color 0.2s, transform 0.2s;
}
.docs-card:hover .docs-card-arrow {
  color: var(--primary);
  transform: translateX(2px);
}
.docs-empty {
  margin: 24px 0 0;
  font-size: 14px;
  color: var(--text-light);
}

@media (max-width: 900px) {
  .docs-grid {
    grid-template-columns: 1fr;
  }
}
</style>