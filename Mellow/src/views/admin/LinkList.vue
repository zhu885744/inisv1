<template>
  <div class="card card-pad">
    <header class="list-head">
      <div>
        <h2 class="block-title">友链列表</h2>
        <p class="block-desc">
          审核通过的友链才会展示在前台友链页
          <template v-if="total > 0"> · 共 {{ total }} 条</template>
        </p>
      </div>
      <div class="head-actions">
        <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
          <i class="bi bi-plus-lg" /> 新建友链
        </button>
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
          v-model="group"
          :options="groupOptions"
          icon="bi bi-collection"
          placeholder="全部分组"
          @change="reload" />

        <FormSelect variant="pill"
          v-model="sortKey"
          :options="sortOptions"
          :disabled="trash"
          icon="bi bi-sort-down"
          placeholder="排序方式"
          @change="reload" />

        <FormSelect variant="pill"
          v-model="searchField"
          :options="searchFields"
          icon="bi bi-funnel"
          placeholder="搜索字段" />

        <div class="search-box">
          <i class="bi bi-search" aria-hidden="true" />
          <FormInput
            v-model="keyword"
            class="search-input"
            type="search"
            placeholder="搜索友链…"
            aria-label="搜索友链"
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
          <button class="btn btn-sm" :disabled="busy" @click="batchAudit(LINK_AUDIT_PASS)">
            <i class="bi bi-check2" /> 通过审核
          </button>
          <button class="btn btn-sm btn-danger" :disabled="busy" @click="batchReject">
            <i class="bi bi-slash-circle" /> 驳回
          </button>
          <button class="btn btn-sm" :disabled="busy" @click="batchAudit(LINK_AUDIT_PENDING)">
            <i class="bi bi-hourglass-split" /> 取消通过
          </button>
          <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
            <i class="bi bi-trash" /> 删除
          </button>
        </template>
        <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
      </div>
    </div>

    <div v-if="trash" class="trash-bar">
      <span><i class="bi bi-trash3" /> 回收站内的友链不再展示，可恢复或彻底删除</span>
      <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
        清空回收站
      </button>
    </div>

    <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

    <div v-else-if="!list.length" class="empty-row">
      <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-link-45deg'" :text="emptyText" />
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

      <ul class="link-list">
        <li v-for="item in list" :key="item.id" class="link-row" :class="{ 'is-selected': isSelected(item.id) }">
          <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
            <FormCheckbox
              :model-value="isSelected(item.id)"
              :disabled="busy"
              :aria-label="`选择友链 ${item.nickname || item.id}`"
              @change="toggleSelect(item.id)" />
          </label>

          <div class="link-avatar">
            <img v-if="item.avatar" :src="item.avatar" :alt="item.nickname" loading="lazy" />
            <i v-else class="bi bi-link-45deg" />
          </div>

          <div class="link-main">
            <div class="link-name-row">
              <span class="link-name">{{ item.nickname || '未命名' }}</span>
              <span class="state-chip" :class="linkAuditClass(item)">
                {{ linkAuditLabel(item) }}
              </span>
              <span class="group-chip">
                <i class="bi bi-collection" /> {{ linkGroupOf(item)?.name || '默认分组' }}
              </span>
              <span class="id-chip">#{{ item.id }}</span>
            </div>
            <div class="link-meta">
              <!-- 申请人：这条友链是谁提交的（uid=0 的老数据 / 管理员直接添加会显示「管理员添加」） -->
              <span class="meta-text is-author" :title="authorTitle(item)">
                <i class="bi bi-person" /> {{ authorName(item) }}
              </span>
              <a class="link-url" :href="safeUrl(item.url) || undefined" target="_blank" rel="noopener noreferrer" :title="item.url">
                <i class="bi bi-box-arrow-up-right" /> {{ item.url || '—' }}
              </a>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
              <!-- 驳回原因：作者也会在通知与「我的友链」里看到
                   只判 reason 是否存在（审核通过时后端会清空它） -->
              <span v-if="item.reason" class="meta-text is-reason" :title="item.reason">
                <i class="bi bi-info-circle" /> {{ item.reason }}
              </span>
              <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
            </div>
            <p v-if="item.description" class="link-desc">{{ item.description }}</p>
          </div>

          <div class="link-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>
            <template v-else>
              <!-- 审核：待审核 / 未通过都能直接「通过」；已通过的可以「驳回」（要填原因） -->
              <button
                v-if="!isLinkAudited(item)"
                class="btn btn-ghost btn-sm"
                title="通过审核"
                aria-label="通过审核"
                :disabled="busy"
                @click="setAudit(item, LINK_AUDIT_PASS)"
              >
                <i class="bi bi-check2-circle" />
              </button>
              <button
                v-else
                class="btn btn-ghost btn-sm danger"
                title="驳回（需要填写原因）"
                aria-label="驳回"
                :disabled="busy"
                @click="askReject(item)"
              >
                <i class="bi bi-slash-circle" />
              </button>
              <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" :disabled="busy" @click="openEdit(item)">
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

    <!-- 新建 / 编辑友链 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑友链 #${edit.id}` : '新建友链'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <!-- 申请人：管理员编辑他人申请的友链时能看到这条是谁提交的（新建时不显示） -->
      <OwnerInfo v-if="edit.id" :uid="edit.uid" label="申请人" class="edit-owner" />

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">站点名称</label>
          <FormInput v-model="edit.nickname" type="text" maxlength="32" placeholder="必填" />
        </div>
        <div class="form-item">
          <label class="form-label">站点链接</label>
          <FormInput v-model="edit.url" type="text" placeholder="必填，需带 http(s)://" />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">站点描述</label>
        <FormTextarea v-model="edit.description" rows="2" placeholder="可选" />
      </div>

      <div class="form-item">
        <label class="form-label">站点头像</label>
        <div class="cover-row">
          <div class="cover-box">
            <img v-if="edit.avatar" :src="edit.avatar" alt="头像预览" loading="lazy" />
            <div v-else class="cover-empty" @click="pickAvatar">
              <i class="bi bi-image" />
              <span>上传</span>
            </div>
            <button v-if="edit.avatar" type="button" class="cover-del" title="移除头像" @click="edit.avatar = ''">
              <i class="bi bi-x" />
            </button>
          </div>
          <div class="cover-fields">
            <FormInput v-model="edit.avatar" type="text" placeholder="粘贴图片链接，或点击下方按钮上传" />
            <div class="cover-actions">
              <button class="btn btn-sm" type="button" :disabled="uploading" @click="pickAvatar">
                <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
              </button>
              <button class="btn btn-sm" type="button" title="从附件库选择" @click="showLibrary = true">
                <i class="bi bi-folder2-open" /> 附件库
              </button>
              <button v-if="edit.avatar" class="btn btn-sm btn-ghost" type="button" @click="edit.avatar = ''">移除</button>
            </div>
          </div>
          <input ref="avatarRef" type="file" accept="image/*" hidden @change="onAvatarChange" />

          <!-- 附件库：选一张已上传的图片作为友链头像 -->
          <AttachmentLibrary
            v-model:visible="showLibrary"
            title="选择友链头像"
            accept="image"
            @select="onLibrarySelect"
          />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">所属分组</label>
          <FormSelect v-model="edit.group" :options="editGroupOptions" />
        </div>
        <div class="form-item">
          <label class="form-label">打开方式</label>
          <FormSelect v-model="edit.target" :options="LINK_TARGET_OPTIONS" />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">审核状态</label>
          <FormSelect v-model="edit.audit" :options="AUDIT_OPTIONS" />
          <p class="form-hint">审核状态、驳回原因与备注仅管理员可修改</p>
        </div>
        <div class="form-item">
          <label class="form-label">备注</label>
          <FormInput v-model="edit.remark" type="text" placeholder="仅管理员可见" />
        </div>
      </div>

      <!-- 驳回原因：选「未通过」时填，作者会在通知与「我的友链」里看到 -->
      <div class="form-item">
        <label class="form-label">
          驳回原因
          <span class="form-hint-inline">选填 · 作者可见</span>
        </label>
        <FormTextarea
          v-model="edit.reason"
          rows="2"
          maxlength="512"
          placeholder="如：站点内容与本站定位不符 / 暂时无法访问" />
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

    <!-- 驳回原因（单条 / 批量共用） -->
    <AuditDialog
      :visible="auditDialog.visible"
      :audit="LINK_AUDIT_REJECT"
      :default-reason="auditDialog.target?.item?.reason || ''"
      :owner-uid="auditDialog.target?.item?.uid || 0"
      owner-label="申请人"
      :loading="auditDialog.loading"
      :message="
        auditDialog.target?.batch
          ? `将驳回已选中的 ${selectedIds.length} 条友链，原因会一并通知各自的申请者。`
          : '驳回后申请者会收到通知，其中包含你填写的原因。'
      "
      @update:visible="closeAuditDialog"
      @confirm="runAudit"
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
 * 友链列表（/admin/links 子路由）
 *
 * 与父级 Links.vue 通过 inject('linksAdmin') 联动：统计卡片点击 → 列表筛选，
 * 列表增删改 → 刷新统计卡片。列表状态（标签 / 回收站 / 分组 / 排序 / 搜索 / 页码）都在这里维护。
 *
 * 后端约束（app/api/controller/links.go、app/model/links.go）：
 * - links/all：非管理员只能查到 audit=1 的友链，后台依赖管理员身份才能看到待审核数据；
 * - links/create、links/update 的允许字段为 nickname/description/url/avatar/target/group/json/text，
 *   管理员（root）额外允许 audit 与 remark，非管理员改他人友链会返回「无权限！」；
 * - 昵称/描述/链接/头像/备注会做 XSS 检测，含恶意代码会被拒绝；
 * - links/count 不支持 onlyTrashed，回收站数量用列表接口的 count；
 * - 分组由后端在 result.group 中解析（未分组时返回「默认分组」）。
 */
