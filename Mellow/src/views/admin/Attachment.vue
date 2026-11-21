<template>
  <div class="attachment-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">
            附件管理
            <span
              v-if="storage.name"
              class="store-badge"
              :class="{ 'is-cloud': storage.driver === 'cos' }"
              :title="`当前启用的存储方式：${storage.name}（新上传的文件会存到这里）`"
            >
              <i :class="storage.driver === 'cos' ? 'bi bi-cloud-check' : 'bi bi-hdd'" />
              {{ storage.name }}
            </span>
          </h2>
          <p class="block-desc">
            管理全站上传附件：预览、重命名、删除与回收站
            <template v-if="total > 0"> · 共 {{ total }} 个</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="uploading" @click="pickFiles">
            <i class="bi bi-cloud-upload" /> {{ uploading ? '上传中...' : '上传附件' }}
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
          <input ref="fileRef" type="file" multiple hidden @change="onFileChange" />
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

    <div class="card card-pad files-card">
      <header class="list-head">
        <div>
          <h2 class="block-title">
            附件列表
            <span v-if="total > 0" class="files-count">{{ total }}</span>
          </h2>
          <p class="block-desc">彻底删除会连同存储文件一起移除，不可恢复</p>
        </div>
      </header>

      <div class="files-layout">
        <!-- 左侧类型导航：取代原来的类型下拉，点一下就能切分类 -->
        <aside class="type-rail" aria-label="按附件类型筛选">
          <button
            v-for="t in typeRail"
            :key="t.value"
            type="button"
            class="type-rail-item"
            :class="{ 'is-active': typeKey === t.value }"
            @click="pickType(t.value)"
          >
            <i :class="t.icon" aria-hidden="true" />
            <span class="type-rail-label">{{ t.label }}</span>
          </button>
        </aside>

        <div class="files-main">

      <!-- 工具栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash }"
            @click="switchTrash(false)"
          >全部附件</button>
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
            v-model="sortKey"
            :options="sortOptions"
            icon="bi bi-sort-down"
            placeholder="排序方式"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索文件名…"
              aria-label="搜索文件名"
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
        <span><i class="bi bi-trash3" /> 回收站内的附件已不参与统计，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-paperclip'" :text="emptyText" />
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
          <span v-if="selectedIds.length" class="list-head-sel">已选 {{ selectedIds.length }} 个</span>
          <span class="list-head-tip"><i class="bi bi-info-circle" aria-hidden="true" /> 悬停可复制、重命名或删除</span>
        </div>

        <ul class="attachment-grid">
          <li
            v-for="item in list"
            :key="item.id"
            class="attachment-card"
            :class="{ 'is-selected': isSelected(item.id) }"
            @click="openPreview(item)"
          >
            <label class="pick card-pick" :title="isSelected(item.id) ? '取消选择' : '选择'" @click.stop>
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择附件 ${item.original_name}`"
                @change="toggleSelect(item.id)" />
            </label>

            <div
              class="attachment-thumb"
              role="button"
              tabindex="0"
              :title="`预览 ${item.original_name}`"
              :aria-label="`预览 ${item.original_name}`"
              @keydown.enter.prevent="openPreview(item)"
            >
              <img
                v-if="isPreviewableImage(item)"
                :src="item.full_url"
                :alt="item.original_name"
                loading="lazy"
              />
              <span v-else class="attachment-filetype" :class="`is-${attachmentTypeOf(item)}`">
                <i :class="attachmentIcon(item)" />
                <em>{{ extText(item) }}</em>
              </span>

              <!-- 悬浮操作：复制 / 重命名 / 删除（回收站内为恢复 / 彻底删除） -->
              <div class="thumb-actions" @click.stop>
                <button class="thumb-action" title="复制链接" aria-label="复制链接" @click="copyUrl(item)">
                  <i class="bi bi-clipboard" />
                </button>
                <template v-if="trash">
                  <button class="thumb-action" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                    <i class="bi bi-arrow-counterclockwise" />
                  </button>
                  <button class="thumb-action is-danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                    <i class="bi bi-x-octagon" />
                  </button>
                </template>
                <template v-else>
                  <button class="thumb-action" title="重命名" aria-label="重命名" :disabled="busy" @click="openRename(item)">
                    <i class="bi bi-pencil" />
                  </button>
                  <button class="thumb-action is-danger" title="删除" aria-label="删除" :disabled="busy" @click="askRemove(item)">
                    <i class="bi bi-trash" />
                  </button>
                </template>
              </div>
            </div>

            <div class="attachment-name" :title="item.original_name">{{ item.original_name || '未命名' }}</div>
            <div class="attachment-meta" :title="`${timeText(item)} · ${uploaderTitle(item)}`">
              <span>{{ formatFileSize(item.file_size) }}</span>
              <span class="attachment-meta-sep" aria-hidden="true" />
              <span>{{ attachmentTypeLabel(item) }}</span>
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
      </div>
    </div>

    <!-- 预览 / 详情 -->
    <AdminFormDialog
      v-model:visible="preview.visible"
      :title="preview.title"
      icon="bi bi-eye"
      :loading="false"
      confirm-text="关闭"
      width="560px"
      @confirm="preview.visible = false"
    >
      <div v-if="preview.video" class="preview-video">
        <video :src="preview.video" controls preload="metadata" playsinline>您的浏览器不支持视频播放</video>
      </div>
      <div v-else-if="preview.image" class="preview-image">
        <img :src="preview.image" :alt="preview.title" loading="lazy" />
      </div>
      <div v-else class="preview-icon">
        <i :class="preview.icon" />
        <span>{{ preview.ext }}</span>
      </div>

      <dl class="preview-meta">
        <div class="preview-row">
          <dt>文件名</dt>
          <dd>{{ preview.name }}</dd>
        </div>
        <div class="preview-row">
          <dt>大小</dt>
          <dd>{{ preview.size }}</dd>
        </div>
        <div class="preview-row">
          <dt>类型</dt>
          <dd>{{ preview.mime || '—' }}</dd>
        </div>
        <div class="preview-row">
          <dt>存储</dt>
          <dd>{{ preview.driver || '—' }}</dd>
        </div>
        <div class="preview-row">
          <dt>上传者</dt>
          <dd>{{ preview.uploader || '—' }}</dd>
        </div>
        <div class="preview-row">
          <dt>上传时间</dt>
          <dd>{{ preview.time }}</dd>
        </div>
        <div class="preview-row">
          <dt>访问地址</dt>
          <dd class="preview-url">
            <a :href="preview.url" target="_blank" rel="noopener noreferrer">{{ preview.url }}</a>
            <button class="btn btn-ghost btn-sm" title="复制链接" aria-label="复制链接" @click="copyUrl(preview.raw)">
              <i class="bi bi-clipboard" />
            </button>
          </dd>
        </div>
      </dl>
    </AdminFormDialog>

    <!-- 重命名 -->
    <AdminFormDialog
      v-model:visible="rename.visible"
      title="重命名附件"
      icon="bi bi-pencil-square"
      :loading="rename.loading"
      confirm-text="保存"
      @confirm="saveRename"
    >
      <div class="form-item">
        <label class="form-label">文件名</label>
        <FormInput v-model="rename.name" type="text" maxlength="256" placeholder="必填" />
        <p class="form-hint">仅修改展示用的原始文件名，不影响实际存储与访问地址</p>
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
 * 附件管理（/admin/attachment）
 *
 * 附件列表（网格）、预览与详情、重命名、删除与回收站。
 *
 * 后端约束（app/api/controller/attachment.go、app/model/attachment.go）：
 * - attachment/all 与 count：非管理员只能查到 uploader_id 为自己的附件，
 *   后台依赖管理员身份才能管理全站附件；count 支持 onlyTrashed；
 * - attachment/all 的每行会额外补 uploader_account / uploader_name（昵称，为空回退账号），
 *   列表展示昵称而不是 uploader_id 数字；
 * - attachment/update 允许字段只有 original_name / target_type / target_id，
 *   非管理员改他人附件返回「无权限！」；
 * - remove 是软删除（不删存储文件）；delete / clear **仅管理员**，
 *   会连同存储文件一起物理删除，不可恢复；
 * - remove / restore 返回 { success_ids, failed_ids, errors }，部分失败时 HTTP 码为 207。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listAttachmentsAdmin,
  countAttachments,
  sumAttachmentSize,
  updateAttachment,
  removeAttachments,
  forceDeleteAttachments,
  restoreAttachments,
  clearAttachmentRecycle,
  uploadAttachments,
  getAttachmentStorage
} from '@/api/attachment'
import {
  ATTACHMENT_FIELD,
  ATTACHMENT_TYPE_OPTIONS,
  attachmentIcon,
  attachmentTypeLabel,
  attachmentTypeOf,
  attachmentWhereJSON,
  isPreviewableImage,
  isPreviewableVideo,
  formatFileSize,
  normalizeExt,
  sanitizeAttachmentKeyword
} from '@/utils/attachment'
import { fromNow, formatTime } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 排序白名单：后端 order 未做校验，必须由前端限定取值
const sortOptions = [
  { value: 'create_time desc', label: '最新上传' },
  { value: 'create_time asc', label: '最早上传' },
  { value: 'file_size desc', label: '体积最大' },
  { value: 'file_size asc', label: '体积最小' },
  { value: 'id desc', label: 'ID 倒序' }
]

// ===== 列表 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const typeKey = ref('')
const sortKey = ref(sortOptions[0].value)
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

// 左侧类型导航：直接复用类型选项，图标按类型区分
const TYPE_ICONS = {
  '': 'bi bi-collection',
  image: 'bi bi-file-earmark-image',
  video: 'bi bi-file-earmark-play',
  audio: 'bi bi-file-earmark-music',
  doc: 'bi bi-file-earmark-text',
  archive: 'bi bi-file-earmark-zip',
  other: 'bi bi-file-earmark'
}

const typeRail = ATTACHMENT_TYPE_OPTIONS.map((item) => ({
  value: item.value,
  label: item.value ? item.label : '全部',
  icon: TYPE_ICONS[item.value] || TYPE_ICONS.other
}))

/** 左侧导航切换类型：回到第一页重新拉取 */
function pickType(value) {
  if (typeKey.value === value) return
  typeKey.value = value
  reload()
}

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ total: 0, size: 0, image: 0, trash: 0 })

// 当前启用的存储方式（后端读 config/storage.toml 的 default，未知值回退本地存储）
const storage = reactive({ driver: '', name: '' })

// ===== 上传 =====
const fileRef = ref(null)
const uploading = ref(false)

// ===== 弹窗 =====
const preview = reactive({
  visible: false,
  title: '',
  image: '',
  video: '',
  icon: '',
  ext: '',
  name: '',
  size: '',
  mime: '',
  driver: '',
  uploader: '',
  time: '',
  url: '',
  raw: null
})

const rename = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: ''
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
  { label: '附件总数', value: stats.total, icon: 'bi bi-paperclip', color: 'var(--primary)' },
  { label: '占用空间', value: formatFileSize(stats.size), icon: 'bi bi-hdd', color: 'var(--success)' },
  { label: '图片数量', value: stats.image, icon: 'bi bi-image', color: '#0ea5e9' },
  { label: '回收站', value: stats.trash, icon: 'bi bi-trash3', color: 'var(--text-muted)' }
])

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的附件'
  if (trash.value) return '回收站是空的'
  return '还没有附件，先上传一个吧'
})

// ---------- 展示辅助 ----------
function extText(item) {
  return normalizeExt(item?.file_ext).toUpperCase() || 'FILE'
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `上传于 ${fromNow(item.create_time)}`
}

// 上传者显示名：后端 attachment/all 已按 uploader_id 批量补好昵称（uploader_name，
// 昵称为空时后端回退成账号 / 「用户 #id」），这里只做兜底展示
function uploaderName(item) {
  if (item?.uploader_name) return item.uploader_name
  return item?.uploader_id ? `#${item.uploader_id}` : '—'
}

