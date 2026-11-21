<template>
  <div class="moment-list">
    <div class="card card-pad">
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
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索动态内容…"
              aria-label="搜索动态内容"
              @input="doSearch" />
          </div>

          <button
            class="btn btn-ghost btn-sm"
            :disabled="loading"
            title="刷新"
            aria-label="刷新列表"
            @click="load"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
        </div>
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
            <button class="btn btn-sm" :disabled="busy" @click="batchAudit(0)">
              <i class="bi bi-hourglass-split" /> 打回待审核
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
        <span><i class="bi bi-trash3" /> 回收站中的动态不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !total" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-lightning-charge'" :text="emptyText" />
      </div>

      <template v-else>
        <!-- 本页全选 -->
        <div class="list-head">
          <label class="pick" title="全选本页">
            <FormCheckbox
              ref="selectAllRef"
              :model-value="pageAllSelected"
              :disabled="busy"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">本页 {{ list.length }} 条 · 共 {{ total }} 条</span>
        </div>

        <ul class="moment-items">
          <li
            v-for="item in list"
            :key="item.id"
            class="moment-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox :model-value="isSelected(item.id)" @change="toggleSelect(item.id)" />
            </label>

            <!-- 配图缩略图：固定 3 列展示，超出部分用 +N 角标提示（解析结果按页缓存） -->
            <div class="row-thumbs" :class="{ 'is-empty': !thumbsOf(item).length }">
              <template v-if="thumbsOf(item).length">
                <button
                  v-for="(img, i) in thumbsOf(item).slice(0, MAX_THUMBS)"
                  :key="img + i"
                  type="button"
                  class="thumb"
                  :title="`查看第 ${i + 1} 张（共 ${thumbsOf(item).length} 张）`"
                  :aria-label="`查看第 ${i + 1} 张配图`"
                  @click="preview(item, i)"
                >
                  <img :src="img" :alt="`配图 ${i + 1}`" loading="lazy" />
                  <span v-if="i === MAX_THUMBS - 1 && moreCount(item)" class="thumb-more">
                    +{{ moreCount(item) }}
                  </span>
                </button>
              </template>
              <i v-else class="bi bi-chat-square-quote" />
            </div>

            <div class="row-main">
              <p class="row-content" @click="onContentClick(item)">
                <EmojiText v-if="item.content" :text="item.content" :size="17" />
                <template v-else>（仅图片，无文字内容）</template>
              </p>

              <div class="row-meta">
                <span class="post-status" :class="statusClass(item)">{{ statusLabel(item) }}</span>

                <!-- 当前可见范围：公开 / 登录可见 / 付费可见（N 积分） -->
                <ScopeBadge :scope="item.scope" :integral="item.scope_integral" />

                <!-- 驳回原因：审核驳回时填写，作者也会在通知与「我的动态」里看到
                     只判 reason 是否存在（审核通过时后端会清空它），避免状态字段口径不一致时标签不显示 -->
                <span v-if="item.reason" class="meta-chip is-reason" :title="item.reason">
                  <i class="bi bi-info-circle" /> {{ item.reason }}
                </span>

                <span v-if="Number(item.top) === 1" class="meta-chip is-top">
                  <i class="bi bi-pin-angle-fill" /> 置顶
                </span>

                <span class="meta-chip">
                  <i class="bi bi-person" /> {{ authorName(item) }}
                </span>

                <span v-if="item.location" class="meta-chip">
                  <i class="bi bi-geo-alt" /> {{ item.location }}
                </span>

                <span class="meta-text"><i class="bi bi-eye" /> {{ item.views || 0 }}</span>
                <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                <span class="meta-text">#{{ item.id }}</span>
              </div>
            </div>

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
                <button class="btn btn-ghost btn-sm" title="查看" aria-label="查看" @click="view(item)">
                  <i class="bi bi-eye" />
                </button>
                <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" @click="openEdit(item)">
                  <i class="bi bi-pencil" />
                </button>
                <button
                  class="btn btn-ghost btn-sm"
                  :title="Number(item.top) === 1 ? '取消置顶' : '置顶'"
                  :aria-label="Number(item.top) === 1 ? '取消置顶' : '置顶'"
                  :disabled="busy"
                  @click="toggleTop(item)"
                >
                  <i :class="Number(item.top) === 1 ? 'bi bi-pin-angle-fill' : 'bi bi-pin-angle'" />
                </button>
                <!-- 审核状态（0 待审核 / 1 已通过 / 2 未通过）：单选切换，当前状态高亮
                     auditRevision 用于「取消驳回弹窗」后强制单选组重建，回到真实状态 -->
                <div :key="`audit-${item.id}-${auditRevision}`" class="audit-switch" role="radiogroup" aria-label="审核状态">
                  <label
                    v-for="opt in AUDIT_OPTIONS"
                    :key="opt.key"
                    class="audit-option"
                    :class="[`is-${opt.key}`, { 'is-active': Number(item.audit) === opt.value, 'is-disabled': busy }]"
                    :title="`设为${opt.label}`"
                  >
                    <FormRadio
                      class="audit-radio"
                      :name="`moment-audit-${item.id}`"
                      :value="opt.value"
                      :model-value="Number(item.audit) === opt.value"
                      :disabled="busy"
                      :aria-label="opt.label"
                      @change="audit(item, opt.value)" />
                    <i :class="opt.icon" aria-hidden="true" />
                  </label>
                </div>
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
          ? `将驳回已选中的 ${selectedIds.length} 条动态，原因会一并通知各自的作者。`
          : '驳回后作者会收到通知，其中包含你填写的原因。'
      "
      @update:visible="closeAuditDialog"
      @confirm="runAudit"
    />

    <!-- 编辑弹窗 -->
    <MomentEditDialog v-model:visible="editVisible" :moment="editTarget" @saved="onSaved" />
  </div>
