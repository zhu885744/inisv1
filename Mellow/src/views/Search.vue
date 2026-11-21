<template>
  <div class="search-page">
    <!-- 搜索区：标题 + 大搜索框（提交按钮内嵌）+ 范围分段 -->
    <section class="search-hero">
      <header class="search-hero-head">
        <h1 class="search-title">搜索</h1>
        <p class="search-desc">文章、页面、话题、用户与动态，一次搜全站</p>
      </header>

      <form class="search-form" role="search" @submit.prevent="submit">
        <i class="bi bi-search search-form-icon" aria-hidden="true" />
        <FormInput
          ref="inputRef"
          v-model="keyword"
          class="search-input"
          type="search"
          :placeholder="placeholder"
          aria-label="搜索"
          autocomplete="off"
          @input="onInput"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
        />
        <button v-if="keyword" type="button" class="search-clear" aria-label="清空" @click="clearKeyword">
          <i class="bi bi-x-lg" />
        </button>
        <button type="submit" class="search-submit">
          <i class="bi bi-search" aria-hidden="true" />
          <span class="search-submit-text">搜索</span>
        </button>
      </form>

      <div class="search-scopes" role="tablist" aria-label="搜索范围">
        <button
          v-for="item in SCOPES"
          :key="item.key"
          type="button"
          class="scope-btn"
          :class="{ 'is-active': scope === item.key }"
          role="tab"
          :aria-selected="scope === item.key"
          @click="changeScope(item.key)"
        >
          {{ item.label }}
        </button>
      </div>
    </section>

    <!-- 未搜索：热门关键词 + 搜索历史（桌面两列，手机单列） -->
    <div v-if="!submitted" class="search-suggest">
      <section class="card suggest-card">
        <header class="block-head">
          <h2 class="block-title"><i class="bi bi-fire" /> 热门搜索</h2>
        </header>
        <div class="tag-list">
          <button
            v-for="(word, i) in HOT_SEARCHES"
            :key="i"
            class="search-tag"
            :class="{ 'is-top': i < 3 }"
            @click="useKeyword(word)"
          >
            <span class="tag-rank">{{ i + 1 }}</span>
            {{ word }}
          </button>
        </div>
      </section>

      <section v-if="history.length" class="card suggest-card">
        <header class="block-head">
          <h2 class="block-title"><i class="bi bi-clock-history" /> 搜索历史</h2>
          <button type="button" class="link-btn" @click="clearHistory">清除全部</button>
        </header>
        <div class="tag-list">
          <span v-for="(word, i) in history" :key="i" class="search-tag search-tag-history">
            <button type="button" class="search-tag-text" @click="useKeyword(word)">{{ word }}</button>
            <button type="button" class="search-tag-remove" aria-label="删除这条历史" @click="removeHistory(i)">
              <i class="bi bi-x" />
            </button>
          </span>
        </div>
      </section>
    </div>

    <!-- 搜索结果 -->
    <template v-else>
      <div class="result-head">
        <p class="result-head-text">
          <template v-if="loading">
            正在搜索 <span class="result-keyword">{{ submitted }}</span>…
          </template>
          <template v-else>
            找到 <strong>{{ results.length }}</strong> 条与
            <span class="result-keyword">{{ submitted }}</span>
            相关的内容
          </template>
        </p>
        <span class="result-hint">
          <kbd>↑</kbd><kbd>↓</kbd> 选择 · <kbd>Enter</kbd> 打开
        </span>
      </div>

      <!-- 加载骨架：与真实条目同构（缩略图 + 两行文字） -->
      <div v-if="loading" class="card result-skeleton">
        <div v-for="i in 4" :key="'sk-' + i" class="skeleton-item">
          <span class="skeleton sk-thumb" />
          <span class="sk-lines">
            <span class="skeleton sk-line is-title" />
            <span class="skeleton sk-line is-text" />
          </span>
        </div>
      </div>

      <div v-else-if="!results.length" class="card">
        <EmptyState icon="bi bi-search" text="没有找到相关内容，换个关键词试试" />
      </div>

      <!-- 按类型分组：组头在卡片外，列表本身是一张卡 -->
      <template v-else>
        <section v-for="group in groups" :key="group.key" class="result-group">
          <header class="group-head">
            <h3 class="group-title">
              <i :class="group.icon" /> {{ group.label }}
            </h3>
            <span class="group-count">{{ group.items.length }}</span>
          </header>
          <ul class="card result-list">
            <li
              v-for="item in group.items"
              :key="item.id || item._index"
              class="result-item"
              :class="{ 'is-selected': selected === item._index }"
              @mouseenter="selected = item._index"
              @click="goResult(item)"
            >
              <!-- 缩略图：文章封面 / 话题、用户、友链头像 / 动态首图；没有或加载失败时退回类型图标 -->
              <span class="result-thumb" :class="`is-${item._type}`">
                <img
                  v-if="thumbOf(item)"
                  :src="thumbOf(item)"
                  :alt="titlePlain(item)"
                  loading="lazy"
                  referrerpolicy="no-referrer"
                  @error="markThumbBroken(item)"
                />
                <i v-else :class="group.icon" aria-hidden="true" />
              </span>

              <div class="result-main">
                <div class="result-line">
                  <span class="result-badge" :class="`is-${item._type}`">{{ typeName(item._type) }}</span>
                  <span class="result-title" v-html="safeHtml(titleText(item))" />
                </div>

                <!-- 摘要 / 正文片段 / 简介（后端已做 <mark> 高亮，其余用户内容在此转义） -->
                <p v-if="summaryOf(item)" class="result-summary" v-html="safeHtml(summaryOf(item))" />

                <!-- 元信息：话题、地址、位置、阅读量、时间等 -->
                <div v-if="metaOf(item).length" class="result-meta">
                  <span v-for="(meta, i) in metaOf(item)" :key="i" class="result-meta-item">
                    <i :class="meta.icon" aria-hidden="true" />
                    <span v-html="safeHtml(meta.text)" />
                  </span>
                </div>
              </div>

              <i class="bi bi-chevron-right result-arrow" aria-hidden="true" />
            </li>
          </ul>
        </section>
      </template>
    </template>
  </div>
