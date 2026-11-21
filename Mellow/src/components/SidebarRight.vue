<template>
  <aside v-if="enabled && modules.length" class="sidebar">
    <template v-for="module in modules" :key="module.id">
      <!-- 用户卡片 -->
      <SidebarUserCard v-if="module.type === 'user'" />

      <!-- 未读通知（未登录不渲染，与改造前一致） -->
      <SidebarNotifyCard
        v-else-if="module.type === 'notify' && userStore.isLogged"
        :title="module.title"
        :icon="module.icon"
        :more-text="module.moreText"
        :more-url="module.moreUrl"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
      />

      <!-- 文章列表（默认随机文章） -->
      <SidebarArticleRank
        v-else-if="module.type === 'articleRank'"
        :title="module.title"
        :icon="module.icon"
        :more-text="module.moreText"
        :more-url="module.moreUrl"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :items="randomArticles"
      />

      <!-- 最新评论 -->
      <SidebarCommentList
        v-else-if="module.type === 'comments'"
        :title="module.title"
        :icon="module.icon"
        :more-text="module.moreText"
        :more-url="module.moreUrl"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :items="latestComments"
      />

      <!-- 分类导航 -->
      <SidebarTagCloud
        v-else-if="module.type === 'categories'"
        :title="module.title"
        :icon="module.icon"
        :more-text="module.moreText"
        :more-url="module.moreUrl"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :items="groups"
        base-path="/category"
        empty-text="暂无分类"
      />

      <!-- 话题云 -->
      <SidebarTagCloud
        v-else-if="module.type === 'topics'"
        :title="module.title"
        :icon="module.icon"
        :more-text="module.moreText"
        :more-url="module.moreUrl"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :items="tags"
        base-path="/topic"
        empty-text="暂无话题"
      />

      <!-- 文章归档 -->
      <SidebarArchiveCard
        v-else-if="module.type === 'archive'"
        :title="module.title"
        :icon="module.icon"
        :more-text="module.moreText"
        :more-url="module.moreUrl"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :total="totalCount"
      />

      <!-- 站点信息 -->
      <SidebarStatsCard
        v-else-if="module.type === 'stats'"
        :title="module.title"
        :icon="module.icon"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :stats="stats"
        :tag-count="tagCount"
        :runtime-days="runtimeDays"
        :site-date-text="siteDateText"
      />

      <!-- 自定义链接 -->
      <SidebarLinkList
        v-else-if="module.type === 'links'"
        :title="module.title"
        :icon="module.icon"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :items="module.items"
        empty-text="暂无链接"
      />

      <!-- 自定义内容（任意 HTML） -->
      <SidebarHtmlCard
        v-else-if="module.type === 'html'"
        :title="module.title"
        :icon="module.icon"
        :collapsible="module.collapsible"
        :default-collapsed="module.collapsed"
        :store-key="module.id"
        :content="module.content"
      />
    </template>
  </aside>
</template>

