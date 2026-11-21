<template>
  <div class="users-admin">
    <!-- 概览：统计卡片点击后联动下方列表筛选 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">用户管理</h2>
          <p class="block-desc">
            管理全站用户：权限组、封禁与解封、冻结与回收站
            <template v-if="total > 0"> · 共 {{ format(total) }} 位</template>
            <template v-if="activeStat && activeKey !== 'all'">
              · 当前「{{ activeStat.label }}」{{ activeStat.error ? '—' : format(activeStat.value) }} 位
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
          <router-link to="/blackroom" class="btn btn-ghost btn-sm">
            <i class="bi bi-shield-slash" /> 小黑屋公示
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

    <!-- 列表：/admin/users 的子路由 -->
    <RouterView />
  </div>
</template>

<script setup>
/**
 * 用户管理概览
 *
 * 只负责两件事：
 * 1. 拉取各状态用户数并渲染统计卡片；
 * 2. 通过 provide 把「统计卡片 ←→ 列表筛选」的联动能力交给子路由 UserList.vue。
 */
import { ref, computed, provide, onMounted, onUnmounted } from 'vue'
import { countUsers, listUsers } from '@/api/users'
import { USER_FILTERS, userCountParams } from '@/utils/user'

// 与列表筛选共用 USER_FILTERS（all/normal/frozen/banned/trash），统计口径不会各写一份
const stats = ref(USER_FILTERS.map((f) => ({ ...f, value: 0, error: '' })))
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
    USER_FILTERS.map(async (f) => {
      try {
        if (f.onlyTrashed) {
          // users/count 未处理 onlyTrashed，改用列表接口的 count 字段统计回收站数量
          const res = await listUsers({ page: 1, limit: 1, onlyTrashed: true, field: 'id' })
          return { key: f.key, value: pickCount(res?.data?.count), error: '' }
        }
        const res = await countUsers(userCountParams(f.key))
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

provide('usersAdmin', {
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
.users-admin {
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
  /* 卡片数量随 USER_FILTERS 增减，用自适应列数，避免每加一个筛选就要改样式 */
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
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
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--bg-muted);
  font-size: 20px;
}
.stat-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.stat-value {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.2;
  min-height: 24px;
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
    grid-template-columns: repeat(3, 1fr);
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
