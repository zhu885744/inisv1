<template>
  <div class="auth-group-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">权限分组</h2>
          <p class="block-desc">
            把权限规则与后台页面打包成组，再把用户加进组即可批量授权
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
            <i class="bi bi-plus-lg" /> 新建分组
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
          <h2 class="block-title">分组列表</h2>
          <p class="block-desc">
            超级管理员分组与系统内置分组不可删除
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
          >全部分组</button>
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
              placeholder="搜索分组名…"
              aria-label="搜索分组名"
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
        <span><i class="bi bi-trash3" /> 回收站内的分组不再参与授权，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-shield-lock'" :text="emptyText" />
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

        <table class="group-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th>分组</th>
              <th class="cell-members">成员</th>
              <th class="cell-count">权限规则</th>
              <th class="cell-count">后台页面</th>
              <th class="cell-act">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in list" :key="item.id" :class="{ 'is-selected': isSelected(item.id) }">
              <td class="cell-pick">
                <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
                  <FormCheckbox
                    :model-value="isSelected(item.id)"
                    :disabled="busy || isProtectedGroup(item)"
                    :aria-label="`选择分组 ${item.name}`"
                    @change="toggleSelect(item.id)" />
                </label>
              </td>
              <td>
                <div class="group-name-row">
                  <span class="group-name">{{ item.name || '未命名分组' }}</span>
                  <code v-if="item.key" class="key-chip">{{ item.key }}</code>
                  <span v-if="Number(item.root) === 1" class="flag-chip is-root">越权</span>
                  <span v-if="Number(item.default) === 1" class="flag-chip">内置</span>
                </div>
                <div class="group-sub">
                  <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                  <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
                </div>
              </td>
              <td class="cell-members">
                <div class="member-stack">
                  <span
                    v-for="member in groupMembers(item).slice(0, 3)"
                    :key="member.id"
                    class="member-avatar"
                    :title="member.nickname || member.account || `用户 ${member.id}`"
                  >
                    <img v-if="member.avatar" :src="member.avatar" :alt="member.nickname" loading="lazy" />
                    <i v-else class="bi bi-person-circle" />
                  </span>
                  <span class="member-count">{{ memberCount(item) }} 人</span>
                </div>
              </td>
              <td class="cell-count">
                <span class="count-chip" :class="{ 'is-all': rulesOf(item).all }">
                  {{ rulesOf(item).all ? '全部规则' : `${rulesOf(item).list.length} 条` }}
                </span>
              </td>
              <td class="cell-count">
                <span class="count-chip" :class="{ 'is-all': pagesOf(item).all }">
                  {{ pagesOf(item).all ? '全部页面' : `${pagesOf(item).list.length} 个` }}
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
                    <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" :disabled="busy" @click="openEdit(item)">
                      <i class="bi bi-pencil" />
                    </button>
                    <button
                      class="btn btn-ghost btn-sm danger"
                      :title="isProtectedGroup(item) ? protectedGroupReason(item) : '删除'"
                      aria-label="删除"
                      :disabled="busy || isProtectedGroup(item)"
                      @click="askRemove(item)"
                    >
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

    <!-- 新建 / 编辑分组 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑分组 #${edit.id}` : '新建分组'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      width="880px"
      @confirm="save"
    >
      <section class="form-section">
        <h4 class="section-title">基本信息</h4>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">分组名称</label>
            <FormInput v-model="edit.name" type="text" placeholder="必填" />
          </div>
          <div class="form-item">
            <label class="form-label">唯一标识 key</label>
            <FormInput v-model="edit.key" type="text" placeholder="可选，需唯一（如 editor）" />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">越权能力</label>
            <FormSelect v-model="edit.root" :options="ROOT_OPTIONS" />
            <p class="form-hint">开启后可越权操作他人数据（如修改他人文章）</p>
          </div>
          <div class="form-item">
            <label class="form-label">备注</label>
            <FormInput v-model="edit.remark" type="text" placeholder="可选" />
          </div>
        </div>
      </section>

      <section class="form-section">
        <h4 class="section-title">权限规则</h4>

        <label class="check-line">
          <FormCheckbox v-model="edit.rulesAll" />
          <span>拥有全部规则（all）</span>
        </label>

        <template v-if="!edit.rulesAll">
          <!-- 快速套用：常见角色一键勾好，不用逐条找 -->
          <div class="preset-bar">
            <span class="preset-label"><i class="bi bi-lightning-charge" /> 快速套用</span>
            <button
              v-for="preset in AUTH_RULE_PRESETS"
              :key="preset.key"
              type="button"
              class="preset-chip"
              :title="`${preset.desc}（叠加勾选，不会取消已选）`"
              @click="applyPreset(preset)"
            >
              <i :class="preset.icon" /> {{ preset.label }}
            </button>
          </div>

          <div class="pick-toolbar">
            <FormInput v-model="ruleKeyword" type="search" placeholder="搜索模块 / 规则名称 / 路由…" />
            <button class="btn btn-sm btn-ghost" type="button" @click="toggleExpandAll">
              {{ allExpanded ? '收起全部' : '展开全部' }}
            </button>
            <button class="btn btn-sm btn-ghost" type="button" @click="selectAllRules">全部选中</button>
            <button class="btn btn-sm btn-ghost" type="button" :disabled="!edit.rules.length" @click="edit.rules = []">
              清空
            </button>
          </div>

          <label class="check-line check-line-inline">
            <FormCheckbox v-model="showUselessRules" />
            <span>显示无需授权的规则（免登录 / 需登录，默认隐藏）</span>
          </label>

          <div v-if="loadingRules" class="pick-loading">规则加载中...</div>
          <div v-else-if="!ruleGroups.length" class="pick-empty">没有匹配的规则</div>
          <div v-else class="group-cards">
            <section v-for="group in ruleGroups" :key="group.key" class="group-card">
              <header class="group-card-head">
                <label class="pick" :title="group.allChecked ? '取消本模块' : '勾选本模块全部规则'">
                  <FormCheckbox
                    :model-value="group.allChecked"
                    :indeterminate.prop="group.someChecked"
                    @change="toggleRuleGroup(group)" />
                </label>
                <button
                  type="button"
                  class="group-card-title"
                  @click="toggleGroupOpen(group.key)"
                >
                  <i :class="group.icon" class="group-card-icon" />
                  <span class="group-card-name">{{ group.label }}</span>
                  <code class="group-card-route">/api/{{ group.key }}</code>
                  <span class="group-card-count" :class="{ 'is-on': group.selectedCount > 0 }">
                    {{ group.selectedCount }}/{{ group.items.length }}
                  </span>
                  <i
                    class="bi group-card-caret"
                    :class="isGroupOpen(group.key) ? 'bi-chevron-up' : 'bi-chevron-down'"
                  />
                </button>
              </header>

              <ul v-if="isGroupOpen(group.key)" class="pick-list pick-list-nested">
                <li v-for="rule in group.items" :key="rule.hash">
                  <label class="pick-item">
                    <FormCheckbox v-model="edit.rules" :value="rule.hash" />
                    <span class="pick-main">
                      <span class="pick-title">{{ authRuleTitle(rule) }}</span>
                      <code class="pick-route">{{ rule.route }}</code>
                    </span>
                    <span class="method-chip" :class="`is-${String(rule.method || 'GET').toLowerCase()}`">
                      {{ String(rule.method || 'GET').toUpperCase() }}
                    </span>
                  </label>
                </li>
              </ul>
            </section>
          </div>

          <p class="form-hint">
            已选 {{ edit.rules.length }} 条 · 当前显示 {{ visibleRuleCount }} 条（共 {{ selectableRules.length }} 条需授权规则）
          </p>
        </template>
      </section>

      <section class="form-section">
        <h4 class="section-title">后台页面</h4>

        <label class="check-line">
          <FormCheckbox v-model="edit.pagesAll" />
          <span>拥有全部页面（all）</span>
        </label>

        <template v-if="!edit.pagesAll">
          <div class="pick-toolbar">
            <FormInput v-model="pageKeyword" type="search" placeholder="搜索页面名称或路径…" />
            <button class="btn btn-sm btn-ghost" type="button" @click="toggleAllPages">
              {{ visiblePages.length && visiblePages.every((p) => edit.pages.includes(p.hash)) ? '取消全选' : '全选当前' }}
            </button>
          </div>

          <div v-if="loadingPages" class="pick-loading">页面加载中...</div>
          <div v-else-if="!visiblePages.length" class="pick-empty">没有匹配的页面</div>
          <ul v-else class="pick-list">
            <li v-for="p in visiblePages" :key="p.hash">
              <label class="pick-item">
                <FormCheckbox v-model="edit.pages" :value="p.hash" />
                <span class="pick-main">
                  <span class="pick-title">{{ p.name }}</span>
                  <code class="pick-route">{{ p.path }}</code>
                </span>
              </label>
            </li>
          </ul>
          <p class="form-hint">已选 {{ edit.pages.length }} 个</p>
        </template>
      </section>

      <section class="form-section">
        <h4 class="section-title">组成员</h4>

        <div class="form-item">
          <label class="form-label">成员用户 ID</label>
          <FormInput v-model="edit.memberIds" type="text" placeholder="多个用逗号分隔，如 1,2,3" />
          <p class="form-hint">以本次输入为准覆盖全部成员；留空表示移除所有成员</p>
        </div>

        <div v-if="memberPreview.length" class="member-chips">
          <span v-for="member in memberPreview" :key="member.id" class="member-chip">
            <img v-if="member.avatar" :src="member.avatar" :alt="member.nickname" loading="lazy" />
            <i v-else class="bi bi-person-circle" />
            {{ member.nickname || member.account || `用户 ${member.id}` }}
          </span>
        </div>
      </section>
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
 * 权限分组（/admin/auth/group）
 *
 * 把权限规则与后台页面打包成组，再把用户加进组完成批量授权。
 *
 * 后端约束（app/api/controller/auth-group.go、app/model/auth-group.go）：
 * - 存储格式：uids 为 "|1|2|"；rules / pages 为 hash 逗号分隔，字面量 "all" 表示全部；
 *   鉴权时按 hash 反查规则，再拼成 [METHOD][route] 比对；
 * - 允许字段只有 name/key/rules/uids/root/pages/remark/json/text，
 *   **default（系统内置标记）不可修改**；
 * - remove / delete / clear 会拒绝 id=1（超级管理员分组）与 default=1（系统内置分组）；
 * - key 重复时后端在 AfterSave 返回「key 已存在！」；
 * - auth-group/count 支持 onlyTrashed，回收站数量可直接统计。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listAuthGroupsAdmin,
  countAuthGroups,
  saveAuthGroup,
  removeAuthGroups,
  forceDeleteAuthGroups,
  restoreAuthGroups,
  clearAuthGroupRecycle,
  AUTH_GROUP_ADMIN_FIELD
} from '@/api/auth-group'
import { listAuthRules } from '@/api/auth-rules'
import {
  AUTH_RULE_FIELD,
  AUTH_RULE_PRESETS,
  authRuleModuleKey,
  authRuleModuleMeta,
  authRuleModuleOrder,
  authRuleMethodRank,
  authRuleTitle,
  authRuleTypeKey
} from '@/utils/auth-rule'
import { getAuthPagesFlat } from '@/api/auth-pages'
import {
  parsePipeIds,
  toPipeIds,
  parseIdInput,
  parseHashList,
  isProtectedGroup,
  protectedGroupReason,
  groupMembers
} from '@/utils/auth-group'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

