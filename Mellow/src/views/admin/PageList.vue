<template>
  <div class="page-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">独立页面</h2>
          <p class="block-desc">
            管理站点独立单页面（关于、协议、自定义页等）
            <template v-if="total > 0"> · 共 {{ total }} 个页面</template>
          </p>
        </div>
        <div class="head-actions">
          <router-link to="/admin/pages/write" class="btn btn-primary btn-sm">
            <i class="bi bi-pencil-square" /> 写页面
          </router-link>
        </div>
      </header>

      <!-- 筛选栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash && status === tab.key }"
            @click="switchTab(tab.key)"
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
            v-model="sortKey"
            :options="sortOptions"
            :disabled="trash"
            icon="bi bi-sort-down"
            placeholder="排序方式"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索页面标题…"
              @input="doSearch" />
          </div>
        </div>
      </div>

      <!-- 批量操作栏 -->
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
            <button class="btn btn-sm" :disabled="busy" @click="batchAudit(1)">
              <i class="bi bi-check2-circle" /> 审核通过
            </button>
            <button class="btn btn-sm" :disabled="busy" @click="batchAudit(2)">
              <i class="bi bi-slash-circle" /> 审核驳回
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 删除
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <!-- 回收站提示 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站中的页面不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !total" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-file-earmark'" :text="emptyText" />
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

        <ul class="page-list">
        <li v-for="item in list" :key="item.id" class="page-row" :class="{ 'is-selected': isSelected(item.id) }">
          <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
            <FormCheckbox :model-value="isSelected(item.id)" @change="toggleSelect(item.id)" />
          </label>

          <span class="page-icon"><i class="bi bi-file-earmark-text" /></span>

          <div class="page-main">
            <h3 class="page-title">
              <a href="javascript:;" :class="{ 'is-trash': trash }" @click="onTitleClick(item)">
                {{ item.title || '无标题页面' }}
              </a>
              <span class="page-status" :class="statusClass(item)">{{ statusLabel(item) }}</span>
              <!-- 当前可见范围：公开 / 登录可见 / 付费可见（N 积分） -->
              <ScopeBadge :scope="item.scope" :integral="item.scope_integral" />
            </h3>

            <div class="page-meta">
              <code v-if="item.key" class="page-key">/{{ item.key }}</code>
              <span v-else class="page-key warning">未设置标识</span>
              <span class="meta-chip"><i class="bi bi-person" /> {{ authorName(item) }}</span>
              <!-- 驳回原因：审核驳回时填写，作者也会在通知与编辑页看到（审核通过后后端会清空） -->
              <span v-if="item.reason" class="meta-chip is-reason" :title="item.reason">
                <i class="bi bi-info-circle" /> {{ item.reason }}
              </span>
              <span class="meta-text"><i class="bi bi-eye" /> {{ item.views || 0 }}</span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
            </div>

            <p v-if="item.remark" class="page-remark">
              <i class="bi bi-bookmark" /> {{ item.remark }}
            </p>
          </div>

          <div class="page-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>

            <template v-else>
              <a
                v-if="item.key"
                :href="`/${item.key}`"
                target="_blank"
                rel="noopener"
                class="btn btn-ghost btn-sm"
                title="前台查看"
              >
                <i class="bi bi-box-arrow-up-right" />
              </a>
              <router-link :to="`/admin/pages/edit/${item.id}`" class="btn btn-ghost btn-sm" title="编辑">
                <i class="bi bi-pencil" />
              </router-link>
              <button
                v-if="Number(item.audit) !== 1"
                class="btn btn-ghost btn-sm ok"
                title="审核通过"
                :disabled="busy"
                @click="audit(item, 1)"
              >
                <i class="bi bi-check2-circle" />
              </button>
              <button
                v-else
                class="btn btn-ghost btn-sm danger"
                title="审核驳回"
                :disabled="busy"
                @click="audit(item, 2)"
              >
                <i class="bi bi-slash-circle" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="删除" :disabled="busy" @click="askRemove(item)">
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

    <!-- 驳回原因（单条 / 批量共用，审核时能看到这条是谁的） -->
    <AuditDialog
      :visible="auditDialog.visible"
      :audit="auditDialog.audit"
      :default-reason="auditDialog.target?.item?.reason || ''"
      :owner-uid="auditDialog.target?.item?.uid || 0"
      owner-label="作者"
      :loading="auditDialog.loading"
      :message="
        auditDialog.target?.batch
          ? `将驳回已选中的 ${selectedIds.length} 个页面，原因会一并通知各自的作者。`
          : '驳回后作者会收到通知，其中包含你填写的原因。'
      "
      @update:visible="closeAuditDialog"
      @confirm="runAudit"
    />

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
import { ref, reactive, computed, onMounted, watchEffect } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import ScopeBadge from '@/components/ScopeBadge.vue'
import AuditDialog from '@/components/admin/AuditDialog.vue'
import {
  listPages,
  updatePage,
  removePage,
  restorePage,
  forceDeletePage,
  clearPageRecycle
} from '@/api/pages'
import { listUsersByIds } from '@/api/users'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)