</template>

<script setup>
import FormRadio from '@/components/form/FormRadio.vue'
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 动态列表（/admin/moments 子路由）
 *
 * 与父级 Moments.vue 通过 inject('momentsAdmin') 联动：统计卡片点击 → 列表筛选，
 * 列表增删改 → 刷新统计卡片。列表状态（标签 / 回收站 / 排序 / 搜索 / 页码）都在这里维护。
 */
import { ref, reactive, computed, inject, watchEffect, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import EmojiText from '@/components/EmojiText.vue'
import ScopeBadge from '@/components/ScopeBadge.vue'
import MomentEditDialog from '@/components/admin/MomentEditDialog.vue'
import AuditDialog from '@/components/admin/AuditDialog.vue'
import {
  listMomentsAdmin,
  updateMoment,
  removeMoment,
  restoreMoment,
  forceDeleteMoment,
  clearMomentRecycle,
  setMomentTop
} from '@/api/moments'
import { listUsersByIds } from '@/api/users'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'
import {
  MOMENT_TABS,
  MOMENT_SORT_OPTIONS,
  MOMENT_TRASH_KEY,
  MOMENT_DEFAULT_ORDER,
  MOMENT_TRASH_ORDER,
  momentListParams,
  isMomentPublic,
  parseMomentImages,
  parseMomentMedia,
  momentStatusLabel,
  momentStatusClass
} from '@/utils/moment'

const router = useRouter()

// 父级 Moments.vue 注入的联动能力（统计卡片 ←→ 列表筛选）
const admin = inject('momentsAdmin', null)

const pageSize = ref(PAGE_SIZE_DEFAULT)
const tabs = MOMENT_TABS
const sortOptions = MOMENT_SORT_OPTIONS

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
// 回收站模式
const trash = ref(false)
const status = ref('all')
const sortKey = ref(MOMENT_DEFAULT_ORDER)
const keyword = ref('')
const searchKey = ref('')
const authorMap = ref({})
const selectedIds = ref([])

const currentTab = computed(() => tabs.find((tab) => tab.key === status.value) || tabs[0])

const emptyText = computed(() => {
  if (trash.value) return '回收站是空的'
  if (searchKey.value) return '没有匹配的动态'
  if (status.value !== 'all') return `暂无「${currentTab.value.label}」的动态`
  return '暂无动态'
})

// ---------- 展示辅助 ----------
// 配图解析结果按当前页缓存：模板里对同一条动态会读多次，避免重复 split
const thumbsMap = computed(() => {
  const map = Object.create(null)
  for (const item of list.value) map[item.id] = parseMomentImages(item.images)
  return map
})

// 列表最多展示 3 张缩略图，其余张数以 +N 角标提示（点击任一缩略图进入大图浏览）
const MAX_THUMBS = 3

function thumbsOf(item) {
  return thumbsMap.value[item.id] || []
}

function moreCount(item) {
  return Math.max(0, thumbsOf(item).length - MAX_THUMBS)
}

function preview(item, index) {
  // 带视频信息：实况图在灯箱里手动点「播放实况」才播
  openLightbox(parseMomentMedia(item.images), index)
}

function authorName(item) {
  const uid = Number(item.uid)
  if (!uid) return '—'
  return authorMap.value[uid]?.nickname || `用户 ${uid}`
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.publish_time || item.create_time)
}

