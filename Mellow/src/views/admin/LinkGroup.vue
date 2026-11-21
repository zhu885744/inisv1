<template>
  <div class="link-group-admin">
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">友链分组</h2>
          <p class="block-desc">
            对友链归类展示；删除分组后，其下友链会回落到「默认分组」
            <template v-if="total > 0"> · 共 {{ total }} 个</template>
          </p>
        </div>
        <div class="head-actions">
          <button
            class="btn btn-ghost btn-sm"
            :disabled="loading"
            title="刷新"
            aria-label="刷新"
            @click="load()"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
            <i class="bi bi-plus-lg" /> 新建分组
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
        <span><i class="bi bi-trash3" /> 回收站内的分组不再出现在友链页，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-collection'" :text="emptyText" />
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

        <ul class="group-list">
          <li v-for="item in list" :key="item.id" class="group-row" :class="{ 'is-selected': isSelected(item.id) }">
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择分组 ${item.name}`"
                @change="toggleSelect(item.id)" />
            </label>

            <div class="group-avatar">
              <img v-if="item.avatar" :src="item.avatar" :alt="item.name" loading="lazy" />
              <i v-else class="bi bi-collection" />
            </div>

            <div class="group-main">
              <div class="group-name-row">
                <span class="group-name">{{ item.name || '未命名分组' }}</span>
                <span class="id-chip">#{{ item.id }}</span>
                <span v-if="linkCountMap[item.id] != null" class="count-chip">
                  <i class="bi bi-link-45deg" /> {{ linkCountMap[item.id] }} 条友链
                </span>
              </div>
              <p v-if="item.description" class="group-desc">{{ item.description }}</p>
              <div class="group-meta">
                <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
              </div>
            </div>

            <div class="group-actions">
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
    </section>

    <!-- 新建 / 编辑分组 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑分组 #${edit.id}` : '新建分组'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">分组名称</label>
        <FormInput v-model="edit.name" type="text" maxlength="32" placeholder="必填" />
      </div>

      <div class="form-item">
        <label class="form-label">分组描述</label>
        <FormTextarea v-model="edit.description" rows="2" placeholder="可选" />
      </div>

      <div class="form-item">
        <label class="form-label">分组图标</label>
        <div class="cover-row">
          <div class="cover-box">
            <img v-if="edit.avatar" :src="edit.avatar" alt="图标预览" loading="lazy" />
            <div v-else class="cover-empty" @click="pickAvatar">
              <i class="bi bi-image" />
              <span>上传</span>
            </div>
            <button v-if="edit.avatar" type="button" class="cover-del" title="移除图标" @click="edit.avatar = ''">
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

          <!-- 附件库：选一张已上传的图片作为分组图标 -->
          <AttachmentLibrary
            v-model:visible="showLibrary"
            title="选择分组图标"
            accept="image"
            @select="onLibrarySelect"
          />
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
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 友链分组（/admin/links/group）
 *
 * 分组的增删改查与回收站；每个分组额外展示其下的友链数量（按 group 统计）。
 *
 * 后端约束（app/api/controller/links-group.go、app/model/links-group.go）：
 * - 允许字段只有 name/description/avatar/json/text，没有排序字段，列表按 id 升序；
 * - save 自动分流 create / update（有 id 走 update）；
 * - links-group/count 不支持 onlyTrashed，回收站数量用列表接口的 count；
 * - 删除分组是软删除，友链表里的 group 值不会自动改写，
 *   后端在查询友链时若发现分组已不存在会回落为「默认分组」。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listLinkGroupsAdmin,
  saveLinkGroup,
  removeLinkGroups,
  forceDeleteLinkGroups,
  restoreLinkGroups,
  clearLinkGroupRecycle,
  countLinks
} from '@/api/links'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { LINK_GROUP_FIELD } from '@/utils/link'
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
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])
// 分组 -> 友链数量（仅统计当前页展示的分组）
const linkCountMap = ref({})

// ===== 弹窗 =====
const avatarRef = ref(null)
const uploading = ref(false)
const showLibrary = ref(false)

// 从附件库选分组图标：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) edit.avatar = url
}

const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  description: '',
  avatar: ''
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
  if (searchKey.value) return '没有匹配的分组'
  if (trash.value) return '回收站是空的'
  return '还没有分组，先新建一个吧'
})

// ---------- 展示辅助 ----------
function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `创建于 ${fromNow(item.create_time)}`
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: pageSize.value,
      field: LINK_GROUP_FIELD,
      order: trash.value ? 'delete_time desc' : 'id asc'
    }
    if (trash.value) params.onlyTrashed = true
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `name|${kw}`

    const res = await listLinkGroupsAdmin(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
    await loadLinkCounts()
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 统计每个分组下的友链数量（回收站里的分组不必统计）
async function loadLinkCounts() {
  linkCountMap.value = {}
  if (trash.value || !list.value.length) return

  const entries = await Promise.all(
    list.value.map(async (g) => {
      try {
        const res = await countLinks({ where: JSON.stringify({ group: Number(g.id) }) })
        return [Number(g.id), Number(res?.data || 0)]
      } catch {
        return [Number(g.id), null]
      }
    })
  )
  linkCountMap.value = Object.fromEntries(entries.filter(([, v]) => v != null))
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

// ---------- 图标上传 ----------
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
  edit.name = ''
  edit.description = ''
  edit.avatar = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.name = item.name || ''
  edit.description = item.description || ''
  edit.avatar = item.avatar || ''
  edit.visible = true
}

async function save() {
  const name = edit.name.trim()
  if (!name) {
    toast.warning('请输入分组名称')
    return
  }

  const payload = { name, description: edit.description.trim(), avatar: edit.avatar.trim() }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveLinkGroup(payload)
    toast.success(edit.id ? '修改已保存' : '分组已创建')
    edit.visible = false
    await afterMutation(0)
  } catch {
    /* 拦截器已提示 */
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
    message: `确定删除分组「${item.name}」吗？其下友链会回落到「默认分组」，可在回收站找回分组。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeLinkGroups([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreLinkGroups([item.id])
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
      await forceDeleteLinkGroups([item.id])
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
      await removeLinkGroups(ids)
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
    await restoreLinkGroups(ids)
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
      await forceDeleteLinkGroups(ids)
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
      await clearLinkGroupRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

onMounted(load)
</script>

<style scoped>
.link-group-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-head {
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

.group-list {
  display: flex;
  flex-direction: column;
}
.group-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.group-row:last-child {
  border-bottom: none;
}
.group-row:hover {
  background: var(--bg-muted);
}
.group-row.is-selected {
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

.group-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 18px;
}
.group-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.group-main {
  flex: 1;
  min-width: 0;
}
.group-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.group-name {
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
.count-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.group-desc {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
}
.group-meta {
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

.group-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.group-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.group-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 弹窗 ---------- */
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
  .group-row {
    flex-wrap: wrap;
  }
  .group-main {
    flex: 1 1 60%;
  }
  .group-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