import { ref, reactive, computed, watchEffect, inject, onMounted, onUnmounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import AuditDialog from '@/components/admin/AuditDialog.vue'
import OwnerInfo from '@/components/admin/OwnerInfo.vue'
import {
  listLinksAdmin,
  listLinkGroupsAdmin,
  createLink,
  updateLink,
  removeLinks,
  forceDeleteLinks,
  restoreLinks,
  clearLinkRecycle
} from '@/api/links'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { listUsersByIds } from '@/api/users'
import {
  LINK_TABS,
  LINK_TRASH_KEY,
  LINK_SORT_OPTIONS,
  LINK_SEARCH_FIELDS,
  LINK_TARGET_OPTIONS,
  LINK_AUDIT_PASS,
  LINK_AUDIT_PENDING,
  LINK_AUDIT_REJECT,
  LINK_GROUP_FIELD,
  linkListParams,
  linkAuditLabel,
  linkAuditClass,
  isLinkAudited,
  linkGroupOf
} from '@/utils/link'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'
import { safeUrl } from '@/utils/url'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 模板直接消费的选项（与父级统计卡片共用同一份定义）
const tabs = LINK_TABS
const sortOptions = LINK_SORT_OPTIONS
const searchFields = LINK_SEARCH_FIELDS

// 审核三态：与文章 / 动态一致（0 待审核 / 1 已通过 / 2 未通过）
const AUDIT_OPTIONS = [
  { value: LINK_AUDIT_PENDING, label: '待审核' },
  { value: LINK_AUDIT_PASS, label: '已通过' },
  { value: LINK_AUDIT_REJECT, label: '未通过' }
]

// 父级 Links.vue 注入的联动能力（统计卡片 ←→ 列表筛选）
const admin = inject('linksAdmin', null)

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const status = ref('all')
const group = ref('')
const sortKey = ref(LINK_SORT_OPTIONS[0].value)
const searchField = ref('nickname')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])
const groups = ref([])
// 申请人信息（uid → 用户）：列表里展示「是谁申请的」，与文章/动态后台同一套做法
const authorMap = ref({})

