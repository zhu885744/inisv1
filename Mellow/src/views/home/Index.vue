<template>
  <div class="home">
    <!-- 公告：单行横幅，内容轮播；右侧短条指示器 -->
    <div
      v-if="placards.length"
      class="placard-card"
      @mouseenter="pause"
      @mouseleave="resume"
    >
      <span class="placard-icon"><i class="bi bi-megaphone" /></span>
      <span class="placard-title">公告</span>
      <span class="placard-divider" aria-hidden="true" />
      <div class="placard-body">
        <transition name="placard-fade" mode="out-in">
          <span
            :key="current.id"
            class="placard-link"
            @click="openPlacard(current)"
          >
            <span class="placard-dot" :style="{ background: placardTypeColor(current.type) }" />
            <span class="placard-name">{{ current.title }}</span>
            <span
              v-if="current.content"
              class="placard-content"
              v-html="current.content"
            />
          </span>
        </transition>
      </div>
      <div v-if="placards.length > 1" class="placard-dots">
        <button
          v-for="(p, i) in placards"
          :key="p.id"
          class="placard-dot-btn"
          :class="{ 'is-active': i === index }"
          @click="goTo(i)"
        />
      </div>
    </div>

    <!-- 发布动态：入口条，点击以弹窗形式编辑发布（未登录时整条即登录入口） -->
    <button v-if="userStore.isLogged" type="button" class="compose" @click="composeOpen = true">
      <img class="compose-avatar" :src="composeAvatar" :alt="composeName" loading="lazy" />
      <span class="compose-placeholder">分享此刻的想法…</span>
      <span class="btn btn-primary btn-sm compose-btn"><i class="bi bi-pencil-square" /> 发布动态</span>
    </button>

    <RouterLink
      v-else
      :to="{ name: 'login', query: { redirect: $route.fullPath } }"
      class="compose compose-guest"
    >
      <i class="bi bi-person-circle compose-guest-icon" aria-hidden="true" />
      <span class="compose-guest-text">登录后即可发布动态、参与讨论</span>
      <span class="btn btn-primary btn-sm">去登录</span>
    </RouterLink>

    <!-- 发布弹窗：正文 / 图片 / 提交都在弹窗里（见 components/MomentComposeDialog.vue） -->
    <MomentComposeDialog v-model:visible="composeOpen" @published="reload" />

    <!-- 发现：文章与动态按时间混排 -->
    <SectionTitle title="发现">
      <template #extra>
        <div class="feed-extra">
          <span class="feed-summary">{{ summaryText }}</span>
          <div class="feed-filter" role="tablist" aria-label="内容类型">
            <button
              v-for="item in FILTERS"
              :key="item.key"
              type="button"
              role="tab"
              class="feed-filter-item"
              :class="{ 'is-active': filter === item.key }"
              :aria-selected="filter === item.key"
              @click="switchFilter(item.key)"
            >
              {{ item.label }}
            </button>
          </div>
        </div>
      </template>
    </SectionTitle>

    <div class="feed">
      <!-- 首屏骨架：首页卡片不带封面图，骨架也不画封面块，避免加载完跳一下 -->
      <template v-if="loading && !items.length">
        <ArticleSkeleton v-for="i in 3" :key="'skeleton-' + i" :cover="false" />
      </template>

      <!-- 混排列表：文章与动态都用动态卡片风格（文章显示正文片段 + 查看更多） -->
      <template v-for="item in pageItems" :key="item.key">
        <ArticleFeedItem
          v-if="item.type === 'article'"
          :article="item.data"
          :like-state="articleLikeStates[item.data.id]"
        />
        <MomentItem
          v-else
          :moment="item.data"
          compact
          :like-state="momentLikeStates[item.data.id]"
          :comment-total="momentCommentCounts[item.data.id]"
          @delete="onMomentDelete"
          @refresh="reload"
        />
      </template>

      <EmptyState
        v-if="!loading && !pageItems.length"
        icon="🔍"
        :text="emptyText"
      />

      <!-- 翻页中：旧内容先留着，给个轻提示 -->
      <div v-if="loading && items.length" class="feed-loading">
        <span class="spinner" /> 加载中…
      </div>

      <!-- 分页：每页 10 条，页码跟着 URL 无关，数据池按需向后补页 -->
      <Pagination
        v-if="total > PAGE_SIZE"
        :current="page"
        :total="total"
        :page-size="PAGE_SIZE"
        @update:current="goPage"
      />
    </div>

    <!-- 公告详情弹窗 -->
    <Teleport to="body">
      <transition name="placard-modal">
        <div v-if="activePlacard" class="placard-mask" @click.self="closePlacard">
          <div class="placard-dialog">
            <div class="placard-dialog-head">
              <!-- 类型小标签：文案 + 对应配色（与后台列表、轮播圆点同源） -->
              <span class="placard-dialog-tag" :style="placardTypeStyle(activePlacard.type)">
                {{ placardTypeLabel(activePlacard.type) }}
              </span>
              <h3 class="placard-dialog-title">{{ activePlacard.title }}</h3>
              <button class="placard-dialog-close" @click="closePlacard" aria-label="关闭">
                <i class="bi bi-x-lg"></i>
              </button>
            </div>
            <div class="placard-dialog-body" v-html="activePlacard.content" />
            <div class="placard-dialog-foot">
              <span class="placard-dialog-date">{{ formatDate(activePlacard.create_time) }}</span>
              <a
                v-if="placardUrl(activePlacard)"
                :href="placardUrl(activePlacard)"
                :target="placardLinkTarget(activePlacard)"
                :rel="placardLinkTarget(activePlacard) === '_blank' ? 'noopener' : null"
                class="btn btn-primary btn-sm"
                @click="onPlacardLinkClick($event, activePlacard)"
              >查看原文</a>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SectionTitle from '@/components/SectionTitle.vue'
