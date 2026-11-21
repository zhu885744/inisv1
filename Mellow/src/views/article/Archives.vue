<template>
  <div>
    <SectionTitle title="文章归档">
      <template #extra>
        <span class="archive-total">
          共 {{ total }} 篇<template v-if="years.length"> · {{ years.length }} 个年份</template>
        </span>
      </template>
    </SectionTitle>

    <div v-if="loading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>

    <div v-else-if="!groups.length" class="card card-pad">
      <EmptyState text="还没有归档" />
    </div>

    <div v-else class="archive-page">
      <!-- 年份筛选：归档多了以后不用一路滚到底 -->
      <div v-if="years.length > 1" class="archive-filter" role="group" aria-label="按年份筛选">
        <button
          type="button"
          class="archive-chip"
          :class="{ 'is-active': activeYear === '' }"
          @click="activeYear = ''"
        >
          全部 <em>{{ total }}</em>
        </button>
        <button
          v-for="y in years"
          :key="y.year"
          type="button"
          class="archive-chip"
          :class="{ 'is-active': activeYear === y.year }"
          @click="activeYear = activeYear === y.year ? '' : y.year"
        >
          {{ y.year }} <em>{{ y.count }}</em>
        </button>
      </div>

      <div class="archive-timeline">
        <section v-for="group in visibleGroups" :key="group.year" class="archive-year-block">
          <header class="archive-year-head">
            <h3 class="archive-year">{{ group.year }}</h3>
            <span class="archive-year-count">{{ group.count }} 篇</span>
          </header>

          <div class="archive-months">
            <div v-for="month in group.months" :key="month.key" class="archive-month">
              <h4 class="archive-month-title">{{ month.label }}</h4>

              <ul class="archive-items">
                <li v-for="a in month.list" :key="a.id" class="archive-item">
                  <router-link class="archive-link" :to="`/archives/${a.id}`">
                    <span class="archive-day">{{ a._day }}</span>
                    <span class="archive-title" :title="a.title">{{ a.title }}</span>
                    <span class="archive-date" :title="formatDate(a._ts)">{{ a._week }}</span>
                    <i class="bi bi-arrow-up-right archive-arrow" aria-hidden="true" />
                  </router-link>
                </li>
              </ul>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import { listArticles } from '@/api/article'
import { formatDate } from '@/utils/time'
import { cache } from '@/utils/cache'
import dayjs from 'dayjs'

// 归档列表缓存键与有效期（12 小时 = 720 分钟）
const ARCHIVE_CACHE_KEY = 'mellow_archive_articles'
const ARCHIVE_CACHE_MINUTES = 12 * 60

const loading = ref(false)
// 原始列表：分组与筛选都在前端算，切换年份不必重新请求
const articles = ref([])
// 当前选中的年份，'' 表示全部
const activeYear = ref('')

// 星期文案：dayjs 的 ddd 会跟随 locale，这里直接用下标更可控
const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

/** 按「年 → 月」两级分组，年月都倒序（新内容在前），组内保持接口的发布时间倒序 */
function groupArticles(list) {
  const byYear = new Map()

  list.forEach((item) => {
    const ts = Number(item.publish_time || item.create_time) || 0
    if (!ts) return
    const d = dayjs(ts * 1000)
    const year = d.format('YYYY')
    const month = d.format('MM')

    if (!byYear.has(year)) byYear.set(year, new Map())
    const months = byYear.get(year)
    if (!months.has(month)) {
      months.set(month, { key: `${year}-${month}`, label: `${Number(month)} 月`, list: [] })
    }
    months.get(month).list.push({
      ...item,
      _ts: ts,
      _day: d.format('DD'),
      _week: WEEKDAYS[d.day()]
    })
  })

  return [...byYear.entries()]
    .sort((a, b) => Number(b[0]) - Number(a[0]))
    .map(([year, months]) => {
      const list = [...months.entries()]
        .sort((a, b) => Number(b[0]) - Number(a[0]))
        .map(([, value]) => value)
      return {
        year,
        count: list.reduce((sum, month) => sum + month.list.length, 0),
        months: list
      }
    })
}

