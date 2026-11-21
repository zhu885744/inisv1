<template>
  <div class="article-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">文章列表</h2>
          <p class="block-desc">
            审核通过的草稿才会发布到前台；置顶与审核均支持批量操作
            <template v-if="total > 0"> · 共 {{ total }} 篇</template>
          </p>
        </div>
        <div class="head-actions">
          <router-link to="/admin/article/write" class="btn btn-primary btn-sm">
            <i class="bi bi-pencil-square" /> 写文章
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
            v-model="groupId"
            :options="groupSelectOptions"
            icon="bi bi-folder2"
            placeholder="全部分类"
            @change="reload" />

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
              placeholder="搜索标题…"
              @input="doSearch" />
          </div>
        </div>
      </div>

      <!-- 批量操作栏（选中后出现） -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 篇</span>
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
            <button class="btn btn-sm" :disabled="busy" @click="batchPin(1)">
              <i class="bi bi-pin-angle" /> 置顶
            </button>
            <button class="btn btn-sm" :disabled="busy" @click="batchPin(0)">
              <i class="bi bi-pin-angle-fill" /> 取消置顶
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 删除
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <!-- 回收站提示条 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站中的文章不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !total" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-file-earmark-text'" :text="emptyText" />
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
          <span class="list-head-text">本页 {{ list.length }} 篇 · 共 {{ total }} 篇</span>
        </div>

        <ul class="post-list">
        <li v-for="item in list" :key="item.id" class="post-row" :class="{ 'is-selected': isSelected(item.id) }">
          <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
            <FormCheckbox
              :model-value="isSelected(item.id)"
              @change="toggleSelect(item.id)" />
          </label>

          <div class="post-cover" :class="{ 'is-trash': trash }" @click="onTitleClick(item)">
            <img v-if="coverOf(item)" :src="coverOf(item)" :alt="item.title" loading="lazy" />
            <i v-else class="bi bi-file-richtext" />
          </div>

          <div class="post-main">
            <h3 class="post-title">
              <i v-if="Number(item.top) === 1" class="bi bi-pin-angle-fill top-flag" title="已置顶" />
              <a href="javascript:;" :class="{ 'is-trash': trash }" @click="onTitleClick(item)">{{ item.title || '无标题' }}</a>
              <span class="post-status" :class="statusClass(item)">{{ statusLabel(item) }}</span>
              <!-- 当前可见范围：公开 / 登录可见 / 付费可见（N 积分） -->
              <ScopeBadge :scope="item.scope" :integral="item.scope_integral" />
            </h3>

            <p class="post-abstract">
              <EmojiText v-if="item.abstract" :text="item.abstract" :size="16" />
              <template v-else>暂无摘要</template>
            </p>

            <div class="post-meta">
              <!-- 驳回原因：审核驳回时填写，作者也会在通知与「我的文章」里看到
                   只判 reason 是否存在（审核通过时后端会清空它） -->
              <span v-if="item.reason" class="meta-chip is-reason" :title="item.reason">
                <i class="bi bi-info-circle" /> {{ item.reason }}
              </span>
              <span class="meta-chip"><i class="bi bi-person" /> {{ authorName(item) }}</span>
              <span v-if="groupNameOf(item)" class="meta-chip">
                <i class="bi bi-folder2" /> {{ groupNameOf(item) }}
              </span>
              <span v-for="t in tagNamesOf(item)" :key="t" class="meta-chip">
                <i class="bi bi-tag" /> {{ t }}
              </span>
              <span class="meta-text"><i class="bi bi-eye" /> {{ item.views || 0 }}</span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
            </div>
          </div>

          <div class="post-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>

            <template v-else>
              <button class="btn btn-ghost btn-sm" title="查看" @click="viewArticle(item)">
                <i class="bi bi-eye" />
              </button>
              <router-link :to="`/admin/article/edit/${item.id}`" class="btn btn-ghost btn-sm" title="编辑">
                <i class="bi bi-pencil" />
              </router-link>
              <button
                class="btn btn-ghost btn-sm"
                :title="Number(item.top) === 1 ? '取消置顶' : '置顶'"
                :disabled="busy"
                @click="toggleTop(item)"
              >
                <i :class="Number(item.top) === 1 ? 'bi bi-pin-angle-fill' : 'bi bi-pin-angle'" />
              </button>
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

    <!-- 驳回原因（单条 / 批量共用） -->
    <AuditDialog
      :visible="auditDialog.visible"
      :audit="auditDialog.audit"
      :default-reason="auditDialog.target?.item?.reason || ''"
      :owner-uid="auditDialog.target?.item?.uid || 0"
      owner-label="作者"
      :loading="auditDialog.loading"
      :message="
        auditDialog.target?.batch
          ? `将驳回已选中的 ${selectedIds.length} 篇文章，原因会一并通知各自的作者。`
          : '驳回后作者会收到通知，其中包含你填写的原因。'
      "
      @update:visible="closeAuditDialog"
      @confirm="runAudit"
    />
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 文章列表（/admin/article 的子路由）
 *
 * 与父级 Articles.vue 通过 inject('articlesAdmin') 联动：统计卡片点击 → 列表筛选，
 * 列表增删改 → 刷新统计卡片。列表状态（标签 / 回收站 / 分类 / 排序 / 搜索 / 页码）都在这里维护。
 *
 * 说明：此前 /admin/article 直接指向本文件，统计概览缺失且全部逻辑堆在一页；
 * 现已与用户管理（Users/UserList）、友链管理（Links/LinkList）保持同一结构。
 */
