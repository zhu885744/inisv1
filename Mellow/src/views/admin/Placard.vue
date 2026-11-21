<template>
  <div class="placard-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">公告管理</h2>
          <p class="block-desc">
            维护首页公告轮播的内容、类型与跳转链接
            <template v-if="list.length"> · 共 {{ list.length }} 条</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd">
            <i class="bi bi-plus-lg" /> 新建公告
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
          >全部公告</button>
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
          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索标题…"
              aria-label="搜索标题"
              @input="doSearch" />
          </div>
          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 回收站提示 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的公告不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 批量操作栏（选中后出现） -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 条</span>
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
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-megaphone'" :text="emptyText" />
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
          <span class="list-head-text">本页 {{ list.length }} 条 · 共 {{ total }} 条</span>
        </div>

        <ul class="placard-list">
          <li
            v-for="item in list"
            :key="item.id"
            class="placard-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择公告 ${item.title || item.id}`"
                @change="toggleSelect(item.id)" />
            </label>

            <div class="placard-main">
            <div class="placard-name-row">
              <span class="type-dot" :style="{ background: typeColor(item) }" :title="typeLabel(item)" />
              <span class="placard-name">{{ item.title || '未命名公告' }}</span>
              <span class="type-chip" :style="typeChipStyle(item)">{{ typeLabel(item) }}</span>
              <span v-if="item.url" class="placard-url">
                <i class="bi bi-link-45deg" /> {{ item.url }}
              </span>
            </div>

            <p class="placard-desc">{{ contentPreview(item) }}</p>

            <div class="placard-meta">
              <span class="meta-text">#{{ item.id }}</span>
              <span class="meta-text">
                <i class="bi bi-box-arrow-up-right" /> {{ targetLabel(item) }}
              </span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
            </div>
          </div>

          <div class="placard-actions">
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

    <!-- 新增 / 编辑公告 -->
    <AdminFormDialog
      v-model:visible="dialog.visible"
      :title="dialog.id ? '编辑公告' : '新建公告'"
      :icon="dialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="dialog.loading"
      confirm-text="保存"
      width="560px"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">公告标题</label>
        <FormInput v-model="dialog.title" type="text" maxlength="32" placeholder="最多 32 字" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">公告类型</label>
          <FormSelect
            v-model="dialog.type"
            :options="PLACARD_TYPES"
            placeholder="请选择类型" />
          <p class="form-hint">决定首页轮播左侧圆点与公告弹窗类型标签的配色</p>
        </div>
        <div class="form-item">
          <label class="form-label">打开方式</label>
          <FormSelect
            v-model="dialog.target"
            :options="TARGET_OPTIONS"
            placeholder="请选择打开方式" />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">公告内容</label>
        <FormTextarea
          v-model="dialog.content"
          rows="4"
          maxlength="512"
          placeholder="最多 512 字，支持 HTML（前台弹窗按原文渲染）" />
      </div>

      <div class="form-item">
        <label class="form-label">跳转链接</label>
        <FormInput v-model="dialog.url" type="text" maxlength="256" placeholder="可选，如 /about 或 https://example.com" />
        <p class="form-hint">留空则前台只展示内容，不显示跳转按钮</p>
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
 * 公告管理（/admin/placard）
 *
 * 公告表字段很少（标题 / 内容 / 类型 / 链接 / 打开方式），也没有状态字段，
 * 因此做成单页 CRUD：全部公告 + 回收站两个视图，配合新增/编辑弹窗。
 *
 * 后端约束（app/api/controller/placard.go）：
 * - placard/save 会自动分流 create / update，故新增与编辑共用同一个弹窗；
 * - 允许字段为 title,content,type,url,target,json,text；url 会走 SanitizeURL；
 * - 标题 32 / 内容 512 / 链接 256 字，超出会被数据库拒绝，前端用 maxlength 兜住；
 * - placard/count 不支持 onlyTrashed，回收站数量不另做统计（列表自带 count）。
 */
