<template>
  <div class="api-keys-admin">
    <!-- 密钥校验状态：只读展示，开关本身在「安全中心 → 安全与限流」里，这里给跳转 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">密钥校验</h2>
          <p class="block-desc">
            开启后调用接口需携带有效密钥（请求头 <code>i-api-key</code> 或 query
            <code>?i-api-key=</code>）；密钥在下方的列表里管理，开关本身在「安全中心 → 安全与限流」里
          </p>
        </div>
        <div class="head-actions">
          <RouterLink to="/admin/security/policy" class="btn btn-primary btn-sm">
            <i class="bi bi-sliders" /> 去安全中心修改
          </RouterLink>
        </div>
      </header>

      <div class="key-state">
        <span class="key-state-item">
          启用校验：
          <strong :class="keyState.enabled === '1' ? 'is-on' : 'is-off'">
            {{ keyState.enabled === '1' ? '开启' : '关闭' }}
          </strong>
        </span>
        <span class="key-state-item">
          放行本站请求：
          <strong :class="keyState.local === '1' ? 'is-on' : 'is-off'">
            {{ keyState.local === '1' ? '开启' : '关闭' }}
          </strong>
        </span>
      </div>

      <p class="block-desc">
        <i class="bi bi-info-circle" />
        「放行本站请求」开启时，同源页面请求与本机·内网直连免密钥；关闭后，打包进二进制的站内前端与后台会 403
        （此时必须由反向代理注入 <code>i-api-key</code>）。
        同源判定（Origin / Referer / Sec-Fetch-Site）可被脚本伪造，它挡的是扫描器与无脑爬虫，不是安全边界。
      </p>
    </section>

    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">接口密钥</h2>
          <p class="block-desc">用于第三方调用接口的凭证；密钥值重复时后端会拒绝创建</p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
            <i class="bi bi-plus-lg" /> 新建密钥
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
          <h2 class="block-title">密钥列表</h2>
          <p class="block-desc">
            密钥只在列表里完整展示一次，请妥善保存
            <template v-if="total > 0"> · 共 {{ total }} 个</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-ghost btn-sm" @click="maskAll = !maskAll">
            <i :class="maskAll ? 'bi bi-eye' : 'bi bi-eye-slash'" /> {{ maskAll ? '显示密钥' : '隐藏密钥' }}
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
            @click="switchTrash(false)"
          >全部密钥</button>
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
              placeholder="搜索备注或密钥…"
              aria-label="搜索备注或密钥"
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
        <span><i class="bi bi-trash3" /> 回收站内的密钥已失效，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-key'" :text="emptyText" />
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

        <table class="key-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th>密钥</th>
              <th class="cell-time">创建时间</th>
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
                    :aria-label="`选择密钥 ${item.value}`"
                    @change="toggleSelect(item.id)" />
                </label>
              </td>
              <td>
                <div class="key-row">
                  <code class="key-value">{{ maskedValue(item) }}</code>
                  <button
                    class="btn btn-ghost btn-sm"
                    :title="item.revealed ? '隐藏' : '显示'"
                    :aria-label="item.revealed ? '隐藏' : '显示'"
                    @click="toggleReveal(item)"
                  >
                    <i :class="item.revealed ? 'bi bi-eye-slash' : 'bi bi-eye'" />
                  </button>
                  <button class="btn btn-ghost btn-sm" title="复制密钥" aria-label="复制密钥" @click="copyValue(item)">
                    <i class="bi bi-clipboard" />
                  </button>
                </div>
                <div class="key-sub">
                  <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
                  <span v-else class="meta-text text-light">未填写备注</span>
                  <!-- 状态与有效期：回收站里的密钥一律已失效，不在这里展示 -->
                  <template v-if="!trash">
                    <span class="meta-chip" :class="statusClass(item)">{{ statusText(item) }}</span>
                    <span class="meta-text" :class="{ 'is-warn': isExpired(item) }">
                      <i class="bi bi-calendar-check" /> {{ expireText(item) }}
                    </span>
                  </template>
                </div>
              </td>
              <td class="cell-time">
                <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                <!-- 用量审计：累计调用次数 + 最后一次使用时间（从未使用时只显示 0 次） -->
                <span v-if="!trash" class="meta-text">
                  <i class="bi bi-activity" /> 已调用 {{ Number(item.use_count) || 0 }} 次
                  <template v-if="Number(item.last_used_at) > 0">
                    · 最近 {{ fromNow(item.last_used_at) }}
                  </template>
                </span>
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
                    <button
                      class="btn btn-ghost btn-sm"
                      :title="Number(item.status) === 1 ? '停用（拒绝调用）' : '启用'"
                      :aria-label="Number(item.status) === 1 ? '停用' : '启用'"
                      :disabled="busy"
                      @click="toggleStatus(item)"
                    >
                      <i :class="Number(item.status) === 1 ? 'bi bi-pause-circle' : 'bi bi-play-circle'" />
                    </button>
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

    <!-- 新建 / 编辑密钥 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑密钥 #${edit.id}` : '新建密钥'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">密钥值</label>
        <div class="key-input-row">
          <FormInput v-model="edit.value" type="text" placeholder="点「随机生成」或自行填写（留空则保存时由后端生成）" />
          <button class="btn btn-sm" type="button" @click="edit.value = generateKey()">
            <i class="bi bi-shuffle" /> 随机生成
          </button>
        </div>
        <p class="form-hint">
          <template v-if="edit.id">
            点「随机生成」会填入一个新密钥，保存后原密钥立即失效（相当于重置）；
            留空保存则交给后端重新生成
          </template>
          <template v-else>
            点「随机生成」直接填入 32 位大写密钥（可连点换一个、也可手工修改）；
            留空保存则由后端生成；后端统一转大写并校验唯一
          </template>
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">备注</label>
        <FormInput v-model="edit.remark" type="text" placeholder="可选，便于区分用途" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">状态</label>
          <FormSelect v-model="edit.status" :options="STATUS_OPTIONS" />
          <p class="form-hint">停用后调用返回 403「该密钥已停用！」，可随时改回启用</p>
        </div>
        <div class="form-item">
          <label class="form-label">有效期</label>
          <FormInput v-model="edit.expireAt" type="datetime-local" />
          <p class="form-hint">留空 = 永久有效；到期后调用返回 403「该密钥已过期！」</p>
        </div>
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
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 接口密钥（/admin/api/keys）
 *
 * 后端约束（app/api/controller/api-keys.go、app/api/middleware/api-key.go）：
 * - 允许字段：value / remark / status / expire_time，value 会被统一转大写；
 * - create：value 为空时自动生成（UUID 去横线转大写），重复则返回「已经存在！」；status 缺省为「启用」；
 * - update：value 为空时**重新生成**密钥，因此编辑弹窗留空即等于重置密钥；
 *   只想改状态 / 有效期 / 备注时必须带上当前 value（见下方的 toggleStatus）；
 * - 校验：开启「系统设置 → 安全 → API KEY」后所有 /api/* 都要带 i-api-key；
 *   密钥「停用」或「已过期」会被拒绝（403 该密钥已停用！/ 该密钥已过期！）；
 * - 用量：校验通过时异步累加 use_count 并刷新 last_used_at（用于审计谁在用、还用不用）；
 * - 删除为软删除，彻底删除不可恢复。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listApiKeys,
  countApiKeys,
  saveApiKey,
  removeApiKeys,
  forceDeleteApiKeys,
  restoreApiKeys,
  clearApiKeyRecycle
} from '@/api/api-keys'
// 顶部只读展示「启用校验 / 放行本站请求」的状态（开关在「系统设置 → 安全 → API KEY」里改）
import { getConfig } from '@/api/config'
import { fromNow, formatTime, toLocalInput, fromLocalInput } from '@/utils/time'
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
// 全局打码开关；单条可通过 revealed 单独显示
const maskAll = ref(false)

// ===== 顶部的密钥校验状态（只读）=====
// enabled: SYSTEM_API_KEY（是否开启密钥校验）
// local:   SYSTEM_API_KEY_LOCAL（开启校验时是否放行本站来源请求）
// 这两项在本页只展示，修改入口在「系统设置 → 安全 → API KEY」
const keyState = reactive({ enabled: '0', local: '1' })

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ total: 0, trash: 0 })

// ===== 弹窗 =====
// 状态下拉（统一走 SelectMenu，不用浏览器原生 select）
const STATUS_OPTIONS = [
  { value: 1, label: '启用（可正常调用）' },
  { value: 0, label: '停用（拒绝调用，但保留记录）' }
]

const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  value: '',
  remark: '',
  // 状态：1 启用 / 0 停用（后端缺省即启用）
  status: 1,
  // 有效期：datetime-local 字符串，空 = 永久有效（提交时转 unix 秒，0 = 永久）
  expireAt: ''
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
  { label: '密钥总数', value: stats.total, icon: 'bi bi-key', color: 'var(--primary)' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的密钥'
  if (trash.value) return '回收站是空的'
  return '还没有密钥，新建一个吧'
})

// ---------- 展示辅助 ----------
function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

// 打码保留首尾各 4 位，便于核对
function maskValue(value) {
  const text = String(value || '')
  if (text.length <= 8) return '•'.repeat(text.length)
  return `${text.slice(0, 4)}${'•'.repeat(Math.min(12, text.length - 8))}${text.slice(-4)}`
}

function maskedValue(item) {
  if (item.revealed || !maskAll.value) return item.value
  return maskValue(item.value)
}

function toggleReveal(item) {
  item.revealed = !item.revealed
}

// 生成 32 位大写密钥：格式与后端 generateAPIKey() 一致（UUID 去横线后转大写）
//
// 为什么放在前端生成而不是"清空让后端生成"：后者要保存之后才看得到值，
// 既不能当场核对/复制，也没法连点几次挑一个再保存。这里只是"帮你填一个字符串"，
// 唯一性仍由后端保存时校验（重复会返回「已经存在！」）。
// 随机源优先 crypto.randomUUID（安全随机），降级到 getRandomValues / Math.random。
function generateKey() {
  let raw = ''

  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    raw = crypto.randomUUID()
  } else if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    raw = [...crypto.getRandomValues(new Uint8Array(16))]
      .map((byte) => byte.toString(16).padStart(2, '0'))
      .join('')
  } else {
    for (let i = 0; i < 32; i += 1) raw += Math.floor(Math.random() * 16).toString(16)
  }

  return raw.replace(/-/g, '').toUpperCase()
}

// 是否已过期（expire_time = 0 表示永久有效）
function isExpired(item) {
  const end = Number(item?.expire_time) || 0
  return end > 0 && Math.floor(Date.now() / 1000) > end
}

// 状态文案 / 样式：已过期优先展示（比「启用中」更贴近实际可用性）
function statusText(item) {
  if (isExpired(item)) return '已过期'
  return Number(item?.status) === 1 ? '启用中' : '已停用'
}

function statusClass(item) {
  if (isExpired(item)) return 'is-expired'
  return Number(item?.status) === 1 ? 'is-on' : 'is-off'
}

function expireText(item) {
  const end = Number(item?.expire_time) || 0
  return end > 0 ? `有效期至 ${formatTime(end)}` : '永久有效'
}

async function copyValue(item) {
  if (!item.value) return
  try {
    await navigator.clipboard.writeText(item.value)
    toast.success('密钥已复制')
  } catch {
    toast.info('复制失败，请手动复制')
  }
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [totalRes, trashRes] = await Promise.all([
      countApiKeys(),
      countApiKeys({ onlyTrashed: true })
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
      order: trash.value ? 'delete_time desc' : 'id desc'
    }
    if (trash.value) params.onlyTrashed = true
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    // 备注或密钥值都可能被搜，密钥用 like，备注用 like 只能二选一，这里按「或」分两次太重，
    // 统一走 remark 字段（后端口径），密钥完整值可在页面上复制
    if (kw) params.like = `remark|${kw}`

    const res = await listApiKeys(params)
    // revealed 是前端局部状态，随数据重建重置
    list.value = (res.data?.data || []).map((item) => ({ ...item, revealed: false }))
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
  edit.value = ''
  edit.remark = ''
  edit.status = 1
  edit.expireAt = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.value = item.value || ''
  edit.remark = item.remark || ''
  edit.status = Number(item.status) === 0 ? 0 : 1
  edit.expireAt = toLocalInput(item.expire_time)
  edit.visible = true
}

async function save() {
  const expireTime = fromLocalInput(edit.expireAt)
  // 留空 = 永久有效；填了已过去的时间等于立刻失效，明显是误操作 → 拦下来
  if (expireTime > 0 && expireTime <= Math.floor(Date.now() / 1000)) {
    toast.warning('有效期必须晚于当前时间（留空表示永久有效）')
    return
  }

  const payload = {
    value: edit.value.trim(),
    remark: edit.remark.trim(),
    status: Number(edit.status) === 1 ? 1 : 0,
    expire_time: expireTime
  }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveApiKey(payload)
    toast.success(edit.id ? '修改已保存' : '密钥已创建')
    edit.visible = false
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出（如密钥已存在）
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

// 快捷停用 / 启用
//
// ⚠️ 必须带上当前 value：后端 update 的语义是「value 为空就重新生成密钥」，
// 只传 { id, status } 会把密钥换掉，导致所有调用方一起失效。
async function toggleStatus(item) {
  const next = Number(item.status) === 1 ? 0 : 1

  busy.value = true
  try {
    await saveApiKey({
      id: item.id,
      value: item.value,
      remark: item.remark || '',
      status: next,
      expire_time: Number(item.expire_time) || 0
    })
    toast.success(next === 1 ? '已启用' : '已停用（该密钥的调用会被立即拒绝）')
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    busy.value = false
  }
}

function keyLabel(item) {
  return item.remark || `密钥 #${item.id}`
}

function askRemove(item) {
  openConfirm({
    title: '删除密钥',
    message: `确定删除「${keyLabel(item)}」吗？删除后使用该密钥的调用会立即失效，可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeApiKeys([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreApiKeys([item.id])
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
    message: `确定彻底删除「${keyLabel(item)}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteApiKeys([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个密钥吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeApiKeys(ids)
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
    await restoreApiKeys(ids)
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
    message: `确定彻底删除选中的 ${ids.length} 个密钥吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteApiKeys(ids)
      toast.success(`已彻底删除 ${ids.length} 个`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有密钥将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearApiKeyRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

// ---------- 校验开关 ----------

// 读配置项（未初始化 / 无权限时静默返回 null，避免弹全局提示）
async function readConfigValue(key) {
  const res = await getConfig(key, { silent: true })
  return res?.data && typeof res.data === 'object' ? res.data : null
}

async function loadKeyState() {
  try {
    const [enabled, local] = await Promise.all([
      readConfigValue('SYSTEM_API_KEY'),
      readConfigValue('SYSTEM_API_KEY_LOCAL')
    ])
    keyState.enabled = enabled && String(enabled.value) === '1' ? '1' : '0'
    // 取不到时按「放行」处理：与系统设置页口径一致，避免老库升级瞬间把站内前端挡住
    keyState.local = local ? (String(local.value) === '1' ? '1' : '0') : '1'
  } catch {
    // 读不到就保持默认值
  }
}

onMounted(() => {
  load()
  loadStats()
  loadKeyState()
})
</script>

<style scoped>
.api-keys-admin {
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

.key-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.key-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.key-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.key-table tr:last-child td {
  border-bottom: none;
}
.key-table tbody tr:hover td {
  background: var(--bg-muted);
}
.key-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-time {
  width: 170px;
}
/* 时间列里「创建时间」与「用量」分两行（两个 inline-flex 会挤在同一行） */
.cell-time .meta-text {
  display: flex;
  margin-top: 2px;
}
.cell-time .meta-text:first-child {
  margin-top: 0;
}
.cell-act {
  width: 132px;
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

.key-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.key-value {
  padding: 2px 8px;
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text);
  background: var(--bg-muted);
  border-radius: 3px;
  word-break: break-all;
}
.key-sub {
  margin-top: 4px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 10px;
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-text.is-warn {
  color: var(--danger);
}
/* 状态标签：启用中 / 已停用 / 已过期 */
.meta-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.meta-chip.is-on {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.meta-chip.is-off {
  color: var(--text-muted);
  background: var(--bg-muted);
}
.meta-chip.is-expired {
  color: var(--danger);
  background: rgba(220, 38, 38, 0.12);
}
/* 顶部校验状态（只读；修改入口在「系统设置 → 安全 → API KEY」） */
.key-state {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 20px;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--text-soft);
}
.key-state-item strong {
  font-weight: 600;
}
.key-state-item strong.is-on {
  color: var(--success);
}
.key-state-item strong.is-off {
  color: var(--text-muted);
}
.text-light {
  color: var(--text-light);
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
.key-input-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.key-input-row .input {
  flex: 1;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}

@media (max-width: 768px) {
  .list-filter,
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .key-table th:nth-child(3),
  .key-table td:nth-child(3) {
    display: none;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
