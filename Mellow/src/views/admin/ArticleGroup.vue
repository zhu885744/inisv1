<template>
  <div class="group-admin">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">文章分类</h2>
          <p class="block-desc">
            管理文章分类层级与标识
            <template v-if="list.length"> · 共 {{ list.length }} 个分类</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd(null)">
            <i class="bi bi-plus-lg" /> 添加分类
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
          >全部分类</button>
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
            <i class="bi bi-search" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索名称 / 标识 / 描述…" />
          </div>
          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 回收站提示 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的分类不会在前台展示，可恢复或彻底删除</span>
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

      <div v-else-if="!rows.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-folder2'" :text="emptyText" />
      </div>

      <template v-else>
        <!-- 本页全选 -->
        <div class="list-head-row">
          <label class="pick" title="全选本页">
            <FormCheckbox
              ref="selectAllRef"
              :model-value="pageAllSelected"
              :disabled="busy || !rows.length"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">共 {{ rows.length }} 个分类</span>
        </div>

        <ul class="group-list">
          <li
            v-for="g in rows"
            :key="g.id"
            class="group-row"
            :class="{ 'is-selected': isSelected(g.id) }"
          >
            <label class="pick" :title="isSelected(g.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(g.id)"
                :disabled="busy"
                :aria-label="`选择分类 ${g.name || g.id}`"
                @change="toggleSelect(g.id)" />
            </label>

            <div class="group-main" :style="{ paddingLeft: g.level * 22 + 'px' }">
            <span v-if="g.level > 0" class="tree-mark">└</span>
            <img v-if="g.avatar" :src="g.avatar" class="group-avatar" alt="" loading="lazy" />
            <span v-else class="group-avatar placeholder"><i class="bi bi-folder2" /></span>

            <div class="group-info">
              <div class="group-name-row">
                <span class="group-name">{{ g.name || '未命名分类' }}</span>
                <code v-if="g.key" class="group-key">{{ g.key }}</code>
                <span v-if="g.flat || g.orphan" class="group-path">
                  <i class="bi bi-diagram-3" /> {{ g.orphan ? '未知父级' : pathOf(g) }}
                </span>
              </div>
              <p v-if="g.description" class="group-desc">{{ g.description }}</p>
            </div>

            <span class="group-count" title="该分类下的文章数（不含子分类）">
              <i class="bi bi-file-earmark-text" /> {{ countText(g) }}
            </span>
          </div>

          <div class="group-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" :disabled="busy" @click="restore(g)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" :disabled="busy" @click="askForceDelete(g)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>

            <template v-else>
              <button class="btn btn-ghost btn-sm" title="添加子分类" @click="openAdd(g)">
                <i class="bi bi-plus-lg" />
              </button>
              <button class="btn btn-ghost btn-sm" title="编辑" @click="openEdit(g)">
                <i class="bi bi-pencil" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="删除" :disabled="busy" @click="askRemove(g)">
                <i class="bi bi-trash" />
              </button>
            </template>
          </div>
        </li>
        </ul>
      </template>
    </div>

    <!-- 新增 / 编辑弹窗 -->
    <AdminFormDialog
      v-model:visible="dialog.visible"
      :title="dialog.title"
      :icon="dialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="dialog.loading"
      confirm-text="保存"
      @confirm="save"
    >
      <div class="form-item">
        <label class="form-label">上级分类</label>
        <FormSelect
          v-model="dialog.pid"
          :options="pidSelectOptions"
          placeholder="请选择上级分类" />
        <p class="form-hint">可选择任意层级作为父分类；编辑时不可选择自身及其子分类</p>
      </div>

      <div class="form-item">
        <label class="form-label">分类名称</label>
        <FormInput v-model="dialog.name" type="text" maxlength="32" placeholder="请输入分类名称" />
      </div>

      <div class="form-item">
        <label class="form-label">唯一识别码</label>
        <FormInput v-model="dialog.key" type="text" maxlength="64" placeholder="可选，如 tech（用于前台 /category/tech）" />
        <p class="form-hint">仅支持字母、数字、下划线、中划线；留空则不使用前台标识路由</p>
      </div>

      <div class="form-item">
        <label class="form-label">分类头像</label>
        <div class="avatar-row">
          <div v-if="dialog.avatar" class="avatar-preview">
            <img :src="dialog.avatar" alt="分类头像" loading="lazy" />
            <button type="button" class="avatar-del" title="移除头像" @click="dialog.avatar = ''">
              <i class="bi bi-x" />
            </button>
          </div>
          <div v-else class="avatar-empty" @click="pickAvatar">
            <i class="bi bi-image" />
            <span>上传</span>
          </div>
          <FormInput
            v-model="dialog.avatar"
            type="text"
            placeholder="或直接粘贴图片链接" />
          <button class="btn btn-sm" type="button" :disabled="uploading" @click="pickAvatar">
            {{ uploading ? '上传中...' : '上传' }}
          </button>
          <button class="btn btn-sm" type="button" title="从附件库选择" @click="showLibrary = true">
            <i class="bi bi-folder2-open" /> 附件库
          </button>
          <input ref="avatarFileRef" type="file" accept="image/*" hidden @change="onAvatarChange" />

          <!-- 附件库：选一张已上传的图片作为分类头像 -->
          <AttachmentLibrary
            v-model:visible="showLibrary"
            title="选择分类头像"
            accept="image"
            @select="onLibrarySelect"
          />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">分类描述</label>
        <FormTextarea
          v-model="dialog.description"
          rows="3"
          maxlength="200"
          placeholder="可选，用于前台分类页展示" />
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
import { ref, reactive, computed, onMounted, watchEffect } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listArticleGroups,
  saveArticleGroup,
  removeArticleGroup,
  restoreArticleGroup,
  forceDeleteArticleGroup,
  clearArticleGroupRecycle
} from '@/api/article-group'
import { countArticlesInGroup } from '@/api/article'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { toast } from '@/utils/toast'

