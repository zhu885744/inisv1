<template>
  <div class="level-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">等级管理</h2>
          <p class="block-desc">
            配置用户等级梯度：等级值、名称与升级所需经验
            <template v-if="total > 0"> · 共 {{ total }} 级</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd">
            <i class="bi bi-plus-lg" /> 新建等级
          </button>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash }"
            @click="switchTab(false)"
          >全部等级</button>
          <button
            type="button"
            class="status-tab trash-tab"
            :class="{ 'is-active': trash }"
            @click="switchTab(true)"
          >
            <i class="bi bi-trash3" /> 回收站
          </button>
        </div>

        <div class="filter-right">
          <FormSelect variant="pill"
            v-model="sortKey"
            :options="sortOptions"
            :disabled="trash"
            icon="bi bi-sort-down"
            placeholder="排序方式"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索名称…"
              aria-label="搜索名称"
              @input="doSearch" />
          </div>

          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 回收站提示 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的等级不参与用户等级计算，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 批量操作栏（选中后出现） -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 个</span>
        <div class="batch-actions">
          <template v-if="trash">
            <button class="btn btn-sm" :disabled="busy" @click="batchRestore()">
              <i class="bi bi-arrow-counterclockwise" /> 批量恢复
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
              <i class="bi bi-x-octagon" /> 批量彻底删除
            </button>
          </template>
          <template v-else>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 批量删除
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-bar-chart'" :text="emptyText" />
      </div>

      <template v-else>
        <!-- 本页全选 -->
        <div class="list-head-row">
          <label class="pick" title="全选本页">
            <FormCheckbox
              ref="selectAllRef"
              :model-value="pageAllSelected"
              :disabled="busy || !list.length"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">本页 {{ list.length }} 个 · 共 {{ total }} 个</span>
        </div>

        <ul class="level-list">
          <li
            v-for="item in list"
            :key="item.id"
            class="level-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择等级 ${item.name || `Lv.${item.value}`}`"
                @change="toggleSelect(item.id)" />
            </label>

            <span class="level-badge">Lv.{{ item.value }}</span>

          <div class="level-main">
            <div class="level-name-row">
              <span class="level-name">{{ item.name || '未命名等级' }}</span>
              <span class="exp-chip">
                <i class="bi bi-lightning-charge" /> {{ expText(item) }}
              </span>
            </div>
            <p class="level-desc">{{ item.description || '暂无描述' }}</p>
            <div class="level-meta">
              <span class="meta-text">#{{ item.id }}</span>
              <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
            </div>
          </div>

          <div class="level-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>

            <template v-else>
              <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" @click="openEdit(item)">
                <i class="bi bi-pencil" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="删除" aria-label="删除" :disabled="busy" @click="askRemove(item)">
                <i class="bi bi-trash" />
              </button>
            </template>
          </div>
        </li>
        </ul>
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

    <!-- 新增 / 编辑等级 -->
    <AdminFormDialog
      v-model:visible="dialog.visible"
      :title="dialog.id ? '编辑等级' : '新建等级'"
      :icon="dialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="dialog.loading"
      confirm-text="保存"
      width="560px"
      @confirm="save"
    >
      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">等级值</label>
          <FormInput
            v-model="dialog.value"
            type="number"
            min="0"
            step="1"
            placeholder="如 0、1、2" />
          <p class="form-hint">用于排序与展示（Lv.N），0 表示起始等级</p>
        </div>
        <div class="form-item">
          <label class="form-label">所需经验</label>
          <FormInput
            v-model="dialog.exp"
            type="number"
            min="0"
            step="1"
            placeholder="达到该等级所需累计经验" />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">等级名称</label>
        <FormInput v-model="dialog.name" type="text" maxlength="32" placeholder="最多 32 字，如「新手」" />
      </div>

      <div class="form-item">
        <label class="form-label">等级描述</label>
        <FormTextarea
          v-model="dialog.description"
          rows="3"
          placeholder="可选，用于用户中心等级说明" />
      </div>

      <div class="form-item">
        <label class="form-label">备注</label>
        <FormInput v-model="dialog.remark" type="text" placeholder="仅后台可见" />
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
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 等级管理（/admin/level）
 *
 * 等级是「按经验值划分的用户梯度」，字段为：等级值 value、名称 name、所需经验 exp、描述、备注。
 * 做成单页 CRUD：全部等级 + 回收站两个视图，配合新增/编辑弹窗。
 *
 * 后端约束（app/api/controller/level.go）：
 * - level/save 自动分流 create / update；允许字段 name,value,description,exp,remark,json,text；
 * - level/count 不支持 onlyTrashed，回收站数量不另做统计；level/all 支持 onlyTrashed；
 * - 等级值 / 经验都是整数，允许 0（后端没有 Is.Empty 校验，0 可正常写入）；
 * - 用户当前等级由 Users 模型按 exp 查询得出（result.level），改动等级会直接影响前台展示。
 */
import { ref, reactive, computed, onMounted, watchEffect } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listLevels,
  saveLevel,
  removeLevels,
  forceDeleteLevels,
  restoreLevels,
  clearLevelRecycle
} from '@/api/level'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 默认按等级值升序，便于按梯度查看
const sortOptions = [
  { value: 'value asc', label: '等级升序' },
  { value: 'value desc', label: '等级降序' },
  { value: 'exp asc', label: '经验升序' },
  { value: 'create_time desc', label: '最新创建' }
]

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const sortKey = ref(sortOptions[0].value)
const keyword = ref('')
// 防抖后的实际搜索词（避免每敲一个字就打一次接口）
const searchKey = ref('')

const dialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  value: 0,
  name: '',
  exp: 0,
  description: '',
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

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的等级'
  if (trash.value) return '回收站是空的'
  return '还没有等级，先新建一个吧'
})

// ---------- 展示辅助 ----------
function expText(item) {
  const exp = Number(item?.exp || 0)
  return `${exp.toLocaleString('zh-CN')} 经验`
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  selectedIds.value = []
  try {
    const params = {
      page: page.value,
      limit: pageSize.value,
      order: trash.value ? 'delete_time desc' : sortKey.value
    }
    if (trash.value) params.onlyTrashed = true
    // 关键词走后端 like 的「字段名|值」格式
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `name|${kw}`
    const res = await listLevels(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
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

// 搜索后回到第一页
const doSearch = debounce(() => {
  searchKey.value = keyword.value
  page.value = 1
  load()
}, 350)

async function switchTab(value) {
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

// ---------- 新增 / 编辑 ----------
function resetDialog() {
  dialog.id = 0
  dialog.value = 0
  dialog.name = ''
  dialog.exp = 0
  dialog.description = ''
  dialog.remark = ''
}

function openAdd() {
  resetDialog()
  // 新建时默认排在最后一级之后
  const max = (list.value || []).reduce((m, i) => Math.max(m, Number(i.value || 0)), -1)
  dialog.value = max + 1
  dialog.visible = true
}

function openEdit(item) {
  dialog.id = Number(item.id)
  dialog.value = Number(item.value || 0)
  dialog.name = item.name || ''
  dialog.exp = Number(item.exp || 0)
  dialog.description = item.description || ''
  dialog.remark = item.remark || ''
  dialog.visible = true
}

// 整数校验：允许 0，拒绝负数与小数
function parseIntOrNull(raw) {
  const text = String(raw ?? '').trim()
  if (text === '') return null
  const num = Number(text)
  if (!Number.isInteger(num) || num < 0) return null
  return num
}

async function save() {
  const name = dialog.name.trim()
  if (!name) {
    toast.warning('请输入等级名称')
    return
  }
  const value = parseIntOrNull(dialog.value)
  if (value === null) {
    toast.warning('等级值必须为不小于 0 的整数')
    return
  }
  const exp = parseIntOrNull(dialog.exp)
  if (exp === null) {
    toast.warning('所需经验必须为不小于 0 的整数')
    return
  }

  const payload = {
    name,
    value,
    exp,
    description: dialog.description.trim(),
    remark: dialog.remark.trim()
  }
  if (dialog.id) payload.id = dialog.id

  dialog.loading = true
  try {
    await saveLevel(payload)
    toast.success(dialog.id ? '修改已保存' : '等级已创建')
    dialog.visible = false
    await load()
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    dialog.loading = false
  }
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
    title: '删除等级',
    message: `确定删除等级「${item.name || `Lv.${item.value}`}」吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeLevels([item.id])
      toast.success('已移入回收站')
      await load()
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreLevels([item.id])
    toast.success('已恢复')
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除等级「${item.name || `Lv.${item.value}`}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteLevels([item.id])
      toast.success('已彻底删除')
      await load()
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有等级将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearLevelRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await load()
    }
  })
}

// ---------- 多选 ----------
const selectedIds = ref([])
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((i) => selectedSet.value.has(i.id))
)
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

// ---------- 批量操作 ----------
function askBatchRemove() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个等级吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeLevels(ids)
      toast.success(`已删除 ${ids.length} 个等级`)
      selectedIds.value = []
      await load()
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  try {
    await restoreLevels(ids)
    toast.success(`已恢复 ${ids.length} 个等级`)
    selectedIds.value = []
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 个等级吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteLevels(ids)
      toast.success(`已彻底删除 ${ids.length} 个等级`)
      selectedIds.value = []
      await load()
    }
  })
}

onMounted(load)
</script>

<style scoped>
.level-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 头部 ---------- */
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
  /* 与下拉 / 搜索框 / 按钮等高 */
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

/* ---------- 回收站提示 ---------- */
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

/* ---------- 批量操作栏 ---------- */
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
.pick {
  display: flex;
  align-items: center;
  align-self: center;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}
.pick input:disabled {
  cursor: not-allowed;
  opacity: 0.4;
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

.level-list {
  display: flex;
  flex-direction: column;
}
.level-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.level-row:last-child {
  border-bottom: none;
}
.level-row:hover {
  background: var(--bg-muted);
}
.level-row.is-selected {
  background: var(--accent-wash);
}

.level-badge {
  flex-shrink: 0;
  min-width: 52px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  text-align: center;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 999px;
  font-variant-numeric: tabular-nums;
}

.level-main {
  flex: 1;
  min-width: 0;
}
.level-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.level-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.exp-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 3px;
  font-variant-numeric: tabular-nums;
}
.level-desc {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.level-meta {
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

.level-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.level-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.level-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 弹窗表单 ---------- */
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}

@media (max-width: 768px) {
  .list-filter {
    gap: 8px;
  }
  .filter-right {
    width: 100%;
    margin-left: 0;
  }
  .search-box {
    width: 100%;
  }
  .level-row {
    flex-wrap: wrap;
  }
  .level-main {
    flex: 1 1 60%;
  }
  .level-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
