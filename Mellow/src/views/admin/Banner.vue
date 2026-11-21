<template>
  <div class="banner-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">轮播管理</h2>
          <p class="block-desc">
            维护首页轮播图：图片、跳转链接与上下线时间
            <template v-if="total > 0"> · 共 {{ total }} 张</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd">
            <i class="bi bi-plus-lg" /> 新建轮播
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
            @click="switchTab(false)"
          >全部轮播</button>
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
          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索标题…"
              aria-label="搜索标题"
              @input="doSearch" />
          </div>
          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 回收站提示 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的轮播不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 批量操作栏（选中后出现） -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 张</span>
        <div class="batch-actions">
          <template v-if="trash">
            <button class="btn btn-sm" :disabled="busy" @click="batchRestore()">
              <i class="bi bi-arrow-counterclockwise" /> 批量恢复
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
              <i class="bi bi-x-octagon" /> 批量彻底删除
            </button>
          </template>
          <template v-else>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 批量删除
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-images'" :text="emptyText" />
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
          <span class="list-head-text">本页 {{ list.length }} 张 · 共 {{ total }} 张</span>
        </div>

        <ul class="banner-list">
          <li
            v-for="item in list"
            :key="item.id"
            class="banner-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择轮播 ${item.title || item.id}`"
                @change="toggleSelect(item.id)" />
            </label>

            <button
              type="button"
              class="banner-cover"
            :title="item.image ? '查看大图' : '未设置图片'"
            :aria-label="`查看轮播 ${item.title || item.id} 的大图`"
            @click="preview(item)"
          >
            <img v-if="item.image" :src="item.image" :alt="item.title || '轮播图'" loading="lazy" />
            <i v-else class="bi bi-image" />
          </button>

          <div class="banner-main">
            <div class="banner-name-row">
              <span class="banner-name">{{ item.title || '未命名轮播' }}</span>
              <span class="state-chip" :class="`is-${stateOf(item).key}`">{{ stateOf(item).label }}</span>
              <span v-if="item.url" class="banner-url">
                <i class="bi bi-link-45deg" /> {{ item.url }}
              </span>
            </div>

            <p v-if="contentPreview(item)" class="banner-desc">{{ contentPreview(item) }}</p>

            <div class="banner-meta">
              <span class="meta-text">#{{ item.id }}</span>
              <span class="meta-text">
                <i class="bi bi-box-arrow-up-right" /> {{ targetLabel(item) }}
              </span>
              <span class="meta-text"><i class="bi bi-calendar-event" /> {{ periodText(item) }}</span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
              <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
            </div>
          </div>

          <div class="banner-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>

            <template v-else>
              <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" @click="openEdit(item)">
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
    </div>

    <!-- 新增 / 编辑轮播 -->
    <AdminFormDialog
      v-model:visible="dialog.visible"
      :title="dialog.id ? '编辑轮播' : '新建轮播'"
      :icon="dialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="dialog.loading"
      confirm-text="保存"
      width="600px"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">轮播标题</label>
        <FormInput v-model="dialog.title" type="text" maxlength="32" placeholder="最多 32 字" />
      </div>

      <div class="form-item">
        <label class="form-label">轮播图片</label>
        <div class="avatar-row">
          <div class="cover-box">
            <img v-if="dialog.image" :src="dialog.image" alt="轮播图预览" loading="lazy" />
            <div v-else class="cover-empty" @click="pickImage">
              <i class="bi bi-image" />
              <span>上传</span>
            </div>
            <button v-if="dialog.image" type="button" class="cover-del" title="移除图片" @click="dialog.image = ''">
              <i class="bi bi-x" />
            </button>
          </div>
          <div class="cover-fields">
            <FormInput v-model="dialog.image" type="text" placeholder="粘贴图片链接，或点击下方按钮上传" />
            <div class="cover-actions">
              <button class="btn btn-sm" type="button" :disabled="uploading" @click="pickImage">
                <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
              </button>
              <button class="btn btn-sm" type="button" title="从附件库选择" @click="showLibrary = true">
                <i class="bi bi-folder2-open" /> 附件库
              </button>
              <button v-if="dialog.image" class="btn btn-sm btn-ghost" type="button" :disabled="uploading" @click="dialog.image = ''">
                移除
              </button>
            </div>
          </div>
          <input ref="fileRef" type="file" accept="image/*" hidden @change="onFileChange" />

          <!-- 附件库：选一张已上传的图片作为轮播图 -->
          <AttachmentLibrary
            v-model:visible="showLibrary"
            title="选择轮播图"
            accept="image"
            @select="onLibrarySelect"
          />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">跳转链接</label>
          <FormInput v-model="dialog.url" type="text" maxlength="256" placeholder="可选，如 /about 或 https://example.com" />
        </div>
        <div class="form-item">
          <label class="form-label">打开方式</label>
          <FormSelect
            v-model="dialog.target"
            :options="TARGET_OPTIONS"
            placeholder="请选择打开方式" />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">开始时间</label>
          <FormInput v-model="dialog.startAt" type="datetime-local" />
          <p class="form-hint">留空表示立即生效</p>
        </div>
        <div class="form-item">
          <label class="form-label">结束时间</label>
          <FormInput v-model="dialog.endAt" type="datetime-local" />
          <p class="form-hint">留空表示长期有效</p>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">说明文字</label>
        <FormTextarea
          v-model="dialog.content"
          rows="3"
          placeholder="可选，部分主题会显示在轮播图上方" />
      </div>

      <div class="form-item">
        <label class="form-label">备注</label>
        <FormInput v-model="dialog.remark" type="text" placeholder="仅后台可见" />
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
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 轮播管理（/admin/banner）
 *
 * 与公告一样字段不多，做成单页 CRUD：全部轮播 + 回收站两个视图，配合新增/编辑弹窗。
 *
 * 后端约束（app/api/controller/banner.go）：
 * - banner/save 会自动分流 create / update，新增与编辑共用一个弹窗；
 * - 允许字段为 title,content,url,image,target,start_time,end_time,remark,json,text；
 * - url / image 走 SanitizeURL（避免 & 被转义），其余文本字段会做 XSS 检测；
 * - start_time / end_time 为 unix 秒，0 表示不限制；判定「生效中」由前端按当前时间计算；
 * - 非管理员删除 / 恢复仅限自己创建的轮播，管理员不受限。
 */