import ArticleFeedItem from '@/components/ArticleFeedItem.vue'
import ArticleSkeleton from '@/components/ArticleSkeleton.vue'
import MomentItem from '@/components/MomentItem.vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import MomentComposeDialog from '@/components/MomentComposeDialog.vue'
import { useUserStore } from '@/stores/user'
import { listArticles } from '@/api/article'
import { listMoments, removeMoment } from '@/api/moments'
import { likeStates } from '@/api/tags'
import { countCommentsBatch } from '@/api/comment'
import { getFullUrl } from '@/utils/runtimeConfig'
import { call } from '@/api/request'
import { placardTypeLabel, placardTypeColor, placardTypeStyle } from '@/utils/placard'
import { formatDate } from '@/utils/time'
import { toast } from '@/utils/toast'
import { safeUrl } from '@/utils/url'

const loading = ref(false)
const placards = ref([])
const index = ref(0)
let timer = null

/* ---------------- 首页发布动态（入口条 + 弹窗） ---------------- */

// 发布逻辑全在弹窗组件里（components/MomentComposeDialog.vue），
// 首页只负责：入口条上显示头像/昵称、控制弹窗显隐、发布成功后刷新「发现」列表。
const userStore = useUserStore()
const composeOpen = ref(false)
const composeAvatar = computed(() => getFullUrl(userStore.user?.avatar || ''))
const composeName = computed(() => userStore.user?.nickname || '我')

// ============ 发现页：文章与动态混排 ============
const FILTERS = [
  { key: 'all', label: '全部' },
  { key: 'article', label: '文章' },
  { key: 'moment', label: '动态' }
]
// 每页展示条数（混排后按页切片，见 pageItems）
const PAGE_SIZE = 10
// 每次向单个来源补页时取的条数（取满一页即可，多余的留给后面的页）
const SOURCE_PAGE = 10

const filter = ref('all')
// 扁平化的混排池：{ key, type, data, time, top }
// 说明：分页只切这个池子，池子不够长时再按来源向后补页（见 ensureItems）
const items = ref([])
const page = ref(1)
const totals = reactive({ article: 0, moment: 0 })
// 两个来源各自的下一页页码
const cursors = reactive({ article: 1, moment: 1 })
// 某个来源是否已经取尽：取尽后不再重复请求空页
const exhausted = reactive({ article: false, moment: false })

const total = computed(() => {
  if (filter.value === 'article') return totals.article
  if (filter.value === 'moment') return totals.moment
  return totals.article + totals.moment
})
// 当前页要渲染的条目：混排池按页切片
const pageItems = computed(() => items.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE))

/* ---------------- 列表互动状态（批量） ----------------
 * 之前每张卡片挂载后各自请求「是否已赞 / 点赞数 / 评论数」，
 * 一页十几张卡片就是几十个请求。
 * 现在由首页在拿到一批数据后批量查一次，再通过 props 下发给卡片：
 *   - 点赞状态：GET /api/user-likes/states（article / moment 各一次）
 *   - 评论数：  GET /api/comment/counts（moment 一次）
 */
const articleLikeStates = reactive({})
const momentLikeStates = reactive({})
const momentCommentCounts = reactive({})