// 状态筛选：独立页面没有草稿概念，只有审核状态
const tabs = [
  { key: 'all', label: '全部', where: {} },
  { key: 'pass', label: '已通过', where: { audit: 1 } },
  { key: 'audit', label: '待审核', where: { audit: 0 } },
  { key: 'reject', label: '未通过', where: { audit: 2 } }
]

// 排序白名单（后端 order 未做白名单校验，必须由前端限定取值）
const sortOptions = [
  { value: 'create_time desc', label: '最新创建' },
  { value: 'create_time asc', label: '最早创建' },
  { value: 'update_time desc', label: '最近更新' },
  { value: 'publish_time desc', label: '最新发布' },
  { value: 'views desc', label: '浏览量最多' }
]

const status = ref('all')
const sortKey = ref('create_time desc')
const keyword = ref('')
const searchKey = ref('')

const authorMap = ref({})
const selectedIds = ref([])

const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  action: null
})

const currentTab = computed(() => tabs.find((tab) => tab.key === status.value) || tabs[0])

const emptyText = computed(() => {
  if (trash.value) return '回收站是空的'
  if (searchKey.value) return '没有匹配的页面'
  return '还没有独立页面，先写一个吧'
})

// ---------- 状态展示 ----------
function statusLabel(item) {
  const audit = Number(item.audit)
  if (audit === 1) return '已通过'
  if (audit === 2) return '未通过'
  return '待审核'
}

function statusClass(item) {
  const audit = Number(item.audit)
  if (audit === 1) return 'is-pub'
  if (audit === 2) return 'is-reject'
  return 'is-audit'
}

function authorName(item) {
  const uid = Number(item.uid)
  if (!uid) return '—'
  const user = authorMap.value[uid]
  return user?.nickname || `用户 ${uid}`
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.update_time || item.create_time)
}

