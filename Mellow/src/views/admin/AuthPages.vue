<template>
  <div class="auth-pages-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">后台页面管理</h2>
          <p class="block-desc">
            维护后台菜单页面；权限分组按页面 hash 授权，删除页面后相关分组需重新配置
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
            <i class="bi bi-plus-lg" /> 新建页面
          </button>
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
        <div v-for="s in statCards" :key="s.label" class="stat-card">
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
          <h2 class="block-title">页面列表</h2>
          <p class="block-desc">
            hash 由后端生成，用于权限分组授权，不可修改
            <template v-if="total > 0"> · 共 {{ total }} 个</template>
          </p>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash }"
            @click="switchTrash(false)"
          >全部页面</button>
          <button
            type="button"
            class="status-tab trash-tab"
            :class="{ 'is-active': trash }"
            @click="switchTrash(true)"
          >
            <i class="bi bi-trash3" /> 回收站
          </button>
        </div>

        <div class="filter-right">
          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索名称或路径…"
              aria-label="搜索名称或路径"
              @input="doSearch" />
          </div>

          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 个</span>
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
        <span><i class="bi bi-trash3" /> 回收站内的页面不再出现在后台菜单，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-window-stack'" :text="emptyText" />
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
          <span class="list-head-text">本页 {{ list.length }} 个</span>
        </div>

        <table class="page-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th>名称</th>
              <th>路径</th>
              <th class="cell-icon">图标</th>
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
                    :aria-label="`选择页面 ${item.name}`"
                    @change="toggleSelect(item.id)" />
                </label>
              </td>
              <td>
                <div class="page-name">{{ item.name || '—' }}</div>
                <div class="page-sub">
                  <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                  <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
                  <span class="hash-text" :title="item.hash">#{{ item.hash }}</span>
                </div>
              </td>
              <td><code class="page-path">{{ item.path }}</code></td>
              <td class="cell-icon">
                <span v-if="item.svg" class="page-icon" v-html="item.svg" />
                <i
                  v-else-if="biIcon(item.icon)"
                  class="icon-preview"
                  :class="biIcon(item.icon)"
                  :title="item.icon"
                />
                <span v-else class="icon-chip">{{ item.icon || '—' }}</span>
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
                    <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" :disabled="busy" @click="openEdit(item)">
                      <i class="bi bi-pencil" />
                    </button>
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

    <!-- 新建 / 编辑页面 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑页面 #${edit.id}` : '新建页面'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">页面名称</label>
          <FormInput v-model="edit.name" type="text" placeholder="必填" />
        </div>
        <div class="form-item">
          <label class="form-label">页面路径</label>
          <FormInput v-model="edit.path" type="text" placeholder="必填，如 /admin/users" />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">图标名</label>
          <FormInput v-model="edit.icon" type="text" placeholder="可选，如 bi bi-bag" />
          <p class="form-hint">填 Bootstrap Icons 类名（https://icons.getbootstrap.com/）</p>
        </div>
        <div class="form-item">
          <label class="form-label">图标大小</label>
          <FormInput v-model="edit.size" type="text" placeholder="默认 16px" />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">SVG 图标</label>
        <FormTextarea v-model="edit.svg" rows="3" placeholder="可选，填写后优先于图标名展示" />
      </div>

      <div class="form-item">
        <label class="form-label">备注</label>
        <FormInput v-model="edit.remark" type="text" placeholder="可选" />
      </div>
    </AdminFormDialog>

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
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 后台页面管理（/admin/security/pages）
 *
 * 后端约束（app/api/controller/auth-pages.go、app/model/auth-pages.go）：
 * - validator 要求 name 与 path 必填；
 * - 允许字段 name/path/icon/svg/size/remark，**hash 由后端生成且不可修改**；
 * - BeforeCreate 校验 hash 唯一；
 * - 权限分组的 pages 字段存页面 hash，删除页面后相关分组需重新配置。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listAuthPages,
  countAuthPages,
  saveAuthPage,
  removeAuthPages,
  forceDeleteAuthPages,
  restoreAuthPages,
  clearAuthPageRecycle
} from '@/api/auth-pages'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// ===== 列表 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ total: 0, trash: 0 })