async function loadLikeStates(type, ids, target) {
  if (!ids.length) return
  try {
    const res = await likeStates(type, ids)
    const states = res?.data?.states || {}
    Object.entries(states).forEach(([id, state]) => {
      target[id] = state
    })
  } catch {
    // 批量失败不阻塞列表：卡片会退化成自己查一次（见各组件的 onMounted）
  }
}

async function loadMomentCommentCounts(ids) {
  if (!ids.length) return
  // 先统一置 0：后端 comment/counts 只返回 count>0 的项，缺项会被卡片当成
  // 「没有批量数据」再各发一次 comment/count（实测 6 条动态里 3 条零评论 → 3 个多余请求）
  ids.forEach((id) => {
    momentCommentCounts[id] = 0
  })
  try {
    const res = await countCommentsBatch('moments', ids)
    const counts = res?.data?.counts || {}
    Object.entries(counts).forEach(([id, count]) => {
      momentCommentCounts[id] = Number(count) || 0
    })
  } catch {
    /* 同上，卡片自行兜底 */
  }
}

// 必须在列表渲染前完成：卡片一旦挂载再补数据，就已经白发了一轮单条请求
async function loadInteractionStates(rows) {
  const articleIds = rows.filter((it) => it.type === 'article').map((it) => Number(it.data.id)).filter(Boolean)
  const momentIds = rows.filter((it) => it.type === 'moment').map((it) => Number(it.data.id)).filter(Boolean)

  await Promise.allSettled([
    loadLikeStates('article', articleIds, articleLikeStates),
    loadLikeStates('moment', momentIds, momentLikeStates),
    loadMomentCommentCounts(momentIds)
  ])
}



const current = computed(() => placards.value[index.value] || {})

const activePlacard = ref(null)
function openPlacard(p) {
  if (!p || !p.id) return
  activePlacard.value = p
  pause()
}
function closePlacard() {
  activePlacard.value = null
  resume()
}
function typeLabel(type) {
  return { notice: '公告', warning: '警告', info: '提示' }[type] || '公告'
}

const route = useRoute()
const router = useRouter()

/**
 * 站内「发布动态」入口（旧地址 /moments，见 router/index.js）会重定向到 /?compose=1。
 * 落地即打开弹窗，随即把参数从地址栏抹掉 —— 否则关掉弹窗后刷新 / 后退会再弹一次。
 */
function openComposeFromQuery() {
  if (route.query.compose !== '1' || !userStore.isLogged) return
  const { compose, ...rest } = route.query
  composeOpen.value = true
  router.replace({ query: rest })
}

// 公告原文跳转方式：仅支持 _blank（新窗口）/ _self（当前窗口），其他值回退为新窗口
function placardLinkTarget(p) {
  return p?.target === '_self' ? '_self' : '_blank'
}

// 公告跳转地址：后台填写，仍走协议白名单 —— 后台富文本/输入框被塞进
// `javascript:` 时不会变成「点一下就执行」的按钮；非法地址直接不渲染链接
function placardUrl(p) {
  return safeUrl(p?.url)
}

// _self 且为站内路径时走路由跳转，避免整页刷新；外链交由浏览器默认行为处理
function onPlacardLinkClick(e, p) {
  if (placardLinkTarget(p) !== '_self') return
  const url = placardUrl(p)
  // 排除协议相对地址（//example.com）与带协议的绝对地址
  if (!url.startsWith('/') || url.startsWith('//')) return
  e.preventDefault()
  router.push(url)
}

function next() {
  if (placards.value.length > 1) {
    index.value = (index.value + 1) % placards.value.length
  }
}
function goTo(i) {
  index.value = i
  restart()
}
function start() {
  stop()
  timer = setInterval(next, 10000)
}
function stop() {
  if (timer) clearInterval(timer)
  timer = null
}
function restart() {
  if (placards.value.length > 1) start()
}
function pause() {
  stop()
}
function resume() {
  restart()
}

async function loadPlacards() {
  try {
    const res = await call('placard', 'all', {
      method: 'GET',
      params: { page: 1, limit: 5, order: 'create_time desc' }
    })
    placards.value = res?.data?.data || []
    index.value = 0
    restart()
  } catch {
    placards.value = []
  }
}

// ============ 发现页数据 ============
const summaryText = computed(() => {
  if (filter.value === 'article') return `${totals.article} 篇文章`
  if (filter.value === 'moment') return `${totals.moment} 条动态`
  return `文章 ${totals.article} · 动态 ${totals.moment}`
})

