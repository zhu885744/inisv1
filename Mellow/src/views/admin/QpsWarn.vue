<template>
  <div class="qps-warn-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">QPS 预警</h2>
          <p class="block-desc">
            记录触发请求频率限制的来源；告警由系统自动写入，后台以查看与清理为主
          </p>
        </div>
        <div class="head-actions">
          <button
            class="btn btn-ghost btn-sm"
            :disabled="loadingStats"
            title="刷新统计"
            aria-label="刷新统计"
            @click="loadStats"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
        </div>
      </header>

      <div class="stat-grid" :aria-busy="loadingStats">
        <div
          v-for="s in statCards"
          :key="s.label"
          class="stat-card"
          :class="{ 'is-link': s.key !== undefined }"
          @click="s.key !== undefined && pickStatus(s.key)"
        >
          <span class="stat-icon" :style="{ color: s.color }"><i :class="s.icon" /></span>
          <span class="stat-body">
            <span class="stat-value">{{ loadingStats ? '···' : s.value }}</span>
            <span class="stat-label">{{ s.label }}</span>
          </span>
        </div>
      </div>
    </section>

    <div class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">告警记录</h2>
          <p class="block-desc">
            同一 IP 频繁出现时可移入黑名单处理
            <template v-if="total > 0"> · 共 {{ total }} 条</template>
          </p>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash && status === tab.key }"
            @click="switchStatus(tab.key)"
          >{{ tab.label }}</button>
          <button
            type="button"
            class="status-tab trash-tab"
            :class="{ 'is-active': trash }"
            @click="toggleTrash"
          >
            <i class="bi bi-trash3" /> 回收站
          </button>
        </div>

        <div class="filter-right">
          <FormSelect variant="pill"
            v-model="methodFilter"
            :options="methodOptions"
            icon="bi bi-signpost-split"
            placeholder="全部方法"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索 IP 或路径…"
              aria-label="搜索 IP 或路径"
              @input="doSearch" />
          </div>

          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 条</span>
        <div class="batch-actions">
          <template v-if="trash">
            <button class="btn btn-sm" :disabled="busy" @click="batchRestore()">
              <i class="bi bi-arrow-counterclockwise" /> 批量恢复
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
              <i class="bi bi-x-octagon" /> 彻底删除
            </button>
          </template>
          <template v-else>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 删除
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的记录仅作留存，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-speedometer2'" :text="emptyText" />
      </div>

      <template v-else>
        <div class="list-head-row">
          <label class="pick" title="全选本页">
            <FormCheckbox
              ref="selectAllRef"
              :model-value="pageAllSelected"
              :disabled="busy"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">本页 {{ list.length }} 条</span>
        </div>

        <table class="warn-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th>IP</th>
              <th>请求</th>
              <th class="cell-act">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in list" :key="item.id" :class="{ 'is-selected': isSelected(item.id) }">
              <td class="cell-pick">
                <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
                  <FormCheckbox
                    :model-value="isSelected(item.id)"
                    :disabled="busy"
                    :aria-label="`选择记录 ${item.ip}`"
                    @change="toggleSelect(item.id)" />
                </label>
              </td>
              <td>
                <code class="ip-value">{{ item.ip || '—' }}</code>
                <div class="warn-sub">
                  <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                  <span v-if="item.agent" class="meta-text user-agent-text" :title="item.agent">
                    <i class="bi bi-browser-chrome" /> {{ item.agent }}
                  </span>
                </div>
              </td>
              <td>
                <span class="method-chip" :class="`is-${methodKey(item)}`">{{ methodLabel(item) }}</span>
                <code class="warn-path">{{ item.path || '—' }}</code>
              </td>
              <td class="cell-act">
                <div class="row-actions">
                  <template v-if="trash">
                    <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                      <i class="bi bi-arrow-counterclockwise" />
                    </button>
                    <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                      <i class="bi bi-x-octagon" />
                    </button>
                  </template>
                  <template v-else>
                    <button class="btn btn-ghost btn-sm danger" title="删除" aria-label="删除" :disabled="busy" @click="askRemove(item)">
                      <i class="bi bi-trash" />
                    </button>
                  </template>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </template>

      <Pagination
        v-if="!loading && total > 0"
        :current="page"
        :total="total"
        :page-size="pageSize"
        :sizes="PAGE_SIZE_OPTIONS"
        @update:page-size="(size) => { pageSize = size; page = 1; load() }"
        @update:current="changePage"
      />
    </div>

    <!-- 操作确认 -->
    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      :confirm-text="confirm.confirmText"
      :danger="confirm.danger"
      :loading="confirm.loading"
      @confirm="runConfirm"
    />
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * QPS 预警（/admin/qps/warn）
 *
 * 后端约束（app/api/controller/qps-warn.go、app/model/qps-warn.go）：
 * - 记录由系统按请求频率自动写入，允许字段只有 ip（agent / path / method 不可写），
 *   因此后台不提供新建与编辑，只做查看与清理；
 * - 非 root 查询时后端会对 ip 做脱敏（MaskIP）；
 * - 删除为软删除，彻底删除不可恢复。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import {
  listQpsWarn,
  countQpsWarn,
  removeQpsWarn,
  forceDeleteQpsWarn,
  restoreQpsWarn,
  clearQpsWarnRecycle
} from '@/api/qps-warn'
import { fromNow, formatTime } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

