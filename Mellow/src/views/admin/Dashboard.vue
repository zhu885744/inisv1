<template>
  <div class="admin-dashboard">
    <!-- 概览横幅：站点 / 版本 / 服务器时间 -->
    <section class="hero">
      <div class="hero-main">
        <div class="hero-user">
          <img class="hero-avatar" :src="avatar" :alt="nickname" @error="onAvatarError" loading="lazy" />
          <div class="hero-line">
            <div class="hero-title">你好，{{ nickname }}</div>
            <div class="hero-sub">{{ siteTitle }} · 管理后台</div>
          </div>
        </div>
        <div class="hero-actions">
          <RouterLink to="/admin/article/write" class="btn btn-sm btn-primary">
            <i class="bi bi-pencil-square" /> 撰写文章
          </RouterLink>
          <RouterLink to="/admin/security/pages" class="btn btn-sm">
            <i class="bi bi-layout-text-window" /> 菜单权限
          </RouterLink>
          <RouterLink to="/admin/system" class="btn btn-sm">
            <i class="bi bi-gear" /> 系统配置
          </RouterLink>
        </div>
      </div>

      <div class="hero-meta">
        <span class="meta-item" title="后端程序版本">
          <i class="bi bi-box-seam" /> INIS
          <b>{{ version.inis || '—' }}</b>
          <em v-if="release.status === 'update'" class="meta-update">可更新 {{ release.tag_name }}</em>
          <button
            type="button"
            class="meta-refresh"
            :class="{ 'is-loading': release.status === 'loading' }"
            :disabled="release.status === 'loading' || !version.inis"
            :title="release.status === 'error' ? '上次检测失败，点击重试' : '检查更新'"
            aria-label="检查更新"
            @click="checkUpdate(true)"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
        </span>
        <span class="meta-item" title="当前主题版本">
          <i class="bi bi-palette" /> Mellow
          <b>{{ appVersion }}</b>
        </span>
        <span class="meta-item" title="后端 Go 运行环境版本">
          <i class="bi bi-cpu" /> Go
          <b>{{ version.go || '—' }}</b>
        </span>
        <span class="meta-item" title="服务器当前时间">
          <i class="bi bi-clock-history" /> 服务器时间
          <b>{{ serverTimeText || '—' }}</b>
        </span>
      </div>
    </section>

    <!-- 版本更新：远端 tag_name 高于本地 INIS 版本时提示（只检测与跳转，不做在线更新） -->
    <section v-if="release.status === 'update'" class="update-banner">
      <span class="update-icon"><i class="bi bi-cloud-arrow-down" /></span>
      <div class="update-main">
        <div class="update-head">
          <span class="update-title">发现新版本 <b>{{ release.tag_name }}</b></span>
          <span class="update-current">当前 INIS {{ version.inis }}</span>
          <a :href="release.html_url" class="update-link" target="_blank" rel="noopener noreferrer">
            查看详情 <i class="bi bi-box-arrow-up-right" />
          </a>
        </div>
        <div v-if="releaseNotesHtml" class="update-notes" v-html="releaseNotesHtml"></div>
        <div v-if="release.published_at" class="update-time">
          <i class="bi bi-clock" /> {{ formatReleaseDate(release.published_at) }} 发布
        </div>
      </div>
    </section>

    <!-- 待办：有未处理事项时高亮 -->
    <section class="todo-row">
      <RouterLink
        v-for="t in todos"
        :key="t.key"
        :to="t.to"
        class="todo-item"
        :class="{ 'is-hot': !t.loading && t.count > 0 }"
      >
        <i :class="t.icon" />
        <span class="todo-label">{{ t.label }}</span>
        <span class="todo-count">{{ t.loading ? '···' : format(t.count) }}</span>
        <i class="bi bi-chevron-right todo-arrow" />
      </RouterLink>
    </section>

    <!-- 快捷跳转：与左侧菜单同一份分组 / 权限口径（见 utils/admin-menu.js） -->
    <section class="shortcut">
      <header class="shortcut-head">
        <div class="shortcut-title">
          <i class="bi bi-lightning-charge-fill" />
          <span>快捷跳转</span>
          <span class="shortcut-hint">共 {{ shortcutTotal }} 个功能入口</span>
        </div>
        <div class="shortcut-search">
          <i class="bi bi-search" />
          <FormInput v-model="keyword" type="search" placeholder="搜索功能" />
          <button v-if="keyword" type="button" title="清空" @click="keyword = ''">
            <i class="bi bi-x-lg" />
          </button>
        </div>
      </header>

      <!-- 最近访问：由 Layout 的路由监听记录（子路由已归一到菜单页） -->
      <div v-if="!keyword.trim() && recentItems.length" class="recent-row">
        <span class="recent-label"><i class="bi bi-clock-history" /> 最近访问</span>
        <RouterLink v-for="item in recentItems" :key="item.path" :to="item.path" class="recent-chip">
          <span v-if="item.isSvg" class="shortcut-icon" v-html="item.svg" />
          <i v-else :class="item.icon" />
          {{ item.name }}
        </RouterLink>
      </div>

      <div v-if="shortcutGroups.length" class="shortcut-groups">
        <div v-for="group in shortcutGroups" :key="group.name" class="shortcut-group">
          <div class="shortcut-group-label">{{ group.label }}</div>
          <div class="shortcut-grid">
            <RouterLink v-for="child in group.children" :key="child.path" :to="child.path" class="shortcut-item">
              <span class="shortcut-icon">
                <span v-if="child.isSvg" v-html="child.svg" />
                <i v-else :class="child.icon" />
              </span>
              <span class="shortcut-name" :title="child.name">{{ child.name }}</span>
            </RouterLink>
          </div>
        </div>
      </div>
      <div v-else class="shortcut-empty">
        <span v-if="!shortcutReady">菜单加载中…</span>
        <span v-else-if="shortcutTotal">没有匹配的功能，换个关键词试试</span>
        <span v-else>当前账号没有可访问的后台页面</span>
      </div>
    </section>

    <!-- 分组统计 -->
    <section v-for="group in groups" :key="group.title" class="stat-group">
      <header class="group-head">
        <i :class="group.icon" />
        <span class="group-title">{{ group.title }}</span>
        <span v-if="group.hint" class="group-hint">{{ group.hint }}</span>
      </header>
      <div class="stat-grid">
        <div v-for="s in group.items" :key="s.key" class="stat-card">
          <div class="stat-label">
            <i :class="s.icon" />
            <span :title="s.label">{{ s.label }}</span>
          </div>
          <div class="stat-value">
            <span v-if="s.loading">···</span>
            <span v-else-if="s.error" class="stat-error" :title="s.error">—</span>
            <span v-else>{{ format(s.value) }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">关于后台</div>
      <ul class="panel-tips">
        <li><i class="bi bi-check-circle" /> 统计卡片并行请求各模块的 <code>/count</code> 接口，失败项显示 <code>—</code> 并悬停查看原因</li>
        <li><i class="bi bi-check-circle" /> 左侧菜单与「菜单权限」由 <code>auth-pages</code> + <code>auth-group</code> 控制，按当前账号权限过滤</li>
        <li><i class="bi bi-check-circle" /> 评论、内容审核与编辑器等开关集中在「系统配置」中统一管理</li>
      </ul>
    </section>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * 后台概览（/admin/dashboard）
 *
 * 数据来源（已与后端 controller 对齐）：
 * - 各模块 <count> 接口返回 { code, data: 数字 }，data 直接是数量；
 * - goods/orders-all 返回 { data: { data: [...], count, page } }，数量取 count；
 * - integral/card-stats 返回 { data: { total, unused, used, expired, ... } }；
 * - 版本信息走 dev 类接口 /dev/info/version（公开，返回 { go, inis, text }）；
 * - 版本更新检测走 GitHub Releases API（公开、跨域，用原生 fetch + 8s 超时），拿 tag_name 与本地
 *   INIS 版本比对，仅提示并跳转 Release 页面，不做在线更新；远端结果缓存 1 小时。
 *   页面加载时自动检测（走缓存、失败静默），版本号后的刷新按钮可手动重检（跳过缓存、toast 反馈）。
 * 统计口径差异：article/comment/goods/users 的 count 支持 where 过滤，
 * 因此「待审核」「今日新增」用 where 传 JSON 字符串（后端 IWhere 支持字符串 JSON）。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import MarkdownIt from 'markdown-it'
import { call, devService } from '@/api/request'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import { useAuthPagesStore } from '@/stores/auth-pages'
import { ADMIN_PINNED, buildAdminMenus, buildAdminPinnedItems, getRecentAdminPaths } from '@/utils/admin-menu'
import { toast } from '@/utils/toast'
import pkg from '../../../package.json'

const userStore = useUserStore()
const siteStore = useSiteStore()
const authPagesStore = useAuthPagesStore()
const { user } = storeToRefs(userStore)

const appVersion = pkg.version || '—'

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/></svg>'

const nickname = computed(() => user.value?.nickname || '管理员')
const avatar = ref(user.value?.avatar || defaultAvatar)
const siteTitle = computed(() => siteStore.config?.title || 'INIS')

function onAvatarError(e) {
  e.target.src = defaultAvatar
}

// ---------- 版本 / 服务器时间 ----------
const version = reactive({ inis: '', go: '' })
const serverTimeText = ref('')

async function loadVersion() {
  try {
    // dev 类接口不走 /api 前缀，响应拦截器已剥离为 { code, msg, data }
    const { code, data } = await devService.get('/dev/info/version')
    if (code === 200 && data) {
      version.inis = data.inis || ''
      version.go = data.go || ''
    }
  } catch {
    /* 版本接口不可用时保持占位 */
  }

  // 本地 INIS 版本就绪后再比对远端 Release（失败不影响其余加载）
  checkUpdate()

  try {
    const { code, data } = await devService.get('/dev/info/time')
    if (code === 200 && data) {
      // 优先用后端本机格式化好的 date（避免按访客浏览器时区二次换算）
      if (data.date) {
        serverTimeText.value = data.date
        return
      }
      if (typeof data.unix === 'number') {
        serverTimeText.value = formatTime(data.unix * 1000)
        return
      }
    }
  } catch {
    /* 回退本地时间 */
  }
  serverTimeText.value = formatTime(Date.now())
}

function formatTime(ms) {
  const d = new Date(ms)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// ---------- 版本更新检测 ----------
// 仓库 inisv1 的 Release 对应后端 INIS 版本（facade.Version），
// 因此用「本地 INIS 版本」而不是主题 package.json 的 Mellow 版本去比对远端 tag_name。
const GITHUB_RELEASE_API = 'https://api.github.com/repos/zhu885744/inisv1/releases/latest'

// 未认证调用 GitHub API 限流 60 次/小时/IP，后台每次打开都请求很容易打满，
// 因此把远端结果缓存 1 小时（缓存的是远端 Release，本地版本每次照常比对）
const RELEASE_CACHE_KEY = 'mellow_github_release'
const RELEASE_CACHE_TTL = 60 * 60 * 1000

// status: idle(未检测) / loading / update(有新版本) / latest(已是最新) / error(检测失败)
const release = reactive({
  status: 'idle',
  tag_name: '',
  body: '',
  published_at: '',
  html_url: ''
})

// Release body 是 Markdown，这里用最普通的 markdown-it 渲染（不接代码高亮，
// 避免把 highlight.js 的 chunk 拖进后台首页）
const mdLite = new MarkdownIt({ html: false, linkify: true, breaks: true })
// 外链统一新窗口打开（与 utils/markdown.js 的正文渲染保持一致）
const mdLinkOpen = mdLite.renderer.rules.link_open
  || function (tokens, idx, options, env, self) { return self.renderToken(tokens, idx, options) }
mdLite.renderer.rules.link_open = function (tokens, idx, options, env, self) {
  const hrefIndex = tokens[idx].attrIndex('href')
  if (hrefIndex >= 0 && String(tokens[idx].attrs[hrefIndex][1]).startsWith('http')) {
    tokens[idx].attrSet('target', '_blank')
    tokens[idx].attrSet('rel', 'noopener noreferrer')
  }
  return mdLinkOpen(tokens, idx, options, env, self)
}
// 更新说明里的配图同样懒加载（与 utils/markdown.js 的正文渲染保持一致）
const mdImage = mdLite.renderer.rules.image
  || function (tokens, idx, options, env, self) { return self.renderToken(tokens, idx, options) }
mdLite.renderer.rules.image = function (tokens, idx, options, env, self) {
  tokens[idx].attrSet('loading', 'lazy')
  tokens[idx].attrSet('decoding', 'async')
  return mdImage(tokens, idx, options, env, self)
}

const releaseNotesHtml = computed(() => (release.body ? mdLite.render(release.body) : ''))

// 去掉 tag 的 v 前缀：v1.0.2 → 1.0.2（本地 version.inis 不带 v）
function normalizeVersion(value) {
  return String(value || '').trim().replace(/^[vV]/, '')
}

function parseVersion(value) {
  return normalizeVersion(value)
    // 兼容 1.0.4+build.7 / 1.0.4-rc1：只比主版本段（/releases/latest 不返回预发布）
    .split(/[-+]/)[0]
    .split('.')
    .map((item) => {
      const num = parseInt(item, 10)
      return Number.isNaN(num) ? 0 : num
    })
}

// 逐段比较点分数字，缺位按 0：1.0.3 > 1.0.2，1.0.2 == v1.0.2
function isNewer(remote, local) {
  const r = parseVersion(remote)
  const l = parseVersion(local)
  const len = Math.max(r.length, l.length)
  for (let i = 0; i < len; i++) {
    const rv = r[i] || 0
    const lv = l[i] || 0
    if (rv !== lv) return rv > lv
  }
  return false
}

// 读缓存：未超过 1 小时的才作数
function readReleaseCache() {
  try {
    const raw = localStorage.getItem(RELEASE_CACHE_KEY)
    if (!raw) return null
    const cached = JSON.parse(raw)
    if (!cached?.ts || Date.now() - cached.ts > RELEASE_CACHE_TTL) return null
    return cached.data || null
  } catch {
    // 缓存损坏 / localStorage 不可用（隐私模式）：当作没有缓存，正常请求
    return null
  }
}

function writeReleaseCache(data) {
  try {
    localStorage.setItem(RELEASE_CACHE_KEY, JSON.stringify({ ts: Date.now(), data }))
  } catch {
    /* 写不进去不影响本次展示 */
  }
}

// GitHub 在部分网络环境下会长时间挂起，超时直接放弃（静默降级为不提示）
const RELEASE_TIMEOUT = 8000

// 用原生 fetch：GitHub API 返回的是裸 JSON，不能走项目 axios 那套 { code, msg, data } 拦截器
async function fetchRelease() {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), RELEASE_TIMEOUT)
  try {
    const res = await fetch(GITHUB_RELEASE_API, {
      headers: {
        Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28'
      },
      signal: controller.signal
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    return {
      tag_name: data?.tag_name || '',
      body: data?.body || '',
      published_at: data?.published_at || '',
      html_url: data?.html_url || ''
    }
  } finally {
    clearTimeout(timer)
  }
}

// 并发去重：手动刷新时自动检测可能还没回来，避免重复打 GitHub
let checking = false

/**
 * 检测更新。
 * 自动检测（页面加载）走 1 小时缓存、出错静默；
 * 手动检测（点版本号后的刷新按钮）跳过缓存，并用 toast 给出结果，避免「点了没反应」。
 */
async function checkUpdate(manual = false) {
  // 本地版本未知时无从比对，保持 idle（不提示、不打扰）
  if (!version.inis || checking) return

  const prevStatus = release.status
  checking = true
  release.status = 'loading'

  try {
    const cached = manual ? null : readReleaseCache()
    const data = cached || (await fetchRelease())
    // 只缓存新鲜拉到的结果，避免把已有缓存的时间戳一直往后推
    if (!cached && data) writeReleaseCache(data)

    release.tag_name = data.tag_name
    release.body = data.body
    release.published_at = data.published_at
    release.html_url = data.html_url

    const remote = normalizeVersion(data.tag_name)
    const newer = !!remote && isNewer(remote, version.inis)
    release.status = newer ? 'update' : 'latest'

    if (manual) {
      if (newer) toast.success(`发现新版本 ${data.tag_name}`)
      else toast.success('当前已是最新版本')
    }
  } catch {
    // 失败不回退已有结论：之前已确认有更新就继续显示横幅，只是本次刷新失败
    release.status = prevStatus === 'update' ? 'update' : 'error'
    if (manual) toast.error('检查更新失败，GitHub 暂时无法访问')
  } finally {
    checking = false
  }
}

function formatReleaseDate(iso) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// ---------- 快捷跳转 ----------
// 菜单口径与侧边栏完全一致（utils/admin-menu.js）：分组顺序、权限过滤、图标解析都复用，
// 所以这里不需要再维护一份页面清单；新增后台页面时侧边栏与快捷入口会同时出现
const keyword = ref('')
const recentPaths = ref([])

const allMenus = computed(() => buildAdminMenus(user.value, authPagesStore.getFlat || []))

// 置顶入口（/admin/stats 等）：侧边栏里贴在「概览」下方，快捷跳转里作为第一个分组
const pinnedItems = computed(() => buildAdminPinnedItems(user.value, authPagesStore.getFlat || []))

// 页面清单是否已就绪：未就绪时不显示「没有可访问页面」，避免首屏闪一下空状态
const shortcutReady = computed(() => authPagesStore.loaded)

// 快捷跳转分组 = 置顶（概览下方那几个）+ 创作 / 管理 / 安全
const baseGroups = computed(() => {
  const pinned = pinnedItems.value.length
    ? [{ name: ADMIN_PINNED.name, label: ADMIN_PINNED.label, children: pinnedItems.value }]
    : []
  return [...pinned, ...allMenus.value]
})

const shortcutTotal = computed(() => baseGroups.value.reduce((sum, g) => sum + g.children.length, 0))

// 菜单 path → 菜单项：用于把「最近访问」里存的 path 还原成名称 + 图标
const menuIndex = computed(() => {
  const map = {}
  baseGroups.value.forEach((g) => g.children.forEach((c) => { map[c.path] = c }))
  return map
})

const recentItems = computed(() => recentPaths.value.map((path) => menuIndex.value[path]).filter(Boolean))

// 关键词过滤：匹配页面名 / 页面路径 / 分组名
const shortcutGroups = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return baseGroups.value
  return baseGroups.value
    .map((g) => ({
      ...g,
      children: g.children.filter((c) =>
        c.name.toLowerCase().includes(kw) || c.path.toLowerCase().includes(kw) || g.label.includes(kw)
      )
    }))
    .filter((g) => g.children.length)
})