// 回收站内的页面没有前台地址，点击不跳转
function onTitleClick(item) {
  if (trash.value) return
  if (item.key) window.open(`/${item.key}`, '_blank', 'noopener')
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  try {
    // 回收站忽略审核状态筛选（被删除的页面可能处于任意状态）
    const where = trash.value ? {} : { ...currentTab.value.where }
    const kw = searchKey.value.replace(/[%_'\\]/g, '').trim()
    if (kw) where.title = { $like: `%${kw}%` }

    const params = {
      page: page.value,
      limit: pageSize.value,
      order: trash.value ? 'delete_time desc' : sortKey.value,
      where: JSON.stringify(where)
    }
    if (trash.value) params.onlyTrashed = true

    const res = await listPages(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
    await loadAuthors(list.value)
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 批量拉取当前页作者昵称（一次请求）
async function loadAuthors(items) {
  const ids = [...new Set((items || []).map((i) => Number(i.uid)).filter(Boolean))]
  if (!ids.length) return
  try {
    const res = await listUsersByIds(ids)
    const map = { ...authorMap.value }
    ;(res.data?.data || []).forEach((u) => { map[u.id] = u })
    authorMap.value = map
  } catch {
    /* 作者信息缺失时回退为「用户 ID」展示，不阻断列表 */
  }
}

function reload() {
  page.value = 1
  load()
}

function switchTab(key) {
  const wasTrash = trash.value
  trash.value = false
  if (!wasTrash && status.value === key) return
  status.value = key
  reload()
}

function toggleTrash() {
  trash.value = !trash.value
  reload()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reload()
}, 350)

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

// ---------- 选中 ----------
function isSelected(id) {
  return selectedIds.value.includes(id)
}

// 本页全选：以当前列表为范围（翻页时 load() 会清空选择）
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((i) => isSelected(i.id))
)
const someSelected = computed(() => selectedIds.value.length > 0 && !pageAllSelected.value)
const selectAllRef = ref(null)

watchEffect(() => {
  if (selectAllRef.value) selectAllRef.value.setIndeterminate(someSelected.value)
})

function toggleSelect(id) {
  const idx = selectedIds.value.indexOf(id)
  if (idx > -1) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}

function toggleSelectAll() {
  selectedIds.value = pageAllSelected.value ? [] : list.value.map((i) => i.id)
}

function clearSelection() {
  selectedIds.value = []
}

// ---------- 确认弹窗 ----------
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

// ---------- 单条操作 ----------
/**
 * 局部更新页面
 * 注意：后端 pages 的校验器要求 key 必填（required,alphaDash），
 * 因此审核等局部更新也必须回传原有 key，否则会返回「唯一键不能为空」。
 */
function patchPayload(item, patch) {
  return {
    id: item.id,
    key: item.key || '',
    ...patch
  }
}

// ---------- 审核（驳回需填原因） ----------
// 驳回（audit=2）要先弹出原因输入框：原因会写进通知（邮件 + 站内信），
// 作者也能在编辑页看到；弹窗里同时显示作者，审核时不必回列表核对
const auditDialog = reactive({
  visible: false,
  audit: 2,
  loading: false,
  // 单条：{ item }；批量：{ batch: true }
  target: null
})

async function applyAudit(item, value, reason = '') {
  if (!item.key) {
    toast.warning('该页面缺少唯一标识，请先编辑补全后再审核')
    return false
  }
  busy.value = true
  try {
    await updatePage(patchPayload(item, { audit: value, reason }))
    item.audit = value
    item.reason = reason
    toast.success(value === 1 ? '已通过审核' : '已驳回')
    return true
  } catch {
    /* 拦截器已提示 */
    return false
  } finally {
    busy.value = false
  }
}

function audit(item, value) {
  // 通过：直接生效；驳回：先让管理员填原因
  if (Number(value) === 2) {
    auditDialog.target = { item }
    auditDialog.audit = 2
    auditDialog.visible = true
    return
  }
  applyAudit(item, value)
}

async function runAudit(reason) {
  const target = auditDialog.target
  if (!target) {
    auditDialog.visible = false
    return
  }

  auditDialog.loading = true
  try {
    if (target.batch) {
      await batchPatch({ audit: auditDialog.audit, reason })
      auditDialog.visible = false
    } else if (await applyAudit(target.item, auditDialog.audit, reason)) {
      auditDialog.visible = false
    }
  } finally {
    auditDialog.loading = false
  }
}

function closeAuditDialog(visible) {
  auditDialog.visible = visible
  if (visible) return
  auditDialog.target = null
}

function askRemove(item) {
  openConfirm({
    title: '删除页面',
    message: `确定删除页面《${item.title || item.key}》吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removePage([item.id])
      toast.success('已移入回收站')
      if (list.value.length === 1 && page.value > 1) page.value -= 1
      await load()
    }
  })
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除页面《${item.title || item.key}》吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeletePage([item.id])
      toast.success('已彻底删除')
      if (list.value.length === 1 && page.value > 1) page.value -= 1
      await load()
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restorePage([item.id])
    toast.success('已恢复')
    if (list.value.length === 1 && page.value > 1) page.value -= 1
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

// ---------- 批量操作 ----------
// 逐条更新（后端 update 仅支持单条 id），统计成功/失败
async function batchPatch(patch) {
  const targets = list.value.filter((i) => isSelected(i.id))
  if (!targets.length) return
  busy.value = true
  let ok = 0
  for (const item of targets) {
    if (!item.key) continue
    try {
      await updatePage(patchPayload(item, patch))
      Object.assign(item, patch)
      ok += 1
    } catch {
      /* 单条失败不影响其它 */
    }
  }
  busy.value = false
  clearSelection()
  const skipped = targets.filter((i) => !i.key).length
  const tail = skipped ? `，${skipped} 个因缺少唯一标识被跳过` : ''
  toast.success(`操作完成：成功 ${ok} 个，失败 ${targets.length - ok - skipped} 个${tail}`)
  if (ok !== targets.length) await load()
}

function batchAudit(value) {
  // 批量驳回同样要填原因（一份原因写给选中的所有页面，逐条各自通知作者）
  if (Number(value) === 2) {
    auditDialog.target = { batch: true }
    auditDialog.audit = 2
    auditDialog.visible = true
    return
  }
  batchPatch({ audit: value })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个页面吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removePage(ids)
      toast.success(`已删除 ${ids.length} 个`)
      clearSelection()
      if (list.value.length === ids.length && page.value > 1) page.value -= 1
      await load()
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  try {
    await restorePage(ids)
    toast.success(`已恢复 ${ids.length} 个`)
    clearSelection()
    if (list.value.length === ids.length && page.value > 1) page.value -= 1
    await load()
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
      await forceDeletePage(ids)
      toast.success(`已彻底删除 ${ids.length} 个`)
      clearSelection()
      if (list.value.length === ids.length && page.value > 1) page.value -= 1
      await load()
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
      await clearPageRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await load()
    }
  })
}

onMounted(load)
</script>

<style scoped>
.page-admin {
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
  font-size: 16px;
  font-weight: 600;
  margin: 0;
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

/* ---------- 筛选栏 ---------- */
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
  /* 与筛选栏内的下拉 / 搜索框 / 按钮等高 */
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

/* ---------- 列表 ---------- */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}

.page-list {
  display: flex;
  flex-direction: column;
}
.page-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 8px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.page-row:last-child {
  border-bottom: none;
}
.page-row.is-selected {
  background: var(--accent-wash);
}
.page-row:hover {
  background: var(--bg-muted);
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
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

.page-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 16px;
}

.page-main {
  flex: 1;
  min-width: 0;
}
.page-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.page-title a {
  color: var(--text);
}
.page-title a:hover {
  color: var(--primary);
}
.page-title a.is-trash {
  cursor: default;
}
.page-status {
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 400;
  border-radius: 3px;
}
.page-status.is-pub {
  background: rgba(108, 154, 77, 0.12);
  color: var(--success);
}
.page-status.is-audit {
  background: var(--gold-wash);
  color: var(--warning);
}
.page-status.is-reject {
  background: var(--accent-soft);
  color: var(--danger);
}

.page-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
}
.page-key {
  padding: 1px 7px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.page-key.warning {
  color: var(--warning);
  background: var(--gold-wash);
}
.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  background: var(--bg-muted);
  border-radius: 3px;
}
/* 驳回原因：单行省略，完整内容看 title */
.meta-chip.is-reason {
  max-width: 260px;
  background: rgba(217, 84, 77, 0.1);
  color: var(--danger);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.page-remark {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--text-light);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.page-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.page-actions .ok:hover:not(:disabled) {
  border-color: var(--success);
  color: var(--success);
}
.page-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
  .page-row {
    flex-wrap: wrap;
  }
  .page-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