import { ref, reactive, computed, inject, onMounted, onUnmounted, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import EmojiText from '@/components/EmojiText.vue'
import ScopeBadge from '@/components/ScopeBadge.vue'
import AuditDialog from '@/components/admin/AuditDialog.vue'
import {
  getArticles,
  updateArticle,
  removeArticle,
  restoreArticle,
  forceDeleteArticle,
  clearArticleRecycle,
  getArticleGroups
} from '@/api/article'
import { listUsersByIds } from '@/api/users'
import { listAllTags } from '@/api/tags'
import { fromNow } from '@/utils/time'
import { parseIdField, debounce } from '@/utils/helper'
import { ARTICLE_TABS, ARTICLE_TRASH_KEY } from '@/utils/article'
import { toast } from '@/utils/toast'

const router = useRouter()

// 父级 Articles.vue 注入的联动能力（统计卡片 ←→ 列表筛选）
const admin = inject('articlesAdmin', null)

const pageSize = ref(PAGE_SIZE_DEFAULT)
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
// 回收站模式
const trash = ref(false)

// 列表字段（避免拉取 content 等大字段）；scope / scope_integral 用于展示「当前可见范围」
const ARTICLE_FIELDS =
  'id,uid,title,abstract,covers,group,tags,top,status,audit,reason,scope,scope_integral,views,create_time,update_time,publish_time,delete_time'

// 状态筛选：与父级统计卡片共用同一份定义（utils/article.js），口径不会各写一份
const tabs = ARTICLE_TABS

// 排序白名单（后端 order 未做白名单校验，必须由前端限定取值）
const sortOptions = [
  { value: 'create_time desc', label: '最新创建' },
  { value: 'create_time asc', label: '最早创建' },
  { value: 'publish_time desc', label: '最新发布' },
  { value: 'update_time desc', label: '最近更新' },
  { value: 'views desc', label: '浏览量最多' }
]

const status = ref('all')
const sortKey = ref('create_time desc')
const groupId = ref('')
const keyword = ref('')
const searchKey = ref('')

const groups = ref([])
const tags = ref([])
const authorMap = ref({})

// 选中集合（用于批量操作）
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

const groupOptions = computed(() => {
  const source = groups.value || []
  const build = (pid, level) =>
    source
      .filter((g) => Number(g.pid || 0) === Number(pid))
      .flatMap((g) => [
        { id: g.id, label: `${'　'.repeat(level)}${level ? '└ ' : ''}${g.name}` },
        ...build(g.id, level + 1)
      ])
  return build(0, 0)
})

const groupMap = computed(() => {
  const map = {}
  ;(groups.value || []).forEach((g) => { map[g.id] = g.name })
  return map
})

// 分类下拉选项：首项为「全部分类」，其余为带缩进的分类树
// （groupId 用字符串比较，故 value 统一取字符串）
const groupSelectOptions = computed(() => [
  { value: '', label: '全部分类' },
  ...groupOptions.value.map((g) => ({ value: String(g.id), label: g.label }))
])

const tagMap = computed(() => {
  const map = {}
  ;(tags.value || []).forEach((t) => { map[t.id] = t.name })
  return map
})

const emptyText = computed(() => {
  if (trash.value) return '回收站是空的'
  if (searchKey.value || groupId.value) return '没有匹配的文章'
  return '暂无文章'
})

// ---------- 状态展示 ----------
function statusLabel(item) {
  if (Number(item.status) === 0) return '草稿'
  const audit = Number(item.audit)
  if (audit === 0) return '待审核'
  if (audit === 2) return '未通过'
  return '已发布'
}

function statusClass(item) {
  if (Number(item.status) === 0) return 'is-draft'
  const audit = Number(item.audit)
  if (audit === 0) return 'is-audit'
  if (audit === 2) return 'is-reject'
  return 'is-pub'
}

function coverOf(item) {
  const c = item.covers
  if (!c) return ''
  if (Array.isArray(c)) return c[0] || ''
  return String(c).split(',')[0] || ''
}

function authorName(item) {
  const uid = Number(item.uid)
  if (!uid) return '—'
  const user = authorMap.value[uid]
  return user?.nickname || `用户 ${uid}`
}

function groupNameOf(item) {
  const ids = parseIdField(item.group)
  return ids.length ? groupMap.value[ids[0]] || '' : ''
}

function tagNamesOf(item) {
  return parseIdField(item.tags)
    .map((id) => tagMap.value[id])
    .filter(Boolean)
    .slice(0, 3)
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.publish_time || item.create_time)
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  try {
    // 回收站模式忽略状态标签的筛选（被删除的文章同样可能带任意 status/audit）
    const where = trash.value ? {} : { ...currentTab.value.where }

    // 分类：group 字段存储为 |id|，用 LIKE 匹配
    if (groupId.value !== '') {
      where.group = { $like: `%|${groupId.value}|%` }
    }
    // 关键词：剔除会干扰 LIKE 的字符
    const kw = searchKey.value.replace(/[%_'\\]/g, '').trim()
    if (kw) {
      where.title = { $like: `%${kw}%` }
    }

    const params = {
      page: page.value,
      limit: pageSize.value,
      field: ARTICLE_FIELDS,
      order: trash.value ? 'delete_time desc' : sortKey.value,
      where: JSON.stringify(where)
    }
    if (trash.value) params.onlyTrashed = true

    const res = await getArticles(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0

    clearSelection()
    await Promise.all([loadAuthors(list.value), ensureMeta()])
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
    // 列表数据变化后同步父级统计（父级内部做了合并节流）
    admin?.refreshStats?.()
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

// 分类 / 话题字典只需加载一次
let metaReady = false
async function ensureMeta() {
  if (metaReady) return
  metaReady = true
  try {
    const res = await getArticleGroups()
    groups.value = res.data?.data || []
  } catch {
    groups.value = []
  }
  try {
    const res = await listAllTags()
    tags.value = res.data?.data || []
  } catch {
    tags.value = []
  }
}

function reload() {
  page.value = 1
  load()
}

function switchTab(key) {
  const wasTrash = trash.value
  trash.value = false
  // 已经在同一标签且原本不在回收站时无需重复加载
  if (!wasTrash && status.value === key) return
  status.value = key
  admin?.setActive(key)
  reload()
}

function toggleTrash() {
  trash.value = !trash.value
  admin?.setActive(trash.value ? ARTICLE_TRASH_KEY : status.value)
  reload()
}

// 父级统计卡片点击后的回调（key 与 tabs / 回收站对应）
function applyFilter(key) {
  if (key === ARTICLE_TRASH_KEY) {
    if (trash.value) return
    trash.value = true
  } else {
    const wasTrash = trash.value
    trash.value = false
    if (!wasTrash && status.value === key) return
    status.value = key
  }
  reload()
}

if (admin) admin.registerFilter(applyFilter)
onUnmounted(() => admin?.unregisterFilter?.(applyFilter))

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

// 本页全选：以当前列表为范围（跨页选择不保留，翻页时 load() 会清空）
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
function viewArticle(item) {
  router.push(Number(item.status) === 0 ? `/admin/article/edit/${item.id}` : `/archives/${item.id}`)
}

// 回收站中的文章没有前台页面，点击不跳转
function onTitleClick(item) {
  if (trash.value) return
  viewArticle(item)
}

/**
 * 管理员局部更新文章
 * 重要：article/update 会把「未传 status」当作草稿处理，因此必须回传原 status；
 * 同时回传原 audit，避免被后端的审核规则重置。
 */
function patchPayload(item, patch) {
  return {
    id: item.id,
    status: Number(item.status),
    audit: Number(item.audit),
    ...patch
  }
}

// ---------- 审核（驳回需填原因） ----------
// 驳回（audit=2）要先弹出原因输入框：原因会写进通知（邮件 + 站内信），
// 作者也能在「我的文章」里看到，避免只有一句「未通过」
const auditDialog = reactive({
  visible: false,
  audit: 2,
  loading: false,
  // 单条：{ item }；批量：{ batch: true }
  target: null
})

async function applyAudit(item, value, reason = '') {
  busy.value = true
  try {
    await updateArticle(patchPayload(item, { audit: value, reason }))
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

async function toggleTop(item) {
  const next = Number(item.top) === 1 ? 0 : 1
  busy.value = true
  try {
    await updateArticle(patchPayload(item, { top: next }))
    item.top = next
    toast.success(next === 1 ? '已置顶' : '已取消置顶')
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askRemove(item) {
  openConfirm({
    title: '删除文章',
    message: `确定删除《${item.title || '无标题'}》吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeArticle(item.id)
      toast.success('已移入回收站')
      if (list.value.length === 1 && page.value > 1) page.value -= 1
      await load()
    }
  })
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除《${item.title || '无标题'}》吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteArticle([item.id])
      toast.success('已彻底删除')
      if (list.value.length === 1 && page.value > 1) page.value -= 1
      await load()
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreArticle([item.id])
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
    try {
      await updateArticle(patchPayload(item, patch))
      Object.assign(item, patch)
      ok += 1
    } catch {
      /* 单条失败不影响其它 */
    }
  }
  busy.value = false
  clearSelection()
  toast.success(`操作完成：成功 ${ok} 篇，失败 ${targets.length - ok} 篇`)
  if (ok !== targets.length) await load()
}

function batchAudit(value) {
  // 批量驳回同样要填原因（一份原因写给选中的所有文章，逐条各自通知作者）
  if (Number(value) === 2) {
    auditDialog.target = { batch: true }
    auditDialog.audit = 2
    auditDialog.visible = true
    return
  }
  batchPatch({ audit: value })
}

function batchPin(value) {
  batchPatch({ top: value })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 篇文章吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeArticle(ids)
      toast.success(`已删除 ${ids.length} 篇`)
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
    await restoreArticle(ids)
    toast.success(`已恢复 ${ids.length} 篇`)
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
    message: `确定彻底删除选中的 ${ids.length} 篇文章吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteArticle(ids)
      toast.success(`已彻底删除 ${ids.length} 篇`)
      clearSelection()
      if (list.value.length === ids.length && page.value > 1) page.value -= 1
      await load()
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有文章将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearArticleRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await load()
    }
  })
}

onMounted(() => {
  ensureMeta()
  load()
})
</script>

<style scoped>
.article-admin {
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

.post-list {
  display: flex;
  flex-direction: column;
}
.post-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 8px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.post-row:last-child {
  border-bottom: none;
}
.post-row.is-selected {
  background: var(--accent-wash);
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

.post-cover {
  width: 108px;
  height: 72px;
  flex-shrink: 0;
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--bg-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-light);
  font-size: 22px;
  cursor: pointer;
}
.post-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}
.post-cover:hover img {
  transform: scale(1.05);
}
/* 回收站内不可跳转，去掉可点击暗示 */
.post-cover.is-trash,
.post-title a.is-trash {
  cursor: default;
}

.post-main {
  flex: 1;
  min-width: 0;
}
.post-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.post-title a {
  color: var(--text);
}
.post-title a:hover {
  color: var(--primary);
}
.top-flag {
  font-size: 12px;
  color: var(--warning);
}
.post-status {
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 400;
  border-radius: 3px;
}
.post-status.is-pub {
  background: rgba(108, 154, 77, 0.12);
  color: var(--success);
}
.post-status.is-draft {
  background: var(--bg-muted);
  color: var(--text-muted);
}
.post-status.is-audit {
  background: var(--gold-wash);
  color: var(--warning);
}
.post-status.is-reject {
  background: var(--accent-soft);
  color: var(--danger);
}

.post-abstract {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--text-soft);
  line-height: 1.7;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.post-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
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

.post-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  align-self: center;
}
.post-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.post-actions .ok:hover:not(:disabled) {
  border-color: var(--success);
  color: var(--success);
}
.post-actions .btn:disabled {
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
  .post-row {
    flex-wrap: wrap;
  }
  .post-cover {
    width: 84px;
    height: 60px;
  }
  .post-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