// ---------- 统计定义 ----------
function startOfToday() {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return Math.floor(d.getTime() / 1000)
}

const C = {
  blue: '#3b82f6',
  amber: '#f59e0b',
  green: '#10b981',
  violet: '#8b5cf6',
  pink: '#ec4899',
  sky: '#0ea5e9',
  red: '#ef4444',
  teal: '#14b8a6'
}

const statGroupDefs = [
  {
    title: '内容',
    icon: 'bi bi-journal-text',
    items: [
      { key: 'article', label: '文章', icon: 'bi bi-file-earmark-text', color: 'var(--primary)', ctrl: 'article' },
      { key: 'moments', label: '动态', icon: 'bi bi-chat-square-quote', color: C.amber, ctrl: 'moments' },
      { key: 'pages', label: '独立页面', icon: 'bi bi-window', color: C.sky, ctrl: 'pages' },
      { key: 'comment', label: '评论', icon: 'bi bi-chat-square-text', color: C.blue, ctrl: 'comment' },
      { key: 'tags', label: '话题', icon: 'bi bi-tags', color: C.pink, ctrl: 'tags' },
      { key: 'articleGroup', label: '文章分类', icon: 'bi bi-collection', color: C.teal, ctrl: 'article-group' }
    ]
  },
  {
    title: '用户与会员',
    icon: 'bi bi-people',
    items: [
      { key: 'users', label: '用户', icon: 'bi bi-people', color: C.violet, ctrl: 'users' },
      { key: 'level', label: '等级', icon: 'bi bi-graph-up-arrow', color: C.green, ctrl: 'level' },
      { key: 'exp', label: '经验流水', icon: 'bi bi-star-half', color: C.amber, ctrl: 'exp' }
    ]
  },
  {
    title: '兑换与附件',
    icon: 'bi bi-bag',
    items: [
      { key: 'goods', label: '商品', icon: 'bi bi-bag', color: C.violet, ctrl: 'goods' },
      {
        key: 'orders',
        label: '兑换订单',
        icon: 'bi bi-receipt',
        color: C.blue,
        ctrl: 'goods',
        method: 'orders-all',
        params: { page: 1, limit: 1 },
        // orders-all 返回 { data: { data, count, page } }
        pick: (payload) => payload?.count ?? 0
      },
      {
        key: 'cards',
        label: '积分卡密',
        icon: 'bi bi-coin',
        color: C.amber,
        ctrl: 'integral',
        method: 'card-stats',
        pick: (payload) => payload?.total ?? 0
      },
      { key: 'attachment', label: '附件', icon: 'bi bi-folder2-open', color: C.green, ctrl: 'attachment' }
    ]
  },
  {
    title: '站点与安全',
    icon: 'bi bi-shield-lock',
    items: [
      { key: 'links', label: '友链', icon: 'bi bi-link-45deg', color: C.green, ctrl: 'links' },
      { key: 'linksGroup', label: '友链分组', icon: 'bi bi-diagram-3', color: C.teal, ctrl: 'links-group' },
      { key: 'banner', label: '轮播', icon: 'bi bi-images', color: C.red, ctrl: 'banner' },
      { key: 'placard', label: '公告', icon: 'bi bi-megaphone', color: C.teal, ctrl: 'placard' },
      { key: 'authPages', label: '后台页面', icon: 'bi bi-layout-text-window', color: C.sky, ctrl: 'auth-pages' },
      { key: 'apiKeys', label: '接口密钥', icon: 'bi bi-key', color: C.amber, ctrl: 'api-keys' }
    ]
  },
  {
    title: '今日新增',
    icon: 'bi bi-calendar-day',
    hint: '按服务器本地时间零点起算',
    items: [
      {
        key: 'todayArticle',
        label: '文章',
        icon: 'bi bi-file-earmark-plus',
        color: 'var(--primary)',
        ctrl: 'article',
        params: { where: JSON.stringify({ create_time: { $gte: startOfToday() } }) }
      },
      {
        key: 'todayComment',
        label: '评论',
        icon: 'bi bi-chat-dots',
        color: C.blue,
        ctrl: 'comment',
        params: { where: JSON.stringify({ create_time: { $gte: startOfToday() } }) }
      },
      {
        key: 'todayUsers',
        label: '新用户',
        icon: 'bi bi-person-plus',
        color: C.violet,
        ctrl: 'users',
        params: { where: JSON.stringify({ create_time: { $gte: startOfToday() } }) }
      }
    ]
  }
]