<script setup>
import { computed, reactive, ref, watch, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { useSiteStore } from '@/stores/site'
import { getArticleGroups } from '@/api/article'
import { call } from '@/api/request'
import { normalizeSidebarConfig } from '@/utils/sidebar'
import { contentLink } from '@/utils/notification'
import { pickCommentAuthor } from '@/utils/helper'
import { renderEmoji } from '@/utils/emoji'
import { thumbUrl } from '@/utils/image'

import SidebarUserCard from '@/components/sidebar/SidebarUserCard.vue'
import SidebarNotifyCard from '@/components/sidebar/SidebarNotifyCard.vue'
import SidebarArticleRank from '@/components/sidebar/SidebarArticleRank.vue'
import SidebarCommentList from '@/components/sidebar/SidebarCommentList.vue'
import SidebarTagCloud from '@/components/sidebar/SidebarTagCloud.vue'
import SidebarArchiveCard from '@/components/sidebar/SidebarArchiveCard.vue'
import SidebarStatsCard from '@/components/sidebar/SidebarStatsCard.vue'
import SidebarLinkList from '@/components/sidebar/SidebarLinkList.vue'
import SidebarHtmlCard from '@/components/sidebar/SidebarHtmlCard.vue'

/**
 * 右侧边栏
 *
 * 模块、顺序、标题、条数、显隐全部来自站点配置 Mellow_functions.sidebar
 * （后台「系统设置 → 网站设置 → 个性侧边栏」），前台只负责按配置渲染：
 *   1. utils/sidebar.js 归一化配置，未配置时回退到默认 8 个模块（与改造前一致）
 *   2. 只为启用的模块发请求（改造前是全量请求，关掉的卡片也在打接口）
 *   3. 每个模块一个组件，样式与交互各自维护
 *
 * 说明：模块标题 / 图标留空时由各组件按类型给默认值；话题、分类等列表的条数
 * 取该模块配置的 limit，多个同类模块取最大值。
 */

const userStore = useUserStore()
const notif = useNotificationStore()
const siteStore = useSiteStore()

/* ---------------- 配置 ---------------- */
const sidebar = computed(() => normalizeSidebarConfig(siteStore.config?.sidebar))
const enabled = computed(() => sidebar.value.enabled)
const modules = computed(() => sidebar.value.modules.filter((item) => item.visible !== false))

/** 某类型模块的最大条数（同类型配置多个模块时取最大，避免重复请求） */
function limitOf(type, fallback) {
  const list = modules.value.filter((item) => item.type === type)
  if (!list.length) return 0
  return Math.max(...list.map((item) => Number(item.limit) || fallback), fallback)
}

const hasType = (type) => modules.value.some((item) => item.type === type)

/* ---------------- 数据 ---------------- */
const groups = ref([])
const tags = ref([])
const tagCount = ref(0)
const totalCount = ref(0)
const stats = ref({ article: 0, moment: 0, link: 0 })
const startTime = ref(Date.now())
const siteDateText = ref('')
const randomArticles = ref([])
const latestComments = ref([])

/** 已按某个条数加载过，避免配置没变时重复请求 */
const loaded = reactive({
  articleRank: 0,
  comments: 0,
  categories: 0,
  topics: 0,
  stats: false
})

const commentColors = ['#c7482a', '#2e7a5c', '#c08a2e', '#4a90e2', '#d9544d', '#9b59b6']

function colorFor(str) {
  let hash = 0
  for (let i = 0; i < (str || '').length; i++) hash = str.charCodeAt(i) + ((hash << 5) - hash)
  return commentColors[Math.abs(hash) % commentColors.length]
}

function stripHtml(html) {
  return (html || '').replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
}

// 生成评论摘要：以 [emoji:url] 占位符为整体单位截断，避免把 URL 切碎导致图片破损
function buildSummary(content, max = 30) {
  const plain = stripHtml(content).trim()
  if (!plain) return ''
  const parts = plain.split(/(\[emoji:[^\]]+\])/g)
  // 纯表情评论：直接完整渲染，不受长度限制
  const emojiOnly = parts.every((p) => !p || p.startsWith('[emoji:'))
  if (emojiOnly) return renderEmoji(plain)
  let out = ''
  for (const p of parts) {
    if (!p) continue
    if (out.length + p.length > max) {
      // 占位符被切断则直接丢弃，不显示半截图片
      if (p.startsWith('[emoji:')) continue
      out += p.slice(0, Math.max(0, max - out.length))
      break
    }
    out += p
  }
  out = out.trim()
  return out ? renderEmoji(out) : ''
}

function parseTime(t) {
  if (!t) return null
  if (typeof t === 'number') return t > 1e10 ? t / 1000 : t
  const d = new Date(t)
  return isNaN(d) ? null : d.getTime() / 1000
}