import { ref, reactive, computed, onMounted, watchEffect } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listBanners,
  saveBanner,
  removeBanners,
  forceDeleteBanners,
  restoreBanners,
  clearBannerRecycle
} from '@/api/banner'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { fromNow, formatTime, toLocalInput, fromLocalInput } from '@/utils/time'
import { truncate, debounce } from '@/utils/helper'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 打开方式：后端默认 _blank，前端仅识别 _self 为当前窗口
const TARGET_OPTIONS = [
  { value: '_blank', label: '新窗口打开' },
  { value: '_self', label: '当前窗口打开' }
]

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const keyword = ref('')
// 防抖后的实际搜索词（避免每敲一个字就打一次接口）
const searchKey = ref('')
// 每次加载时刷新，用于计算「生效中 / 未开始 / 已结束」
const nowTs = ref(Math.floor(Date.now() / 1000))

const fileRef = ref(null)
const uploading = ref(false)
const showLibrary = ref(false)

// 从附件库选图：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) dialog.image = url
}

const dialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  title: '',
  image: '',
  url: '',
  target: '_blank',
  startAt: '',
  endAt: '',
  content: '',
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

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的轮播'
  if (trash.value) return '回收站是空的'
  return '还没有轮播图，先新建一张吧'
})

// ---------- 展示辅助 ----------
// 生效状态：0 / 空值表示不限制
function stateOf(item) {
  const start = Number(item?.start_time || 0)
  const end = Number(item?.end_time || 0)
  if (end > 0 && end < nowTs.value) return { key: 'ended', label: '已结束' }
  if (start > 0 && start > nowTs.value) return { key: 'pending', label: '未开始' }
  return { key: 'active', label: '生效中' }
}

function targetLabel(item) {
  return item?.target === '_self' ? '当前窗口' : '新窗口'
}

function periodText(item) {
  const start = Number(item?.start_time || 0)
  const end = Number(item?.end_time || 0)
  if (!start && !end) return '长期有效'
  if (!start) return `至 ${formatTime(end)}`
  if (!end) return `${formatTime(start)} 起`
  return `${formatTime(start)} ~ ${formatTime(end)}`
}