const list = ref([])
const loading = ref(false)
const busy = ref(false)
const uploading = ref(false)
const showLibrary = ref(false)

// 从附件库选分类头像：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) dialog.avatar = url
}
const trash = ref(false)
const keyword = ref('')

// 分类下的文章数（id -> number|null；null 表示统计失败，数量过多时不统计）
const counts = ref({})

const avatarFileRef = ref(null)

const dialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  pid: 0,
  name: '',
  key: '',
  avatar: '',
  description: ''
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

const groupMap = computed(() => {
  const map = {}
  ;(list.value || []).forEach((g) => { map[g.id] = g })
  return map
})

// 扁平化树（按 pid 递归，使用 level 控制缩进）
const flatTree = computed(() => {
  const source = list.value || []
  const childrenMap = new Map()
  source.forEach((g) => {
    const pid = Number(g.pid || 0)
    if (!childrenMap.has(pid)) childrenMap.set(pid, [])
    childrenMap.get(pid).push(g)
  })

  const out = []
  const walk = (pid, level) => {
    const nodes = childrenMap.get(pid) || []
    nodes.forEach((g) => {
      out.push({ ...g, level })
      walk(Number(g.id), level + 1)
    })
  }
  walk(0, 0)

  // 兜底：父级不存在（数据异常）的节点单独列出，避免在树中丢失
  const shown = new Set(out.map((i) => i.id))
  source.forEach((g) => {
    if (!shown.has(g.id)) out.push({ ...g, level: 0, orphan: true })
  })
  return out
})

const searching = computed(() => !!keyword.value.trim())

// 搜索时为平铺结果（附带所属路径），避免树形过滤造成上下文丢失
const rows = computed(() => {
  if (!searching.value) return flatTree.value
  const kw = keyword.value.trim().toLowerCase()
  return (list.value || [])
    .filter((g) =>
      [g.name, g.key, g.description].some((v) => String(v || '').toLowerCase().includes(kw))
    )
    .map((g) => ({ ...g, level: 0, flat: true }))
})

