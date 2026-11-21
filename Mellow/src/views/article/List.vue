<template>
  <div>
    <SectionTitle title="全部文章">
      <template #extra>
        <span class="text-muted">共 {{ total }} 篇</span>
      </template>
    </SectionTitle>

    <!-- 过滤栏 -->
    <div class="filter-bar card card-pad-sm">
      <div class="filter-tags">
        <button
          :class="['filter-tag', !filter.group && 'is-active']"
          @click="setFilter({ group: null, tag: null })"
        >全部</button>
        <span class="filter-divider">|</span>
        <FormSelect variant="pill"
          v-model="filter.sort"
          :options="sortOptions"
          icon="bi bi-sort-down"
          placeholder="排序方式"
          @change="reload" />
      </div>
    </div>

    <div class="article-list">
      <template v-if="loading">
        <ArticleSkeleton v-for="i in pageSize" :key="'sk-' + i" :cover="i % 2 === 1" />
      </template>
      <ArticleCard v-for="a in articles" :key="a.id" :article="a" :abstract-limit="50" />
      <EmptyState v-if="!loading && !articles.length" text="没有找到文章" />
    </div>

    <Pagination
      :current="page"
      :total="total"
      :page-size="pageSize"
      @update:current="(p) => { page = p; load() }"
    />
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import SectionTitle from '@/components/SectionTitle.vue'
import ArticleCard from '@/components/ArticleCard.vue'
import ArticleSkeleton from '@/components/ArticleSkeleton.vue'
import Pagination from '@/components/Pagination.vue'
import EmptyState from '@/components/EmptyState.vue'
import { listArticles } from '@/api/article'

const route = useRoute()
const articles = ref([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const pageSize = 10

const filter = ref({
  sort: 'create_time desc',
  group: null,
  tag: null
})

// 排序下拉选项（主题内置 SelectMenu）
const sortOptions = [
  { value: 'create_time desc', label: '最新发布' },
  { value: 'views desc', label: '最多阅读' },
  { value: 'create_time asc', label: '最早发布' }
]

function setFilter(f) {
  filter.value.group = f.group
  filter.value.tag = f.tag
  page.value = 1
  load()
}

async function load() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: pageSize,
      where: { audit: 1, status: 1 },
      order: filter.value.sort
    }
    // 分组过滤：group 字段直接匹配
    if (filter.value.group) {
      params.where.group = filter.value.group
    }
    // 话题过滤：用 like 匹配 tags 字段中的 |id|
    if (filter.value.tag) {
      params.like = `tags|%7C${filter.value.tag}%7C`
    }
    const res = await listArticles(params)
    articles.value = res.data?.data || []
    total.value = res.data?.count || 0
  } catch {
    articles.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

watch(() => route.query, (q) => {
  filter.value.group = q.group ? Number(q.group) : null
  filter.value.tag = q.tag ? Number(q.tag) : null
  page.value = 1
  load()
}, { immediate: false })

onMounted(() => {
  filter.value.group = route.query.group ? Number(route.query.group) : null
  filter.value.tag = route.query.tag ? Number(route.query.tag) : null
  load()
})
</script>

<style scoped>
.filter-bar {
  margin-bottom: 16px;
  padding: 12px 16px !important;
}
.filter-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.filter-tag {
  /* 与筛选栏内的排序下拉等高 */
  display: inline-flex;
  align-items: center;
  height: var(--control-h-sm);
  padding: 0 10px;
  border-radius: var(--radius);
  color: var(--text-soft);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.15s;
}
.filter-tag.is-active {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.filter-tag:hover {
  color: var(--primary);
}
.filter-divider {
  color: var(--border);
  margin: 0 4px;
}
.article-list {
  padding: 8px 0;
}
.is-loading {
  padding: 32px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}
</style>