// 待办项：数量 > 0 时高亮，点击直达对应页面
const todoDefs = [
  {
    key: 'todoOrders',
    label: '待发货订单',
    icon: 'bi bi-truck',
    to: '/admin/goods',
    ctrl: 'goods',
    method: 'orders-all',
    // 后端 orders-all 用 utils.Is.Empty 判断 status，数字 0 会被当作空值跳过，
    // 必须传字符串 '0'（tinyint 与 '0' 比较由 MySQL 隐式转换）
    params: { page: 1, limit: 1, status: '0' },
    pick: (payload) => payload?.count ?? 0
  },
  {
    key: 'todoAudit',
    label: '待审核文章',
    icon: 'bi bi-clock-history',
    to: '/admin/article',
    ctrl: 'article',
    params: { where: JSON.stringify({ audit: 0, status: 1 }) }
  }
]

const groups = ref(
  statGroupDefs.map((g) => ({
    ...g,
    items: g.items.map((item) => ({ ...item, value: 0, error: '', loading: true }))
  }))
)
const todos = ref(todoDefs.map((t) => ({ ...t, count: 0, loading: true })))

function format(n) {
  return typeof n === 'number' ? n.toLocaleString('zh-CN') : n
}

// 从接口返回里取出数量：count 接口 data 直接是数字，其余用 pick 自定义解析
function resolveCount(item, payload) {
  if (typeof item.pick === 'function') return Number(item.pick(payload)) || 0
  if (typeof payload === 'number') return payload
  if (payload && typeof payload === 'object') {
    const value = payload.count ?? payload.total ?? payload.data ?? 0
    return Number(value) || 0
  }
  return 0
}