// 悬浮提示：把昵称 / 账号 / 用户 ID 都列出来，便于管理员核对是哪个账号上传的
function uploaderTitle(item) {
  const lines = [`上传者：${uploaderName(item)}`]
  if (item.uploader_account) lines.push(`账号：${item.uploader_account}`)
  if (item.uploader_id) lines.push(`用户 ID：${item.uploader_id}`)
  return lines.join('\n')
}

async function copyUrl(item) {
  const url = typeof item === 'string' ? item : item?.full_url
  if (!url) return
  try {
    await navigator.clipboard.writeText(url)
    toast.success('链接已复制')
  } catch {
    toast.info('复制失败，请手动复制')
  }
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [totalRes, sizeRes, imageRes, trashRes] = await Promise.all([
      countAttachments(),
      sumAttachmentSize(),
      countAttachments({ where: attachmentWhereJSON('image') }),
      countAttachments({ onlyTrashed: true })
    ])
    stats.total = Number(totalRes?.data || 0)
    stats.size = Number(sizeRes?.data?.file_size || 0)
    stats.image = Number(imageRes?.data || 0)
    stats.trash = Number(trashRes?.data || 0)
  } catch {
    /* 统计失败不阻塞列表 */
  } finally {
    loadingStats.value = false
  }
}