function formatMd(t) {
  const ts = parseTime(t)
  if (!ts) return ''
  const d = new Date(ts * 1000)
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${m}/${day}`
}

async function loadRandomArticles(limit) {
  if (loaded.articleRank === limit) return
  try {
    const res = await call('article', 'rand', {
      method: 'GET',
      params: { limit, field: 'id,title' }
    })
    randomArticles.value = Array.isArray(res?.data) ? res.data : (res?.data?.data || [])
    loaded.articleRank = limit
  } catch {
    randomArticles.value = []
  }
}

async function loadLatestComments(limit) {
  if (loaded.comments === limit) return
  try {
    const res = await call('comment', 'all', {
      method: 'GET',
      params: { limit, order: 'create_time desc' }
    })
    const data = res?.data?.data || []
    latestComments.value = data.map((c) => {
      const author = pickCommentAuthor(c)
      return {
        ...c,
        _nickname: author.nickname || '匿名',
        _avatar: author.avatar || '',
        _thumb: thumbUrl(author.avatar || '', 44),
        _summary: buildSummary(c.content, 30),
        _color: colorFor(author.nickname || String(c.uid)),
        _date: formatMd(c.create_time),
        // 与消息中心同一套内容地址规则（见 utils/notification.js）
        _link: contentLink(c.bind_type, c.bind_id, c.id) || '/archives'
      }
    })
    loaded.comments = limit
  } catch {
    latestComments.value = []
  }
}

async function loadGroups(limit) {
  if (loaded.categories === limit) return
  try {
    const res = await getArticleGroups()
    const list = res.data?.data || []
    // 基于 pid 构建层级，按 pid 排序（pid=0 为顶级）
    const flat = []
    const walk = (parentId) => {
      list
        .filter((n) => n.pid === parentId)
        .forEach((n) => {
          flat.push({ id: n.id, name: n.name })
          walk(n.id)
        })
    }
    walk(0)
    // 处理孤儿节点
    list
      .filter((n) => !flat.find((f) => f.id === n.id))
      .forEach((n) => flat.push({ id: n.id, name: n.name }))
    groups.value = flat.slice(0, limit)
    loaded.categories = limit
  } catch {
    groups.value = []
  }
}

async function loadTags(limit) {
  if (loaded.topics === limit) return
  try {
    const res = await call('tags', 'all', {
      method: 'GET',
      params: { limit, order: 'create_time desc', field: 'id,name' }
    })
    tags.value = res.data?.data || []
    loaded.topics = limit
  } catch {}
}

// 站点信息：建站日期（来自 Mellow_functions 配置的 date 字段，秒级时间戳）+ 话题总数
async function loadSiteInfo() {
  try {
    const cfg = (await siteStore.load()) || {}
    const dateTs = cfg?.date ? Number(cfg.date) : 0
    if (dateTs > 0) {
      startTime.value = dateTs * 1000
      const d = new Date(dateTs * 1000)
      const y = d.getFullYear()
      const m = String(d.getMonth() + 1).padStart(2, '0')
      const day = String(d.getDate()).padStart(2, '0')
      siteDateText.value = `${y}-${m}-${day}`
    }
  } catch {}
  try {
    const res = await call('tags', 'count', { method: 'GET' })
    tagCount.value = res?.data?.data || res?.data || 0
  } catch {}
}

async function loadStats() {
  if (loaded.stats) return
  try {
    const [a, m, l] = await Promise.all([
      call('article', 'count', { method: 'GET', params: { where: { audit: 1, status: 1 } } }),
      call('moments', 'count', { method: 'GET', params: { where: { audit: 1, status: 1 } } }),
      call('links', 'count', { method: 'GET' })
    ])
    stats.value = {
      article: a.data || 0,
      moment: m.data || 0,
      link: l.data || 0
    }
    totalCount.value = a.data || 0
    loaded.stats = true
  } catch {}
}

const runtimeDays = computed(() => {
  const diff = Date.now() - startTime.value
  return Math.floor(diff / (1000 * 60 * 60 * 24)) + 1
})

/* ---------------- 按需加载 ---------------- */
function loadNeeded() {
  if (!enabled.value) return

  const articleLimit = limitOf('articleRank', 6)
  if (articleLimit > 0) loadRandomArticles(articleLimit)

  const commentLimit = limitOf('comments', 6)
  if (commentLimit > 0) loadLatestComments(commentLimit)

  const categoryLimit = limitOf('categories', 20)
  if (categoryLimit > 0) loadGroups(categoryLimit)

  const topicLimit = limitOf('topics', 20)
  if (topicLimit > 0) loadTags(topicLimit)

  if (hasType('archive') || hasType('stats')) {
    loadStats()
    loadSiteInfo()
  }
}

// 配置就绪 / 改动后重新评估需要哪些数据（只补缺，不重复打接口）
watch(
  () => `${enabled.value}|${modules.value.map((item) => `${item.type}:${item.limit}:${item.visible}`).join('|')}`,
  () => loadNeeded(),
  { immediate: true }
)

watch(
  () => userStore.isLogged,
  (v) => {
    if (v) notif.startPolling()
    else notif.stopPolling()
  }
)

onMounted(() => {
  if (!siteStore.loaded) siteStore.load()
  if (userStore.isLogged) notif.startPolling()
})
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