const tabs = [
  { key: 'all', label: '全部' },
  { key: 'recent', label: '近 24 小时' }
]

const methodOptions = [
  { value: '', label: '全部方法' },
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'PATCH', label: 'PATCH' }
]

// ===== 列表 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const status = ref('all')
const methodFilter = ref('')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ total: 0, recent: 0, trash: 0 })

const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  action: null
})

const statCards = computed(() => [
  { label: '告警总数', value: stats.total, icon: 'bi bi-speedometer2', color: 'var(--primary)', key: 'all' },
  { label: '近 24 小时', value: stats.recent, icon: 'bi bi-clock-history', color: 'var(--warning)', key: 'recent' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的记录'
  if (trash.value) return '回收站是空的'
  if (status.value === 'recent') return '近 24 小时没有告警'
  return '暂无 QPS 告警'
})

// ---------- 展示辅助 ----------
function nowUnix() {
  return Math.floor(Date.now() / 1000)
}

function methodKey(item) {
  return String(item?.method || 'get').toLowerCase()
}

function methodLabel(item) {
  return String(item?.method || 'GET').toUpperCase()
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `${fromNow(item.create_time)} · ${formatTime(item.create_time)}`
}

// ---------- 筛选 ----------
function buildWhere() {
  const where = {}
  if (status.value === 'recent') where.create_time = { $gte: nowUnix() - 86400 }
  if (methodFilter.value) where.method = methodFilter.value
  return Object.keys(where).length ? JSON.stringify(where) : null
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [totalRes, recentRes, trashRes] = await Promise.all([
      countQpsWarn(),
      countQpsWarn({ where: JSON.stringify({ create_time: { $gte: nowUnix() - 86400 } }) }),
      countQpsWarn({ onlyTrashed: true })
    ])
    stats.total = Number(totalRes?.data || 0)
    stats.recent = Number(recentRes?.data || 0)
    stats.trash = Number(trashRes?.data || 0)
  } catch {
    /* 统计失败不阻塞列表 */
  } finally {
    loadingStats.value = false
  }
}

// ---------- 列表 ----------
async function load() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: pageSize.value,
      order: trash.value ? 'delete_time desc' : 'create_time desc'
    }
    if (trash.value) params.onlyTrashed = true
    else {
      const where = buildWhere()
      if (where) params.where = where
    }

    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `ip|${kw}`

    const res = await listQpsWarn(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reload()
}, 350)

function switchStatus(key) {
  if (!trash.value && status.value === key) return
  trash.value = false
  status.value = key
  reload()
}

function pickStatus(key) {
  switchStatus(key)
}