// ===== 弹窗 =====
const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  path: '',
  icon: '',
  svg: '',
  size: '16px',
  remark: ''
})

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
  { label: '页面总数', value: stats.total, icon: 'bi bi-window-stack', color: 'var(--primary)' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

// icon 是 Bootstrap Icons 类名（"bi bi-bag" 或 "bi-bag"）时返回可直接用于 :class 的值，否则返回空串
function biIcon(name) {
  if (typeof name !== 'string' || !name.trim()) return ''
  const value = name.trim()
  if (!value.includes('bi-')) return ''
  return value.startsWith('bi ') ? value : `bi ${value}`
}

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的页面'
  if (trash.value) return '回收站是空的'
  return '还没有后台页面，新建一个吧'
})

// ---------- 展示辅助 ----------
function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `创建于 ${fromNow(item.create_time)}`
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [totalRes, trashRes] = await Promise.all([
      countAuthPages(),
      countAuthPages({ onlyTrashed: true })
    ])
    stats.total = Number(totalRes?.data || 0)
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
      order: trash.value ? 'delete_time desc' : 'id asc'
    }
    if (trash.value) params.onlyTrashed = true
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `name|${kw}`

    const res = await listAuthPages(params)
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

async function switchTrash(value) {
  if (trash.value === value) return
  trash.value = value
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

// ---------- 新建 / 编辑 ----------
function resetEdit() {
  edit.id = 0
  edit.name = ''
  edit.path = ''
  edit.icon = ''
  edit.svg = ''
  edit.size = '16px'
  edit.remark = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.name = item.name || ''
  edit.path = item.path || ''
  edit.icon = item.icon || ''
  edit.svg = item.svg || ''
  edit.size = item.size || '16px'
  edit.remark = item.remark || ''
  edit.visible = true
}

async function save() {
  const name = edit.name.trim()
  const path = edit.path.trim()
  if (!name) {
    toast.warning('请输入页面名称')
    return
  }
  if (!path) {
    toast.warning('请输入页面路径')
    return
  }

  const payload = {
    name,
    path,
    icon: edit.icon.trim(),
    svg: edit.svg.trim(),
    size: edit.size.trim() || '16px',
    remark: edit.remark.trim()
  }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveAuthPage(payload)
    toast.success(edit.id ? '修改已保存' : '页面已创建')
    edit.visible = false
    await afterMutation(0)
  } catch {
    /* 拦截器已提示 */
  } finally {
    edit.loading = false
  }
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
    title: '删除页面',
    message: `确定删除页面「${item.name}」吗？已授权该页面的权限分组需重新配置，可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeAuthPages([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreAuthPages([item.id])
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
    message: `确定彻底删除页面「${item.name}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAuthPages([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个页面吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeAuthPages(ids)
      toast.success(`已删除 ${ids.length} 个`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    await restoreAuthPages(ids)
    toast.success(`已恢复 ${ids.length} 个`)
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
    message: `确定彻底删除选中的 ${ids.length} 个页面吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAuthPages(ids)
      toast.success(`已彻底删除 ${ids.length} 个`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有页面将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearAuthPageRecycle()
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
.auth-pages-admin {
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
  width: 190px;
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

.page-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.page-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.page-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.page-table tr:last-child td {
  border-bottom: none;
}
.page-table tbody tr:hover td {
  background: var(--bg-muted);
}
.page-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-icon {
  width: 72px;
  text-align: center;
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

.page-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.page-sub {
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
.hash-text {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.page-path {
  padding: 1px 6px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 3px;
  word-break: break-all;
}
.page-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  color: var(--text-soft);
}
.page-icon :deep(svg) {
  width: 100%;
  height: 100%;
}
.icon-chip {
  font-size: 11px;
  color: var(--text-muted);
}
.icon-preview {
  font-size: 16px;
  color: var(--text-soft);
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

/* ---------- 弹窗 ---------- */
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
.form-hint {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-muted);
}

@media (max-width: 768px) {
  .list-filter,
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .page-table th:nth-child(3),
  .page-table td:nth-child(3) {
    display: none;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
