<template>
  <div class="auth-rules-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">权限规则</h2>
          <p class="block-desc">
            接口级权限点：规则由「请求方法 + 路由」唯一确定，权限分组按规则授权
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
            <i class="bi bi-plus-lg" /> 新建规则
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
          :class="{ 'is-link': s.type !== undefined }"
          :aria-pressed="s.type !== undefined && typeFilter === s.type"
          @click="s.type !== undefined && pickType(s.type)"
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
          <h2 class="block-title">规则列表</h2>
          <p class="block-desc">
            修改路由或方法会重新计算 hash，已授权分组需要重新配置
            <template v-if="total > 0"> · 共 {{ total }} 条</template>
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
          >全部规则</button>
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
          <FormSelect variant="pill"
            v-model="typeFilter"
            :options="AUTH_RULE_TYPE_OPTIONS"
            icon="bi bi-shield-check"
            placeholder="全部类型"
            @change="reload" />

          <FormSelect variant="pill"
            v-model="methodFilter"
            :options="AUTH_RULE_METHOD_OPTIONS"
            icon="bi bi-signpost-split"
            placeholder="全部方法"
            @change="reload" />

          <FormSelect variant="pill"
            v-model="sortKey"
            :options="sortOptions"
            icon="bi bi-sort-down"
            placeholder="排序方式"
            @change="reload" />

          <FormSelect variant="pill"
            v-model="searchField"
            :options="AUTH_RULE_SEARCH_FIELDS"
            icon="bi bi-funnel"
            placeholder="搜索字段" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索规则…"
              aria-label="搜索规则"
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
        <span><i class="bi bi-trash3" /> 回收站内的规则不再参与授权，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-shield-check'" :text="emptyText" />
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

        <table class="rule-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th>规则名称</th>
              <th class="cell-method">方法</th>
              <th>路由</th>
              <th class="cell-type">类型</th>
              <th class="cell-num">费用</th>
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
                    :aria-label="`选择规则 ${item.name}`"
                    @change="toggleSelect(item.id)" />
                </label>
              </td>
              <td>
                <div class="rule-name">{{ item.name || '—' }}</div>
                <div class="rule-sub">
                  <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                  <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
                  <span class="meta-text hash-text" :title="item.hash">#{{ item.hash }}</span>
                </div>
              </td>
              <td class="cell-method">
                <span class="method-chip" :class="methodClass(item)">{{ authRuleMethodText(item) }}</span>
              </td>
              <td><code class="rule-route">{{ item.route }}</code></td>
              <td class="cell-type">
                <span class="type-chip" :class="typeClass(item)">{{ authRuleTypeLabel(item) }}</span>
              </td>
              <td class="cell-num">{{ item.cost ?? 0 }}</td>
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

    <!-- 新建 / 编辑规则 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑规则 #${edit.id}` : '新建规则'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">规则名称</label>
        <FormInput v-model="edit.name" type="text" placeholder="如「【用户 API】获取全部」" />
        <p class="form-hint">留空时后端会自动使用路由作为名称</p>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">请求方法</label>
          <FormSelect v-model="edit.method" :options="methodEditOptions" />
        </div>
        <div class="form-item">
          <label class="form-label">规则类型</label>
          <FormSelect v-model="edit.type" :options="AUTH_RULE_TYPES" />
          <p class="form-hint">公共＝免登录放行；需登录＝登录即放行；默认＝需对应权限点</p>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">路由</label>
        <FormInput v-model="edit.route" type="text" placeholder="如 /api/users/all" />
        <p class="form-hint">格式：<code>/api/控制器/方法</code>；修改后 hash 会重新计算</p>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">费用（cost）</label>
          <FormInput v-model="edit.cost" type="number" min="0" step="1" placeholder="默认 1" />
        </div>
        <div class="form-item">
          <label class="form-label">备注</label>
          <FormInput v-model="edit.remark" type="text" placeholder="可选" />
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
 * 权限规则（/admin/auth/rules）
 *
 * 接口级权限点管理：规则由「请求方法 + 路由」唯一确定，权限分组按规则授权。
 *
 * 后端约束（app/api/controller/auth-rules.go、app/api/middleware/rule.go）：
 * - 规则类型共三种：common（中间件直接放行）/ login（需登录）/ default（需对应权限点）；
 *   早期初始化数据误用过 type=root，中间件从未对其特殊判定（行为等同 default），
 *   种子数据已统一为 default，老库由 model.normalizeAuthRuleTypes 在迁移时纠正；
 * - 允许字段只有 cost/name/route/method/type/remark/json/text；
 * - **hash 由后端按 [METHOD]route 计算**，前端不传；改动 method/route 会让 hash 变化，
 *   已授权分组需要重新配置（已在界面提示）；
 * - 名称为空时后端会用路由兜底，避免规则树按名称分组解析失败；
 * - auth-rules/count 支持 onlyTrashed，回收站数量可直接用计数接口；
 * - 规则的增删改没有额外的 root 校验，依赖路由级权限控制。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listAuthRules,
  countAuthRules,
  saveAuthRule,
  removeAuthRules,
  forceDeleteAuthRules,
  restoreAuthRules,
  clearAuthRuleRecycle
} from '@/api/auth-rules'
import {
  AUTH_RULE_FIELD,
  AUTH_RULE_TYPES,
  AUTH_RULE_TYPE_OPTIONS,
  AUTH_RULE_METHOD_OPTIONS,
  AUTH_RULE_METHODS,
  AUTH_RULE_SEARCH_FIELDS,
  authRuleTypeLabel,
  authRuleTypeKey,
  authRuleMethodText,
  authRuleWhereJSON,
  sanitizeAuthRuleKeyword
} from '@/utils/auth-rule'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 排序白名单：后端 order 未做校验，必须由前端限定取值
const sortOptions = [
  { value: 'id asc', label: '按 ID 升序' },
  { value: 'id desc', label: '按 ID 降序' },
  { value: 'create_time desc', label: '最新创建' },
  { value: 'create_time asc', label: '最早创建' }
]

