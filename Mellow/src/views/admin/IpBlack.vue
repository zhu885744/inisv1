<template>
  <div class="ip-black-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">IP 黑名单</h2>
          <p class="block-desc">
            封禁恶意 IP：支持 1 小时 / 24 小时 / 7 天 / 永久四个等级；非管理员看到的 IP 会被后端脱敏
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
            <i class="bi bi-plus-lg" /> 添加封禁
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
        <div
          v-for="s in statCards"
          :key="s.label"
          class="stat-card"
          :class="{ 'is-link': s.key !== undefined }"
          @click="s.key !== undefined && pickFilter(s.key)"
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
          <h2 class="block-title">封禁列表</h2>
          <p class="block-desc">
            已过期的临时封禁不再拦截请求，可清理或转为永久
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
            v-model="levelFilter"
            :options="levelOptions"
            icon="bi bi-layers"
            placeholder="全部等级"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索 IP 或原因…"
              aria-label="搜索 IP 或原因"
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
              <i class="bi bi-trash" /> 解除封禁
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的记录不再拦截请求，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-shield-x'" :text="emptyText" />
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

        <table class="ip-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th>IP</th>
              <th class="cell-level">等级</th>
              <th class="cell-state">状态</th>
              <th class="cell-count">违规</th>
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
                <code class="ip-value">{{ item.ip }}</code>
                <div class="ip-sub">
                  <span v-if="item.cause" class="meta-text"><i class="bi bi-exclamation-circle" /> {{ item.cause }}</span>
                  <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
                  <span v-if="item.agent" class="meta-text user-agent-text" :title="item.agent">
                    <i class="bi bi-browser-chrome" /> {{ item.agent }}
                  </span>
                  <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                </div>
              </td>
              <td class="cell-level">
                <span class="level-chip" :class="`is-${levelKey(item)}`">{{ levelLabel(item) }}</span>
              </td>
              <td class="cell-state">
                <template v-if="isPermanent(item)">
                  <span class="state-chip is-permanent">永久封禁</span>
                </template>
                <template v-else-if="isExpired(item)">
                  <span class="state-chip is-expired">已过期</span>
                </template>
                <template v-else>
                  <span class="state-chip is-active">{{ fromNow(item.expire_time) }}解封</span>
                </template>
              </td>
              <td class="cell-count">{{ item.violation_count ?? 0 }} 次</td>
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
                    <button class="btn btn-ghost btn-sm danger" title="解除封禁" aria-label="解除封禁" :disabled="busy" @click="askRemove(item)">
                      <i class="bi bi-shield-check" />
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

    <!-- 新建 / 编辑封禁 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑封禁 #${edit.id}` : '添加封禁'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-shield-plus'"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">IP 地址</label>
        <FormInput v-model="edit.ip" type="text" placeholder="必填，如 192.168.1.1" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">封禁等级</label>
          <FormSelect v-model="edit.level" :options="editLevelOptions" />
        </div>
        <div class="form-item">
          <label class="form-label">自定义时长（小时）</label>
          <FormInput
            v-model="edit.duration"
            type="number"
            min="0"
            step="1"
            :disabled="edit.level === 4"
            placeholder="留空按等级默认时长" />
          <p class="form-hint">留空则按等级取默认（1 / 24 / 168 小时）</p>
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">封禁原因</label>
          <FormInput v-model="edit.cause" type="text" placeholder="可选" />
        </div>
        <div class="form-item">
          <label class="form-label">备注</label>
          <FormInput v-model="edit.remark" type="text" placeholder="可选，仅管理端可见" />
        </div>
      </div>

      <p class="dialog-tip">
        选择「4 级（永久）」会置 <strong>is_permanent=1</strong>，此时时长与解封时间都会被清零；
        改为 1~3 级或提交自定义时长即恢复为临时封禁。
      </p>
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
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * IP 黑名单（/admin/ip/black）
 *
 * 后端约束（app/api/controller/ip-black.go、app/model/ip-black.go）：
 * - 封禁等级 level 1~3 对应 1 / 24 / 168 小时；**4 级不会自动变永久**，
 *   永久要显式提交 is_permanent=true（后端会把 duration / expire_time 归零）；
 * - 未传 duration 时按 level 取默认时长并推出 expire_time；
 * - BeforeCreate 校验 IP 不重复；IP 字段会被统一转大写；
 * - 非 root 查询时后端会对 ip 做脱敏（MaskIP）；
 * - 「已过期」= 非永久且 expire_time 早于当前时间。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listIpBlack,
  countIpBlack,
  saveIpBlack,
  removeIpBlack,
  forceDeleteIpBlack,
  restoreIpBlack,
  clearIpBlackRecycle
} from '@/api/ip-black'
import { fromNow, formatTime } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 状态筛选（expired 需要按当前时间比较，单独在列表里拼条件）
const tabs = [
  { key: 'all', label: '全部' },
  { key: 'permanent', label: '永久' },
  { key: 'temporary', label: '临时' },
  { key: 'expired', label: '已过期' }
]