</template>

<script setup>
/**
 * 搜索（/search）
 *
 * 由原来的搜索弹窗改成独立页面：
 * - 关键词与搜索范围同步到地址栏（/search?q=xxx&scope=article），可分享、可刷新、可后退；
 * - scope=all 时并行查询全部类型并按「文章 / 页面 / 用户 / 友链 / 动态 / 话题」分组展示，
 *   指定范围时只查该类型；
 * - 键盘可用：↑↓ 选择、Enter 打开（Enter 在输入框内触发提交，选中项优先）；
 * - 后端接口：search/{article,pages,tags,links,users,moments}（type=common，无需登录）。
 */
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { escapeHtml } from '@/utils/emoji'
import { parseLiveItem } from '@/utils/livePhoto'
import FormInput from '@/components/form/FormInput.vue'
import EmptyState from '@/components/EmptyState.vue'
import { call } from '@/api/request'
import { fromNow } from '@/utils/time'

const route = useRoute()
const router = useRouter()

const STORAGE_KEY = 'mellow_search_history'

// 搜索范围：key 与后端 search 控制器的方法名对应（scope 是前端自己的名字）
const SCOPES = [
  { key: 'all', label: '全部', icon: 'bi bi-grid', api: '' },
  { key: 'article', label: '文章', icon: 'bi bi-file-text', api: 'article' },
  { key: 'page', label: '页面', icon: 'bi bi-file-earmark', api: 'pages' },
  { key: 'tag', label: '话题', icon: 'bi bi-tag', api: 'tags' },
  { key: 'links', label: '友链', icon: 'bi bi-link-45deg', api: 'links' },
  { key: 'users', label: '用户', icon: 'bi bi-person', api: 'users' },
  { key: 'moments', label: '动态', icon: 'bi bi-chat-square-text', api: 'moments' }
]