const emptyText = computed(() => {
  if (filter.value === 'article') return '还没有文章'
  if (filter.value === 'moment') return '还没有动态'
  return '还没有内容'
})

// 当前筛选下要拉取的来源
function activeSources() {
  if (filter.value === 'article') return ['article']
  if (filter.value === 'moment') return ['moment']
  return ['article', 'moment']
}

// 拉取某一类的一页（两类接口的返回结构一致：data.data 列表 + data.count 总数）
async function fetchSource(type, p) {
  if (type === 'article') {
    const res = await listArticles({
      page: p,
      limit: SOURCE_PAGE,
      where: { audit: 1, status: 1 },
      order: 'top desc, publish_time desc'
    })
    return { list: res.data?.data || [], count: Number(res.data?.count) || 0 }
  }

  const res = await listMoments({ page: p, limit: SOURCE_PAGE })
  return { list: res.data?.data || [], count: Number(res.data?.count) || 0 }
}

// 清空混排池与游标（首屏重载 / 切换筛选时用）
function resetPool() {
  page.value = 1
  cursors.article = 1
  cursors.moment = 1
  exhausted.article = false
  exhausted.moment = false
  totals.article = 0
  totals.moment = 0
  items.value = []
  // 互动状态跟着列表一起清，避免上一轮筛选的残留数据被复用
  Object.keys(articleLikeStates).forEach((key) => delete articleLikeStates[key])
  Object.keys(momentLikeStates).forEach((key) => delete momentLikeStates[key])
  Object.keys(momentCommentCounts).forEach((key) => delete momentCommentCounts[key])
}

/**
 * 保证混排池里至少有 count 条
 *
 * 分页只切池子，池子不够长才向后补页：每个未取尽的来源各拉一页（SOURCE_PAGE 条），
 * 合并进池后按「置顶优先 + 时间倒序」重排。补到够用或两个来源都取尽为止。
 */
async function ensureItems(count) {
  // 兜底循环上限：服务端 count 与实际条数不一致时不至于转死
  for (let guard = 0; guard < 20 && items.value.length < count; guard++) {
    const sources = activeSources().filter((type) => !exhausted[type])
    if (!sources.length) return

    const results = await Promise.all(
      sources.map(async (type) => {
        const loadedForType = items.value.filter((it) => it.type === type).length
        const p = cursors[type]
        try {
          const data = await fetchSource(type, p)
          cursors[type] = p + 1
          totals[type] = data.count
          // 返回不足一页，或已拿满该来源的总数 → 视为取尽，后面不再请求
          if (data.list.length < SOURCE_PAGE || loadedForType + data.list.length >= data.count) {
            exhausted[type] = true
          }
          return { type, list: data.list }
        } catch {
          // 单个来源失败不影响另一类展示，且不再重试（避免同一页反复请求）
          exhausted[type] = true
          return { type, list: [] }
        }
      })
    )

    const seen = new Set(items.value.map((it) => it.key))
    const incoming = []

    results.forEach((res) => {
      const prefix = res.type === 'article' ? 'a' : 'm'
      res.list.forEach((row) => {
        const key = `${prefix}:${row.id}`
        if (seen.has(key)) return
        seen.add(key)
        incoming.push({
          key,
          type: res.type,
          data: row,
          // 文章按发布时间、动态按创建时间参与排序；置顶统一浮到最前
          time: Number(row.publish_time || row.create_time) || 0,
          top: Number(row.top) || 0
        })
      })
    })

    // 一条新的都没有：说明服务端没有更多了，直接收尾
    if (!incoming.length) return

    // 先把这一批的点赞 / 评论数批量取回来：卡片挂载时 props 已就绪，
    // 就不会再各自发一次单条请求（挂载后再补会白发一轮）
    await loadInteractionStates(incoming)

    items.value = items.value.concat(incoming).sort((a, b) => b.top - a.top || b.time - a.time)
  }
}

/**
 * 切到第 target 页
 * 先把池子补到 target * PAGE_SIZE 条，再切页码；目标页无内容时回退到最后有数据的页
 */
async function loadPage(target = 1) {
  if (loading.value) return
  loading.value = true
  try {
    await ensureItems(target * PAGE_SIZE)
    page.value = target
    if (!pageItems.value.length && target > 1) {
      page.value = Math.max(1, Math.ceil(items.value.length / PAGE_SIZE))
    }
  } finally {
    loading.value = false
  }
}