// ===== 列表 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const typeFilter = ref('')
const methodFilter = ref('')
const sortKey = ref(sortOptions[0].value)
const searchField = ref('name')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ total: 0, login: 0, common: 0, defaultCount: 0, trash: 0 })

// ===== 弹窗 =====
const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  method: 'GET',
  route: '',
  type: 'default',
  cost: 1,
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

// 编辑弹窗的方法下拉不带「全部」
const methodEditOptions = computed(() => AUTH_RULE_METHODS.map((m) => ({ value: m, label: m })))

const statCards = computed(() => [
  { label: '规则总数', value: stats.total, icon: 'bi bi-shield-check', color: 'var(--primary)', type: '' },
  { label: '需登录', value: stats.login, icon: 'bi bi-box-arrow-in-right', color: '#0ea5e9', type: 'login' },
  { label: '公共接口', value: stats.common, icon: 'bi bi-globe', color: 'var(--success)', type: 'common' },
  { label: '默认', value: stats.defaultCount, icon: 'bi bi-shield-lock', color: 'var(--text-muted)', type: 'default' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--warning)' }
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的规则'
  if (trash.value) return '回收站是空的'
  return '暂无权限规则'
})

// ---------- 展示辅助 ----------
function typeClass(item) {
  // root 等遗留标注按 default 归一（后端只认 common / login / default 三类）
  return `is-${authRuleTypeKey(item)}`
}

function methodClass(item) {
  return `is-${authRuleMethodText(item).toLowerCase()}`
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `创建于 ${fromNow(item.create_time)}`
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [totalRes, loginRes, commonRes, defaultRes, trashRes] = await Promise.all([
      countAuthRules(),
      countAuthRules({ where: JSON.stringify({ type: 'login' }) }),
      countAuthRules({ where: JSON.stringify({ type: 'common' }) }),
      countAuthRules({ where: JSON.stringify({ type: 'default' }) }),
      countAuthRules({ onlyTrashed: true })
    ])
    stats.total = Number(totalRes?.data || 0)
    stats.login = Number(loginRes?.data || 0)
    stats.common = Number(commonRes?.data || 0)
    stats.defaultCount = Number(defaultRes?.data || 0)
    stats.trash = Number(trashRes?.data || 0)
  } catch {
    /* 统计失败不阻塞列表 */
  } finally {
    loadingStats.value = false
  }
}

