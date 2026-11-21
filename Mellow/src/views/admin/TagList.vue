<template>
  <div class="tag-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">话题管理</h2>
          <p class="block-desc">
            维护文章话题：名称、描述与话题图
            <template v-if="total > 0"> · 共 {{ total }} 个</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd">
            <i class="bi bi-plus-lg" /> 新建话题
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
          >全部话题</button>
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
              placeholder="搜索名称…"
              aria-label="搜索名称"
              @input="doSearch" />
          </div>
          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 回收站提示 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的话题不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 批量操作栏（选中后出现） -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 个</span>
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
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-tags'" :text="emptyText" />
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
          <span class="list-head-text">本页 {{ list.length }} 个 · 共 {{ total }} 个</span>
        </div>

        <ul class="tag-list">
          <li
            v-for="item in list"
            :key="item.id"
            class="tag-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择话题 ${item.name || item.id}`"
                @change="toggleSelect(item.id)" />
            </label>

            <div class="tag-avatar">
            <img v-if="item.avatar" :src="item.avatar" :alt="item.name" loading="lazy" />
            <i v-else class="bi bi-hash" />
          </div>

          <div class="tag-main">
            <div class="tag-name-row">
              <span class="tag-name">#{{ item.name || '未命名话题' }}</span>
              <span class="count-chip" :title="countTitle(item)">
                <i class="bi bi-file-earmark-text" /> {{ countText(item) }}
              </span>
            </div>
            <p v-if="item.description" class="tag-desc">{{ item.description }}</p>
            <div class="tag-meta">
              <span class="meta-text">#{{ item.id }}</span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
            </div>
          </div>

          <div class="tag-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>

            <template v-else>
              <router-link
                :to="`/topic/${item.id}`"
                class="btn btn-ghost btn-sm"
                title="前台查看"
                aria-label="前台查看"
              >
                <i class="bi bi-eye" />
              </router-link>
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

    <!-- 新增 / 编辑话题 -->
    <AdminFormDialog
      v-model:visible="dialog.visible"
      :title="dialog.id ? '编辑话题' : '新建话题'"
      :icon="dialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="dialog.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">话题名称</label>
        <FormInput v-model="dialog.name" type="text" maxlength="32" placeholder="最多 32 字" />
      </div>

      <div class="form-item">
        <label class="form-label">话题描述</label>
        <FormTextarea
          v-model="dialog.description"
          rows="3"
          placeholder="可选，用于前台话题页展示" />
      </div>

      <div class="form-item">
        <label class="form-label">话题主持人（用户 ID）</label>
        <FormInput
          v-model.number="dialog.moderator_uid"
          type="number"
          min="0"
          placeholder="0 表示暂无主持人" />
        <p class="form-hint">
          主持人可在前台话题页编辑该话题的简介与封面；留 0 表示不设置
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">精选话题</label>
        <FormSelect v-model="dialog.recommend" :options="RECOMMEND_OPTIONS" />
      </div>

      <div class="form-item">
        <label class="form-label">话题图</label>
        <div class="avatar-row">
          <div class="cover-box">
            <img v-if="dialog.avatar" :src="dialog.avatar" alt="话题图预览" loading="lazy" />
            <div v-else class="cover-empty" @click="pickImage">
              <i class="bi bi-image" />
              <span>上传</span>
            </div>
            <button v-if="dialog.avatar" type="button" class="cover-del" title="移除话题图" @click="dialog.avatar = ''">
              <i class="bi bi-x" />
            </button>
          </div>
          <div class="cover-fields">
            <FormInput v-model="dialog.avatar" type="text" placeholder="粘贴图片链接，或点击下方按钮上传" />
            <div class="cover-actions">
              <button class="btn btn-sm" type="button" :disabled="uploading" @click="pickImage">
                <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
              </button>
              <button class="btn btn-sm" type="button" title="从附件库选择" @click="showLibrary = true">
                <i class="bi bi-folder2-open" /> 附件库
              </button>
              <button v-if="dialog.avatar" class="btn btn-sm btn-ghost" type="button" :disabled="uploading" @click="dialog.avatar = ''">
                移除
              </button>
            </div>
          </div>
          <input ref="fileRef" type="file" accept="image/*" hidden @change="onFileChange" />

          <!-- 附件库：选一张已上传的图片作为话题图 -->
          <AttachmentLibrary
            v-model:visible="showLibrary"
            title="选择话题图"
            accept="image"
            @select="onLibrarySelect"
          />
        </div>
        <p class="form-hint">可选，留空时前台用默认样式展示</p>
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
 * 话题管理（/admin/tags）
 *
 * 话题字段很少（名称 / 描述 / 话题图），做成单页 CRUD：全部话题 + 回收站两个视图。
 *
 * 后端约束（app/api/controller/tags.go）：
 * - tags/all 会额外注入 article_count（该话题被「已审核且未删除」文章引用的数量）；
 *   传 field 时必须带上它，否则会被字段过滤掉；
 * - tags/count 不支持 onlyTrashed，回收站数量不另做统计；
 * - tags/save 自动分流 create / update；允许字段为 name,description,avatar,json,text；
 * - avatar 走 SanitizeURL，name / description / avatar / text 会做 XSS 检测；
 * - 话题没有唯一索引，同名话题可以重复创建（后端不做重名校验）。
 */