// 等级下拉（与后端 BanLevel 常量对应）
const levelOptions = [
  { value: '', label: '全部等级' },
  { value: 1, label: '1 级（1 小时）' },
  { value: 2, label: '2 级（24 小时）' },
  { value: 3, label: '3 级（7 天）' },
  { value: 4, label: '4 级（永久）' }
]

const editLevelOptions = [
  { value: 1, label: '1 级（1 小时）' },
  { value: 2, label: '2 级（24 小时）' },
  { value: 3, label: '3 级（7 天）' },
  { value: 4, label: '4 级（永久）' }
]

// ===== 列表 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const status = ref('all')
const levelFilter = ref('')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ total: 0, permanent: 0, temporary: 0, expired: 0, trash: 0 })

// ===== 弹窗 =====
const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  ip: '',
  level: 1,
  duration: '',
  cause: '',
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
  { label: '封禁总数', value: stats.total, icon: 'bi bi-shield-x', color: 'var(--primary)', key: 'all' },
  { label: '永久封禁', value: stats.permanent, icon: 'bi bi-infinity', color: 'var(--danger)', key: 'permanent' },
  { label: '临时封禁', value: stats.temporary, icon: 'bi bi-hourglass-split', color: 'var(--warning)', key: 'temporary' },
  { label: '已过期', value: stats.expired, icon: 'bi bi-clock-history', color: 'var(--text-muted)', key: 'expired' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的记录'
  if (trash.value) return '回收站是空的'
  if (status.value === 'expired') return '没有已过期的封禁'
  return '黑名单是空的'
})

// ---------- 展示辅助 ----------
function nowUnix() {
  return Math.floor(Date.now() / 1000)
}

function isPermanent(item) {
  return Number(item?.is_permanent) === 1
}

function isExpired(item) {
  if (isPermanent(item)) return false
  const expire = Number(item?.expire_time || 0)
  return expire > 0 && expire < nowUnix()
}

function levelKey(item) {
  return isPermanent(item) ? 4 : Number(item?.level || 1)
}

function levelLabel(item) {
  return levelKey(item) === 4 ? '4 级（永久）' : `${levelKey(item)} 级`
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  if (!isPermanent(item) && Number(item.expire_time) > 0) {
    return `封禁于 ${fromNow(item.create_time)} · 解封 ${formatTime(item.expire_time)}`
  }
  return `封禁于 ${fromNow(item.create_time)}`
}

// ---------- 筛选条件 ----------
// 状态与等级叠加到同一个 where
function buildWhere() {
  const where = {}
  const now = nowUnix()

  if (status.value === 'permanent') {
    where.is_permanent = 1
  } else if (status.value === 'temporary') {
    where.is_permanent = 0
    where.expire_time = { $gte: now }
  } else if (status.value === 'expired') {
    where.is_permanent = 0
    where.expire_time = { $lt: now }
  }

  if (levelFilter.value !== '') where.level = Number(levelFilter.value)

  return Object.keys(where).length ? JSON.stringify(where) : null
}