// 存储方式只在挂载时取一次：它来自配置文件，页面停留期间不会变；
// 取不到就不显示徽章（不影响其余功能）
async function loadStorage() {
  try {
    const res = await getAttachmentStorage()
    storage.driver = res?.data?.driver || ''
    storage.name = res?.data?.name || ''
  } catch {
    /* 忽略：请求失败由拦截器统一提示 */
  }
}

// ---------- 列表 ----------
async function load() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      limit: pageSize.value,
      field: ATTACHMENT_FIELD,
      order: trash.value ? 'delete_time desc' : sortKey.value
    }
    if (trash.value) params.onlyTrashed = true

    const where = attachmentWhereJSON(typeKey.value)
    if (where) params.where = where

    const kw = sanitizeAttachmentKeyword(searchKey.value)
    if (kw) params.like = `original_name|${kw}`

    const res = await listAttachmentsAdmin(params)
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

// ---------- 上传 ----------
function pickFiles() {
  fileRef.value?.click()
}

async function onFileChange(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  if (!files.length) return

  uploading.value = true
  try {
    const fd = new FormData()
    files.forEach((file) => fd.append('files', file))
    const res = await uploadAttachments(fd)
    const results = res?.data?.results || []
    toast.success(`上传完成：成功 ${results.length} 个`)
    await Promise.all([load(), loadStats()])
  } catch {
    // 失败提示由请求拦截器统一给出（如类型不允许、体积超限）
  } finally {
    uploading.value = false
  }
}