// 结果分组顺序（scope=all 时的合并顺序，与后端返回的字段命名一致）
const GROUP_ORDER = [
  { key: 'article', label: '文章', icon: 'bi bi-file-text' },
  { key: 'page', label: '页面', icon: 'bi bi-file-earmark' },
  { key: 'users', label: '用户', icon: 'bi bi-person' },
  { key: 'links', label: '友链', icon: 'bi bi-link-45deg' },
  { key: 'moments', label: '动态', icon: 'bi bi-chat-square-text' },
  { key: 'tag', label: '话题', icon: 'bi bi-tag' }
]

const HOT_SEARCHES = ['随记', '旅行', '博客', 'Vue', 'Node.js', '前端开发', '算法', '数据库']

const TYPE_NAMES = {
  article: '文章',
  page: '页面',
  tag: '话题',
  links: '友链',
  users: '用户',
  moments: '动态'
}

const PLACEHOLDER = {
  article: '搜索文章...',
  page: '搜索页面...',
  tag: '搜索话题...',
  links: '搜索友链...',
  users: '搜索用户...',
  moments: '搜索动态...'
}

const inputRef = ref(null)
const keyword = ref('')
// 已提交的关键词（与输入框解耦：输入过程中不清空上一轮结果）
const submitted = ref('')
const scope = ref('all')
const loading = ref(false)
const results = ref([])
const history = ref([])
const selected = ref(-1)

let debounceTimer = null

const placeholder = computed(
  () => PLACEHOLDER[scope.value] || '搜索文章、页面、话题、友链、用户或动态...'
)

// 按类型分组（results 是扁平数组，_index 用于键盘选中）
const groups = computed(() => {
  const map = new Map()
  results.value.forEach((item) => {
    if (!map.has(item._type)) map.set(item._type, [])
    map.get(item._type).push(item)
  })

  return GROUP_ORDER.filter((group) => map.has(group.key)).map((group) => ({
    ...group,
    items: map.get(group.key)
  }))
})

// ---------- 历史记录 ----------
function loadHistory() {
  try {
    const raw = JSON.parse(localStorage.getItem(STORAGE_KEY))
    history.value = Array.isArray(raw) ? raw : []
  } catch {
    history.value = []
  }
}

function saveHistory(word) {
  if (!word) return
  history.value = [word, ...history.value.filter((item) => item !== word)].slice(0, 10)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(history.value))
}

function clearHistory() {
  history.value = []
  localStorage.removeItem(STORAGE_KEY)
}

function removeHistory(index) {
  history.value.splice(index, 1)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(history.value))
}

// ---------- 交互 ----------
function onInput() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    if (keyword.value.trim()) runSearch(keyword.value.trim())
    else resetResults()
  }, 300)
}

function submit() {
  const word = keyword.value.trim()
  if (!word) {
    inputRef.value?.focus()
    return
  }
  // 输入框内回车：有选中项时优先打开选中项，否则按当前关键词搜索
  if (selected.value >= 0 && results.value[selected.value]) {
    goResult(results.value[selected.value])
    return
  }
  runSearch(word)
}

function useKeyword(word) {
  keyword.value = word
  runSearch(word)
}

function clearKeyword() {
  keyword.value = ''
  resetResults()
  nextTick(() => inputRef.value?.focus())
}

function resetResults() {
  submitted.value = ''
  results.value = []
  selected.value = -1
  syncQuery('', scope.value)
}

function changeScope(key) {
  if (scope.value === key) return
  scope.value = key
  selected.value = -1
  if (keyword.value.trim()) runSearch(keyword.value.trim())
  else syncQuery('', key)
}

function move(step) {
  if (!results.value.length) return
  const total = results.value.length
  selected.value = (selected.value + step + total) % total
}