function goPage(target) {
  if (target === page.value) return
  window.scrollTo({ top: 0, behavior: 'smooth' })
  loadPage(target)
}

function reload() {
  resetPool()
  loadPage(1)
}

function switchFilter(key) {
  if (filter.value === key) return
  filter.value = key
  reload()
}

// 动态的删除（管理员或有权限的作者才会看到按钮，见 MomentItem 的 canManage）
async function onMomentDelete(m) {
  if (!confirm('确定要删除这条动态吗？')) return
  try {
    await removeMoment(String(m.id))
    toast.success('删除成功')
    // 本地移除即可，不整页重排；总数同步减一
    items.value = items.value.filter((it) => !(it.type === 'moment' && it.data.id === m.id))
    totals.moment = Math.max(0, totals.moment - 1)
    // 删掉的是当前页最后一条时回退一页，避免停在空白页
    if (!pageItems.value.length && page.value > 1) {
      page.value = Math.max(1, Math.ceil(items.value.length / PAGE_SIZE))
    }
  } catch {}
}

// 参数可能来自「已在本页时再点一次发布入口」，所以除了首屏还要跟着 watch
watch(() => route.query.compose, openComposeFromQuery)

onMounted(() => {
  loadPlacards()
  reload()
  openComposeFromQuery()
})

onBeforeUnmount(() => {
  stop()
})
</script>

<style scoped>
.home {
  padding-bottom: 32px;
}

/* 公告：单行横幅（图标 + 标签 + 轮播内容 + 短条指示器） */
.placard-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 14px;
  margin-bottom: 16px;
  /* 左侧一层极淡的朱砂底色，强调但不抢正文 */
  background: linear-gradient(90deg, var(--accent-wash), transparent 42%), var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  transition: border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
}
.placard-card:hover {
  border-color: var(--primary-soft);
  box-shadow: var(--shadow);
}
/* 图标：小方块底色，与站内 tag / 徽标同一套圆角 */
.placard-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 26px;
  height: 26px;
  font-size: 14px;
  color: var(--primary);
  background: var(--accent-soft);
  border-radius: var(--radius-sm);
}
.placard-title {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: 0.02em;
}
.placard-divider {
  flex-shrink: 0;
  width: 1px;
  height: 14px;
  background: var(--border);
}
.placard-body {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}
.placard-link {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--text-soft);
  line-height: 1.6;
  cursor: pointer;
}
.placard-link:hover .placard-name {
  color: var(--primary);
}
/* 类型圆点：颜色由 utils/placard.js 的行内样式给出（避免各页面配色漂移） */
.placard-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary);
}
.placard-name {
  flex-shrink: 0;
  max-width: 40%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  color: var(--text);
  transition: color var(--dur) var(--ease);
}
.placard-content {
  flex: 1;
  min-width: 0;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.placard-content :deep(p) {
  display: inline;
  margin: 0;
}
/* 指示器：短条，当前项拉长 —— 比小圆点更易瞄准 */
.placard-dots {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
}
.placard-dot-btn {
  width: 6px;
  height: 6px;
  padding: 0;
  border: none;
  border-radius: 999px;
  background: var(--text-muted);
  opacity: 0.35;
  cursor: pointer;
  transition: width var(--dur) var(--ease), opacity var(--dur) var(--ease),
    background var(--dur) var(--ease);
}
.placard-dot-btn:hover {
  opacity: 0.7;
}
.placard-dot-btn.is-active {
  width: 16px;
  opacity: 1;
  background: var(--primary);
}
/* 淡入淡出轮播动画 */
.placard-fade-enter-active,
.placard-fade-leave-active {
  transition: opacity 0.4s ease;
}
.placard-fade-enter-from,
.placard-fade-leave-to {
  opacity: 0;
}

/* ---------- 发现页 ---------- */
/* 类型筛选：胶囊分段控件 */
.feed-filter {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  border-radius: 999px;
  background: var(--bg-muted);
}
.feed-filter-item {
  padding: 3px 12px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.7;
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.feed-filter-item:hover {
  color: var(--text);
}
.feed-filter-item.is-active {
  color: #fff;
  background: var(--primary);
}

/* 「发现」标题右侧：统计 + 筛选同一行，编号用等宽数字避免跳动 */
.feed-extra {
  display: flex;
  align-items: center;
  gap: 12px;
}
.feed-summary {
  font-size: 12px;
  color: var(--text-light, var(--text-muted));
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.feed {
  display: flex;
  flex-direction: column;
}

/* 骨架屏仍是「通栏列表行」，这里覆盖成卡片，与文章 / 动态卡片对齐 */
.feed :deep(div.skeleton-article) {
  padding: 20px;
  margin-bottom: 16px;
  /* 与文章 / 动态卡片同一套：浅描边 + 轻阴影 */
  border: 1px solid var(--border-soft);
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}
/* MomentItem 自带浅描边卡片：只补一层同款阴影，描边保留（跟随后台配色的 --line-soft） */
.feed :deep(.moment-card) {
  box-shadow: var(--shadow-sm);
  transition: box-shadow var(--dur) var(--ease);
}
/* 悬停与文章卡片同一表现：只加阴影，不做位移（列表滚动时更稳） */
.feed :deep(.moment-card:hover) {
  box-shadow: var(--shadow);
}

/* 翻页时的轻提示：分页后旧内容会留到新数据回来为止，给个反馈避免「点了没反应」 */
.feed-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 0;
  font-size: 13px;
  color: var(--text-muted);
}

/* 公告详情弹窗 */
.placard-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}
.placard-dialog {
  width: 100%;
  max-width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.25);
  overflow: hidden;
}
.placard-dialog-head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 16px 18px 12px;
  border-bottom: 1px solid var(--border);
}
/* 类型小标签：文字色 + 同色淡底由 utils/placard.js 的行内样式给出 */
.placard-dialog-tag {
  flex-shrink: 0;
  margin-top: 2px;
  padding: 1px 8px;
  font-size: 11px;
  line-height: 1.6;
  border-radius: 999px;
  white-space: nowrap;
}
.placard-dialog-title {
  flex: 1;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  line-height: 1.4;
}
.placard-dialog-close {
  flex-shrink: 0;
  border: none;
  background: none;
  color: var(--text-muted);
  font-size: 15px;
  cursor: pointer;
  padding: 2px 4px;
}
.placard-dialog-close:hover {
  color: var(--text);
}
.placard-dialog-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 18px;
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-soft);
  white-space: pre-line;
  word-break: break-word;
}
.placard-dialog-body :deep(img) {
  max-width: 100%;
  border-radius: 6px;
}
.placard-dialog-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 18px;
  border-top: 1px solid var(--border);
}
.placard-dialog-date {
  font-size: 12px;
  color: var(--text-muted);
}
.placard-modal-enter-active,
.placard-modal-leave-active {
  transition: opacity 0.25s ease;
}
.placard-modal-enter-from,
.placard-modal-leave-to {
  opacity: 0;
}
.placard-modal-enter-active .placard-dialog,
.placard-modal-leave-active .placard-dialog {
  transition: transform 0.25s ease;
}
.placard-modal-enter-from .placard-dialog,
.placard-modal-leave-to .placard-dialog {
  transform: translateY(12px) scale(0.98);
}