// 点击统计卡片直接按类型筛选
function pickType(type) {
  if (trash.value) {
    trash.value = false
  }
  if (typeFilter.value === type) return
  typeFilter.value = type
  reload()
}

// ---------- 列表 ----------
async function load() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: pageSize.value,
      field: AUTH_RULE_FIELD,
      order: trash.value ? 'delete_time desc' : sortKey.value
    }
    if (trash.value) params.onlyTrashed = true

    const where = authRuleWhereJSON(typeFilter.value, methodFilter.value)
    if (where) params.where = where

    const kw = sanitizeAuthRuleKeyword(searchKey.value)
    if (kw) {
      const field = AUTH_RULE_SEARCH_FIELDS.some((f) => f.value === searchField.value) ? searchField.value : 'name'
      params.like = `${field}|${kw}`
    }

    const res = await listAuthRules(params)
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
  edit.method = 'GET'
  edit.route = ''
  edit.type = 'default'
  edit.cost = 1
  edit.remark = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.name = item.name || ''
  edit.method = authRuleMethodText(item)
  edit.route = item.route || ''
  edit.type = String(item.type || 'default')
  edit.cost = Number(item.cost ?? 1)
  edit.remark = item.remark || ''
  edit.visible = true
}

async function save() {
  const route = edit.route.trim()
  if (!route) {
    toast.warning('请输入路由')
    return
  }
  const cost = Number(String(edit.cost).trim())
  if (!Number.isInteger(cost) || cost < 0) {
    toast.warning('费用必须为不小于 0 的整数')
    return
  }

  const payload = {
    name: edit.name.trim(),
    method: String(edit.method || 'GET').toUpperCase(),
    route,
    type: edit.type,
    cost,
    remark: edit.remark.trim()
  }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveAuthRule(payload)
    toast.success(edit.id ? '修改已保存' : '规则已创建')
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
    title: '删除规则',
    message: `确定删除规则「${item.name || item.route}」吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeAuthRules([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreAuthRules([item.id])
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
    message: `确定彻底删除规则「${item.name || item.route}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAuthRules([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 条规则吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeAuthRules(ids)
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
    await restoreAuthRules(ids)
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
    message: `确定彻底删除选中的 ${ids.length} 条规则吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAuthRules(ids)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有规则将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearAuthRuleRecycle()
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
.auth-rules-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 概览 ---------- */
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
  width: 170px;
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

.rule-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.rule-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.rule-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: top;
}
.rule-table tr:last-child td {
  border-bottom: none;
}
.rule-table tbody tr:hover td {
  background: var(--bg-muted);
}
.rule-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-method {
  width: 84px;
}
.cell-type {
  width: 108px;
}
.cell-num {
  width: 72px;
  text-align: right;
  font-variant-numeric: tabular-nums;
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

.rule-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  line-height: 1.5;
}
.rule-sub {
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

.method-chip {
  display: inline-block;
  min-width: 58px;
  padding: 2px 8px;
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

.rule-route {
  padding: 1px 6px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 3px;
  word-break: break-all;
}

.type-chip {
  display: inline-block;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.type-chip.is-default {
  color: var(--text-muted);
  background: var(--bg-muted);
}
.type-chip.is-common {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.type-chip.is-login {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
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
.form-hint code {
  padding: 0 4px;
  font-size: 11px;
  background: var(--bg-muted);
  border-radius: 3px;
}

@media (max-width: 768px) {
  .list-filter {
    gap: 8px;
  }
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  /* 窄屏隐藏路由与类型列，避免表格横向溢出 */
  .rule-table th:nth-child(4),
  .rule-table td:nth-child(4),
  .rule-table th:nth-child(5),
  .rule-table td:nth-child(5) {
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