// 把关键词 / 范围写进地址栏（replace：不污染后退历史，同时可分享、可刷新）
function syncQuery(word, nextScope) {
  const query = {}
  if (word) query.q = word
  if (nextScope && nextScope !== 'all') query.scope = nextScope
  const current = route.query
  if ((current.q || '') === (query.q || '') && (current.scope || '') === (query.scope || '')) return
  router.replace({ path: '/search', query })
}

// ---------- 搜索 ----------
async function runSearch(word) {
  submitted.value = word
  selected.value = -1
  loading.value = true
  syncQuery(word, scope.value)

  try {
    const list = scope.value === 'all' ? await searchAll(word) : await searchType(scope.value, word)
    // 补 _index 供键盘选中（分组后仍能定位到扁平数组里的位置）
    results.value = list.map((item, index) => ({ ...item, _index: index }))
    if (results.value.length) saveHistory(word)
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

// 全部范围：并行查询 + 合并去重（话题最多 3 条，避免刷屏）
async function searchAll(word) {
  const [articles, pages, tags, links, users, moments] = await Promise.all([
    searchType('article', word),
    searchType('page', word),
    searchType('tag', word),
    searchType('links', word),
    searchType('users', word),
    searchType('moments', word)
  ])

  const seen = new Set()
  const merged = []
  const push = (items, max) => {
    ;(items || []).slice(0, max || items.length).forEach((item) => {
      if (seen.has(item.id)) return
      seen.add(item.id)
      merged.push(item)
    })
  }

  push(articles)
  push(pages)
  push(users)
  push(links)
  push(moments)
  push(tags, 3)

  return merged
}

// 按范围查询：scope 是前端命名，api 是后端方法名
function apiOf(scopeKey) {
  return SCOPES.find((item) => item.key === scopeKey)?.api || scopeKey
}

async function searchType(scopeKey, word) {
  try {
    const res = await call('search', apiOf(scopeKey), {
      method: 'GET',
      params: { keyword: word, page: 1, limit: 50 }
    })
    // INIS 列表返回：data 可能直接是数组，也可能包在 data.data 里
    const arr = res.data?.data || res.data
    if (!Array.isArray(arr)) return []
    return arr.map((item) => ({ ...item, _type: scopeKey }))
  } catch {
    return []
  }
}

// ---------- 结果展示 ----------

// 图片加载失败的条目（退回类型图标，避免留下空白方块）
const brokenThumbs = ref(new Set())

function typeName(type) {
  return TYPE_NAMES[type] || '内容'
}

function titleText(item) {
  const type = item._type
  if (type === 'users' || type === 'links') return item.nickname || item.name || item.title || '未知'
  if (type === 'moments') return truncateHtml(item.content || '', 40) || '动态'
  return item.title || item.name || '未知'
}

// 纯文本标题（用于 img 的 alt / 无障碍）
function titlePlain(item) {
  return stripTags(titleText(item))
}

function stripTags(text) {
  return String(text || '').replace(/<[^>]*>/g, '')
}

// 逗号分隔的多图字段（covers / images 都是这种存法）取第一张
function firstImage(value) {
  if (!value) return ''
  const list = Array.isArray(value) ? value : String(value).split(',')
  // 实况图项是「图片::视频」，缩略图只能取图片那一段
  const found = list.map(parseLiveItem).find((item) => item.image)
  return found ? found.image : ''
}

function markThumbBroken(item) {
  brokenThumbs.value = new Set(brokenThumbs.value).add(item._index)
}

// 缩略图：文章 → 封面；话题 / 用户 / 友链 → 头像；动态 → 首图
function thumbOf(item) {
  if (brokenThumbs.value.has(item._index)) return ''

  switch (item._type) {
    case 'article':
      return firstImage(item.covers)
    case 'moments':
      return firstImage(item.images)
    case 'tag':
    case 'users':
    case 'links':
      return String(item.avatar || '').trim()
    default:
      return ''
  }
}

// 摘要 / 简介：搜索结果里最有信息量的那一段（保留后端返回的 <mark> 高亮）
//
// 文章优先用手写的 abstract，没有就用后端从正文里截的 snippet；
// 页面没有摘要，直接用 snippet（正文片段）。
function summaryOf(item) {
  const limit = 120
  switch (item._type) {
    case 'article':
      return truncateHtml(item.abstract || item.snippet || '', limit)
    case 'page':
      return truncateHtml(item.snippet || '', limit)
    case 'moments':
      return truncateHtml(item.content || '', limit)
    case 'tag':
    case 'users':
    case 'links':
      return truncateHtml(item.description || '', limit)
    default:
      return ''
  }
}

function hostOf(url) {
  try {
    return new URL(url).host
  } catch {
    return String(url || '')
  }
}

// 元信息：随类型展示不同维度（地址 / 位置 / 阅读量 / 时间 / 邮箱）
// 注：文章不展示话题（tags），搜索结果里话题噪声大且与摘要重复
function metaOf(item) {
  const meta = []
  const time = item.create_time ? fromNow(item.create_time) : ''

  // 除摘要外的高亮字段（如用户职位）也要保留 <mark>
  const mark = (text) => String(text || '').trim()

  switch (item._type) {
    case 'article':
      if (Number(item.views) > 0) meta.push({ icon: 'bi bi-eye', text: `${item.views} 阅读` })
      if (time) meta.push({ icon: 'bi bi-clock', text: time })
      break
    case 'page':
      if (item.key) meta.push({ icon: 'bi bi-link-45deg', text: `/${item.key}` })
      if (Number(item.views) > 0) meta.push({ icon: 'bi bi-eye', text: `${item.views} 阅读` })
      if (time) meta.push({ icon: 'bi bi-clock', text: time })
      break
    case 'users':
      if (item.title) meta.push({ icon: 'bi bi-award', text: mark(item.title) })
      if (item.email) meta.push({ icon: 'bi bi-envelope', text: item.email })
      break
    case 'links':
      if (item.url) meta.push({ icon: 'bi bi-box-arrow-up-right', text: hostOf(item.url) })
      break
    case 'moments': {
      if (item.location) meta.push({ icon: 'bi bi-geo-alt', text: mark(item.location) })
      const images = firstImage(item.images) ? String(item.images).split(',').filter(Boolean).length : 0
      if (images > 1) meta.push({ icon: 'bi bi-images', text: `${images} 张图` })
      if (time) meta.push({ icon: 'bi bi-clock', text: time })
      break
    }
    default:
      break
  }

  return meta
}

/**
 * 搜索结果安全化：整体转义，只放行后端注入的 `<mark>` 高亮
 *
 * 后端 search.go 的 highlightResult 只在命中处加 <mark>/</mark>，其余全是用户内容
 * （标题 / 摘要 / 动态正文 / 简介），且后端不做 XSS 清洗，所以必须在渲染前转义，
 * 否则一条 `<img src=x onerror=...>` 的动态被别人搜到就会执行。
 */
function safeHtml(text) {
  return escapeHtml(text).replace(/&lt;(\/?mark)&gt;/g, '<$1>')
}

// 截断带 <mark> 高亮的文本（保留标签闭合，避免截断破坏高亮）
function truncateHtml(text, max = 100) {
  if (!text) return ''
  const plain = text.replace(/<[^>]*>/g, '')
  if (plain.length <= max) return text

  let count = 0
  let out = ''
  let open = 0
  for (let i = 0; i < text.length && count < max; i++) {
    if (text[i] === '<') {
      const end = text.indexOf('>', i)
      if (end !== -1) {
        const tag = text.slice(i, end + 1)
        out += tag
        if (!tag.startsWith('</')) open++
        else open--
        i = end
        continue
      }
    }
    out += text[i]
    count++
  }
  while (open-- > 0) out += '</mark>'
  return `${out}...`
}

function goResult(item) {
  switch (item._type) {
    case 'article':
      if (item.id) router.push(`/archives/${item.id}`)
      break
    case 'page':
      if (item.key) router.push(`/${item.key}`)
      break
    case 'tag':
      if (item.id) router.push(`/topic/${item.id}`)
      break
    case 'links':
      // noopener：外链页面拿不到 window.opener，避免被反向跳转（tabnabbing）
      if (item.url) window.open(item.url, '_blank', 'noopener,noreferrer')
      break
    case 'users':
      if (item.id) router.push(`/author/${item.id}`)
      break
    case 'moments':
      // 动态列表页已下线，直接进动态详情（/moments/:id）
      if (item.id) router.push(`/moments/${item.id}`)
      break
  }
}

onMounted(() => {
  loadHistory()

  // 支持 /search?q=xxx&scope=article 直达（刷新、分享、后退都不丢状态）
  const word = String(route.query.q || '').trim()
  const queryScope = String(route.query.scope || 'all')
  if (SCOPES.some((item) => item.key === queryScope)) scope.value = queryScope

  if (word) {
    keyword.value = word
    runSearch(word)
  } else {
    nextTick(() => inputRef.value?.focus())
  }
})
</script>

<style scoped>
.search-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
}

/* ---------- 搜索区 ---------- */
/* 一整块「搜索台」：左上角淡朱砂光晕，与站内其它纸感卡片区分层级 */
.search-hero {
  position: relative;
  overflow: hidden;
  padding: 22px 20px 18px;
  background:
    radial-gradient(120% 150% at 0% 0%, var(--accent-wash), transparent 58%),
    var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}
.search-hero-head {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 4px 10px;
}
.search-title {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 22px;
  font-weight: 900;
  letter-spacing: 0.02em;
  color: var(--text);
}
.search-desc {
  margin: 0;
  font-size: 12px;
  color: var(--text-muted);
}

/* 搜索框：胶囊形，提交按钮内嵌在右端（手机端也不换行） */
.search-form {
  position: relative;
  display: flex;
  align-items: center;
  margin-top: 16px;
}
.search-form-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  z-index: 1;
  font-size: 15px;
  color: var(--text-muted);
  pointer-events: none;
}
.search-input {
  width: 100%;
  height: 50px;
  padding: 0 128px 0 42px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-soft);
  color: var(--text);
  font-size: 15px;
  transition: border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease),
    background-color var(--dur) var(--ease);
}
/* 隐藏 type=search 的原生清除按钮：否则会和自定义的 ❌ 同时出现（两个叉）
   各内核的伪元素都写一遍，避免浏览器差异 */