const statusLabel = momentStatusLabel
const statusClass = momentStatusClass

// ---------- 数据加载 ----------
// 请求序号：快速切换筛选 / 搜索时丢弃过期响应，避免旧数据覆盖新结果
let reqSeq = 0

function buildParams() {
  return momentListParams({
    // 回收站忽略状态标签（已删除的动态同样带任意 status/audit）
    key: trash.value ? MOMENT_TRASH_KEY : status.value,
    page: page.value,
    limit: pageSize.value,
    keyword: searchKey.value,
    // 回收站固定按删除时间倒序，其余跟随排序白名单
    order: trash.value ? MOMENT_TRASH_ORDER : sortKey.value
  })
}

async function load() {
  const run = ++reqSeq
  loading.value = true
  try {
    const res = await listMomentsAdmin(buildParams())
    if (run !== reqSeq) return
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
    loadAuthors(list.value) // 作者昵称不阻塞列表首屏
  } catch {
    if (run !== reqSeq) return
    list.value = []
    total.value = 0
  } finally {
    if (run === reqSeq) loading.value = false
  }
}

// 批量拉取当前页作者昵称：只请求尚未缓存的 uid，一次请求搞定
const requestedAuthors = new Set()

async function loadAuthors(items) {
  const ids = [...new Set((items || []).map((i) => Number(i.uid)).filter(Boolean))]
  const missing = ids.filter((id) => !authorMap.value[id] && !requestedAuthors.has(id))
  if (!missing.length) return
  missing.forEach((id) => requestedAuthors.add(id))
  try {
    const res = await listUsersByIds(missing)
    const map = { ...authorMap.value }
    ;(res.data?.data || []).forEach((u) => {
      if (u?.id) map[u.id] = u
    })
    authorMap.value = map
  } catch {
    // 作者信息缺失时回退为「用户 ID」展示，不阻断列表；下次进入允许重试
    missing.forEach((id) => requestedAuthors.delete(id))
  }
}

function reload() {
  page.value = 1
  load()
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reload()
}, 350)

// ---------- 筛选联动 ----------
function switchTab(key) {
  if (!trash.value && status.value === key) return
  trash.value = false
  status.value = key
  admin?.setActive(key)
  reload()
}

function toggleTrash() {
  trash.value = !trash.value
  admin?.setActive(trash.value ? MOMENT_TRASH_KEY : status.value)
  reload()
}

// 父级统计卡片点击后的回调（key 与 tabs / trash 对应）
function applyFilter(key) {
  if (key === MOMENT_TRASH_KEY) {
    if (trash.value && page.value === 1) return
    trash.value = true
    reload()
    return
  }
  const next = tabs.some((t) => t.key === key) ? key : 'all'
  if (!trash.value && status.value === next) return
  trash.value = false
  status.value = next
  reload()
}

admin?.registerFilter(applyFilter)
onUnmounted(() => admin?.unregisterFilter?.(applyFilter))

// ---------- 选中 ----------
// 列表整页可能上百条，用 Set 做 O(1) 命中判断
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((i) => selectedSet.value.has(i.id))
)
const someSelected = computed(() => selectedIds.value.length > 0 && !pageAllSelected.value)

const selectAllRef = ref(null)

// 部分选中时展示半选态（indeterminate 只能通过 DOM 属性设置）
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

// ---------- 确认弹窗 ----------
const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  action: null
})

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

// ---------- 增删改统一收尾 ----------
// removedCount：本次从当前页移除的条数；当前页被清空时自动回退一页
async function afterMutation(removedCount = 0) {
  if (removedCount > 0 && list.value.length <= removedCount && page.value > 1) {
    page.value -= 1
  }
  await load()
  refreshStats()
}

function refreshStats() {
  admin?.refreshStats?.()
}