async function toggleTrash() {
  trash.value = !trash.value
  keyword.value = ''
  searchKey.value = ''
  page.value = 1
  await load()
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

// ---------- 选中 ----------
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(() => list.value.length > 0 && list.value.every((i) => selectedSet.value.has(i.id)))
const someSelected = computed(() => selectedIds.value.length > 0 && !pageAllSelected.value)

const selectAllRef = ref(null)

watchEffect(() => {
  if (selectAllRef.value) selectAllRef.value.setIndeterminate(someSelected.value)
})

function isSelected(id) {
  return selectedSet.value.has(id)
}

function toggleSelect(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = [...next]
}

function toggleSelectAll() {
  selectedIds.value = pageAllSelected.value ? [] : list.value.map((i) => i.id)
}

function clearSelection() {
  selectedIds.value = []
}

async function afterMutation(removedCount = 0) {
  if (removedCount > 0 && list.value.length <= removedCount && page.value > 1) {
    page.value -= 1
  }
  await load()
  loadStats()
}

// ---------- 删除 / 恢复 ----------
function openConfirm({ title, message, confirmText = '确定', danger = false, action }) {
  confirm.title = title
  confirm.message = message
  confirm.confirmText = confirmText
  confirm.danger = danger
  confirm.action = action
  confirm.visible = true
}

async function runConfirm() {
  if (typeof confirm.action !== 'function') {
    confirm.visible = false
    return
  }
  confirm.loading = true
  try {
    const ok = await confirm.action()
    if (ok !== false) confirm.visible = false
  } catch {
    // 请求失败时保持弹窗打开，错误提示已由请求拦截器统一给出
  } finally {
    confirm.loading = false
  }
}

function askRemove(item) {
  openConfirm({
    title: '删除告警',
    message: `确定删除 IP「${item.ip || '—'}」的告警记录吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeQpsWarn([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreQpsWarn([item.id])
    toast.success('已恢复')
    await afterMutation(1)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除 IP「${item.ip || '—'}」的告警记录吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteQpsWarn([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 条告警吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeQpsWarn(ids)
      toast.success(`已删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    await restoreQpsWarn(ids)
    toast.success(`已恢复 ${ids.length} 条`)
    clearSelection()
    await afterMutation(ids.length)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 条告警吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteQpsWarn(ids)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有告警将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearQpsWarnRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

onMounted(() => {
  load()
  loadStats()
})
</script>

<style scoped>
.qps-warn-admin {
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
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.stat-card.is-link {
  cursor: pointer;
  transition: box-shadow 0.15s;
}
.stat-card.is-link:hover {
  box-shadow: 0 0 0 1px var(--primary) inset;
}
.stat-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--bg-card);
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
}
.stat-label {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 工具栏 ---------- */
.list-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.status-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.status-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: var(--control-h-sm);
  padding: 0 14px;
  font-size: 13px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  background: var(--bg-muted);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.status-tab:hover {
  color: var(--primary);
}
.status-tab.is-active {
  background: var(--primary);
  color: #fff;
}
.filter-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-left: auto;
}
.search-box {
  position: relative;
  width: 180px;
}
.search-box .bi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: var(--text-light);
}
.search-input {
  width: 100%;
  height: var(--control-h-sm);
  padding: 0 12px 0 30px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

/* ---------- 批量 / 回收站 ---------- */
.batch-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--accent-wash);
  border: 1px solid var(--accent-soft);
  border-radius: var(--radius);
}
.batch-count {
  font-size: 13px;
  color: var(--text-soft);
}
.batch-count strong {
  color: var(--primary-deep);
}
.batch-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.trash-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  font-size: 12px;
  color: var(--warning);
  background: var(--gold-wash);
  border-radius: var(--radius);
}

/* ---------- 列表 ---------- */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}

.list-head-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px 4px;
}
.list-head-text {
  font-size: 12px;
  color: var(--text-muted);
}

.warn-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.warn-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.warn-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.warn-table tr:last-child td {
  border-bottom: none;
}
.warn-table tbody tr:hover td {
  background: var(--bg-muted);
}
.warn-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-act {
  width: 96px;
  text-align: right;
}

.pick {
  display: flex;
  align-items: center;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

.ip-value {
  padding: 2px 8px;
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text);
  background: var(--bg-muted);
  border-radius: 3px;
  word-break: break-all;
}
.warn-sub {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.user-agent-text {
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.method-chip {
  display: inline-block;
  min-width: 52px;
  margin-right: 8px;
  padding: 2px 6px;
  font-size: 11px;
  font-weight: 600;
  text-align: center;
  border-radius: 3px;
}
.method-chip.is-get {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.method-chip.is-post {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.method-chip.is-put {
  color: var(--warning);
  background: var(--gold-wash);
}
.method-chip.is-delete {
  color: var(--danger);
  background: var(--accent-soft);
}
.method-chip.is-patch {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.12);
}
.warn-path {
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text-muted);
  word-break: break-all;
}

.row-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}
.row-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.row-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .list-filter,
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