.search-input::-webkit-search-cancel-button,
.search-input::-webkit-search-decoration,
.search-input::-webkit-search-results-button,
.search-input::-webkit-search-results-decoration {
  -webkit-appearance: none;
  appearance: none;
  display: none;
}
.search-input::-ms-clear {
  display: none;
  width: 0;
  height: 0;
}
.search-input::placeholder {
  color: var(--text-light);
}
.search-input:focus {
  outline: none;
  background: var(--bg-card);
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.search-clear {
  position: absolute;
  right: 96px;
  top: 50%;
  transform: translateY(-50%);
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--text-muted);
  background: transparent;
  -webkit-tap-highlight-color: transparent;
}
.search-clear:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.search-submit {
  position: absolute;
  right: 5px;
  top: 50%;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  height: 40px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--primary), var(--primary-deep));
  color: #fff;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 3px 10px var(--accent-glow);
  transition: box-shadow var(--dur) var(--ease), filter var(--dur) var(--ease);
}
.search-submit:hover {
  box-shadow: 0 5px 16px var(--accent-glow);
  filter: brightness(1.04);
}
.search-submit:active {
  box-shadow: 0 2px 8px var(--accent-glow);
}
.search-submit i {
  font-size: 13px;
}

/* 范围切换：分段控件（与首页发现流同一套胶囊语言） */
.search-scopes {
  display: flex;
  gap: 2px;
  margin-top: 14px;
  padding: 3px;
  border-radius: 999px;
  background: var(--bg-muted);
  overflow-x: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.search-scopes::-webkit-scrollbar {
  display: none;
}
.scope-btn {
  flex: 1 0 auto;
  padding: 6px 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-muted);
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  white-space: nowrap;
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease),
    box-shadow var(--dur) var(--ease);
  -webkit-tap-highlight-color: transparent;
}
.scope-btn:hover {
  color: var(--text);
}
.scope-btn.is-active {
  color: var(--primary);
  font-weight: 600;
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
}