// ---------- 统计 ----------
async function loadStats() {
  const now = nowUnix()
  loadingStats.value = true
  try {
    const [totalRes, permRes, tempRes, expiredRes, trashRes] = await Promise.all([
      countIpBlack(),
      countIpBlack({ where: JSON.stringify({ is_permanent: 1 }) }),
      countIpBlack({ where: JSON.stringify({ is_permanent: 0, expire_time: { $gte: now } }) }),
      countIpBlack({ where: JSON.stringify({ is_permanent: 0, expire_time: { $lt: now } }) }),
      countIpBlack({ onlyTrashed: true })
    ])
    stats.total = Number(totalRes?.data || 0)
    stats.permanent = Number(permRes?.data || 0)
    stats.temporary = Number(tempRes?.data || 0)
    stats.expired = Number(expiredRes?.data || 0)
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
      order: trash.value ? 'delete_time desc' : 'id desc'
    }
    if (trash.value) params.onlyTrashed = true
    else {
      const where = buildWhere()
      if (where) params.where = where
    }

    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `ip|${kw}`

    const res = await listIpBlack(params)
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

async function toggleTrash() {
  trash.value = !trash.value
  keyword.value = ''
  searchKey.value = ''
  page.value = 1
  await load()
}

function pickFilter(key) {
  switchStatus(key)
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
  edit.ip = ''
  edit.level = 1
  edit.duration = ''
  edit.cause = ''
  edit.remark = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.ip = item.ip || ''
  edit.level = isPermanent(item) ? 4 : Number(item.level || 1)
  edit.duration = isPermanent(item) ? '' : String(item.duration || '')
  edit.cause = item.cause || ''
  edit.remark = item.remark || ''
  edit.visible = true
}

async function save() {
  const ip = edit.ip.trim()
  if (!ip) {
    toast.warning('请输入 IP 地址')
    return
  }
  const duration = Number(String(edit.duration).trim())
  if (String(edit.duration).trim() !== '' && (!Number.isInteger(duration) || duration <= 0)) {
    toast.warning('自定义时长必须为大于 0 的整数（小时）')
    return
  }

  const permanent = Number(edit.level) === 4
  const payload = {
    ip,
    level: permanent ? 4 : Number(edit.level),
    is_permanent: permanent,
    cause: edit.cause.trim(),
    remark: edit.remark.trim()
  }
  // 永久封禁时长归零；临时封禁可显式指定小时数
  if (!permanent) {
    payload.duration = String(edit.duration).trim() === '' ? 0 : duration
  } else {
    payload.duration = 0
  }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveIpBlack(payload)
    toast.success(edit.id ? '修改已保存' : '已加入黑名单')
    edit.visible = false
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出（如 IP 已存在）
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

// ---------- 解除 / 恢复 ----------
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
    title: '解除封禁',
    message: `确定解除 IP「${item.ip}」的封禁吗？解除后该 IP 可正常访问，可在回收站找回记录。`,
    confirmText: '解除',
    danger: true,
    action: async () => {
      await removeIpBlack([item.id])
      toast.success('已解除封禁')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreIpBlack([item.id])
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
    message: `确定彻底删除 IP「${item.ip}」的封禁记录吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteIpBlack([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量解除封禁',
    message: `确定解除选中的 ${ids.length} 条封禁吗？`,
    confirmText: '解除',
    danger: true,
    action: async () => {
      await removeIpBlack(ids)
      toast.success(`已解除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    await restoreIpBlack(ids)
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
    message: `确定彻底删除选中的 ${ids.length} 条记录吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteIpBlack(ids)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有封禁记录将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearIpBlackRecycle()
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
.ip-black-admin {
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

.ip-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.ip-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.ip-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.ip-table tr:last-child td {
  border-bottom: none;
}
.ip-table tbody tr:hover td {
  background: var(--bg-muted);
}
.ip-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-level {
  width: 108px;
}
.cell-state {
  width: 130px;
}
.cell-count {
  width: 72px;
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
.ip-sub {
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
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.level-chip {
  display: inline-block;
  padding: 2px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.level-chip.is-1 {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.level-chip.is-2 {
  color: var(--warning);
  background: var(--gold-wash);
}
.level-chip.is-3 {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.12);
}
.level-chip.is-4 {
  color: var(--danger);
  background: var(--accent-soft);
}

.state-chip {
  display: inline-block;
  padding: 2px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.state-chip.is-permanent {
  color: var(--danger);
  background: var(--accent-soft);
}
.state-chip.is-active {
  color: var(--warning);
  background: var(--gold-wash);
}
.state-chip.is-expired {
  color: var(--text-muted);
  background: var(--bg-muted);
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
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.dialog-tip {
  margin: 0;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius);
}

@media (max-width: 768px) {
  .list-filter,
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .ip-table th:nth-child(5),
  .ip-table td:nth-child(5) {
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