// ===== 弹窗 =====
const avatarRef = ref(null)
const uploading = ref(false)
const showLibrary = ref(false)

// 从附件库选头像：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) edit.avatar = url
}

const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  // 申请人 uid：只用于展示「申请人：昵称」，不参与提交
  uid: 0,
  nickname: '',
  url: '',
  description: '',
  avatar: '',
  group: 0,
  target: '_blank',
  audit: LINK_AUDIT_PENDING,
  reason: '',
  remark: ''
})

// 驳回原因弹窗（单条 / 批量共用）
const auditDialog = reactive({
  visible: false,
  loading: false,
  // 单条：{ item }；批量：{ batch: true }
  target: null
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

const groupOptions = computed(() => [
  { value: '', label: '全部分组' },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

// 编辑弹窗里的分组：0 为「默认分组」（后端未分组即 0）
const editGroupOptions = computed(() => [
  { value: 0, label: '默认分组' },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的友链'
  if (trash.value) return '回收站是空的'
  if (status.value === 'pending') return '没有待审核的友链'
  return '还没有友链，先添加一条吧'
})

// ---------- 展示辅助 ----------
function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `添加于 ${fromNow(item.create_time)}`
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  try {
    const params = linkListParams({
      key: trash.value ? LINK_TRASH_KEY : status.value,
      page: page.value,
      limit: pageSize.value,
      keyword: searchKey.value,
      searchField: searchField.value,
      group: group.value,
      order: trash.value ? undefined : sortKey.value
    })
    const res = await listLinksAdmin(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
    loadAuthors(list.value) // 申请人昵称不阻塞列表首屏
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 批量拉取当前页申请人信息：只请求尚未缓存的 uid，一次请求搞定
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
    // 申请人信息缺失时回退为「用户 ID」展示，不阻断列表；下次进入允许重试
    missing.forEach((id) => requestedAuthors.delete(id))
  }
}

// 申请人显示名：昵称优先，其次账号，没有则「用户 ID」
function authorName(item) {
  const uid = Number(item.uid)
  if (!uid) return '管理员添加'
  const user = authorMap.value[uid]
  return user?.nickname || user?.account || `用户 ${uid}`
}

// 申请人 hover 提示：账号 / 昵称 / UID 都摆出来，方便核对是谁申请的
function authorTitle(item) {
  const uid = Number(item.uid)
  if (!uid) return '这条友链由管理员直接添加'
  const user = authorMap.value[uid]
  if (!user) return `用户 UID：${uid}`
  const parts = []
  if (user.account) parts.push(`账号：${user.account}`)
  if (user.nickname) parts.push(`昵称：${user.nickname}`)
  parts.push(`UID：${uid}`)
  return parts.join(' · ')
}

function reload() {
  page.value = 1
  load()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reload()
}, 350)

async function loadGroups() {
  try {
    const res = await listLinkGroupsAdmin({ page: 1, limit: 100, order: 'id asc', field: LINK_GROUP_FIELD })
    groups.value = res.data?.data || []
  } catch {
    groups.value = []
  }
}

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
  admin?.setActive(trash.value ? LINK_TRASH_KEY : status.value)
  keyword.value = ''
  searchKey.value = ''
  page.value = 1
  load()
}

// 父级统计卡片点击后的回调（key 与 tabs / trash 对应）
function applyFilter(key) {
  const next = key === LINK_TRASH_KEY ? null : key
  if (next === null) {
    if (trash.value) return
    trash.value = true
  } else {
    if (!trash.value && status.value === next) return
    trash.value = false
    status.value = next
  }
  keyword.value = ''
  searchKey.value = ''
  page.value = 1
  load()
}

if (admin) admin.registerFilter(applyFilter)
onUnmounted(() => admin?.unregisterFilter?.(applyFilter))

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

// ---------- 头像上传 ----------
function pickAvatar() {
  avatarRef.value?.click()
}

async function onAvatarChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')
    edit.avatar = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

// ---------- 新建 / 编辑 ----------
function resetEdit() {
  edit.id = 0
  edit.uid = 0
  edit.nickname = ''
  edit.url = ''
  edit.description = ''
  edit.avatar = ''
  edit.group = 0
  edit.target = '_blank'
  edit.audit = LINK_AUDIT_PENDING
  edit.reason = ''
  edit.remark = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.uid = Number(item.uid) || 0
  edit.nickname = item.nickname || ''
  edit.url = item.url || ''
  edit.description = item.description || ''
  edit.avatar = item.avatar || ''
  edit.group = Number(item.group || 0)
  edit.target = item.target || '_blank'
  edit.audit = Number(item.audit || LINK_AUDIT_PENDING)
  edit.reason = item.reason || ''
  edit.remark = item.remark || ''
  edit.visible = true
}

async function save() {
  const nickname = edit.nickname.trim()
  const url = edit.url.trim()
  if (!nickname) {
    toast.warning('请输入站点名称')
    return
  }
  if (!url) {
    toast.warning('请输入站点链接')
    return
  }
  if (!/^https?:\/\//i.test(url)) {
    toast.warning('站点链接需以 http:// 或 https:// 开头')
    return
  }

  const payload = {
    nickname,
    url,
    description: edit.description.trim(),
    avatar: edit.avatar.trim(),
    target: edit.target,
    group: Number(edit.group || 0)
  }
  // 审核、驳回原因与备注仅管理员可写（非管理员会被后端忽略）
  payload.audit = Number(edit.audit)
  // 只有「未通过」才保留原因：改为通过 / 待审核时一并清掉，避免作者看到过期原因
  payload.reason = payload.audit === LINK_AUDIT_REJECT ? edit.reason.trim() : ''
  payload.remark = edit.remark.trim()

  edit.loading = true
  try {
    if (edit.id) {
      await updateLink({ id: edit.id, ...payload })
      toast.success('修改已保存')
    } else {
      await createLink(payload)
      toast.success('友链已创建')
    }
    edit.visible = false
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出（如 XSS 拦截、无权限）
  } finally {
    edit.loading = false
  }
}

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

// ---------- 审核 ----------
// 变更审核状态（驳回时由弹窗把原因传进来）
async function setAudit(item, value, reason = '') {
  busy.value = true
  try {
    await updateLink({ id: Number(item.id), audit: value, reason })
    item.audit = value
    item.reason = reason
    toast.success(AUDIT_TIPS[value] || '操作成功')
    // 状态筛选下该条目可能已不属于当前标签
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

const AUDIT_TIPS = {
  [LINK_AUDIT_PENDING]: '已取消通过，回到待审核',
  [LINK_AUDIT_PASS]: '已通过审核',
  [LINK_AUDIT_REJECT]: '已驳回'
}

// 驳回要先填原因：原因会写进通知（邮件 + 站内信），申请者也能在「我的友链」里看到
function askReject(item) {
  auditDialog.target = { item }
  auditDialog.visible = true
}

function batchReject() {
  if (!selectedIds.value.length) return
  auditDialog.target = { batch: true }
  auditDialog.visible = true
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
      await batchAudit(LINK_AUDIT_REJECT, reason)
      auditDialog.visible = false
    } else if (await setAudit(target.item, LINK_AUDIT_REJECT, reason)) {
      auditDialog.visible = false
    }
  } finally {
    auditDialog.loading = false
  }
}

function closeAuditDialog(visible) {
  auditDialog.visible = visible
  if (!visible) auditDialog.target = null
}

async function batchAudit(value, reason = '') {
  const ids = [...selectedIds.value]
  busy.value = true
  let ok = 0
  for (const id of ids) {
    try {
      await updateLink({ id, audit: value, reason: value === LINK_AUDIT_REJECT ? reason : '' })
      ok++
    } catch {
      // 单条失败不中断，继续处理其它友链
    }
  }
  busy.value = false
  clearSelection()
  toast.success(`操作完成：成功 ${ok} 条，失败 ${ids.length - ok} 条`)
  // 有失败条目或处于状态筛选下，重新拉取保证列表与统计准确
  if (ok !== ids.length || status.value !== 'all') await afterMutation(0)
  else refreshStats()
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
    title: '删除友链',
    message: `确定删除友链「${item.nickname || item.url}」吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeLinks([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreLinks([item.id])
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
    message: `确定彻底删除友链「${item.nickname || item.url}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteLinks([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 条友链吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeLinks(ids)
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
    await restoreLinks(ids)
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
    message: `确定彻底删除选中的 ${ids.length} 条友链吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteLinks(ids)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有友链将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearLinkRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

onMounted(() => {
  loadGroups()
  load()
})
</script>

<style scoped>
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

.link-list {
  display: flex;
  flex-direction: column;
}
.link-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.link-row:last-child {
  border-bottom: none;
}
.link-row:hover {
  background: var(--bg-muted);
}
.link-row.is-selected {
  background: var(--accent-wash);
}

.pick {
  display: flex;
  align-items: center;
  height: 40px;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

.link-avatar {
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
  font-size: 18px;
}
.link-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.link-main {
  flex: 1;
  min-width: 0;
}
.link-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.link-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.state-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.state-chip.is-pass {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.state-chip.is-pending {
  color: var(--warning);
  background: var(--gold-wash);
}
.state-chip.is-reject {
  color: var(--danger);
  background: rgba(217, 84, 77, 0.12);
}
/* 申请人：稍微突出一点，便于审核时核对是谁申请的 */
.meta-text.is-author {
  max-width: 180px;
  font-weight: 500;
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 驳回原因：单行省略，完整内容看 title */
.meta-text.is-reason {
  max-width: 260px;
  color: var(--danger);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 申请人一行：与下面的表单留点距离 */
.edit-owner {
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px dashed var(--border-soft);
}
/* 表单里的行内提示（驳回原因旁的「作者可见」） */
.form-hint-inline {
  margin-left: 4px;
  font-size: 12px;
  font-weight: 400;
  color: var(--text-light);
}
.group-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.id-chip {
  padding: 1px 6px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 3px;
}
.link-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.link-url {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  max-width: 340px;
  color: var(--text-muted);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.link-url:hover {
  color: var(--primary);
  text-decoration: underline;
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.link-desc {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
}

.link-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.link-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.link-actions .btn:disabled {
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
.cover-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.cover-box {
  position: relative;
  width: 96px;
  height: 72px;
  flex-shrink: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
}
.cover-box img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  font-size: 11px;
  color: var(--text-light);
  cursor: pointer;
  transition: color 0.2s;
}
.cover-empty:hover {
  color: var(--primary);
}
.cover-del {
  position: absolute;
  right: 0;
  top: 0;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  line-height: 1;
  cursor: pointer;
}
.cover-fields {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cover-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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
  .link-row {
    flex-wrap: wrap;
  }
  .link-main {
    flex: 1 1 60%;
  }
  .link-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