const ROOT_OPTIONS = [
  { value: 0, label: '否' },
  { value: 1, label: '是（可越权操作他人数据）' }
]

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
const stats = reactive({ total: 0, builtin: 0, root: 0, trash: 0 })

// ===== 规则 / 页面（编辑弹窗的多选来源）=====
const allRules = ref([])
const allPages = ref([])
const loadingRules = ref(false)
const loadingPages = ref(false)
const ruleKeyword = ref('')
const pageKeyword = ref('')
// 展开的模块卡片（key 列表）；搜索时全部视为展开
const openGroups = ref([])
// 默认只显示 type=default（真正校验权限点的规则），另两类勾了也没用
const showUselessRules = ref(false)

// ===== 弹窗 =====
const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  key: '',
  remark: '',
  root: 0,
  rulesAll: false,
  rules: [],
  pagesAll: false,
  pages: [],
  memberIds: ''
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
  { label: '分组总数', value: stats.total, icon: 'bi bi-shield-lock', color: 'var(--primary)' },
  { label: '系统内置', value: stats.builtin, icon: 'bi bi-award', color: 'var(--success)' },
  { label: '越权分组', value: stats.root, icon: 'bi bi-shield-fill-exclamation', color: 'var(--warning)' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的分组'
  if (trash.value) return '回收站是空的'
  return '还没有权限分组，先新建一个吧'
})

