<template>
  <article class="docs-overview">
    <header class="docs-overview-head">
      <h1 class="docs-title">文档概览</h1>
      <p v-if="!loading && total" class="docs-meta">
        {{ groups.length }} 个分类 · {{ total }} 篇文档
      </p>
    </header>

    <p v-if="loading" class="docs-hint">加载中…</p>
    <p v-else-if="!groups.length" class="docs-hint">
      还没有文档内容，请先在后台「文档库」中进入该项目并添加目录和页面。
    </p>

    <!-- 按分类罗列：分类名 + 该分类下的文档，子分类缩进一级 -->
    <section
      v-for="group in groups"
      :key="group.id"
      class="docs-group"
      :class="{ 'is-sub': group.depth > 0 }"
    >
      <h2 class="docs-group-title">
        <span class="docs-group-name">{{ group.name }}</span>
        <span class="docs-group-count">{{ group.docs.length }}</span>
      </h2>
      <p v-if="group.description" class="docs-group-desc">{{ group.description }}</p>
      <ul class="docs-cards">
        <li v-for="doc in group.docs" :key="doc.id">
          <RouterLink
            :to="`/docs/${version}/${group.slug}/${doc.slug}`"
            class="docs-card"
            :title="doc.title"
          >
            {{ doc.title }}
          </RouterLink>
        </li>
      </ul>
    </section>
  </article>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { call } from '@/api/request'

const props = defineProps({
  version: { type: String, default: 'latest' }
})

// 展平后的分类：[{ id, name, description, slug, depth, docs }]，空分类不展示
const groups = ref([])
const loading = ref(true)

const total = computed(() => groups.value.reduce((sum, group) => sum + group.docs.length, 0))

async function load() {
  loading.value = true
  try {
    const res = await call('docs-group', 'tree', {
      method: 'GET',
      params: { version: props.version, with_docs: 1 }
    })

    // 后端返回的是树（children 无限层级 + docs），按先序展平，depth 决定缩进
    const flat = []
    const walk = (nodes, depth = 0) => {
      for (const node of nodes || []) {
        const docs = node.docs || []
        // 没有文档的分类不占版面（目录本身在左侧栏可见）
        if (docs.length) {
          flat.push({
            id: node.id,
            name: node.name || '未命名分类',
            description: node.description || '',
            // 没有 slug 的是虚拟「未分类」节点，正文链接用 - 占位（与文章详情页一致）
            slug: node.slug || '-',
            depth,
            docs
          })
        }
        walk(node.children, depth + 1)
      }
    }
    walk(res?.data || [])

    groups.value = flat
  } catch {
    groups.value = []
  } finally {
    loading.value = false
  }
}

watch(() => props.version, load, { immediate: true })
</script>

<style scoped>
.docs-overview {
  /* 卡片网格不需要太宽，但也别太窄导致一行只放两三个 */
  max-width: 900px;
  padding-bottom: 8px;
}
.docs-overview-head {
  margin-bottom: 22px;
}
.docs-title {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 26px;
  font-weight: 700;
  color: var(--text);
  letter-spacing: 0.5px;
}
.docs-meta {
  margin: 8px 0 0;
  font-size: 12.5px;
  color: var(--text-light);
}
.docs-hint {
  margin: 20px 0 0;
  font-size: 13.5px;
  color: var(--text-light);
}

/* 每个分类一块：标题 + 文档链接列表 */
.docs-group {
  margin-bottom: 24px;
}
.docs-group:last-child {
  margin-bottom: 0;
}
/* 子分类缩进一级，用更小的标题 */
.docs-group.is-sub {
  margin: 10px 0 18px 14px;
}
.docs-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
/* 分类前的小竖条：一眼分开各分类，不用额外图标 */
.docs-group-title::before {
  content: '';
  width: 3px;
  height: 13px;
  border-radius: 2px;
  background: var(--primary);
}
.docs-group.is-sub .docs-group-title {
  font-size: 13px;
  color: var(--text-soft);
}
.docs-group.is-sub .docs-group-title::before {
  height: 11px;
  background: var(--border);
}
.docs-group-count {
  font-size: 11.5px;
  font-weight: 400;
  color: var(--text-light);
}
.docs-group-desc {
  margin: 2px 0 8px;
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--text-light);
}

/* 小卡片：一行能放好几个，比目录行更容易扫读 */
.docs-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(176px, 1fr));
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.docs-card {
  display: block;
  padding: 9px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-soft);
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: border-color 0.15s, color 0.15s;
}
.docs-card:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}

@media (max-width: 640px) {
  .docs-title {
    font-size: 22px;
  }
  .docs-overview-head {
    margin-bottom: 18px;
  }
}
</style>
