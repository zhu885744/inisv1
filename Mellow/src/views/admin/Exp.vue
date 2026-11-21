<template>
  <div class="exp-admin">
    <!-- 概览：随当前模块切换统计口径 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">经验管理</h2>
          <p class="block-desc">{{ activeView.desc }}</p>
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
      <!-- 模块切换：用户经验 / 经验流水 / 经验规则 -->
      <div class="view-tabs">
        <button
          v-for="v in views"
          :key="v.key"
          type="button"
          class="status-tab"
          :class="{ 'is-active': view === v.key }"
          @click="switchView(v.key)"
        >
          <i :class="v.icon" /> {{ v.label }}
        </button>
      </div>

      <!-- ============ 1. 用户经验管理 ============ -->
      <template v-if="view === 'users'">
        <header class="list-head">
          <div>
            <h2 class="block-title">用户经验</h2>
            <p class="block-desc">
              调整后经验不会低于 0（后端会拦截「经验值不足」）；批量调整将按顺序逐个提交
              <template v-if="userTotal > 0"> · 共 {{ userTotal }} 位</template>
            </p>
          </div>
        </header>

        <div class="list-filter">
          <div class="status-tabs">
            <button
              v-for="tab in userTabs"
              :key="tab.key"
              type="button"
              class="status-tab"
              :class="{ 'is-active': userFilter === tab.key }"
              @click="switchUserFilter(tab.key)"
            >{{ tab.label }}</button>
          </div>

          <div class="filter-right">
            <FormSelect variant="pill"
              v-model="userSort"
              :options="userSortOptions"
              icon="bi bi-sort-down"
              placeholder="排序方式"
              @change="reloadUsers" />

            <FormSelect variant="pill"
              v-model="userSearchField"
              :options="USER_SEARCH_FIELDS"
              icon="bi bi-funnel"
              placeholder="搜索字段" />

            <div class="search-box">
              <i class="bi bi-search" aria-hidden="true" />
              <FormInput
                v-model="userKeyword"
                class="search-input"
                type="search"
                placeholder="搜索用户…"
                aria-label="搜索用户"
                @input="doUserSearch" />
            </div>

            <button class="btn btn-sm" :disabled="userLoading" @click="loadUsers()">
              <i class="bi bi-arrow-clockwise" /> 刷新
            </button>
          </div>
        </div>

        <!-- 批量调整 -->
        <div v-if="selectedUids.length" class="batch-bar">
          <span class="batch-count">已选 <strong>{{ selectedUids.length }}</strong> 位用户</span>
          <div class="batch-actions">
            <button class="btn btn-sm btn-primary" :disabled="busy" @click="openBatchAdjust">
              <i class="bi bi-star" /> 批量调整经验
            </button>
            <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
          </div>
        </div>

        <div v-if="userLoading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!users.length" class="empty-row">
          <EmptyState icon="bi bi-people" :text="userEmptyText" />
        </div>

        <template v-else>
          <div class="list-head-row">
            <label class="pick" title="全选本页">
              <FormCheckbox
                ref="userAllRef"
                :model-value="userPageAllSelected"
                :disabled="busy"
                aria-label="全选本页"
                @change="toggleUserAll" />
            </label>
            <span class="list-head-text">本页 {{ users.length }} 位</span>
          </div>

          <ul class="user-list">
            <li v-for="item in users" :key="item.id" class="user-row" :class="{ 'is-selected': isUserSelected(item.id) }">
              <label class="pick" :title="isUserSelected(item.id) ? '取消选择' : '选择'">
                <FormCheckbox
                  :model-value="isUserSelected(item.id)"
                  :disabled="busy"
                  :aria-label="`选择用户 ${item.nickname || item.id}`"
                  @change="toggleUser(item.id)" />
              </label>

              <div class="user-avatar">
                <img v-if="item.avatar" :src="item.avatar" :alt="item.nickname" loading="lazy" />
                <i v-else class="bi bi-person-circle" />
              </div>

              <div class="user-main">
                <div class="user-name-row">
                  <span class="user-name">{{ item.nickname || '未设置昵称' }}</span>
                  <span class="id-chip">#{{ item.id }}</span>
                  <span class="level-chip">{{ levelNameOf(item.exp) }}</span>
                </div>
                <div class="user-meta">
                  <span class="meta-text"><i class="bi bi-coin" /> 经验 {{ item.exp || 0 }}</span>
                  <span class="meta-text"><i class="bi bi-clock" /> 注册于 {{ fromNow(item.create_time) }}</span>
                </div>
              </div>

              <div class="user-actions">
                <button class="btn btn-primary btn-sm" :disabled="busy" @click="openAdjust(item)">
                  <i class="bi bi-star" /> 调整经验
                </button>
              </div>
            </li>
          </ul>
        </template>

        <Pagination
          v-if="!userLoading && userTotal > 0"
          :current="userPage"
          :total="userTotal"
          :page-size="userPageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { userPageSize = size; userPage = 1; loadUsers() }"
          @update:current="changeUserPage"
        />
      </template>

      <!-- ============ 2. 经验流水 ============ -->
      <template v-else-if="view === 'logs'">
        <header class="list-head">
          <div>
            <h2 class="block-title">经验流水</h2>
            <p class="block-desc">
              全站经验获取记录，可按类型筛选、删除与回收站恢复
              <template v-if="total > 0"> · 共 {{ total }} 条</template>
            </p>
          </div>
        </header>

        <div class="list-filter">
          <div class="status-tabs">
            <button
              type="button"
              class="status-tab"
              :class="{ 'is-active': !trash }"
              @click="switchTab(false)"
            >全部记录</button>
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
              v-model="typeFilter"
              :options="typeOptions"
              icon="bi bi-tags"
              placeholder="全部类型"
              @change="reload" />

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
                placeholder="搜索说明…"
                aria-label="搜索说明"
                @input="doSearch" />
            </div>

            <button class="btn btn-sm" :disabled="loading" @click="load()">
              <i class="bi bi-arrow-clockwise" /> 刷新
            </button>
          </div>
        </div>

        <!-- 回收站提示 -->
        <div v-if="trash" class="trash-bar">
          <span><i class="bi bi-trash3" /> 回收站内的记录不参与统计，可恢复或彻底删除</span>
          <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
            清空回收站
          </button>
        </div>

        <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!list.length" class="empty-row">
          <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-star'" :text="logEmptyText" />
        </div>

        <ul v-else class="exp-list">
          <li v-for="item in list" :key="item.id" class="exp-row">
            <div class="exp-value" :class="{ 'is-minus': Number(item.value) < 0 }">
              {{ valueText(item) }}
            </div>

            <div class="exp-main">
              <div class="exp-name-row">
                <span class="exp-user">{{ authorName(item) }}</span>
                <span class="type-chip">{{ typeLabel(item) }}</span>
                <span v-if="Number(item.bind_id) > 0" class="bind-text">{{ bindText(item) }}</span>
              </div>
              <p class="exp-desc">{{ item.description || '—' }}</p>
              <div class="exp-meta">
                <span class="meta-text">#{{ item.id }}</span>
                <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
              </div>
            </div>

            <div class="exp-actions">
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
          </li>
        </ul>

        <Pagination
          v-if="!loading && total > 0"
          :current="page"
          :total="total"
          :page-size="pageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { pageSize = size; page = 1; load() }"
          @update:current="changePage"
        />
      </template>

      <!-- ============ 3. 经验规则 ============ -->
      <template v-else>
        <header class="list-head">
          <div>
            <h2 class="block-title">经验规则</h2>
            <p class="block-desc">各类行为可获得的经验值与每日上限，修改后立即影响后续发放</p>
          </div>
        </header>

        <div v-if="loadingRules" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!rules.length" class="empty-row">
          <EmptyState icon="bi bi-star" text="暂无经验规则" />
        </div>

        <table v-else class="rule-table">
          <thead>
            <tr>
              <th>类型</th>
              <th>名称</th>
              <th class="cell-num">单次经验</th>
              <th class="cell-num">每日上限</th>
              <th class="cell-act">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rules" :key="r.type">
              <td><code class="rule-type">{{ r.type }}</code></td>
              <td>{{ r.name || '—' }}</td>
              <td class="cell-num">{{ r.value }}</td>
              <td class="cell-num">{{ limitText(r) }}</td>
              <td class="cell-act">
                <button class="btn btn-ghost btn-sm" title="编辑规则" aria-label="编辑规则" @click="openRuleEdit(r)">
                  <i class="bi bi-pencil" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </template>
    </div>

    <!-- 调整经验（单个 / 批量） -->
    <AdminFormDialog
      v-model:visible="adjust.visible"
      :title="adjust.title"
      icon="bi bi-star"
      :loading="adjust.loading"
      confirm-text="确认调整"
      @confirm="submitAdjust"
    >
      <!-- 单个用户：展示当前经验与等级，便于核对 -->
      <div v-if="adjust.mode === 'single'" class="adjust-user">
        <div class="user-avatar sm">
          <img v-if="adjust.user?.avatar" :src="adjust.user.avatar" :alt="adjust.user.nickname" loading="lazy" />
          <i v-else class="bi bi-person-circle" />
        </div>
        <div class="user-main">
          <div class="user-name-row">
            <span class="user-name">{{ adjust.user?.nickname || '未设置昵称' }}</span>
            <span class="id-chip">#{{ adjust.user?.id }}</span>
            <span class="level-chip">{{ levelNameOf(adjust.user?.exp) }}</span>
          </div>
          <div class="user-meta">
            <span class="meta-text">当前经验 <strong>{{ adjust.user?.exp || 0 }}</strong></span>
          </div>
        </div>
      </div>

      <div v-else class="adjust-user">
        <div class="user-avatar sm multi">
          <i class="bi bi-people" />
        </div>
        <div class="user-main">
          <div class="user-name-row">
            <span class="user-name">已选 {{ adjust.uids.length }} 位用户</span>
          </div>
          <div class="user-meta">
            <span class="meta-text">后端仅支持单个用户调整，批量将按顺序逐个提交</span>
          </div>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">经验调整值</label>
        <FormInput v-model="adjust.value" type="number" step="1" placeholder="正数发放，负数扣除" />
        <div class="quick-row">
          <button v-for="n in [10, 50, 100]" :key="`add-${n}`" class="btn btn-sm" type="button" @click="adjust.value = n">
            +{{ n }}
          </button>
          <button v-for="n in [10, 50, 100]" :key="`sub-${n}`" class="btn btn-sm" type="button" @click="adjust.value = -n">
            -{{ n }}
          </button>
        </div>
        <p class="form-hint">
          正数发放、负数扣除；扣除超过用户当前经验时后端会拒绝并提示「用户经验值不足」
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">说明</label>
        <FormInput v-model="adjust.description" type="text" placeholder="可选，默认「管理员发放/扣除经验值 N」" />
      </div>
    </AdminFormDialog>

    <!-- 编辑经验规则 -->
    <AdminFormDialog
      v-model:visible="rule.visible"
      :title="`编辑规则：${rule.name || rule.type}`"
      icon="bi bi-pencil-square"
      :loading="rule.loading"
      confirm-text="保存"
      @confirm="saveRule"
    >
      <div class="form-item">
        <label class="form-label">规则名称</label>
        <FormInput v-model="rule.name" type="text" maxlength="32" placeholder="如「点赞」" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">单次经验</label>
          <FormInput v-model="rule.value" type="number" min="0" step="1" placeholder="每次动作获得的经验" />
        </div>
        <div class="form-item">
          <label class="form-label">每日上限</label>
          <FormInput v-model="rule.limit" type="number" min="0" step="1" placeholder="0 表示不限" />
          <p class="form-hint">0 表示不限制次数</p>
        </div>
      </div>

      <p class="dialog-tip">
        规则类型 <strong>{{ rule.type }}</strong> 不可修改；保存只覆盖该类型的配置，其余类型保持原样。
        签到奖励已独立，请到「签到管理」里配置。
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
/**
 * 经验管理（/admin/exp）
 *
 * 三个模块：
 *  1. 用户经验管理 —— 用户列表 + 单个 / 批量调整经验值（exp/give）
 *  2. 经验流水   —— exp 表全站记录，支持类型筛选、删除与回收站恢复
 *  3. 经验规则   —— SYSTEM_EXP_RULES 配置（读取走 exp/rules，保存走 config/save）
 *
 * 后端约束（app/api/controller/exp.go、app/model/exp.go）：
 * - exp/give 仅 root 可用，且**只接受单个 uid**（无批量接口），批量由前端顺序调用；
 *   value 为 0 会被拒绝，负数时要求用户当前经验足够（否则返回「用户经验值不足！」）；
 * - exp/rules 只返回 type/name/value/daily_limit（签到已独立，不再返回 check-in）；
 *   保存规则以 config/one 的原始 json 为基底，只覆盖当前编辑的那一条，避免把其它字段写丢；
 * - exp/count 不支持 onlyTrashed，回收站数量改用列表接口的 count 字段；
 * - 流水删除只是移除记录，不会回退用户已获得的经验值（后端未做回退）；
 * - 用户列表走 users/all：order 未做白名单校验由前端限定，where 支持 $gt / 等值条件。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import { listUsers, countUsers } from '@/api/users'
import {
  listExps,
  countExps,
  sumExp,
  giveExp,
  removeExps,
  forceDeleteExps,
  restoreExps,
  clearExpRecycle,
  getExpRules,
  getExpConfig,
  saveExpConfig
} from '@/api/exp'
import { listLevels } from '@/api/level'
import { USER_SEARCH_FIELDS, sanitizeUserKeyword } from '@/utils/user'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'

// 用户列表与经验流水是两个列表，各自记住自己的每页条数
const userPageSize = ref(PAGE_SIZE_DEFAULT)
const pageSize = ref(PAGE_SIZE_DEFAULT)

const views = [
  {
    key: 'users',
    label: '用户经验',
    icon: 'bi bi-people',
    desc: '查询用户经验值并直接发放 / 扣除，支持批量调整'
  },
  {
    key: 'logs',
    label: '经验流水',
    icon: 'bi bi-star',
    desc: '全站经验获取记录，可按类型筛选、删除与回收站恢复'
  },
  {
    key: 'rules',
    label: '经验规则',
    icon: 'bi bi-list-check',
    desc: '配置各类行为可获得的经验值与每日上限'
  }
]

// 用户经验筛选（where 由后端 IWhere 解析，$gt / 等值均支持）
const userTabs = [
  { key: 'all', label: '全部', where: null },
  { key: 'has', label: '有经验', where: { exp: { $gt: 0 } } },
  { key: 'zero', label: '零经验', where: { exp: 0 } }
]

// 排序白名单：后端 users/all 的 order 未校验，必须由前端限定
const userSortOptions = [
  { value: 'exp desc', label: '经验最多' },
  { value: 'exp asc', label: '经验最少' },
  { value: 'id desc', label: '最新注册' },
  { value: 'id asc', label: '最早注册' },
  { value: 'login_time desc', label: '最近登录' }
]

const sortOptions = [
  { value: 'create_time desc', label: '最新记录' },
  { value: 'create_time asc', label: '最早记录' },
  { value: 'value desc', label: '经验最多' },
  { value: 'id desc', label: 'ID 倒序' }
]

// ===== 当前模块 =====
const view = ref('users')
const activeView = computed(() => views.find((v) => v.key === view.value) || views[0])

// ===== 用户经验 =====
// 只取列表展示需要的字段，避免拉取 result / json 等大字段
const USER_FIELD = 'id,nickname,avatar,exp,integral,login_time,create_time'
const users = ref([])
const userTotal = ref(0)
const userPage = ref(1)
const userLoading = ref(false)
const busy = ref(false)
const userFilter = ref('all')
const userSort = ref(userSortOptions[0].value)
const userSearchField = ref('nickname')
const userKeyword = ref('')
const userSearchKey = ref('')
const selectedUids = ref([])
const userStats = reactive({ total: 0, hasExp: 0, topExp: 0 })

// ===== 等级（用于按经验推算等级名） =====
const levels = ref([])

// ===== 经验流水 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const trash = ref(false)
const typeFilter = ref('')
const sortKey = ref(sortOptions[0].value)
const keyword = ref('')
// 防抖后的实际搜索词
const searchKey = ref('')
const logStats = reactive({ total: 0, sum: 0, give: 0, trash: 0 })

// ===== 经验规则 =====
const rules = ref([])
const loadingRules = ref(false)

const rule = reactive({
  visible: false,
  loading: false,
  type: '',
  name: '',
  value: 0,
  limit: 0
})

// ===== 调整经验 =====
const adjust = reactive({
  visible: false,
  loading: false,
  mode: 'single', // single | batch
  user: null,
  uids: [],
  value: '',
  description: '',
  title: '调整经验'
})

// ===== 统计 =====
const loadingStats = ref(false)

const userStatCards = computed(() => [
  { label: '全部用户', value: userStats.total, icon: 'bi bi-people', color: 'var(--primary)' },
  { label: '有经验用户', value: userStats.hasExp, icon: 'bi bi-star', color: 'var(--success)' },
  { label: '零经验用户', value: Math.max(0, userStats.total - userStats.hasExp), icon: 'bi bi-dash-circle', color: 'var(--text-muted)' },
  { label: '最高经验', value: userStats.topExp, icon: 'bi bi-trophy', color: 'var(--warning)' },
  { label: '等级数量', value: levels.value.length, icon: 'bi bi-bar-chart', color: '#0ea5e9' }
])

const logStatCards = computed(() => [
  { label: '流水总数', value: logStats.total, icon: 'bi bi-list-ul', color: 'var(--primary)' },
  { label: '经验合计', value: logStats.sum, icon: 'bi bi-star', color: 'var(--success)' },
  { label: '管理员发放', value: logStats.give, icon: 'bi bi-gift', color: 'var(--warning)' },
  { label: '回收站', value: logStats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

const ruleStatCards = computed(() => {
  const list = rules.value
  // 每日上限合计：只统计有次数上限的规则（不限次数的无法估算）
  const daily = list.reduce(
    (sum, r) => sum + (Number(r.daily_limit) > 0 ? Number(r.daily_limit) * Number(r.value || 0) : 0),
    0
  )
  const maxOne = list.reduce((max, r) => Math.max(max, Number(r.value || 0)), 0)
  const unlimited = list.filter((r) => Number(r.daily_limit) <= 0).length
  return [
    { label: '规则数量', value: list.length, icon: 'bi bi-list-check', color: 'var(--primary)' },
    { label: '每日上限合计', value: daily, icon: 'bi bi-calendar-day', color: 'var(--success)' },
    { label: '单次最高经验', value: maxOne, icon: 'bi bi-star', color: 'var(--warning)' },
    { label: '不限次数规则', value: unlimited, icon: 'bi bi-infinity', color: '#0ea5e9' }
  ]
})

const statCards = computed(() => {
  if (view.value === 'logs') return logStatCards.value
  if (view.value === 'rules') return ruleStatCards.value
  return userStatCards.value
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

// 类型下拉：全部 + 后端返回的规则类型 + 管理员发放
const typeOptions = computed(() => [
  { value: '', label: '全部类型' },
  ...rules.value.map((r) => ({ value: r.type, label: r.name || r.type })),
  { value: 'give', label: '管理员发放' }
])

const userEmptyText = computed(() => {
  if (userSearchKey.value) return '没有匹配的用户'
  if (userFilter.value === 'has') return '暂无有经验的用户'
  if (userFilter.value === 'zero') return '暂无零经验用户'
  return '暂无用户'
})

const logEmptyText = computed(() => {
  if (searchKey.value) return '没有匹配的记录'
  if (trash.value) return '回收站是空的'
  return '暂无经验记录'
})

// ---------- 展示辅助 ----------
// 等级按经验阈值匹配：取「所需经验不超过当前经验」的最高等级
function levelNameOf(exp) {
  const value = Number(exp || 0)
  let match = null
  levels.value.forEach((lv) => {
    if (Number(lv.exp || 0) <= value) match = lv
  })
  return match?.name || 'LV0'
}

function findRule(type) {
  return rules.value.find((r) => r.type === type)
}

function typeLabel(item) {
  if (item?.type === 'give') return '管理员发放'
  return findRule(item?.type)?.name || item?.type || '未知'
}

function limitText(ruleItem) {
  const limit = Number(ruleItem?.daily_limit || 0)
  return limit > 0 ? `${limit} 次` : '不限'
}

function valueText(item) {
  const value = Number(item?.value || 0)
  return value > 0 ? `+${value}` : String(value)
}

// 后端把经验记录的所属用户放在 result.author
function authorName(item) {
  const author = item?.result?.author
  return author?.nickname || `用户 ${item?.uid || '—'}`
}

function bindText(item) {
  const bindType = item?.bind_type || 'default'
  return `${bindType} #${item.bind_id}`
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

// ---------- 统计加载 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    if (view.value === 'users') await loadUserStats()
    else if (view.value === 'logs') await loadLogStats()
    else await loadRules()
  } finally {
    loadingStats.value = false
  }
}

async function loadUserStats() {
  const [totalRes, hasRes, topRes] = await Promise.all([
    countUsers(),
    countUsers({ where: JSON.stringify({ exp: { $gt: 0 } }) }),
    listUsers({ page: 1, limit: 1, order: 'exp desc', field: 'id,nickname,exp' })
  ])
  userStats.total = Number(totalRes?.data || 0)
  userStats.hasExp = Number(hasRes?.data || 0)
  userStats.topExp = Number(topRes?.data?.data?.[0]?.exp || 0)
}

async function loadLogStats() {
  const [totalRes, sumRes, giveRes, trashRes] = await Promise.all([
    countExps(),
    sumExp(),
    countExps({ where: JSON.stringify({ type: 'give' }) }),
    // exp/count 不支持 onlyTrashed，回收站数量用列表接口的 count
    listExps({ page: 1, limit: 1, onlyTrashed: true, field: 'id' })
  ])
  logStats.total = Number(totalRes?.data || 0)
  logStats.sum = Number(sumRes?.data?.value || 0)
  logStats.give = Number(giveRes?.data || 0)
  logStats.trash = Number(trashRes?.data?.count || 0)
}

// ---------- 用户加载 ----------
async function loadUsers() {
  userLoading.value = true
  try {
    const params = {
      page: userPage.value,
      limit: userPageSize.value,
      field: USER_FIELD,
      order: userSort.value
    }
    const tab = userTabs.find((t) => t.key === userFilter.value)
    if (tab?.where) params.where = JSON.stringify(tab.where)

    // 关键词走后端 like 的「字段名|值」格式
    const kw = sanitizeUserKeyword(userSearchKey.value)
    if (kw) {
      const field = USER_SEARCH_FIELDS.some((f) => f.value === userSearchField.value)
        ? userSearchField.value
        : 'nickname'
      params.like = `${field}|${kw}`
    }

    const res = await listUsers(params)
    users.value = res.data?.data || []
    userTotal.value = res.data?.count || 0
    clearSelection()
  } catch {
    users.value = []
    userTotal.value = 0
  } finally {
    userLoading.value = false
  }
}

function reloadUsers() {
  userPage.value = 1
  loadUsers()
}

const doUserSearch = debounce(() => {
  userSearchKey.value = userKeyword.value
  reloadUsers()
}, 350)

async function switchUserFilter(key) {
  if (userFilter.value === key) return
  userFilter.value = key
  userKeyword.value = ''
  userSearchKey.value = ''
  reloadUsers()
}

function changeUserPage(p) {
  if (p < 1 || p === userPage.value) return
  userPage.value = p
  loadUsers()
}

// ---------- 用户选择 ----------
const userSelectedSet = computed(() => new Set(selectedUids.value))
const userPageAllSelected = computed(
  () => users.value.length > 0 && users.value.every((i) => userSelectedSet.value.has(i.id))
)
const userSomeSelected = computed(() => selectedUids.value.length > 0 && !userPageAllSelected.value)

const userAllRef = ref(null)

watchEffect(() => {
  if (userAllRef.value) userAllRef.value.setIndeterminate(userSomeSelected.value)
})

function isUserSelected(id) {
  return userSelectedSet.value.has(id)
}

function toggleUser(id) {
  const next = new Set(selectedUids.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedUids.value = [...next]
}

function toggleUserAll() {
  selectedUids.value = userPageAllSelected.value ? [] : users.value.map((i) => i.id)
}

function clearSelection() {
  selectedUids.value = []
}

// ---------- 调整经验 ----------
function openAdjust(item) {
  adjust.mode = 'single'
  adjust.user = {
    id: Number(item.id),
    nickname: item.nickname,
    avatar: item.avatar,
    exp: Number(item.exp || 0)
  }
  adjust.uids = []
  adjust.value = ''
  adjust.description = ''
  adjust.title = `调整经验：${item.nickname || '未设置昵称'}`
  adjust.visible = true
}

function openBatchAdjust() {
  adjust.mode = 'batch'
  adjust.user = null
  adjust.uids = [...selectedUids.value]
  adjust.value = ''
  adjust.description = ''
  adjust.title = `批量调整经验（${adjust.uids.length} 位用户）`
  adjust.visible = true
}

async function submitAdjust() {
  const value = Number(String(adjust.value).trim())
  if (!Number.isInteger(value) || value === 0) {
    toast.warning('经验调整值必须为不为 0 的整数（正数发放，负数扣除）')
    return
  }

  const ids = adjust.mode === 'single' ? [adjust.user?.id] : [...adjust.uids]
  const uidList = ids.map((i) => Number(i)).filter((n) => Number.isInteger(n) && n > 0)
  if (!uidList.length) {
    toast.warning('请选择有效的用户')
    return
  }

  adjust.loading = true
  try {
    const description = adjust.description.trim()

    // 后端 exp/give 只支持单个 uid，批量按顺序逐个提交
    if (uidList.length === 1) {
      await giveExp({ uid: uidList[0], value, description })
      toast.success(value > 0 ? `已发放 ${value} 经验` : `已扣除 ${Math.abs(value)} 经验`)
    } else {
      let success = 0
      for (const uid of uidList) {
        try {
          await giveExp({ uid, value, description })
          success++
        } catch {
          // 单个失败（如经验不足）不中断，继续处理其它用户
        }
      }
      const failed = uidList.length - success
      toast.success(`批量调整完成：成功 ${success} 个${failed ? `，失败 ${failed} 个` : ''}`)
    }

    adjust.visible = false
    clearSelection()
    await Promise.all([loadUsers(), loadStats()])
  } catch {
    // 失败提示由请求拦截器统一给出（如无权限、用户不存在、经验不足）
  } finally {
    adjust.loading = false
  }
}

// ---------- 经验流水 ----------
async function load() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: pageSize.value,
      order: trash.value ? 'delete_time desc' : sortKey.value
    }
    if (trash.value) params.onlyTrashed = true
    if (typeFilter.value) params.where = JSON.stringify({ type: typeFilter.value })
    // 关键词走后端 like 的「字段名|值」格式
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `description|${kw}`
    const res = await listExps(params)
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

// ---------- 经验规则 ----------
async function loadRules() {
  loadingRules.value = true
  try {
    const res = await getExpRules()
    rules.value = Array.isArray(res?.data) ? res.data : []
  } catch {
    rules.value = []
  } finally {
    loadingRules.value = false
  }
}

async function loadLevels() {
  try {
    const res = await listLevels({ page: 1, limit: 100, order: 'value asc', field: 'id,name,value,exp' })
    levels.value = res.data?.data || []
  } catch {
    levels.value = []
  }
}

function openRuleEdit(item) {
  rule.type = item.type || ''
  rule.name = item.name || ''
  rule.value = Number(item.value || 0)
  rule.limit = Number(item.daily_limit || 0)
  rule.visible = true
}

// 整数校验（允许 0）
function parseIntOrNull(raw) {
  const text = String(raw ?? '').trim()
  if (text === '') return null
  const num = Number(text)
  if (!Number.isInteger(num) || num < 0) return null
  return num
}

async function saveRule() {
  const name = rule.name.trim()
  if (!name) {
    toast.warning('请输入规则名称')
    return
  }
  const value = parseIntOrNull(rule.value)
  if (value === null) {
    toast.warning('单次经验必须为不小于 0 的整数')
    return
  }
  const limit = parseIntOrNull(rule.limit)
  if (limit === null) {
    toast.warning('每日上限必须为不小于 0 的整数')
    return
  }

  rule.loading = true
  try {
    // 以 config 中的原始配置为基底：exp/rules 只是规则的展示视图，
    // 直接把 rules 写回去会丢掉配置里其它字段
    const res = await getExpConfig()
    const raw = res?.data?.json
    const config = typeof raw === 'string' ? JSON.parse(raw) : { ...(raw || {}) }

    // 首次编辑时配置里可能没有对应类型，用当前规则列表补全
    rules.value.forEach((r) => {
      if (!config[r.type]) {
        config[r.type] = { name: r.name, value: Number(r.value || 0), daily_limit: Number(r.daily_limit || 0) }
      }
    })

    // 只覆盖当前编辑的这一条
    config[rule.type] = { ...config[rule.type], name, value, daily_limit: limit }

    await saveExpConfig(config)
    toast.success('规则已保存')
    rule.visible = false
    await loadRules()
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    rule.loading = false
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
    title: '删除记录',
    message: `确定删除记录 #${item.id} 吗？删除不会回退用户已获得的经验值，可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeExps([item.id])
      toast.success('已移入回收站')
      await Promise.all([load(), loadStats()])
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreExps([item.id])
    toast.success('已恢复')
    await Promise.all([load(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除记录 #${item.id} 吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteExps([item.id])
      toast.success('已彻底删除')
      await Promise.all([load(), loadStats()])
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有经验记录将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearExpRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await Promise.all([load(), loadStats()])
    }
  })
}

// ---------- 模块切换 ----------
async function switchView(key) {
  if (view.value === key) return
  view.value = key
  if (key === 'users' && !users.value.length) await loadUsers()
  if (key === 'logs' && !list.value.length) await load()
  if (key === 'rules' && !rules.value.length) await loadRules()
  await loadStats()
}

onMounted(async () => {
  loadLevels()
  loadRules() // 流水类型的中文名依赖规则
  await Promise.all([loadUsers(), loadStats()])
})
</script>

<style scoped>
.exp-admin {
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

/* ---------- 模块切换 ---------- */
.view-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.view-tabs .status-tab,
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
.view-tabs .status-tab:hover,
.status-tab:hover {
  color: var(--primary);
}
.view-tabs .status-tab.is-active,
.status-tab.is-active {
  background: var(--primary);
  color: #fff;
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

.user-list {
  display: flex;
  flex-direction: column;
}
.user-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.user-row:last-child {
  border-bottom: none;
}
.user-row.is-selected {
  background: var(--accent-wash);
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

.user-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 22px;
}
.user-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.user-avatar.sm {
  width: 36px;
  height: 36px;
  font-size: 20px;
}
.user-avatar.multi {
  color: var(--primary);
}

.user-main {
  flex: 1;
  min-width: 0;
}
.user-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.user-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.id-chip {
  padding: 1px 6px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 3px;
}
.level-chip {
  padding: 1px 8px;
  font-size: 11px;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.user-meta {
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

.user-exp {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
  border-radius: var(--radius-sm);
  font-variant-numeric: tabular-nums;
}

.user-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.user-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 经验流水 ---------- */
.exp-list {
  display: flex;
  flex-direction: column;
}
.exp-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.exp-row:last-child {
  border-bottom: none;
}
.exp-row:hover {
  background: var(--bg-muted);
}

.exp-value {
  flex-shrink: 0;
  min-width: 62px;
  padding: 4px 10px;
  font-size: 13px;
  font-weight: 700;
  text-align: center;
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
  border-radius: 999px;
  font-variant-numeric: tabular-nums;
}
.exp-value.is-minus {
  color: var(--danger);
  background: var(--accent-soft);
}

.exp-main {
  flex: 1;
  min-width: 0;
}
.exp-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.exp-user {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.type-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.bind-text {
  font-size: 12px;
  color: var(--text-light);
}
.exp-desc {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
}
.exp-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}

.exp-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.exp-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.exp-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 规则表 ---------- */
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
}
.rule-table tr:last-child td {
  border-bottom: none;
}
.rule-table tbody tr:hover td {
  background: var(--bg-muted);
}
.rule-type {
  padding: 1px 6px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.cell-num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.cell-act {
  width: 64px;
  text-align: right;
}

/* ---------- 弹窗 ---------- */
.form-item {
  margin-bottom: 14px;
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
.dialog-tip {
  margin: 0;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius);
}

.adjust-user {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.quick-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
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
  .user-row,
  .exp-row {
    flex-wrap: wrap;
  }
  .user-main,
  .exp-main {
    flex: 1 1 60%;
  }
  .user-actions,
  .exp-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