// ---------- 预览 ----------
function openPreview(item) {
  preview.title = item.original_name || '附件详情'
  preview.image = isPreviewableImage(item) ? item.full_url : ''
  preview.video = isPreviewableVideo(item) ? item.full_url : ''
  preview.icon = attachmentIcon(item)
  preview.ext = extText(item)
  preview.name = item.original_name || '—'
  preview.size = formatFileSize(item.file_size)
  preview.mime = item.mime_type || ''
  preview.driver = item.storage_driver || ''
  preview.uploader = uploaderName(item)
  preview.time = formatTime(item.create_time)
  preview.url = item.full_url || ''
  preview.raw = item
  preview.visible = true
}

// ---------- 重命名 ----------
function openRename(item) {
  rename.id = Number(item.id)
  rename.name = item.original_name || ''
  rename.visible = true
}

async function saveRename() {
  const name = rename.name.trim()
  if (!name) {
    toast.warning('请输入文件名')
    return
  }
  rename.loading = true
  try {
    await updateAttachment({ id: rename.id, original_name: name })
    toast.success('已重命名')
    rename.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    rename.loading = false
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

// remove / restore 返回成功与失败明细，统一给出提示
function summary(res, okText) {
  const success = res?.data?.success_ids || []
  const failed = res?.data?.failed_ids || []
  if (failed.length) {
    toast.warning(`${okText}：成功 ${success.length} 个，失败 ${failed.length} 个`)
  } else {
    toast.success(`${okText}：${success.length} 个`)
  }
}

async function afterMutation(removedCount = 0) {
  if (removedCount > 0 && list.value.length <= removedCount && page.value > 1) {
    page.value -= 1
  }
  await load()
  loadStats()
}

function askRemove(item) {
  openConfirm({
    title: '删除附件',
    message: `确定删除附件「${item.original_name}」吗？删除后可在回收站找回（不会删除存储文件）。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      const res = await removeAttachments([item.id])
      summary(res, '已删除')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    const res = await restoreAttachments([item.id])
    summary(res, '已恢复')
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
    message: `确定彻底删除附件「${item.original_name}」吗？存储文件会一并删除，此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAttachments([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个附件吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      const res = await removeAttachments(ids)
      summary(res, '已删除')
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    const res = await restoreAttachments(ids)
    summary(res, '已恢复')
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
    message: `确定彻底删除选中的 ${ids.length} 个附件吗？存储文件会一并删除，不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteAttachments(ids)
      toast.success(`已彻底删除 ${ids.length} 个`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有附件的存储文件会被一并删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearAttachmentRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

onMounted(() => {
  load()
  loadStats()
  loadStorage()
})
</script>

<style scoped>
.attachment-admin {
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
/* 标题后的总数徽章 */
.files-count {
  display: inline-block;
  margin-left: 6px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 500;
  line-height: 1.8;
  vertical-align: middle;
  font-variant-numeric: tabular-nums;
}

/* ---------- 文件管理器布局：左类型导航 + 右主区 ---------- */
.files-layout {
  display: flex;
  align-items: flex-start;
  gap: 18px;
}
.type-rail {
  flex: 0 0 132px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px;
  background: var(--bg);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
}
.type-rail-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 34px;
  padding: 0 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}
.type-rail-item .bi {
  flex-shrink: 0;
  font-size: 15px;
  color: var(--text-light);
  transition: color var(--dur-fast) var(--ease);
}
.type-rail-item:hover {
  background: var(--bg-card);
  color: var(--text);
}
.type-rail-item:hover .bi {
  color: var(--primary);
}
.type-rail-item.is-active {
  background: var(--bg-card);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}
.type-rail-item.is-active .bi {
  color: var(--primary);
}
.type-rail-label {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.files-main {
  flex: 1 1 auto;
  min-width: 0;
}

/* 标题右侧的存储方式徽章 */
.store-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 12px;
  font-weight: 500;
  line-height: 1.6;
  vertical-align: middle;
}
.store-badge.is-cloud {
  border-color: var(--primary-soft);
  background: var(--primary-soft);
  color: var(--primary-deep);
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

/* ---------- 网格 ---------- */
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
  padding: 10px 4px 6px;
}
.list-head-text {
  font-size: 12px;
  color: var(--text-muted);
}
.list-head-sel {
  font-size: 12px;
  color: var(--primary);
}
.list-head-tip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  font-size: 12px;
  color: var(--text-light);
}

/* 文件管理器风格：小格子 + 大类型图标 + 居中文件名 */
.attachment-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(108px, 1fr));
  gap: 6px;
  margin: 4px 0 0;
  padding: 0;
  list-style: none;
}
.attachment-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 10px 6px 8px;
  border: 1px solid transparent;
  border-radius: var(--radius);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.attachment-card:hover {
  background: var(--bg-muted);
}
.attachment-card.is-selected {
  background: var(--accent-wash);
  border-color: var(--accent-soft);
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
.card-pick {
  position: absolute;
  left: 8px;
  top: 8px;
  z-index: 3;
  padding: 2px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.9);
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease);
}
.attachment-card:hover .card-pick,
.attachment-card.is-selected .card-pick,
.card-pick:focus-within {
  opacity: 1;
}

/* 缩略图 / 文件图标区 */
.attachment-thumb {
  position: relative;
  width: 100%;
  aspect-ratio: 1 / 1;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  cursor: pointer;
  outline: none;
}
.attachment-card:hover .attachment-thumb {
  border-color: var(--border);
}
.attachment-thumb:focus-visible {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.attachment-thumb img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.attachment-filetype {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  color: var(--text-light);
}
.attachment-filetype .bi {
  font-size: 38px;
  line-height: 1;
}
.attachment-filetype em {
  font-size: 10px;
  font-style: normal;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
/* 类型配色：图标颜色即文件类型，和文件名下方的类型文字对应 */
.attachment-filetype.is-image .bi {
  color: #0ea5e9;
}
.attachment-filetype.is-video .bi {
  color: #8b5cf6;
}
.attachment-filetype.is-audio .bi {
  color: #d946ef;
}
.attachment-filetype.is-doc .bi {
  color: #3b82f6;
}
.attachment-filetype.is-archive .bi {
  color: #eab308;
}
.attachment-filetype.is-other .bi {
  color: #94a3b8;
}

/* 悬浮操作条：不占布局，叠在缩略图底部 */
.thumb-actions {
  position: absolute;
  inset: auto 0 0 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 4px;
  background: linear-gradient(180deg, rgba(32, 28, 22, 0), rgba(32, 28, 22, 0.62));
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease);
}
.attachment-card:hover .thumb-actions,
.attachment-card:focus-within .thumb-actions {
  opacity: 1;
}
.thumb-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.92);
  color: var(--text-soft);
  font-size: 12px;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.thumb-action:hover:not(:disabled) {
  background: #fff;
  color: var(--primary);
}
.thumb-action.is-danger:hover:not(:disabled) {
  color: var(--danger);
}
.thumb-action:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.attachment-name {
  width: 100%;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text);
  text-align: center;
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.attachment-meta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  width: 100%;
  font-size: 11px;
  color: var(--text-light);
  white-space: nowrap;
  overflow: hidden;
}
.attachment-meta-sep {
  width: 2px;
  height: 2px;
  flex-shrink: 0;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
}

/* ---------- 弹窗 ---------- */
.preview-image,
.preview-video {
  display: flex;
  align-items: center;
  justify-content: center;
  max-height: 320px;
  overflow: hidden;
  border-radius: var(--radius);
  background: var(--bg-muted);
}
.preview-image img {
  max-width: 100%;
  max-height: 320px;
  object-fit: contain;
}
.preview-video {
  background: #000;
}
.preview-video video {
  width: 100%;
  max-height: 320px;
  border-radius: var(--radius);
  background: #000;
}
.preview-icon {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 24px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.preview-icon .bi {
  font-size: 44px;
}
.preview-icon span {
  font-size: 12px;
  letter-spacing: 0.06em;
}

.preview-meta {
  margin: 14px 0 0;
}
.preview-row {
  display: flex;
  gap: 12px;
  padding: 6px 0;
  font-size: 13px;
}
.preview-row dt {
  width: 72px;
  flex-shrink: 0;
  color: var(--text-muted);
}
.preview-row dd {
  margin: 0;
  min-width: 0;
  color: var(--text-soft);
  word-break: break-all;
}
.preview-url {
  display: flex;
  align-items: center;
  gap: 6px;
}
.preview-url a {
  min-width: 0;
  color: var(--primary);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.preview-url a:hover {
  text-decoration: underline;
}

.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}

/* 触屏没有 hover：勾选框与操作条常驻，否则无法重命名 / 删除 */
@media (hover: none) {
  .card-pick,
  .thumb-actions {
    opacity: 1;
  }
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
  /* 左侧导航改为横向滚动条 */
  .files-layout {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }
  .type-rail {
    flex: 0 0 auto;
    flex-direction: row;
    gap: 2px;
    padding: 5px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .type-rail::-webkit-scrollbar {
    display: none;
  }
  .type-rail-item {
    width: auto;
    height: 30px;
    padding: 0 10px;
    flex-shrink: 0;
  }
  .attachment-grid {
    grid-template-columns: repeat(auto-fill, minmax(92px, 1fr));
  }
  .attachment-filetype .bi {
    font-size: 32px;
  }
  .list-head-tip {
    display: none;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