// ---------- 展示辅助 ----------
function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `创建于 ${fromNow(item.create_time)}`
}

function rulesOf(item) {
  return parseHashList(item?.rules)
}

function pagesOf(item) {
  return parseHashList(item?.pages)
}

// 成员数优先用后端解析好的结果，缺失时退回解析 uids 字符串
function memberCount(item) {
  const members = groupMembers(item)
  return members.length || parsePipeIds(item?.uids).length
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [totalRes, builtinRes, rootRes, trashRes] = await Promise.all([
      countAuthGroups(),
      countAuthGroups({ where: JSON.stringify({ default: 1 }) }),
      countAuthGroups({ where: JSON.stringify({ root: 1 }) }),
      countAuthGroups({ onlyTrashed: true })
    ])
    stats.total = Number(totalRes?.data || 0)
    stats.builtin = Number(builtinRes?.data || 0)
    stats.root = Number(rootRes?.data || 0)
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
      field: AUTH_GROUP_ADMIN_FIELD,
      order: trash.value ? 'delete_time desc' : 'id asc'
    }
    if (trash.value) params.onlyTrashed = true
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `name|${kw}`

    const res = await listAuthGroupsAdmin(params)
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

// ---------- 选中（受保护分组不可选）----------
const selectedSet = computed(() => new Set(selectedIds.value))
const selectableGroups = computed(() => list.value.filter((g) => !isProtectedGroup(g)))
const pageAllSelected = computed(
  () => selectableGroups.value.length > 0 && selectableGroups.value.every((g) => selectedSet.value.has(g.id))
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
  selectedIds.value = pageAllSelected.value ? [] : selectableGroups.value.map((g) => g.id)
}