// ---------- 单条操作 ----------
// 已公开发布的动态去前台详情页，否则打开编辑弹窗
function view(item) {
  if (!trash.value && isMomentPublic(item)) {
    router.push(`/moments/${item.id}`)
    return
  }
  toast.info('该动态未公开发布，已打开编辑')
  openEdit(item)
}

function onContentClick(item) {
  if (trash.value) return
  view(item)
}

/**
 * 管理员局部更新动态
 * 重要：moments/update 在「首次发布」时会按审核开关重算 audit（开启审核 → 0 待审核），
 * 因此必须回传原 status 与原 audit，避免「编辑一下就被打回待审核」。
 */
function patchPayload(item, patch) {
  return {
    id: item.id,
    status: Number(item.status),
    audit: Number(item.audit),
    ...patch
  }
}

// 审核三态：0 待审核 / 1 已通过 / 2 未通过（列表内以单选框直接切换）
const AUDIT_TIPS = { 0: '已打回待审核', 1: '已通过审核', 2: '已驳回' }

const AUDIT_OPTIONS = [
  { value: 0, key: 'pending', label: '待审核', icon: 'bi bi-hourglass-split' },
  { value: 1, key: 'pass', label: '已通过', icon: 'bi bi-check2-circle' },
  { value: 2, key: 'reject', label: '未通过', icon: 'bi bi-slash-circle' }
]

// ---------- 审核（驳回需填原因） ----------
// 驳回（audit=2）要先弹出原因输入框：原因会写进通知（邮件 + 站内信），
// 作者也能在「我的动态」里看到，避免只有一句「未通过」
const auditDialog = reactive({
  visible: false,
  audit: 2,
  loading: false,
  // 单条：{ item }；批量：{ batch: true }
  target: null
})

// 取消弹窗后强制单选组重建：点过的单选框已被浏览器选中，绑定的 audit 没变不会自动回退
const auditRevision = ref(0)

async function applyAudit(item, value, reason = '') {
  busy.value = true
  try {
    await updateMoment(patchPayload(item, { audit: value, reason }))
    item.audit = value
    item.reason = reason
    toast.success(AUDIT_TIPS[value] || '操作成功')
    // 状态筛选下，审核状态变更都可能让条目不再属于当前标签
    if (status.value !== 'all') await afterMutation(0)
    else refreshStats()
    return true
  } catch {
    /* 拦截器已提示 */
    return false
  } finally {
    busy.value = false
  }
}

function audit(item, value) {
  // 通过 / 打回待审核：直接生效；驳回：先让管理员填原因
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
  auditRevision.value += 1
}