const groups = computed(() => groupArticles(articles.value))
const total = computed(() => articles.value.length)
const years = computed(() => groups.value.map((g) => ({ year: g.year, count: g.count })))
const visibleGroups = computed(() =>
  activeYear.value ? groups.value.filter((g) => g.year === activeYear.value) : groups.value
)

async function load() {
  // 优先读缓存：命中且未过期时直接渲染，不再发起请求
  const cached = cache.get(ARCHIVE_CACHE_KEY)
  if (Array.isArray(cached)) {
    articles.value = cached
    return
  }

  loading.value = true
  try {
    const res = await listArticles({
      page: 1,
      limit: 999,
      where: { audit: 1, status: 1 },
      order: 'publish_time desc',
      field: 'id,title,create_time,publish_time'
    })
    const list = res.data?.data || []
    articles.value = list
    // 空结果不写缓存，避免新文章发布后被空缓存挡住
    if (list.length) cache.set(ARCHIVE_CACHE_KEY, list, ARCHIVE_CACHE_MINUTES)
  } catch {
    articles.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.archive-total {
  font-variant-numeric: tabular-nums;
}

.is-loading {
  padding: 64px;
  text-align: center;
  color: var(--text-muted);
}

/* ---------- 年份筛选 ---------- */
.archive-filter {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 20px;
}
.archive-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: var(--control-h-sm);
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: all var(--dur-fast) var(--ease);
}
.archive-chip em {
  font-style: normal;
  font-size: 11px;
  color: var(--text-light);
}
.archive-chip:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.archive-chip.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
}
.archive-chip.is-active em {
  color: var(--primary-deep);
}

/* ---------- 年份区块 ---------- */
/* 每年一张卡片，与首页文章卡片 / 文章详情页同一套外观（bg-card + border-soft + radius-lg + 20px）。
   归档是分组列表，按年拆卡片比整页一张长卡片更好滚，也比空白底更像「列表页」。 */
.archive-timeline {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.archive-year-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
}
.archive-year-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
}
.archive-year {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 26px;
  font-weight: 900;
  line-height: 1.1;
  letter-spacing: 0.02em;
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}
.archive-year-count {
  font-size: 12px;
  color: var(--text-light);
}

/* ---------- 月份 + 时间轴 ---------- */
.archive-months {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-left: 5px;
  padding-left: 18px;
  border-left: 1px solid var(--border-soft);
}
.archive-month {
  position: relative;
}
/* 时间轴节点：月份圆点 */
.archive-month::before {
  content: '';
  position: absolute;
  left: -23px;
  top: 7px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--bg-card);
  border: 2px solid var(--primary-soft);
}
.archive-month-title {
  margin: 0 0 6px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: var(--text-muted);
}

.archive-items {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}
.archive-item + .archive-item {
  border-top: 1px dashed var(--border-soft);
}
.archive-link {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 10px;
  margin: 0 -10px;
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 14px;
  text-decoration: none;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.archive-link:hover {
  background: var(--accent-wash);
  color: var(--primary-deep);
}
.archive-day {
  flex-shrink: 0;
  width: 26px;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  color: var(--text-light);
}
.archive-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.archive-date {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
}
.archive-arrow {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--primary);
  opacity: 0;
  transform: translateX(-2px);
  transition: opacity var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease);
}
.archive-link:hover .archive-arrow {
  opacity: 1;
  transform: none;
}

/* ---------- 窄屏 ---------- */
@media (max-width: 640px) {
  .archive-filter {
    flex-wrap: nowrap;
    overflow-x: auto;
    padding-bottom: 2px;
    scrollbar-width: none;
  }
  .archive-filter::-webkit-scrollbar {
    display: none;
  }
  .archive-chip {
    flex-shrink: 0;
    padding: 0 12px;
  }
  .archive-year {
    font-size: 22px;
  }
  .archive-months {
    margin-left: 3px;
    padding-left: 14px;
  }
  .archive-month::before {
    left: -19px;
  }
  .archive-link {
    gap: 8px;
    padding: 10px 8px;
    margin: 0 -8px;
  }
  /* 窄屏放不下两处日期：只留左侧的「日」，完整日期在 title 里 */
  .archive-date {
    display: none;
  }
}
</style>