function clearSelection() {
  selectedIds.value = []
}

// ---------- 规则 / 页面数据源 ----------
async function loadRulesOnce() {
  if (allRules.value.length || loadingRules.value) return
  loadingRules.value = true
  try {
    // 一次性拉全量规则：目前约 700 条且随功能增长，留足余量避免被静默截断
    const res = await listAuthRules({ page: 1, limit: 2000, order: 'id asc', field: AUTH_RULE_FIELD })
    allRules.value = res.data?.data || []
  } catch {
    allRules.value = []
  } finally {
    loadingRules.value = false
  }
}

async function loadPagesOnce() {
  if (allPages.value.length || loadingPages.value) return
  loadingPages.value = true
  try {
    const res = await getAuthPagesFlat()
    allPages.value = res?.data?.data || []
  } catch {
    allPages.value = []
  } finally {
    loadingPages.value = false
  }
}

/**
 * 真正需要授权的规则：type=default。
 * common（免登录）与 login（登录即放行）在中间件里根本不查权限点，勾了也没用，
 * 默认隐藏，避免上千条里混进一堆「勾了不生效」的开关。
 * （历史分组里若已存了这类 hash，隐藏不影响提交 —— edit.rules 原样保留。）
 */
const selectableRules = computed(() =>
  allRules.value.filter((rule) => showUselessRules.value || authRuleTypeKey(rule) === 'default')
)

/**
 * 按模块归类：一个模块一张卡片，卡片头可整组勾选。
 * 搜索命中「模块中文名 / 模块 key / 规则名 / 路由」任意一项。
 */