// 内容可能是 HTML，列表里只取纯文本摘要
function contentPreview(item) {
  return truncate(item?.content || '', 100)
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

function preview(item) {
  if (!item?.image) return
  openLightbox([item.image], 0)
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  selectedIds.value = []
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (trash.value) params.onlyTrashed = true
    // 关键词走后端 like 的「字段名|值」格式
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `title|${kw}`
    const res = await listBanners(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    nowTs.value = Math.floor(Date.now() / 1000)
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
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

// ---------- 图片上传 ----------
function pickImage() {
  fileRef.value?.click()
}

async function onFileChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.warning('图片不能超过 5MB')
    return
  }
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')
    dialog.image = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

// ---------- 新增 / 编辑 ----------
function resetDialog() {
  dialog.id = 0
  dialog.title = ''
  dialog.image = ''
  dialog.url = ''
  dialog.target = '_blank'
  dialog.startAt = ''
  dialog.endAt = ''
  dialog.content = ''
  dialog.remark = ''
}

function openAdd() {
  resetDialog()
  dialog.visible = true
}

function openEdit(item) {
  dialog.id = Number(item.id)
  dialog.title = item.title || ''
  dialog.image = item.image || ''
  dialog.url = item.url || ''
  dialog.target = item.target === '_self' ? '_self' : '_blank'
  dialog.startAt = toLocalInput(item.start_time)
  dialog.endAt = toLocalInput(item.end_time)
  dialog.content = item.content || ''
  dialog.remark = item.remark || ''
  dialog.visible = true
}

async function save() {
  const title = dialog.title.trim()
  if (!title) {
    toast.warning('请输入轮播标题')
    return
  }
  if (!dialog.image.trim()) {
    toast.warning('请上传或填写轮播图片')
    return
  }

  const startTime = fromLocalInput(dialog.startAt)
  const endTime = fromLocalInput(dialog.endAt)
  if (startTime > 0 && endTime > 0 && endTime <= startTime) {
    toast.warning('结束时间必须晚于开始时间')
    return
  }

  const payload = {
    title,
    image: dialog.image.trim(),
    url: dialog.url.trim(),
    target: dialog.target,
    start_time: startTime,
    end_time: endTime,
    content: dialog.content.trim(),
    remark: dialog.remark.trim()
  }
  if (dialog.id) payload.id = dialog.id

  dialog.loading = true
  try {
    await saveBanner(payload)
    toast.success(dialog.id ? '修改已保存' : '轮播已创建')
    dialog.visible = false
    await load()
  } catch {
    // 失败提示由请求拦截器统一给出（如内容含恶意代码）
  } finally {
    dialog.loading = false
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
    title: '删除轮播',
    message: `确定删除轮播「${item.title || item.id}」吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeBanners([item.id])
      toast.success('已移入回收站')
      await load()
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreBanners([item.id])
    toast.success('已恢复')
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除轮播「${item.title || item.id}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteBanners([item.id])
      toast.success('已彻底删除')
      await load()
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有轮播将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearBannerRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await load()
    }
  })
}

// ---------- 多选 ----------
const selectedIds = ref([])
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((i) => selectedSet.value.has(i.id))
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
  selectedIds.value = pageAllSelected.value ? [] : list.value.map((i) => i.id)
}

function clearSelection() {
  selectedIds.value = []
}

// ---------- 批量操作 ----------
function askBatchRemove() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 张轮播吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeBanners(ids)
      toast.success(`已删除 ${ids.length} 张轮播`)
      selectedIds.value = []
      await load()
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  try {
    await restoreBanners(ids)
    toast.success(`已恢复 ${ids.length} 张轮播`)
    selectedIds.value = []
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 张轮播吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteBanners(ids)
      toast.success(`已彻底删除 ${ids.length} 张轮播`)
      selectedIds.value = []
      await load()
    }
  })
}

onMounted(load)
</script>

<style scoped>
.banner-admin {
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
.pick input:disabled {
  cursor: not-allowed;
  opacity: 0.4;
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

.banner-list {
  display: flex;
  flex-direction: column;
}
.banner-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.banner-row:last-child {
  border-bottom: none;
}
.banner-row:hover {
  background: var(--bg-muted);
}
.banner-row.is-selected {
  background: var(--accent-wash);
}

.banner-cover {
  position: relative;
  width: 132px;
  height: 74px;
  flex-shrink: 0;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: zoom-in;
  transition: transform 0.2s, border-color 0.2s;
}
.banner-cover img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.banner-cover:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
}
.banner-cover:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 1px;
}

.banner-main {
  flex: 1;
  min-width: 0;
}
.banner-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.banner-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.state-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.state-chip.is-active {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.state-chip.is-pending {
  color: var(--warning);
  background: var(--gold-wash);
}
.state-chip.is-ended {
  color: var(--text-muted);
  background: var(--bg-muted);
}
.banner-url {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  max-width: 260px;
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.banner-desc {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  word-break: break-word;
}
.banner-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.banner-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.banner-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.banner-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 弹窗表单 ---------- */
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

.avatar-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.cover-box {
  position: relative;
  width: 132px;
  height: 74px;
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
  .filter-right {
    width: 100%;
    margin-left: 0;
  }
  .search-box {
    width: 100%;
  }
  .banner-row {
    flex-wrap: wrap;
  }
  .banner-main {
    flex: 1 1 60%;
  }
  .banner-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