async function fetchCount(item) {
  const res = await call(item.ctrl, item.method || 'count', {
    method: 'GET',
    params: { ...(item.params || {}) }
  })
  return resolveCount(item, res?.data)
}

async function loadStats() {
  const tasks = []
  groups.value.forEach((group) => {
    group.items.forEach((item) => {
      tasks.push(
        (async () => {
          try {
            item.value = await fetchCount(item)
          } catch (e) {
            item.error = e?.msg || e?.response?.data?.msg || '获取失败'
          } finally {
            item.loading = false
          }
        })()
      )
    })
  })
  todos.value.forEach((item) => {
    tasks.push(
      (async () => {
        try {
          item.count = await fetchCount(item)
        } catch {
          /* 待办失败不提示，保持 0 */
        } finally {
          item.loading = false
        }
      })()
    )
  })
  await Promise.all(tasks)
}

onMounted(async () => {
  avatar.value = user.value?.avatar || defaultAvatar
  siteStore.load()
  loadVersion()
  loadStats()

  // 快捷跳转：等页面清单就绪后再取「最近访问」，避免首屏闪一下空列表
  await authPagesStore.ensureLoaded()
  recentPaths.value = getRecentAdminPaths()
})
</script>

<style scoped>
.admin-dashboard {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* ---------- 概览横幅 ---------- */
.hero {
  padding: 20px 22px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.hero-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 14px;
}
.hero-user {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.hero-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--bg-muted);
}
.hero-title {
  font-size: 18px;
  font-weight: 600;
  line-height: 1.3;
}
.hero-sub {
  margin-top: 2px;
  font-size: 13px;
  color: var(--text-muted);
}
.hero-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px dashed var(--border-soft);
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-item .bi {
  color: var(--primary);
}
.meta-item b {
  font-weight: 600;
  color: var(--text-soft);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
/* INIS 版本号后面的「可更新」徽章：仅在检测到新版本时出现 */
.meta-update {
  font-style: normal;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--gold-wash);
  border: 1px solid var(--gold-line);
  color: var(--warning);
  white-space: nowrap;
}
/* 版本号后的「检查更新」小按钮：点击强制拉取远端 Release（跳过 1 小时缓存） */
.meta-refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-left: -2px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--text-light);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.meta-refresh:hover:not(:disabled) {
  color: var(--primary);
  background: var(--bg-muted);
}
.meta-refresh:disabled {
  cursor: default;
  opacity: 0.55;
}
.meta-refresh.is-loading i {
  animation: meta-refresh-spin 0.9s linear infinite;
}
@keyframes meta-refresh-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ---------- 版本更新横幅 ---------- */
.update-banner {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 18px;
  background: var(--gold-wash);
  border: 1px solid var(--gold-line);
  border-radius: var(--radius);
}
.update-icon {
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: var(--bg-card);
  color: var(--warning);
  font-size: 19px;
}
.update-main {
  flex: 1;
  min-width: 0;
}
.update-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 12px;
}
.update-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.update-title b {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--warning);
}
.update-current {
  font-size: 12px;
  color: var(--text-muted);
}
.update-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--primary);
  text-decoration: none;
  transition: opacity 0.18s;
}
.update-link:hover {
  opacity: 0.75;
}
/* Release body 经 markdown-it 渲染后插入：v-html 的内容不带 scoped 标记，需用 :deep */
.update-notes {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  word-break: break-word;
}
.update-notes :deep(p) {
  margin: 0 0 6px;
}
.update-notes :deep(p:last-child) {
  margin-bottom: 0;
}
.update-notes :deep(h1),
.update-notes :deep(h2),
.update-notes :deep(h3),
.update-notes :deep(h4),
.update-notes :deep(h5),
.update-notes :deep(h6) {
  margin: 8px 0 4px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.update-notes :deep(ul),
.update-notes :deep(ol) {
  margin: 0 0 6px;
  padding-left: 20px;
}
.update-notes :deep(li) {
  margin: 2px 0;
}
.update-notes :deep(a) {
  color: var(--primary);
  text-decoration: none;
}
.update-notes :deep(a:hover) {
  text-decoration: underline;
}
.update-notes :deep(code) {
  padding: 1px 5px;
  background: var(--bg-card);
  border-radius: 4px;
  font-size: 12px;
}
.update-notes :deep(pre) {
  margin: 6px 0;
  padding: 10px 12px;
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  overflow-x: auto;
}
.update-notes :deep(pre code) {
  padding: 0;
  background: transparent;
}
.update-notes :deep(blockquote) {
  margin: 6px 0;
  padding-left: 10px;
  border-left: 3px solid var(--border);
  color: var(--text-muted);
}
.update-notes :deep(hr) {
  margin: 8px 0;
  border: none;
  border-top: 1px solid var(--border);
}
.update-notes :deep(img) {
  max-width: 100%;
}
.update-time {
  margin-top: 8px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-light);
}