const ruleGroups = computed(() => {
  const kw = ruleKeyword.value.trim().toLowerCase()
  const picked = new Set(edit.rules)
  const map = new Map()

  for (const rule of selectableRules.value) {
    const key = authRuleModuleKey(rule)
    const meta = authRuleModuleMeta(key)
    if (kw) {
      const hay = `${meta.label} ${key} ${rule.name || ''} ${rule.route || ''}`.toLowerCase()
      if (!hay.includes(kw)) continue
    }
    if (!map.has(key)) {
      map.set(key, { key, label: meta.label, icon: meta.icon, items: [], selectedCount: 0 })
    }
    const group = map.get(key)
    group.items.push(rule)
    if (picked.has(rule.hash)) group.selectedCount += 1
  }

  const groups = [...map.values()]
  for (const group of groups) {
    // 组内按「查看 → 新增 → 修改 → 删除」排，再按名称
    group.items.sort(
      (a, b) =>
        authRuleMethodRank(a) - authRuleMethodRank(b) ||
        authRuleTitle(a).localeCompare(authRuleTitle(b), 'zh')
    )
    group.allChecked = group.items.length > 0 && group.selectedCount === group.items.length
    group.someChecked = group.selectedCount > 0 && !group.allChecked
  }

  return groups.sort(
    (a, b) => authRuleModuleOrder(a.key) - authRuleModuleOrder(b.key) || a.key.localeCompare(b.key)
  )
})

const visibleRuleCount = computed(() => ruleGroups.value.reduce((sum, group) => sum + group.items.length, 0))

const allExpanded = computed(
  () => ruleGroups.value.length > 0 && ruleGroups.value.every((group) => openGroups.value.includes(group.key))
)

// 搜索时一律展开，命中项直接可见
function isGroupOpen(key) {
  return !!ruleKeyword.value.trim() || openGroups.value.includes(key)
}