// 父级选择项：排除自身及其所有后代，防止形成环
const parentOptions = computed(() => {
  const banned = new Set()
  if (dialog.id) {
    banned.add(Number(dialog.id))
    const childrenMap = new Map()
    ;(list.value || []).forEach((g) => {
      const pid = Number(g.pid || 0)
      if (!childrenMap.has(pid)) childrenMap.set(pid, [])
      childrenMap.get(pid).push(g.id)
    })
    const stack = [Number(dialog.id)]
    while (stack.length) {
      const current = stack.pop()
      ;(childrenMap.get(current) || []).forEach((id) => {
        if (!banned.has(Number(id))) {
          banned.add(Number(id))
          stack.push(Number(id))
        }
      })
    }
  }

  // 以「顶级分类」为根生成带缩进的下拉项
  const source = (list.value || []).filter((g) => !banned.has(Number(g.id)))
  const byPid = new Map()
  source.forEach((g) => {
    const pid = Number(g.pid || 0)
    if (!byPid.has(pid)) byPid.set(pid, [])
    byPid.get(pid).push(g)
  })
  const out = []
  const walk = (pid, level) => {
    ;(byPid.get(pid) || []).forEach((g) => {
      out.push({ id: Number(g.id), label: `${'　'.repeat(level)}${level ? '└ ' : ''}${g.name}` })
      walk(Number(g.id), level + 1)
    })
  }
  walk(0, 0)
  return out
})

// 上级分类下拉选项：首项为「顶级分类」，其余为排除自身及后代后的层级树
const pidSelectOptions = computed(() => [
  { value: 0, label: '顶级分类' },
  ...parentOptions.value.map((opt) => ({ value: opt.id, label: opt.label }))
])

const emptyText = computed(() => {
  if (searching.value) return '没有匹配的分类'
  if (trash.value) return '回收站是空的'
  return '还没有分类，先添加一个吧'
})

// ---------- 展示辅助 ----------
function pathOf(g) {
  const pid = Number(g.pid || 0)
  if (!pid) return '顶级分类'
  const parts = []
  let cur = pid
  let guard = 0
  while (cur > 0 && guard < 10) {
    const parent = groupMap.value[cur]
    if (!parent) break
    parts.unshift(parent.name)
    cur = Number(parent.pid || 0)
    guard++
  }
  return parts.length ? parts.join(' / ') : '未知父级'
}

function countText(g) {
  const value = counts.value[g.id]
  if (value === undefined || value === null) return '—'
  return value
}

function childCountOf(id) {
  return (list.value || []).filter((g) => Number(g.pid || 0) === Number(id)).length
}

// ---------- 数据加载 ----------
async function load() {
  loading.value = true
  selectedIds.value = []
  try {
    const params = { order: 'id asc' }
    if (trash.value) params.onlyTrashed = true
    const res = await listArticleGroups(params)
    list.value = res.data?.data || []
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
  loadCounts()
}

/**
 * 分类文章数统计
 * 后端没有按分类聚合的接口，这里退化为「每个分类一次 count」；
 * 分类数量较多（> 30）时跳过统计，避免一次页面产生过多请求。
 */
async function loadCounts() {
  const groups = list.value || []
  counts.value = {}
  if (!groups.length || groups.length > 30) return
  const entries = await Promise.all(
    groups.map(async (g) => {
      try {
        const res = await countArticlesInGroup(g.id)
        return [g.id, Number(res.data) || 0]
      } catch {
        return [g.id, null]
      }
    })
  )
  const map = {}
  entries.forEach(([id, n]) => { map[id] = n })
  counts.value = map
}

async function switchTab(value) {
  if (trash.value === value) return
  trash.value = value
  keyword.value = ''
  await load()
}

// ---------- 新增 / 编辑 ----------
function resetDialog() {
  dialog.id = 0
  dialog.pid = 0
  dialog.name = ''
  dialog.key = ''
  dialog.avatar = ''
  dialog.description = ''
}

function openAdd(parent) {
  resetDialog()
  dialog.pid = parent ? Number(parent.id) : 0
  dialog.title = parent ? `在「${parent.name}」下添加子分类` : '添加分类'
  dialog.visible = true
}

function openEdit(g) {
  dialog.id = Number(g.id)
  dialog.pid = Number(g.pid || 0)
  dialog.name = g.name || ''
  dialog.key = g.key || ''
  dialog.avatar = g.avatar || ''
  dialog.description = g.description || ''
  dialog.title = '编辑分类'
  dialog.visible = true
}

function pickAvatar() {
  avatarFileRef.value?.click()
}

async function onAvatarChange(e) {
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
    toast.success('头像已上传')
  } catch {
    toast.error('头像上传失败')
  } finally {
    uploading.value = false
  }
}