/* ---------- 待办 ---------- */
.todo-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}
.todo-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: inherit;
  text-decoration: none;
  transition: border-color 0.2s, box-shadow 0.2s, transform 0.2s;
}
.todo-item:hover {
  border-color: var(--primary-soft);
  box-shadow: var(--shadow-sm);
  color: var(--text);
}
.todo-item:active {
  transform: translateY(1px);
}
.todo-item > .bi:first-child {
  font-size: 18px;
  color: var(--text-muted);
}
.todo-item.is-hot {
  border-color: var(--gold-line);
  background: var(--gold-wash);
}
.todo-item.is-hot > .bi:first-child {
  color: var(--warning);
}
.todo-label {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  color: var(--text-soft);
}
.todo-count {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-muted);
}
.todo-item.is-hot .todo-count {
  color: var(--warning);
}
.todo-arrow {
  font-size: 12px;
  color: var(--text-light);
}

/* ---------- 快捷跳转 ---------- */
.shortcut {
  padding: 16px 18px 18px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.shortcut-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 12px;
}
.shortcut-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
}
.shortcut-title .bi {
  color: var(--text-light);
}
.shortcut-hint {
  font-size: 12px;
  font-weight: 400;
  color: var(--text-light);
}
.shortcut-search {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  height: 30px;
  width: 200px;
  max-width: 100%;
  padding: 0 9px;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-light);
  font-size: 12px;
  transition: border-color 0.18s;
}
.shortcut-search:focus-within {
  border-color: var(--primary-soft);
}
.shortcut-search input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  outline: none;
  color: var(--text);
  font-size: 13px;
  font-family: inherit;
}
/* 去掉 type=search 的原生清除按钮，统一用自己的按钮 */
.shortcut-search input::-webkit-search-cancel-button {
  display: none;
}
.shortcut-search button {
  border: none;
  background: transparent;
  color: var(--text-light);
  font-size: 12px;
  line-height: 1;
  padding: 2px;
  cursor: pointer;
  transition: color 0.18s;
}
.shortcut-search button:hover {
  color: var(--primary);
}
.recent-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
  padding-bottom: 14px;
  border-bottom: 1px dashed var(--border-soft);
}
.recent-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-light);
}
.recent-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 11px;
  background: var(--bg-muted);
  border-radius: 999px;
  color: var(--text-soft);
  font-size: 12px;
  text-decoration: none;
  transition: background 0.18s, color 0.18s;
}
.recent-chip:hover {
  background: var(--accent-soft);
  color: var(--primary);
}
.recent-chip .shortcut-icon {
  font-size: 12px;
}
.shortcut-groups {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.shortcut-group-label {
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--text-light);
}
.shortcut-grid {
  display: grid;
  /* 纯文字入口，列可以更窄：一行能放下更多，不用翻着找 */
  grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
  gap: 6px;
}
.shortcut-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 9px;
  border-radius: 6px;
  color: var(--text-soft);
  font-size: 13px;
  text-decoration: none;
  transition: background 0.18s, color 0.18s;
}
.shortcut-item:hover {
  background: var(--bg-muted);
  color: var(--primary);
}
/* 入口不再配彩色图标块：13px 小图标，hover 时点亮 */
.shortcut-icon {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-light);
  font-size: 13px;
  transition: color 0.18s;
}
.shortcut-icon :deep(svg) {
  width: 14px;
  height: 14px;
}
.shortcut-item:hover .shortcut-icon {
  color: var(--primary);
}
.shortcut-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.shortcut-empty {
  padding: 6px 0;
  font-size: 13px;
  color: var(--text-muted);
}