function toggleGroupOpen(key) {
  const next = new Set(openGroups.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  openGroups.value = [...next]
}

function toggleExpandAll() {
  openGroups.value = allExpanded.value ? [] : ruleGroups.value.map((group) => group.key)
}

const visiblePages = computed(() => {
  const kw = pageKeyword.value.trim().toLowerCase()
  if (!kw) return allPages.value
  return allPages.value.filter(
    (p) => String(p.name || '').toLowerCase().includes(kw) || String(p.path || '').toLowerCase().includes(kw)
  )
})

/** 勾 / 取消整个模块 */
function toggleRuleGroup(group) {
  const next = new Set(edit.rules)
  const allChecked = group.items.length > 0 && group.items.every((rule) => next.has(rule.hash))
  group.items.forEach((rule) => (allChecked ? next.delete(rule.hash) : next.add(rule.hash)))
  edit.rules = [...next]
}

/** 全选当前可见的规则（受搜索与「显示无需授权」影响） */
function selectAllRules() {
  const next = new Set(edit.rules)
  ruleGroups.value.forEach((group) => group.items.forEach((rule) => next.add(rule.hash)))
  edit.rules = [...next]
}

/** 快速套用模板：叠加勾选，不会取消已选的权限（可再手动微调） */
function applyPreset(preset) {
  const modules = preset.modules ? new Set(preset.modules) : null
  const methods = preset.methods ? new Set(preset.methods.map((item) => item.toUpperCase())) : null
  // 路由区分大小写，只把方法部分统一成大写再比对
  const permits = preset.permits
    ? new Set(
        preset.permits.map((item) => {
          const [method, ...rest] = item.split(' ')
          return `${method.toUpperCase()} ${rest.join(' ')}`
        })
      )
    : null
  const next = new Set(edit.rules)
  let added = 0

  allRules.value.forEach((rule) => {
    // 只有 default 类型才真正校验权限点
    if (authRuleTypeKey(rule) !== 'default') return
    if (modules && !modules.has(authRuleModuleKey(rule))) return
    if (methods && !methods.has(String(rule.method || 'GET').toUpperCase())) return
    if (permits && !permits.has(`${String(rule.method || 'GET').toUpperCase()} ${rule.route}`)) return
    if (next.has(rule.hash)) return
    next.add(rule.hash)
    added += 1
  })

  edit.rules = [...next]

  // 套用后把受影响的模块展开，方便核对
  const touched = ruleGroups.value
    .filter((group) => group.items.some((rule) => next.has(rule.hash)))
    .map((group) => group.key)
  openGroups.value = [...new Set([...openGroups.value, ...touched])]

  if (added) toast.success(`已套用「${preset.label}」，新增 ${added} 条权限`)
  else toast.info(`「${preset.label}」没有需要新增的权限`)
}

function toggleAllPages() {
  const hashes = visiblePages.value.map((p) => p.hash)
  const allChecked = hashes.length && hashes.every((h) => edit.pages.includes(h))
  const next = new Set(edit.pages)
  hashes.forEach((h) => (allChecked ? next.delete(h) : next.add(h)))
  edit.pages = [...next]
}

// 成员预览：优先用后端解析的 result.users，缺失时按输入展示
const memberPreview = computed(() => {
  const ids = parseIdInput(edit.memberIds)
  if (edit.id && !ids.length) return []
  const known = new Map()
  list.value.forEach((g) => groupMembers(g).forEach((m) => known.set(Number(m.id), m)))
  return [...new Set(ids)].map((id) => known.get(id) || { id })
})

// ---------- 新建 / 编辑 ----------
function resetEdit() {
  edit.id = 0
  edit.name = ''
  edit.key = ''
  edit.remark = ''
  edit.root = 0
  edit.rulesAll = false
  edit.rules = []
  edit.pagesAll = false
  edit.pages = []
  edit.memberIds = ''
  ruleKeyword.value = ''
  pageKeyword.value = ''
  openGroups.value = []
}

async function openCreate() {
  resetEdit()
  edit.visible = true
  await Promise.all([loadRulesOnce(), loadPagesOnce()])
}

async function openEdit(item) {
  const rules = parseHashList(item.rules)
  const pages = parseHashList(item.pages)
  edit.id = Number(item.id)
  edit.name = item.name || ''
  edit.key = item.key || ''
  edit.remark = item.remark || ''
  edit.root = Number(item.root) === 1 ? 1 : 0
  edit.rulesAll = rules.all
  edit.rules = [...rules.list]
  edit.pagesAll = pages.all
  edit.pages = [...pages.list]
  edit.memberIds = parsePipeIds(item.uids).join(',')
  ruleKeyword.value = ''
  pageKeyword.value = ''
  edit.visible = true
  await Promise.all([loadRulesOnce(), loadPagesOnce()])

  // 自动展开已有授权的模块：打开弹窗就能看清这个组到底给了哪些模块的权限
  openGroups.value = ruleGroups.value
    .filter((group) => group.items.some((rule) => edit.rules.includes(rule.hash)))
    .map((group) => group.key)
}

async function save() {
  const name = edit.name.trim()
  if (!name) {
    toast.warning('请输入分组名称')
    return
  }

  const payload = {
    name,
    key: edit.key.trim(),
    remark: edit.remark.trim(),
    root: Number(edit.root) === 1 ? 1 : 0,
    // all 与 hash 列表二选一：all 为真时忽略具体列表
    rules: edit.rulesAll ? 'all' : [...new Set(edit.rules)].join(','),
    pages: edit.pagesAll ? 'all' : [...new Set(edit.pages)].join(','),
    uids: toPipeIds(parseIdInput(edit.memberIds))
  }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveAuthGroup(payload)
    toast.success(edit.id ? '修改已保存' : '分组已创建')
    edit.visible = false
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出（如 key 已存在）
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
    title: '删除分组',
    message: `确定删除分组「${item.name}」吗？组内成员会失去该组授权，可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeAuthGroups([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreAuthGroups([item.id])
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
    message: `确定彻底删除分组「${item.name}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAuthGroups([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个分组吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeAuthGroups(ids)
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
    await restoreAuthGroups(ids)
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
    message: `确定彻底删除选中的 ${ids.length} 个分组吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAuthGroups(ids)
      toast.success(`已彻底删除 ${ids.length} 个`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有分组将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearAuthGroupRecycle()
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
.auth-group-admin {
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

.group-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.group-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.group-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.group-table tr:last-child td {
  border-bottom: none;
}
.group-table tbody tr:hover td {
  background: var(--bg-muted);
}
.group-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-members {
  width: 160px;
}
.cell-count {
  width: 108px;
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
.pick input:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.group-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.group-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.key-chip {
  padding: 1px 6px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.flag-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.flag-chip.is-root {
  color: var(--warning);
  background: var(--gold-wash);
}
.group-sub {
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

.member-stack {
  display: flex;
  align-items: center;
  gap: 6px;
}
.member-avatar {
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 13px;
}
.member-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.member-count {
  font-size: 12px;
  color: var(--text-muted);
}

.count-chip {
  display: inline-block;
  padding: 2px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.count-chip.is-all {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
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
.form-section + .form-section {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px dashed var(--border-soft);
}
.section-title {
  margin: 0 0 12px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: var(--text-muted);
}
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

.check-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
}
.check-line input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

.pick-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 8px 0;
}
.pick-toolbar .input {
  flex: 1;
}
.pick-loading,
.pick-empty {
  padding: 16px;
  font-size: 12px;
  text-align: center;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.pick-list {
  max-height: 240px;
  overflow-y: auto;
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
/* 模块卡片里的规则列表：卡片自带边框，内层不再重复描边 */
.pick-list-nested {
  max-height: 300px;
  border: none;
  border-top: 1px dashed var(--border-soft);
  border-radius: 0;
}

/* ---------- 快速套用 ---------- */
.preset-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin: 6px 0 10px;
}
.preset-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.preset-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--primary-deep);
  background: var(--accent-wash);
  border: 1px solid var(--accent-soft);
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s;
}
.preset-chip:hover {
  background: var(--accent-soft);
  border-color: var(--primary);
}

.check-line-inline {
  padding: 0 0 8px;
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 按模块分组的卡片 ---------- */
.group-cards {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 360px;
  overflow-y: auto;
  padding-right: 2px;
}
.group-card {
  /* 必须禁止收缩：外层是 max-height + overflow 的列向 flex，
     卡片又有 overflow:hidden（会让 min-height:auto 失效），
     不写 flex-shrink:0 的话几十张卡片会被平均压成几像素高的细线，内容全被裁掉 */
  flex-shrink: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}
.group-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px 6px 8px;
  background: var(--bg-muted);
}
.group-card-title {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 0;
  font-size: 13px;
  color: var(--text);
  background: transparent;
  border: none;
  cursor: pointer;
  text-align: left;
}
.group-card-icon {
  flex-shrink: 0;
  color: var(--primary);
}
.group-card-name {
  flex-shrink: 0;
  font-weight: 600;
}
.group-card-route {
  min-width: 0;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text-light);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.group-card-count {
  flex-shrink: 0;
  margin-left: auto;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-card);
  border-radius: 999px;
}
.group-card-count.is-on {
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.group-card-caret {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
}
.pick-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  cursor: pointer;
  border-bottom: 1px dashed var(--border-soft);
}
.pick-item:last-child {
  border-bottom: none;
}
.pick-item:hover {
  background: var(--bg-muted);
}
.pick-item input {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  accent-color: var(--primary);
  cursor: pointer;
}
.pick-main {
  flex: 1;
  min-width: 0;
}
.pick-title {
  display: block;
  font-size: 13px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pick-route {
  display: inline-block;
  margin-top: 2px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text-muted);
  word-break: break-all;
}

.method-chip {
  flex-shrink: 0;
  min-width: 52px;
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

.member-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.member-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px 3px 4px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 999px;
}
.member-chip img {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  object-fit: cover;
}
.member-chip .bi {
  font-size: 16px;
  color: var(--text-light);
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
  /* 窄屏隐藏成员与页面列，避免横向溢出 */
  .group-table th:nth-child(3),
  .group-table td:nth-child(3),
  .group-table th:nth-child(5),
  .group-table td:nth-child(5) {
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
