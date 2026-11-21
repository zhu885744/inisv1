<template>
  <div>
    <div v-if="loading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>
    <EmptyState v-else-if="error" :text="errorMsg" />
    <div v-else>
      <div class="category-info card card-pad">
        <img
          v-if="categoryInfo.avatar"
          :src="categoryInfo.avatar"
          class="category-avatar"
          @error="onAvatarError"
          loading="lazy"
        />
        <div class="category-info-main">
          <h1 class="category-title">
            {{ categoryInfo.name }}
            <span class="category-count">({{ total }})</span>
          </h1>
          <p v-if="categoryInfo.description" class="category-desc">{{ categoryInfo.description }}</p>
        </div>
      </div>

      <SectionTitle :title="categoryInfo.name">
        <template #extra>
          <span class="text-muted">共 {{ total }} 篇</span>
        </template>
      </SectionTitle>

      <div class="article-list">
        <ArticleCard v-for="a in articles" :key="a.id" :article="a" :abstract-limit="50" />
        <EmptyState v-if="!articles.length" text="该分类下暂无文章" />
      </div>

      <Pagination
        :current="page"
        :total="total"
        :page-size="limit"
        @update:current="(p) => { page = p; loadArticles() }"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import SectionTitle from '@/components/SectionTitle.vue'
import ArticleCard from '@/components/ArticleCard.vue'
import Pagination from '@/components/Pagination.vue'
import EmptyState from '@/components/EmptyState.vue'
import { call } from '@/api/request'

const route = useRoute()

const loading = ref(true)
const error = ref(false)
const errorMsg = ref('')
const categoryInfo = ref({})
const articles = ref([])
const page = ref(1)
const total = ref(0)
const limit = 10

function onAvatarError(e) {
  e.target.style.display = 'none'
}

// 获取分类信息，解析出分类 id（文章 group 字段用的是 |id|）
// 路由参数可能是 id 或 key/中文名：
//  - 纯数字 → article-group/one?id=
//  - key    → article-group/all?where={"key":...}（one 不支持按 key 查）
//  - 中文名  → 在 article-group/all 列表里按 name 匹配
async function loadCategory() {
  const key = route.params.key
  try {
    if (/^\d+$/.test(String(key))) {
      const one = await call('article-group', 'one', {
        method: 'GET',
        params: { id: key }
      })
      if (one?.data?.id) {
        categoryInfo.value = one.data
        error.value = false
        return one.data
      }
    } else {
      // 用 key 反查分类 id
      const byKey = await call('article-group', 'all', {
        method: 'GET',
        params: {
          where: JSON.stringify({ key }),
          field: 'id,key,name,avatar,description'
        }
      })
      const hit = (byKey?.data?.data || [])[0]
      if (hit?.id) {
        categoryInfo.value = hit
        error.value = false
        return hit
      }
    }
    // 兜底：全部列表里按 key / 中文名匹配
    const res = await call('article-group', 'all', { method: 'GET' })
    const list = res?.data?.data || []
    const matched =
      list.find((c) => String(c.id) === String(key)) ||
      list.find((c) => c.key === key) ||
      list.find((c) => c.name === decodeURIComponent(key))
    if (matched) {
      categoryInfo.value = matched
      error.value = false
      return matched
    }
    error.value = true
    errorMsg.value = '未找到该分类'
    return null
  } catch {
    error.value = true
    errorMsg.value = '获取分类信息失败'
    return null
  }
}

// 获取分类下文章列表：where={"group":"|分类ID|"}，竖线必须
async function loadArticles() {
  const groupId = categoryInfo.value?.id
  if (!groupId) return
  try {
    const res = await call('article', 'all', {
      method: 'GET',
      params: {
        page: page.value,
        limit,
        where: JSON.stringify({ group: `|${groupId}|` }),
        order: 'top desc, create_time desc'
      }
    })
    articles.value = res.data?.data || []
    total.value = res.data?.count || 0
  } catch {
    articles.value = []
    total.value = 0
  }
}

async function init() {
  loading.value = true
  error.value = false
  const matched = await loadCategory()
  if (matched) {
    await loadArticles()
  }
  loading.value = false
}

// 从 /category/a 跳转到 /category/b 时组件复用，监听参数变化重新加载
watch(() => route.params.key, () => {
  page.value = 1
  init()
})

onMounted(init)
</script>

<style scoped>
.is-loading {
  padding: 64px;
  text-align: center;
  color: var(--text-muted);
}
.category-info {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 24px;
  background: linear-gradient(135deg, var(--accent-wash), var(--bg-card));
}
.category-avatar {
  width: 80px;
  height: 80px;
  border-radius: var(--radius);
  object-fit: cover;
  flex-shrink: 0;
  background: var(--bg-muted);
}
.category-info-main {
  flex: 1;
  min-width: 0;
}
.category-title {
  font-family: var(--font-serif);
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 6px;
}
.category-count {
  font-size: 16px;
  font-weight: normal;
  color: var(--text-muted);
}
.category-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0;
  line-height: 1.6;
}
.article-list {
  padding: 8px 0;
}
</style>