/* ---------- 未搜索时的推荐区（热门 / 历史） ---------- */
.search-suggest {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  align-items: start;
}
.suggest-card {
  padding: 18px 20px 16px;
}

/* ---------- 区块头 ---------- */
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.block-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 700;
  color: var(--text);
}
.block-title .bi {
  font-size: 14px;
  color: var(--primary);
}
.link-btn {
  font-size: 12px;
  color: var(--text-muted);
  font-family: inherit;
  background: none;
  border: none;
  cursor: pointer;
  transition: color var(--dur) var(--ease);
}
.link-btn:hover {
  color: var(--danger);
}

/* ---------- 关键词标签 ---------- */
.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.search-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 4px 14px;
  font-family: inherit;
  font-size: 13px;
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  color: var(--text-soft);
  background: var(--bg-soft);
  cursor: pointer;
  transition: color var(--dur) var(--ease), border-color var(--dur) var(--ease),
    background-color var(--dur) var(--ease);
  -webkit-tap-highlight-color: transparent;
}
.search-tag:hover {
  color: var(--primary);
  border-color: var(--primary-soft);
  background: var(--accent-wash);
}
/* 热搜排名：前三名用实心朱砂，其余淡底 */
.tag-rank {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  border-radius: 4px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
}
.search-tag.is-top .tag-rank {
  background: var(--primary);
  color: #fff;
}
.search-tag-history {
  padding-right: 6px;
}
.search-tag-text {
  font-family: inherit;
  font-size: inherit;
  color: inherit;
  background: none;
  border: none;
  cursor: pointer;
}
.search-tag-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-right: -4px;
  border-radius: 50%;
  font-size: 13px;
  line-height: 1;
  color: var(--text-light);
  background: none;
  border: none;
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease);
}
.search-tag-remove:hover {
  color: var(--danger);
  background: var(--bg-muted);
}