async function toggleTop(item) {
  const next = Number(item.top) === 1 ? 0 : 1
  busy.value = true
  try {
    await setMomentTop([item.id], next)
    item.top = next
    toast.success(next === 1 ? '已置顶' : '已取消置顶')
    // 「置顶优先」排序下置顶结果会改变行序，重新拉取保持一致
    if (!trash.value && sortKey.value.startsWith('top')) await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askRemove(item) {
  openConfirm({
    title: '删除动态',
    message: `确定删除动态 #${item.id} 吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeMoment([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除动态 #${item.id} 吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteMoment([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreMoment([item.id])
    toast.success('已恢复')
    await afterMutation(1)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

// ---------- 批量操作 ----------
// 逐条更新（后端 update 仅支持单条 id），统计成功/失败
async function batchPatch(patch) {
  const targets = list.value.filter((i) => selectedSet.value.has(i.id))
  if (!targets.length) return
  busy.value = true
  let ok = 0
  try {
    for (const item of targets) {
      try {
        await updateMoment(patchPayload(item, patch))
        Object.assign(item, patch)
        ok += 1
      } catch {
        /* 单条失败不影响其它 */
      }
    }
  } finally {
    busy.value = false
  }
  clearSelection()
  toast.success(`操作完成：成功 ${ok} 条，失败 ${targets.length - ok} 条`)
  // 有失败条目或处于状态筛选下，重新拉取保证列表与统计准确
  if (ok !== targets.length || status.value !== 'all') await afterMutation(0)
  else refreshStats()
}

function batchAudit(value) {
  // 批量驳回同样要填原因（一份原因写给选中的所有条目，逐条各自通知作者）
  if (Number(value) === 2) {
    auditDialog.target = { batch: true }
    auditDialog.audit = 2
    auditDialog.visible = true
    return
  }
  batchPatch({ audit: value })
}

// 置顶接口原生支持批量，无需逐条
async function batchPin(value) {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  try {
    await setMomentTop(ids, value)
    toast.success(value === 1 ? `已置顶 ${ids.length} 条` : `已取消置顶 ${ids.length} 条`)
    clearSelection()
    await afterMutation(0)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 条动态吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeMoment(ids)
      toast.success(`已删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 条动态吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteMoment(ids)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  try {
    await restoreMoment(ids)
    toast.success(`已恢复 ${ids.length} 条`)
    clearSelection()
    await afterMutation(ids.length)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有动态将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearMomentRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

// ---------- 编辑弹窗 ----------
const editVisible = ref(false)
const editTarget = ref(null)

function openEdit(item) {
  editTarget.value = item
  editVisible.value = true
}

async function onSaved() {
  await afterMutation(0)
}

onMounted(load)
</script>

<style scoped>
.moment-list {
  display: flex;
  flex-direction: column;
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

.list-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px 4px;
}
.list-head-text {
  font-size: 12px;
  color: var(--text-muted);
}

.moment-items {
  display: flex;
  flex-direction: column;
}
.moment-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 8px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.moment-row:last-child {
  border-bottom: none;
}
.moment-row.is-selected {
  background: var(--accent-wash);
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

.row-thumbs {
  display: grid;
  /* 固定 3 列：图片不足 3 张也不改变列宽，保证每行正文左边缘对齐 */
  grid-template-columns: repeat(3, 56px);
  gap: 4px;
  flex-shrink: 0;
  align-self: center;
}
.row-thumbs.is-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  /* 与有图时的 3 列等宽（3 * 56 + 2 * 4） */
  width: 176px;
  min-height: 56px;
  color: var(--text-light);
  font-size: 22px;
}
.thumb {
  position: relative;
  width: 56px;
  height: 56px;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  cursor: zoom-in;
  transition: transform 0.2s, border-color 0.2s;
}
/* 只有一张配图时跨两列，避免右侧留一大块空白 */
.thumb:only-child {
  grid-column: span 2;
  width: auto;
}
.thumb img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.thumb:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
}
.thumb:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 1px;
}
.thumb-more {
  position: absolute;
  right: 0;
  bottom: 0;
  min-width: 20px;
  padding: 1px 4px;
  font-size: 11px;
  line-height: 1.5;
  text-align: center;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  border-radius: 6px 0 0 0;
  /* 点击落在缩略图上，直接预览最后一张 */
  pointer-events: none;
}

.row-main {
  flex: 1;
  min-width: 0;
}
.row-content {
  margin: 0 0 8px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  cursor: pointer;
}
.row-content:hover {
  color: var(--primary);
}
.row-meta {
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
.meta-chip.is-top {
  background: var(--gold-wash);
  color: var(--warning);
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

.row-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  align-self: center;
}
.row-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.row-actions .ok:hover:not(:disabled) {
  border-color: var(--success);
  color: var(--success);
}
.row-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 审核状态单选（待审核 / 已通过 / 未通过） ---------- */
.audit-switch {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
}
.audit-option {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 22px;
  border-radius: 3px;
  color: var(--text-light);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
/* 原生单选框只保留语义与键盘操作，视觉交给 .audit-option */
.audit-radio {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.audit-option:hover:not(.is-active):not(.is-disabled) {
  background: var(--bg-card);
  color: var(--primary);
}
.audit-option:focus-within {
  outline: 2px solid var(--primary);
  outline-offset: 1px;
}
.audit-option.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.audit-option.is-active.is-pending {
  background: var(--gold-wash);
  color: var(--warning);
}
.audit-option.is-active.is-pass {
  background: rgba(108, 154, 77, 0.14);
  color: var(--success);
}
.audit-option.is-active.is-reject {
  background: var(--accent-soft);
  color: var(--danger);
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
  .moment-row {
    flex-wrap: wrap;
  }
  /* 窄屏下缩略图独占一行，正文另起一行，避免多图行把正文挤成窄条 */
  .row-main {
    flex: 1 1 100%;
  }
  .row-thumbs {
    grid-template-columns: repeat(3, 44px);
  }
  .row-thumbs.is-empty {
    width: 140px;
    min-height: 44px;
  }
  .thumb {
    width: 44px;
    height: 44px;
  }
  .row-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
