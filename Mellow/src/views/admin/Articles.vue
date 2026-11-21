<template>
  <div class="articles-admin">
    <!-- 概览：统计卡片点击后联动下方列表筛选 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">文章管理</h2>
          <p class="block-desc">
            管理全站文章：审核、置顶、编辑与批量操作
            <template v-if="total > 0"> · 共 {{ format(total) }} 篇</template>
            <template v-if="activeStat && activeKey !== 'all'">
              · 当前「{{ activeStat.label }}」{{ activeStat.error ? '—' : format(activeStat.value) }} 篇
            </template>
          </p>
        </div>
        <div class="head-actions">
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            :class="{ 'is-loading': refreshing }"
            :disabled="refreshing"
            title="刷新统计"
            aria-label="刷新统计"
            @click="loadCounts"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
          <router-link to="/admin/article/write" class="btn btn-primary btn-sm">
            <i class="bi bi-pencil-square" /> 写文章
          </router-link>
        </div>
      </header>

      <div class="stat-grid" :aria-busy="refreshing">
        <button
          v-for="s in stats"
          :key="s.key"
          type="button"
          class="stat-card"
          :class="{ 'is-active': s.key === activeKey, 'is-error': !!s.error }"
          :aria-pressed="s.key === activeKey"
          @click="pick(s.key)"
        >
          <span class="stat-icon" :style="{ color: s.color }"><i :class="s.icon" /></span>
          <span class="stat-body">
            <span class="stat-value">
              <template v-if="!loaded">···</template>
              <template v-else-if="s.error">—</template>
              <template v-else>{{ format(s.value) }}</template>
            </span>
            <span class="stat-label">{{ s.label }}</span>
          </span>
        </button>
      </div>

      <p v-if="hasError" class="stat-error">
        <i class="bi bi-exclamation-circle" aria-hidden="true" />
        <span>部分统计获取失败，数值可能不准确</span>
        <button type="button" class="btn btn-ghost btn-sm" :disabled="refreshing" @click="loadCounts">
          重试
        </button>
      </p>
    </section>

    <!-- 列表：/admin/article 的子路由 -->
    <RouterView />
  </div>
</template>

<script setup>
/**
 * 文章管理概览
 *
 * 只负责两件事：
 * 1. 拉取各状态文章数并渲染统计卡片；
 * 2. 通过 provide 把「统计卡片 ←→ 列表筛选」的联动能力交给子路由 ArticleList.vue。
 *
 * 此前 /admin/article 直接指向 ArticleList.vue，本文件未接入路由，
 * 导致文章模块缺少统计概览、全部逻辑堆在列表页；现在与用户管理 / 友链管理保持一致。
 */
import { ref, computed, provide, onMounted, onUnmounted } from 'vue'
import { countArticles } from '@/api/article'
import { ARTICLE_FILTERS, articleCountParams } from '@/utils/article'

// 与列表筛选共用 ARTICLE_FILTERS（all/pub/audit/reject/draft/top/trash），口径不会各写一份
const stats = ref(ARTICLE_FILTERS.map((f) => ({ ...f, value: 0, error: '' })))
const refreshing = ref(false)
// 首屏加载完成前显示占位符；之后的刷新保留旧数值，避免数字来回闪烁
const loaded = ref(false)
const activeKey = ref('all')

const total = computed(() => stats.value.find((s) => s.key === 'all')?.value || 0)
const activeStat = computed(() => stats.value.find((s) => s.key === activeKey.value) || null)
const hasError = computed(() => stats.value.some((s) => s.error))

// 请求序号：快速连点刷新时丢弃过期响应，组件卸载后也不再回写状态
let reqSeq = 0

// 列表增删改后会连续触发多次刷新，合并到一次请求，避免统计数字反复跳动
let refreshTimer = null
const REFRESH_DELAY = 250

function format(n) {
  return typeof n === 'number' ? n.toLocaleString('zh-CN') : n
}

// count 接口在 number / { count } / { data } 几种返回结构下都可能出现
function pickCount(payload) {
  if (typeof payload === 'number') return payload
  if (payload && typeof payload === 'object') {
    const value = Number(payload.count ?? payload.data)
    return Number.isFinite(value) ? value : 0
  }
  return 0
}

async function loadCounts() {
  const run = ++reqSeq
  refreshing.value = true

  const results = await Promise.all(
    ARTICLE_FILTERS.map(async (f) => {
      try {
        const res = await countArticles(articleCountParams(f.key))
        return { key: f.key, value: pickCount(res?.data), error: '' }
      } catch {
        return { key: f.key, value: 0, error: '获取失败' }
      }
    })
  )

  if (run !== reqSeq) return // 已有更新的请求在途，本次结果作废

  results.forEach((r) => {
    const target = stats.value.find((s) => s.key === r.key)
    if (!target) return
    target.error = r.error
    if (!r.error) target.value = r.value
  })
  loaded.value = true
  refreshing.value = false
}

// 合并短时间内的多次刷新请求（列表批量操作会连续调用）
function scheduleRefreshStats() {
  if (refreshTimer) clearTimeout(refreshTimer)
  refreshTimer = setTimeout(() => {
    refreshTimer = null
    loadCounts()
  }, REFRESH_DELAY)
}

// 子列表注册进来的筛选回调（点击统计卡片时切换列表状态）
let filterHandler = null

provide('articlesAdmin', {
  registerFilter(fn) {
    filterHandler = typeof fn === 'function' ? fn : null
  },
  unregisterFilter(fn) {
    if (!fn || filterHandler === fn) filterHandler = null
  },
  setActive(key) {
    activeKey.value = key
  },
  refreshStats: scheduleRefreshStats
})

function pick(key) {
  activeKey.value = key
  filterHandler?.(key)
}

onMounted(loadCounts)

onUnmounted(() => {
  reqSeq += 1
  filterHandler = null
  if (refreshTimer) {
    clearTimeout(refreshTimer)
    refreshTimer = null
  }
})
</script>

<style scoped>
.articles-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.block-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.stat-grid {
  display: grid;
  /* 卡片数量随 ARTICLE_FILTERS 增减，用自适应列数 */
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  text-align: left;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.stat-card:hover {
  border-color: var(--primary);
}
.stat-card.is-active {
  border-color: var(--primary);
  background: var(--accent-wash);
}
.stat-card.is-error {
  border-color: var(--danger);
}
.stat-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--bg-muted);
  font-size: 18px;
}
.stat-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.stat-value {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
  min-height: 22px;
}
.stat-label {
  font-size: 12px;
  color: var(--text-muted);
}

.stat-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 12px 0 0;
  font-size: 12px;
  color: var(--danger);
}
.stat-error .bi {
  font-size: 13px;
}

@media (max-width: 1100px) {
  .stat-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}
@media (max-width: 768px) {
  .panel-head {
    flex-wrap: wrap;
  }
  .head-actions {
    width: 100%;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