/* ---------- 结果 ---------- */
.result-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}
.result-head-text {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}
.result-head-text strong {
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
/* 关键词：淡朱砂底的小块，比加粗更醒目 */
.result-keyword {
  margin: 0 2px;
  padding: 1px 8px;
  border-radius: 6px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-weight: 600;
  word-break: break-all;
}
.result-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-light);
}
.result-hint kbd {
  padding: 1px 5px;
  border: 1px solid var(--border);
  border-bottom-width: 2px;
  border-radius: 3px;
  background: var(--bg-card);
  font-family: var(--font-mono);
  font-size: 11px;
}

/* 结果分组：组头在卡片外，列表本身是一张卡 */
.result-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 2px;
}
.group-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 700;
  color: var(--text);
}
.group-title .bi {
  font-size: 13px;
  color: var(--primary);
}
.group-count {
  margin-left: auto;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.7;
  font-variant-numeric: tabular-nums;
}

.result-list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 6px;
  list-style: none;
  box-shadow: var(--shadow-sm);
}
/* 覆盖全局 .card:hover：整张列表不做悬浮反馈，反馈落在条目上 */
.result-list:hover {
  box-shadow: var(--shadow-sm);
}
.result-item {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-radius: var(--radius);
  cursor: pointer;
  transition: background-color var(--dur) var(--ease);
}
/* 条目分隔线：虚线但不贴边，左右留出缩进 */
.result-item + .result-item::before {
  content: '';
  position: absolute;
  top: 0;
  left: 12px;
  right: 12px;
  border-top: 1px dashed var(--border-soft);
}
.result-item:hover,
.result-item.is-selected {
  background: var(--accent-wash);
}
/* 键盘选中：左侧朱砂短条，和 hover 区分开 */
.result-item.is-selected::after {
  content: '';
  position: absolute;
  left: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 0 3px 3px 0;
  background: var(--primary);
}
/* 缩略图：没有图 / 图挂了时展示类型图标 */
.result-thumb {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--radius);
  border: 1px solid var(--border-soft);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 18px;
}
.result-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
/* 头像类（话题 / 用户 / 友链）整块圆形：容器也圆，避免方形描边露出 */
.result-thumb.is-users,
.result-thumb.is-links,
.result-thumb.is-tag {
  border-radius: 50%;
}
.result-badge {
  flex-shrink: 0;
  padding: 1px 7px;
  font-size: 11px;
  line-height: 1.7;
  border-radius: 4px;
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.result-badge.is-page { color: #6f6f68; background: rgba(138, 138, 130, 0.14); }
.result-badge.is-tag { color: #3d7fb8; background: rgba(91, 155, 213, 0.14); }
.result-badge.is-links { color: #4f8a3a; background: rgba(106, 168, 79, 0.14); }
.result-badge.is-users { color: #a5701f; background: rgba(201, 138, 45, 0.14); }
.result-badge.is-moments { color: #6b5aa8; background: rgba(127, 109, 189, 0.14); }
.result-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.result-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.result-title {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color var(--dur) var(--ease);
}
.result-item:hover .result-title {
  color: var(--primary-deep);
}
.result-title :deep(mark),
.result-summary :deep(mark),
.result-meta :deep(mark) {
  padding: 0 1px;
  border-radius: 2px;
  background: var(--accent-glow);
  color: inherit;
}
/* 摘要：最多两行，超出省略 */
.result-summary {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
/* 元信息：话题 / 地址 / 位置 / 阅读量 / 时间 */
.result-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 12px;
  font-size: 12px;
  color: var(--text-light);
}
.result-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.result-meta-item i {
  color: var(--text-muted);
}
.result-arrow {
  flex-shrink: 0;
  align-self: center;
  color: var(--text-light);
  transition: transform var(--dur) var(--ease), color var(--dur) var(--ease);
}
.result-item:hover .result-arrow {
  color: var(--primary);
  transform: translateX(2px);
}

/* 加载骨架：与真实条目同构（缩略图 + 标题行 + 摘要行） */
.result-skeleton {
  padding: 6px;
  box-shadow: var(--shadow-sm);
}
.skeleton-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
}
.skeleton-item + .skeleton-item {
  border-top: 1px dashed var(--border-soft);
}
.sk-thumb {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: var(--radius);
}
.sk-lines {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 6px;
}
.sk-line {
  height: 12px;
  border-radius: 4px;
}
.sk-line.is-title {
  width: 45%;
  height: 14px;
}
.sk-line.is-text {
  width: 80%;
}

@media (max-width: 900px) {
  .search-suggest {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 640px) {
  .search-hero {
    padding: 18px 14px 14px;
  }
  .search-title {
    font-size: 19px;
  }
  .search-desc {
    display: none;
  }
  .search-form {
    margin-top: 12px;
  }
  .search-input {
    height: 46px;
    /* 16px 可避免 iOS 聚焦输入框时页面被放大 */
    font-size: 16px;
    padding: 0 100px 0 38px;
  }
  .search-form-icon {
    left: 14px;
  }
  /* 窄屏提交键只留图标，把宽度还给输入框 */
  .search-submit {
    height: 36px;
    padding: 0 12px;
  }
  .search-submit-text {
    display: none;
  }
  .search-clear {
    right: 52px;
  }
  .search-scopes {
    margin-top: 12px;
  }
  .scope-btn {
    flex: 0 0 auto;
    padding: 6px 12px;
  }
  .result-hint {
    display: none;
  }
  .result-item {
    gap: 10px;
    padding: 10px;
  }
  .result-thumb {
    width: 48px;
    height: 48px;
    font-size: 16px;
  }
  .result-title {
    white-space: normal;
  }
  .result-summary {
    -webkit-line-clamp: 3;
    line-clamp: 3;
  }
  .sk-thumb {
    width: 48px;
    height: 48px;
  }
}
</style>