async function save() {
  const name = dialog.name.trim()
  if (!name) {
    toast.warning('请输入分类名称')
    return
  }
  const key = dialog.key.trim()
  if (key && !/^[A-Za-z0-9_-]+$/.test(key)) {
    toast.warning('唯一识别码只能包含字母、数字、下划线或中划线')
    return
  }

  dialog.loading = true
  try {
    const payload = {
      pid: Number(dialog.pid) || 0,
      name,
      key,
      avatar: dialog.avatar.trim(),
      description: dialog.description.trim()
    }
    if (dialog.id) payload.id = dialog.id
    await saveArticleGroup(payload)
    toast.success(dialog.id ? '修改已保存' : '分类已创建')
    dialog.visible = false
    await load()
  } catch {
    // 失败提示由请求拦截器统一给出（如 key 已存在）
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

function askRemove(g) {
  const children = childCountOf(g.id)
  const articles = counts.value[g.id]
  const extra = []
  if (children > 0) extra.push(`该分类下有 ${children} 个子分类`)
  if (Number(articles) > 0) extra.push(`该分类下有 ${articles} 篇文章`)
  const tail = extra.length ? `（${extra.join('，')}）删除后这些内容不会自动处理，请先确认。` : ''

  openConfirm({
    title: '删除分类',
    message: `确定删除分类「${g.name || g.id}」吗？${tail}删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeArticleGroup([g.id])
      toast.success('已移入回收站')
      await load()
    }
  })
}

async function restore(g) {
  busy.value = true
  try {
    await restoreArticleGroup([g.id])
    toast.success('已恢复')
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(g) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除分类「${g.name || g.id}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteArticleGroup([g.id])
      toast.success('已彻底删除')
      await load()
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有分类将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearArticleGroupRecycle()
      toast.success('回收站已清空')
      await load()
    }
  })
}

// ---------- 多选 ----------
const selectedIds = ref([])
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => rows.value.length > 0 && rows.value.every((i) => selectedSet.value.has(i.id))
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
  selectedIds.value = pageAllSelected.value ? [] : rows.value.map((i) => i.id)
}

function clearSelection() {
  selectedIds.value = []
}

// ---------- 批量操作 ----------
function askBatchRemove() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  const withChildren = ids.filter((id) => childCountOf(id) > 0).length
  const tail = withChildren
    ? `其中 ${withChildren} 个分类含子分类，删除后子分类将变为「未知父级」，请谨慎操作。`
    : ''
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 个分类吗？${tail}删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeArticleGroup(ids)
      toast.success(`已删除 ${ids.length} 个分类`)
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
    await restoreArticleGroup(ids)
    toast.success(`已恢复 ${ids.length} 个分类`)
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
    message: `确定彻底删除选中的 ${ids.length} 个分类吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteArticleGroup(ids)
      toast.success(`已彻底删除 ${ids.length} 个分类`)
      selectedIds.value = []
      await load()
    }
  })
}

onMounted(load)
</script>

<style scoped>
.group-admin {
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
  width: 230px;
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

.group-list {
  display: flex;
  flex-direction: column;
}
.group-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 6px;
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

.group-main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}
.tree-mark {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-light);
}
.group-avatar {
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  border-radius: var(--radius-sm);
  object-fit: cover;
  border: 1px solid var(--border);
}
.group-avatar.placeholder {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 15px;
}
.group-info {
  flex: 1;
  min-width: 0;
}
.group-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.group-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.group-key {
  padding: 1px 6px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.group-path {
  font-size: 11px;
  color: var(--text-light);
}
.group-desc {
  margin: 3px 0 0;
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.group-count {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 3px;
  font-variant-numeric: tabular-nums;
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

/* ---------- 弹窗表单 ---------- */
.form-item {
  margin-bottom: 16px;
}
.form-item:last-child {
  margin-bottom: 0;
}
.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.avatar-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.avatar-preview {
  position: relative;
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border);
}
.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.avatar-del {
  position: absolute;
  right: 1px;
  top: 1px;
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
.avatar-empty {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  font-size: 10px;
  color: var(--text-light);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
}
.avatar-empty:hover {
  border-color: var(--primary);
  color: var(--primary);
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
    flex: 1;
    width: auto;
  }
  .group-row {
    flex-wrap: wrap;
  }
  .group-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