/* ---------- 统计分组 ---------- */
.stat-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
}
.group-head .bi {
  color: var(--primary);
}
.group-hint {
  font-weight: 400;
  font-size: 12px;
  color: var(--text-light);
}
.stat-grid {
  display: grid;
  /* 小卡片自适应列数：窄屏也能一排放两三个，不用一路往下滚 */
  grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
  gap: 10px;
}
.stat-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: border-color 0.2s;
}
.stat-card:hover {
  border-color: var(--primary);
}
.stat-label {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 12px;
  color: var(--text-muted);
}
.stat-label i {
  flex-shrink: 0;
  font-size: 13px;
  color: var(--text-light);
  transition: color 0.2s;
}
.stat-card:hover .stat-label i {
  color: var(--primary);
}
.stat-label span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.stat-value {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.3;
  min-height: 26px;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.stat-error {
  color: var(--text-muted);
  font-weight: 400;
  cursor: help;
}

/* ---------- 说明面板 ---------- */
.panel {
  padding: 22px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.panel-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 10px;
}
.panel-tips {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.panel-tips li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--text-soft);
}
.panel-tips i {
  color: var(--primary);
}
.panel-tips code {
  background: var(--bg-muted);
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
}

@media (max-width: 640px) {
  /* 手机上固定两列，卡片本来就小，一列会滚很长 */
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .hero-actions {
    width: 100%;
  }
}
</style>
