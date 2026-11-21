<template>
  <div class="log-admin">
    <!-- 概览 + 筛选 -->
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">日志解析</h2>
          <p class="block-desc">
            解析 <code>runtime/logs/&lt;日期&gt;/{info,warn,error,debug}.log</code> 与
            <code>runtime/sms/{sms,email}.log</code>，只读查看；
            JSON 行按字段展开，解析不出来的行也会原样展示
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm" :disabled="loading || loadingMeta" @click="refreshAll()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            v-for="tab in sourceTabs"
            :key="tab.key"
            type="button"
            class="status-tab"
            :class="{ 'is-active': source === tab.key }"
            @click="switchSource(tab.key)"
          >
            <i :class="tab.icon" /> {{ tab.label }}
          </button>
        </div>

        <div class="filter-right">
          <FormSelect variant="pill"
            v-if="source === 'system'"
            v-model="date"
            :options="dateOptions"
            icon="bi bi-calendar3"
            placeholder="选择日期"
            @change="onDateChange" />
          <FormSelect variant="pill"
            v-model="file"
            :options="fileOptions"
            icon="bi bi-file-earmark-text"
            placeholder="选择文件"
            @change="reloadFromFirstPage" />
          <FormSelect variant="pill"
            v-model="level"
            :options="levelOptions"
            icon="bi bi-funnel"
            placeholder="全部级别"
            @change="reloadFromFirstPage" />
          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索关键字…"
              aria-label="搜索日志关键字"
              @input="doSearch" />
          </div>
        </div>
      </div>

      <!-- 统计：级别可点击筛选 -->
      <div class="stat-grid" :aria-busy="loading">
        <button
          v-for="item in levelStats"
          :key="item.key"
          type="button"
          class="stat-card"
          :class="{ 'is-link': item.key !== 'other', 'is-active': level === item.key }"
          @click="pickLevel(item.key)"
        >
          <span class="stat-icon" :style="{ color: item.color }"><i :class="item.icon" /></span>
          <span class="stat-body">
            <span class="stat-value">{{ loading ? '···' : item.value }}</span>
            <span class="stat-label">{{ item.label }}</span>
          </span>
        </button>

        <div class="stat-card is-plain">
          <span class="stat-icon"><i class="bi bi-list-ol" /></span>
          <span class="stat-body">
            <span class="stat-value">{{ data.lines || 0 }}</span>
            <span class="stat-label">文件总行数</span>
          </span>
        </div>
      </div>

      <p v-if="data.truncated" class="block-desc warn-text">
        <i class="bi bi-exclamation-triangle" />
        文件过大，仅解析了前 {{ maxLines }} 行（可先按级别或关键字筛选）
      </p>
    </section>

    <!-- 条目列表 -->
    <div class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">
            {{ currentFileName || '日志条目' }}
          </h2>
          <p class="block-desc">
            <template v-if="loading">加载中…</template>
            <template v-else-if="!total">暂无匹配的日志</template>
            <template v-else>
              匹配 <strong>{{ total }}</strong> 条 · 第 {{ page }} / {{ data.pages || 1 }} 页 · 默认最新在前
            </template>
          </p>
        </div>
      </header>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 解析中...</div>

      <EmptyState
        v-else-if="!list.length"
        icon="bi bi-journal-x"
        :text="emptyHint"
      />

      <template v-else>
        <ul class="log-list">
          <li
            v-for="(item, index) in list"
            :key="`${page}-${index}`"
            class="log-item"
            :class="[`is-${item.level || 'raw'}`, { 'is-expanded': expanded === index }]"
          >
            <div class="log-row" @click="toggle(index)">
              <span class="log-time">{{ item.time || '—' }}</span>
              <span class="log-level" :class="`is-${item.level || 'raw'}`">{{ levelLabel(item) }}</span>
              <span class="log-msg">{{ item.msg || item.raw }}</span>
              <span v-if="item.caller" class="log-caller">{{ item.caller }}</span>
              <i class="bi log-toggle" :class="expanded === index ? 'bi-chevron-up' : 'bi-chevron-down'" />
            </div>

            <!-- 展开：字段 + 原始行 -->
            <div v-if="expanded === index" class="log-detail">
              <div v-if="item.status" class="detail-row">
                <span class="detail-label">状态</span>
                <span class="detail-value">{{ item.status }}</span>
              </div>
              <div v-if="hasFields(item)" class="detail-row">
                <span class="detail-label">字段</span>
                <pre class="detail-pre">{{ prettyFields(item) }}</pre>
              </div>
              <div class="detail-row">
                <span class="detail-label">原始行</span>
                <pre class="detail-pre">{{ item.raw }}</pre>
              </div>
              <div class="detail-actions">
                <button class="btn btn-sm btn-ghost" @click.stop="copy(item.raw)">
                  <i class="bi bi-clipboard" /> 复制原始行
                </button>
              </div>
            </div>
          </li>
        </ul>

        <Pagination
          v-if="!loading && total > 0"
          :current="page"
          :total="total"
          :page-size="limit"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { limit = size; page = 1; load() }"
          @update:current="(p) => { page = p; load() }"
        />
      </template>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { computed, onMounted, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import { listLogDates, listLogFiles, readLog } from '@/api/log'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
import { toast } from '@/utils/toast'

// 与后端 logMaxLines 保持一致（仅用于提示文案）
const maxLines = 200000

const sourceTabs = [
  { key: 'system', label: '系统日志', icon: 'bi bi-file-earmark-code' },
  { key: 'notify', label: '通知日志', icon: 'bi bi-envelope-paper' }
]

const levelMeta = {
  error: { label: '错误', color: 'var(--danger)', icon: 'bi bi-x-octagon' },
  warn: { label: '警告', color: 'var(--warning, #b8860b)', icon: 'bi bi-exclamation-triangle' },
  info: { label: '信息', color: 'var(--primary, #2f7d6b)', icon: 'bi bi-info-circle' },
  debug: { label: '调试', color: 'var(--text-muted)', icon: 'bi bi-bug' },
  other: { label: '未解析', color: 'var(--text-light, #999)', icon: 'bi bi-question-circle' }
}

const source = ref('system')
const dates = ref([])
const notifyFiles = ref([])
const files = ref([])

const date = ref('')
const file = ref('')
const level = ref('')
const keyword = ref('')
const page = ref(1)
const limit = ref(PAGE_SIZE_DEFAULT)

const loading = ref(false)
const loadingMeta = ref(false)
const expanded = ref(-1)

const data = ref({
  list: [],
  total: 0,
  lines: 0,
  pages: 0,
  levels: {},
  truncated: false,
  empty: false
})

const list = computed(() => data.value.list || [])
const total = computed(() => Number(data.value.total) || 0)

const dateOptions = computed(() =>
  dates.value.map((item) => ({
    value: item.date,
    label: `${item.date}（${item.files} 个文件 · ${formatSize(item.size)}）`
  }))
)

const fileOptions = computed(() =>
  (source.value === 'notify' ? notifyFiles.value : files.value).map((item) => ({
    value: item.name,
    label: `${item.name}（${formatSize(item.size)}）`
  }))
)

const levelOptions = [
  { value: '', label: '全部级别' },
  { value: 'error', label: '错误 error' },
  { value: 'warn', label: '警告 warn' },
  { value: 'info', label: '信息 info' },
  { value: 'debug', label: '调试 debug' }
]

const levelStats = computed(() => {
  const levels = data.value.levels || {}
  return ['error', 'warn', 'info', 'debug', 'other'].map((key) => ({
    key,
    label: levelMeta[key].label,
    color: levelMeta[key].color,
    icon: levelMeta[key].icon,
    value: Number(levels[key]) || 0
  }))
})

const currentFileName = computed(() => file.value || '')

const emptyHint = computed(() => {
  if (data.value.empty) return '该日志文件不存在或已被清理，换个日期或文件看看'
  if (keyword.value || level.value) return '当前筛选条件下没有日志，试试清空关键字或级别'
  if (source.value === 'notify') return '还没有短信 / 邮件发送记录'
  return '这一天还没有日志'
})

function formatSize(size) {
  const value = Number(size) || 0
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(2)} MB`
}

function levelLabel(item) {
  const key = item.level || 'raw'
  if (key === 'raw') return '未解析'
  return levelMeta[key]?.label || key
}

function hasFields(item) {
  const fields = item?.fields
  return !!fields && typeof fields === 'object' && Object.keys(fields).length > 0
}

function prettyFields(item) {
  try {
    return JSON.stringify(item.fields || {}, null, 2)
  } catch {
    return String(item.fields)
  }
}

function toggle(index) {
  expanded.value = expanded.value === index ? -1 : index
}

async function copy(text) {
  const value = String(text || '')
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    toast.success('已复制')
  } catch {
    toast.warning('当前环境不支持自动复制，请手动选择文本')
  }
}

// ---------- 数据加载 ----------

async function loadMeta() {
  loadingMeta.value = true
  try {
    const res = await listLogDates()
    const payload = res?.data || {}
    dates.value = Array.isArray(payload.dates) ? payload.dates : []
    notifyFiles.value = Array.isArray(payload.notify) ? payload.notify : []

    if (!date.value) {
      // 默认看最近一天（没有历史日期时用今天，交给后端回空态）
      date.value = dates.value[0]?.date || payload.today || ''
    }
  } catch {
    dates.value = []
    notifyFiles.value = []
  } finally {
    loadingMeta.value = false
  }
}

async function loadFiles() {
  try {
    const res = await listLogFiles(source.value === 'notify' ? { scope: 'notify' } : { date: date.value })
    const list_ = res?.data?.list || []
    files.value = source.value === 'notify' ? [] : list_
    if (source.value === 'notify') notifyFiles.value = list_

    const options = source.value === 'notify' ? notifyFiles.value : files.value
    // 当前文件已不在列表里（换日期/换来源）→ 自动选中第一个
    if (!options.some((item) => item.name === file.value)) {
      file.value = options[0]?.name || ''
    }
  } catch {
    files.value = []
    file.value = ''
  }
}

async function load() {
  if (!file.value) {
    data.value = { list: [], total: 0, lines: 0, pages: 0, levels: {}, truncated: false, empty: false }
    return
  }

  loading.value = true
  expanded.value = -1
  try {
    const res = await readLog({
      scope: source.value,
      date: date.value,
      file: file.value,
      level: level.value,
      keyword: keyword.value.trim(),
      page: page.value,
      limit: limit.value,
      order: 'desc'
    })
    const payload = res?.data || {}
    data.value = {
      list: payload.list || [],
      total: payload.total || 0,
      lines: payload.lines || 0,
      pages: payload.pages || 0,
      levels: payload.levels || {},
      truncated: !!payload.truncated,
      empty: !!payload.empty
    }
    if (payload.empty && page.value === 1) toast.info('该日志文件不存在或已被清理')
  } catch (e) {
    // silent 请求：这里自己提示，避免与拦截器重复
    toast.error(e?.msg || '读取日志失败')
    data.value = { list: [], total: 0, lines: 0, pages: 0, levels: {}, truncated: false, empty: false }
  } finally {
    loading.value = false
  }
}

function reloadFromFirstPage() {
  page.value = 1
  load()
}

async function onDateChange() {
  await loadFiles()
  reloadFromFirstPage()
}

async function switchSource(key) {
  if (source.value === key) return
  source.value = key
  level.value = ''
  keyword.value = ''
  await loadFiles()
  reloadFromFirstPage()
}

function pickLevel(key) {
  // 「未解析」不是可筛选的级别（解析失败的行没有 level 字段）
  if (key === 'other') return
  level.value = level.value === key ? '' : key
  reloadFromFirstPage()
}

// 关键字输入防抖，避免每敲一个字都扫一遍文件
let searchTimer = null
function doSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    load()
  }, 350)
}

async function refreshAll() {
  await loadMeta()
  await loadFiles()
  await load()
}

onMounted(async () => {
  await loadMeta()
  await loadFiles()
  await load()
})
</script>

<style scoped>
/* 概览卡与条目列表卡是上下两块，需要竖向间距（与其它后台页统一 16px） */
.log-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.panel-head,
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}
.block-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.block-desc code {
  padding: 1px 5px;
  font-size: 11px;
  border-radius: 3px;
  background: var(--bg-muted);
}
/* 截断提示紧跟在统计卡下面，单独给一点上间距，否则会贴着卡片 */
.warn-text {
  margin-top: 12px;
  color: var(--warning, #b8860b);
}
.head-actions {
  display: flex;
  gap: 8px;
}

/* 筛选栏 */
.list-filter {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
}
.status-tabs {
  display: flex;
  gap: 6px;
}
.status-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.status-tab:hover {
  border-color: var(--primary);
  color: var(--text);
}
.status-tab.is-active {
  border-color: var(--primary);
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.filter-right {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.search-box {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  height: 32px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-muted);
}
.search-input {
  width: 170px;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13px;
  color: var(--text);
}

/* 统计卡 */
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 10px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  text-align: left;
  cursor: default;
}
.stat-card.is-link {
  cursor: pointer;
}
.stat-card.is-link:hover {
  border-color: var(--primary);
}
.stat-card.is-active {
  border-color: var(--primary);
  background: var(--accent-soft);
}
.stat-icon {
  font-size: 16px;
  color: var(--text-light);
}
.stat-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.stat-value {
  font-size: 16px;
  font-weight: 600;
  line-height: 1.2;
}
.stat-label {
  font-size: 11px;
  color: var(--text-muted);
}

/* 日志列表 */
.log-list {
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.log-item + .log-item {
  border-top: 1px solid var(--border);
}
.log-item.is-expanded {
  background: var(--bg-muted);
}
.log-row {
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 8px 12px;
  font-size: 12px;
  cursor: pointer;
}
.log-row:hover {
  background: var(--bg-muted);
}
.log-time {
  flex-shrink: 0;
  width: 138px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.log-level {
  flex-shrink: 0;
  min-width: 44px;
  padding: 1px 6px;
  font-size: 11px;
  text-align: center;
  border-radius: 3px;
  color: #fff;
  background: var(--text-light);
}
.log-level.is-error {
  background: var(--danger);
}
.log-level.is-warn {
  background: var(--warning, #b8860b);
}
.log-level.is-info {
  background: var(--primary, #2f7d6b);
}
.log-level.is-debug {
  background: var(--text-muted);
}
.log-level.is-raw {
  background: var(--text-light);
}
.log-msg {
  flex: 1;
  min-width: 0;
  color: var(--text);
  word-break: break-all;
}
.log-item.is-error .log-msg {
  color: var(--danger);
}
.log-caller {
  flex-shrink: 0;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-light);
}
.log-toggle {
  flex-shrink: 0;
  color: var(--text-light);
}

/* 展开明细 */
.log-detail {
  padding: 4px 12px 12px;
}
.detail-row {
  display: flex;
  gap: 10px;
  margin-bottom: 8px;
}
.detail-label {
  flex-shrink: 0;
  width: 56px;
  padding-top: 6px;
  font-size: 11px;
  color: var(--text-muted);
}
.detail-value {
  font-size: 12px;
}
.detail-pre {
  flex: 1;
  min-width: 0;
  margin: 0;
  padding: 8px 10px;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-soft);
}
.detail-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