/* ---------- 首页发布动态：入口条（点击弹窗），间距与公告一致 24px ---------- */
.compose {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 12px 16px;
  margin-bottom: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  color: var(--text-muted);
  font-family: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  transition: border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease),
    transform var(--dur) var(--ease);
}
.compose:hover {
  border-color: var(--primary-soft);
  box-shadow: var(--shadow);
  transform: translateY(-1px);
}
.compose:active {
  transform: none;
}
.compose-avatar {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  box-shadow: 0 0 0 2px var(--bg-card), 0 0 0 3px var(--border);
}
.compose-placeholder {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 右侧按钮：图标 + 文案，与 .btn 的 inline-flex 对齐 */
.compose-btn {
  flex-shrink: 0;
  gap: 6px;
}
.compose-btn i {
  font-size: 13px;
}
/* 未登录：整条即登录入口 */
.compose-guest {
  color: var(--text-muted);
  text-decoration: none;
}
.compose-guest-icon {
  font-size: 24px;
  color: var(--text-light, var(--text-muted));
}
.compose-guest-text {
  flex: 1;
  min-width: 0;
}
@media (max-width: 640px) {
  .compose {
    padding: 10px 12px;
    margin-bottom: 18px;
    gap: 10px;
  }
  /* 窄屏：公告只留标题（摘要截断后没有信息量），统计数字让位给筛选 */
  .placard-card {
    gap: 8px;
    padding: 8px 12px;
    margin-bottom: 12px;
  }
  .placard-content,
  .placard-divider {
    display: none;
  }
  .placard-name {
    max-width: none;
  }
  .feed-summary {
    display: none;
  }
  .feed-extra {
    gap: 0;
  }
}
</style>