import { ref, reactive, computed, onMounted, watchEffect } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listPlacards,
  savePlacard,
  removePlacards,
  forceDeletePlacards,
  restorePlacards,
  clearPlacardRecycle
} from '@/api/placard'
import { fromNow } from '@/utils/time'
import { truncate, debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'
// 公告类型（文案 + 配色）来自共享定义 utils/placard.js：与首页轮播圆点、公告内容弹窗的
// 类型标签同源，改配色只动那一处（历史上「公告」用主题朱砂、「警告」用 #d9544d，
// 两个红色几乎看不出区别，现改为 朱砂 / 琥珀 / 天蓝 三个色相）
import {
  PLACARD_TYPES,
  placardTypeKey,
  placardTypeLabel,
  placardTypeColor,
  placardTypeStyle
} from '@/utils/placard'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 打开方式：后端默认 _blank，前端仅识别 _self 为当前窗口
const TARGET_OPTIONS = [
  { value: '_blank', label: '新窗口打开' },
  { value: '_self', label: '当前窗口打开' }
]

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const keyword = ref('')
// 防抖后的实际搜索词（避免每敲一个字就打一次接口）
const searchKey = ref('')

const dialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  title: '',
  type: 'notice',
  content: '',
  url: '',
  target: '_blank'
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
  if (searchKey.value) return '没有匹配的公告'
  if (trash.value) return '回收站是空的'
  return '还没有公告，先新建一条吧'
})

// ---------- 展示辅助 ----------
function typeColor(item) {
  return placardTypeColor(item?.type)
}

function typeLabel(item) {
  return placardTypeLabel(item?.type)
}

// 类型标签的行内配色（文字 + 同色淡底），与首页弹窗里的标签一致
function typeChipStyle(item) {
  return placardTypeStyle(item?.type)
}

function targetLabel(item) {
  return item?.target === '_self' ? '当前窗口' : '新窗口'
}

// 内容可能是 HTML（前台用 v-html 渲染），列表里只取纯文本摘要
function contentPreview(item) {
  const text = truncate(item?.content || '', 120)
  return text || '暂无内容'
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
    const params = { page: page.value, limit: pageSize.value }
    if (trash.value) params.onlyTrashed = true
    // 关键词走后端 like 的「字段名|值」格式：公告表只有 title 适合模糊匹配
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `title|${kw}`
    const res = await listPlacards(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
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
  dialog.title = ''
  dialog.type = 'notice'
  dialog.content = ''
  dialog.url = ''
  dialog.target = '_blank'
}

function openAdd() {
  resetDialog()
  dialog.visible = true
}

function openEdit(item) {
  dialog.id = Number(item.id)
  dialog.title = item.title || ''
  dialog.type = placardTypeKey(item?.type)
  dialog.content = item.content || ''
  dialog.url = item.url || ''
  dialog.target = item.target === '_self' ? '_self' : '_blank'
  dialog.visible = true
}

async function save() {
  const title = dialog.title.trim()
  if (!title) {
    toast.warning('请输入公告标题')
    return
  }

  const payload = {
    title,
    type: dialog.type,
    content: dialog.content.trim(),
    url: dialog.url.trim(),
    target: dialog.target
  }
  if (dialog.id) payload.id = dialog.id

  dialog.loading = true
  try {
    await savePlacard(payload)
    toast.success(dialog.id ? '修改已保存' : '公告已创建')
    dialog.visible = false
    await load()
  } catch {
    // 失败提示由请求拦截器统一给出（如内容含恶意代码）
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
    title: '删除公告',
    message: `确定删除公告「${item.title || item.id}」吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removePlacards([item.id])
      toast.success('已移入回收站')
      await load()
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restorePlacards([item.id])
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
    message: `确定彻底删除公告「${item.title || item.id}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeletePlacards([item.id])
      toast.success('已彻底删除')
      await load()
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有公告将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearPlacardRecycle()
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
    message: `确定删除选中的 ${ids.length} 条公告吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removePlacards(ids)
      toast.success(`已删除 ${ids.length} 条公告`)
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
    await restorePlacards(ids)
    toast.success(`已恢复 ${ids.length} 条公告`)
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
    message: `确定彻底删除选中的 ${ids.length} 条公告吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeletePlacards(ids)
      toast.success(`已彻底删除 ${ids.length} 条公告`)
      selectedIds.value = []
      await load()
    }
  })
}

onMounted(load)
</script>

<style scoped>
.placard-admin {
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
  /* 与搜索框 / 按钮等高 */
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
  width: 200px;
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

.placard-list {
  display: flex;
  flex-direction: column;
}
.placard-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.placard-row:last-child {
  border-bottom: none;
}
.placard-row:hover {
  background: var(--bg-muted);
}
.placard-row.is-selected {
  background: var(--accent-wash);
}

.placard-main {
  flex: 1;
  min-width: 0;
}
.placard-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.type-dot {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 50%;
}
.placard-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
/* 类型标签：文字色与底色由 utils/placard.js 的行内样式给出（与前台弹窗一致） */
.type-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 999px;
  white-space: nowrap;
}
.placard-url {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  max-width: 260px;
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.placard-desc {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  word-break: break-word;
}
.placard-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.placard-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.placard-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.placard-actions .btn:disabled {
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
  .placard-row {
    flex-wrap: wrap;
  }
  .placard-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