import { ref, reactive, computed, onMounted, watchEffect } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listTagsAdmin,
  saveTag,
  removeTags,
  forceDeleteTags,
  restoreTags,
  clearTagRecycle
} from '@/api/tags'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
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
// 防抖后的实际搜索词（避免每敲一个字就打一次接口）
const searchKey = ref('')

// 下拉选项（统一走 SelectMenu，不用浏览器原生 select）
const RECOMMEND_OPTIONS = [
  { value: 0, label: '普通' },
  { value: 1, label: '精选（话题广场可筛选展示）' }
]

const fileRef = ref(null)
const uploading = ref(false)
const showLibrary = ref(false)

// 从附件库选话题图：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) dialog.avatar = url
}

const dialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  description: '',
  avatar: '',
  // 话题（Hashtag）扩展字段：话题即 tags 表，见 Mellow/src/views/topic/*
  moderator_uid: 0,
  recommend: 0
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
  if (searchKey.value) return '没有匹配的话题'
  if (trash.value) return '回收站是空的'
  return '还没有话题，先新建一个吧'
})

// ---------- 展示辅助 ----------
function countText(item) {
  // article_count 由后端注入；回收站数据可能没有该字段，用「—」表示未知
  const value = Number(item?.article_count)
  return Number.isFinite(value) ? value : '—'
}

function countTitle(item) {
  const value = Number(item?.article_count)
  return Number.isFinite(value) ? `被 ${value} 篇已发布文章引用` : '暂无引用统计'
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
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
    if (kw) params.like = `name|${kw}`
    const res = await listTagsAdmin(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
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
    dialog.avatar = url
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
  dialog.name = ''
  dialog.description = ''
  dialog.avatar = ''
  dialog.moderator_uid = 0
  dialog.recommend = 0
}

function openAdd() {
  resetDialog()
  dialog.visible = true
}

function openEdit(item) {
  dialog.id = Number(item.id)
  dialog.name = item.name || ''
  dialog.description = item.description || ''
  dialog.avatar = item.avatar || ''
  dialog.moderator_uid = Number(item.moderator_uid) || 0
  dialog.recommend = Number(item.recommend) || 0
  dialog.visible = true
}

async function save() {
  const name = dialog.name.trim()
  if (!name) {
    toast.warning('请输入话题名称')
    return
  }

  const payload = {
    name,
    description: dialog.description.trim(),
    avatar: dialog.avatar.trim(),
    moderator_uid: Number(dialog.moderator_uid) || 0,
    recommend: Number(dialog.recommend) ? 1 : 0
  }
  if (dialog.id) payload.id = dialog.id

  dialog.loading = true
  try {
    await saveTag(payload)
    toast.success(dialog.id ? '修改已保存' : '话题已创建')
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
  const used = Number(item.article_count) > 0
  const extra = used ? '已有文章引用该话题，删除后这些文章不会携带此话题。' : ''
  openConfirm({
    title: '删除话题',
    message: `确定删除话题「${item.name || item.id}」吗？${extra}删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeTags([item.id])
      toast.success('已移入回收站')
      await load()
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreTags([item.id])
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
    message: `确定彻底删除话题「${item.name || item.id}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteTags([item.id])
      toast.success('已彻底删除')
      await load()
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有话题将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearTagRecycle()
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
    message: `确定删除选中的 ${ids.length} 个话题吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeTags(ids)
      toast.success(`已删除 ${ids.length} 个话题`)
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
    await restoreTags(ids)
    toast.success(`已恢复 ${ids.length} 个话题`)
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
    message: `确定彻底删除选中的 ${ids.length} 个话题吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteTags(ids)
      toast.success(`已彻底删除 ${ids.length} 个话题`)
      selectedIds.value = []
      await load()
    }
  })
}

onMounted(load)
</script>

<style scoped>
.tag-admin {
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

.tag-list {
  display: flex;
  flex-direction: column;
}
.tag-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.tag-row:last-child {
  border-bottom: none;
}
.tag-row:hover {
  background: var(--bg-muted);
}
.tag-row.is-selected {
  background: var(--accent-wash);
}

.tag-avatar {
  width: 42px;
  height: 42px;
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
.tag-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tag-main {
  flex: 1;
  min-width: 0;
}
.tag-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.tag-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  word-break: break-word;
}
.count-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 3px;
  font-variant-numeric: tabular-nums;
}
.tag-desc {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tag-meta {
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

.tag-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.tag-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.tag-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 弹窗表单 ---------- */
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
  width: 64px;
  height: 64px;
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
  .tag-row {
    flex-wrap: wrap;
  }
  .tag-main {
    flex: 1 1 60%;
  }
  .tag-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
