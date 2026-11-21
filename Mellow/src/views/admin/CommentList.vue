<template>
  <div class="comment-list">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">评论列表</h2>
          <p class="block-desc">
            按来源与层级筛选评论，支持编辑内容、删除与回收站恢复
            <template v-if="total > 0"> · 共 {{ total }} 条</template>
          </p>
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
            v-model="level"
            :options="COMMENT_LEVEL_OPTIONS"
            icon="bi bi-diagram-3"
            placeholder="全部层级"
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
            placeholder="搜索范围"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              :placeholder="`搜索${searchFieldLabel}…`"
              :aria-label="`搜索${searchFieldLabel}`"
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
            <button class="btn btn-sm" :disabled="busy" @click="batchPass()">
              <i class="bi bi-check2-circle" /> 批量通过
            </button>
            <button class="btn btn-sm" :disabled="busy" @click="askBatchReject()">
              <i class="bi bi-x-circle" /> 批量驳回
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
        <span><i class="bi bi-trash3" /> 回收站中的评论不会在前台展示，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !total" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-chat-square-text'" :text="emptyText" />
      </div>

      <template v-else>
        <!-- 本页全选 -->
        <div class="list-head-row">
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

        <ul class="comment-items">
          <li
            v-for="item in list"
            :key="item.id"
            class="comment-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择评论 ${item.id}`"
                @change="toggleSelect(item.id)" />
            </label>

            <img class="comment-avatar" :src="authorOf(item).avatar || defaultAvatar" :alt="authorOf(item).nickname" loading="lazy" />

            <div class="comment-main">
              <div class="comment-name-row">
                <span class="comment-name">{{ authorOf(item).nickname || '匿名' }}</span>
                <span v-if="authorOf(item).title" class="flag is-title">{{ authorOf(item).title }}</span>
                <span class="bind-chip" :class="`is-${item.bind_type}`">
                  <i :class="bindIcon(item)" /> {{ bindLabel(item) }}
                </span>
                <!-- 当前可见范围：评论有「公开 / 登录可见 / 仅作者可见」三档（allow-pay=false） -->
                <ScopeBadge
                  :scope="item.scope"
                  :integral="item.scope_integral"
                  :allow-pay="false"
                />
                <span v-if="isReply(item)" class="flag is-reply">
                  <i class="bi bi-reply" /> 回复 #{{ item.pid }}
                </span>
                <!-- 审核状态：已通过不显示标签，减少噪音（开关见「系统设置 → 评论配置 → 内容审核」） -->
                <span
                  v-if="auditFlag(item)"
                  class="flag"
                  :class="auditFlag(item).cls"
                  :title="auditFlag(item).tip"
                >
                  <i :class="auditFlag(item).icon" /> {{ auditFlag(item).text }}
                </span>
                <span class="comment-time">{{ timeText(item) }}</span>
              </div>

              <p class="comment-content">
                <EmojiText v-if="item.content" :text="item.content" :size="17" />
                <template v-else>（无内容）</template>
              </p>

              <!-- 配图缩略图：最多 3 张，超出用 +N 提示（解析结果按页缓存） -->
              <div v-if="imagesOf(item).length" class="comment-thumbs">
                <button
                  v-for="(img, i) in imagesOf(item).slice(0, MAX_THUMBS)"
                  :key="img + i"
                  type="button"
                  class="thumb"
                  :title="`查看第 ${i + 1} 张（共 ${imagesOf(item).length} 张）`"
                  :aria-label="`查看第 ${i + 1} 张配图`"
                  @click="preview(item, i)"
                >
                  <img :src="img" :alt="`配图 ${i + 1}`" loading="lazy" />
                  <span v-if="i === MAX_THUMBS - 1 && moreCount(item)" class="thumb-more">
                    +{{ moreCount(item) }}
                  </span>
                </button>
              </div>

              <div class="comment-meta">
                <span class="meta-text">#{{ item.id }}</span>
                <router-link
                  v-if="bindLink(item)"
                  :to="bindLink(item)"
                  class="meta-chip is-link"
                  :title="`前往：${bindTitle(item)}`"
                >
                  <i class="bi bi-box-arrow-up-right" /> {{ bindTitleText(item) }}
                </router-link>
                <span v-else-if="bindTitle(item)" class="meta-chip">
                  <i class="bi bi-file-earmark" /> {{ bindTitleText(item) }}
                </span>
                <span v-if="item.ip" class="meta-text"><i class="bi bi-geo-alt" /> {{ item.ip }}</span>
                <span v-if="isEdited(item)" class="meta-text">已编辑</span>
              </div>
            </div>

            <div class="comment-actions">
              <template v-if="trash">
                <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                  <i class="bi bi-arrow-counterclockwise" />
                </button>
                <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                  <i class="bi bi-x-octagon" />
                </button>
              </template>

              <template v-else>
                <!-- 审核：待审核 / 未通过时才显示对应操作（已通过的只留「驳回」） -->
                <button
                  v-if="Number(item.audit) !== 1"
                  class="btn btn-ghost btn-sm ok"
                  title="审核通过（前台可见）"
                  aria-label="审核通过"
                  :disabled="busy"
                  @click="passItem(item)"
                >
                  <i class="bi bi-check2-circle" />
                </button>
                <button
                  v-if="Number(item.audit) !== 2"
                  class="btn btn-ghost btn-sm danger"
                  title="驳回（需填写原因）"
                  aria-label="驳回"
                  :disabled="busy"
                  @click="askReject(item)"
                >
                  <i class="bi bi-x-circle" />
                </button>
                <router-link
                  v-if="bindLink(item)"
                  :to="bindLink(item)"
                  class="btn btn-ghost btn-sm"
                  title="查看来源"
                  aria-label="查看来源"
                >
                  <i class="bi bi-eye" />
                </router-link>
                <button class="btn btn-ghost btn-sm" title="编辑内容" aria-label="编辑内容" @click="openEdit(item)">
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

    <!-- 编辑评论 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="`编辑评论 #${edit.id}`"
      icon="bi bi-pencil-square"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="saveEdit"
    >
      <div class="form-item">
        <label class="form-label">评论内容</label>
        <FormTextarea v-model="edit.content" rows="4" placeholder="请输入评论内容" />
      </div>

      <div class="form-item">
        <label class="form-label">评论图片</label>
        <FormInput v-model="edit.images" type="text" placeholder="多个图片链接用英文逗号分隔" />

        <div class="image-actions">
          <button type="button" class="btn btn-sm" :disabled="uploading" @click="pickCommentImages">
            <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
          </button>
          <button type="button" class="btn btn-sm" title="从附件库选择" @click="showImageLibrary = true">
            <i class="bi bi-folder2-open" /> 附件库
          </button>
          <button v-if="editImageList.length" type="button" class="btn btn-sm btn-ghost" @click="edit.images = ''">
            清空
          </button>
        </div>
        <input ref="imageFileRef" type="file" accept="image/*" multiple hidden @change="onCommentImagesChange" />

        <div v-if="editImageList.length" class="thumb-row">
          <img v-for="(img, i) in editImageList" :key="`${img}-${i}`" :src="img" alt="评论图片" loading="lazy" />
        </div>
        <p class="form-hint">留空表示移除该评论的全部图片</p>
      </div>

      <AttachmentLibrary
        v-model:visible="showImageLibrary"
        title="选择评论图片"
        accept="image"
        multiple
        :max="9"
        @select="onImageLibrarySelect"
      />
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

    <!-- 驳回评论（须填原因；单条与批量共用，目标记在 auditDialog.target） -->
    <AuditDialog
      :visible="auditDialog.visible"
      :audit="auditDialog.audit"
      :default-reason="auditDialog.reason"
      :owner-uid="auditDialog.ownerUid"
      owner-label="评论者"
      :loading="auditDialog.loading"
      title="驳回评论"
      message="驳回后该评论在前台不可见，评论者会收到通知（含你填写的原因）。"
      confirm-text="驳回"
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
 * 评论列表（/admin/comment 子路由）
 *
 * 与父级 Comments.vue 通过 inject('commentsAdmin') 联动：统计卡片点击 → 列表筛选，
 * 列表增删改 → 刷新统计卡片。列表状态（来源标签 / 回收站 / 层级 / 排序 / 搜索 / 页码）都在这里维护。
 *
 * 关键约束（后端 comment 控制器）：
 * - 评论表没有审核 / 状态字段，可筛选维度只有 bind_type、pid（层级）与回收站；
 * - comment/update 允许 content / images（root 才能改 pid / bind_id / bind_type），内容不能为空；
 * - comment/remove|delete 对非管理员仅允许删除自己的评论，管理员不受限；
 * - result 内已带 author / article / page / moments，作者与来源无需额外请求。
 */
import { ref, reactive, computed, inject, watchEffect, onMounted, onUnmounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import EmojiText from '@/components/EmojiText.vue'
import ScopeBadge from '@/components/ScopeBadge.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import AuditDialog from '@/components/admin/AuditDialog.vue'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { uploadAttachments } from '@/api/attachment'
import {
  listComments,
  updateComment,
  removeComments,
  forceDeleteComments,
  restoreComments,
  clearCommentRecycle
} from '@/api/comment'
import { fromNow } from '@/utils/time'
import { debounce, pickCommentAuthor } from '@/utils/helper'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'
import {
  COMMENT_TABS,
  COMMENT_LEVEL_OPTIONS,
  COMMENT_SORT_OPTIONS,
  COMMENT_SEARCH_FIELDS,
  COMMENT_TRASH_KEY,
  COMMENT_TRASH_ORDER,
  COMMENT_DEFAULT_ORDER,
  BIND_TYPE_ARTICLE,
  BIND_TYPE_PAGE,
  BIND_TYPE_MOMENTS,
  commentListParams,
  commentBindLabel,
  commentBindTitle,
  commentBindLink,
  parseCommentImages,
  parseCommentMedia,
  parseCommentRawList,
  isCommentReply
} from '@/utils/comment'
import { parseLiveItem } from '@/utils/livePhoto'

// 父级 Comments.vue 注入的联动能力（统计卡片 ←→ 列表筛选）
const admin = inject('commentsAdmin', null)

const pageSize = ref(PAGE_SIZE_DEFAULT)
const tabs = COMMENT_TABS
const sortOptions = COMMENT_SORT_OPTIONS
const searchFields = COMMENT_SEARCH_FIELDS

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="34" fill="%238a8a82" font-family="serif">用</text></svg>'

// 列表最多展示 3 张缩略图，其余张数以 +N 角标提示
const MAX_THUMBS = 3

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
// 回收站模式
const trash = ref(false)
const status = ref('all')
const sortKey = ref(COMMENT_DEFAULT_ORDER)
const level = ref('')
const searchField = ref('content')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

const currentTab = computed(() => tabs.find((tab) => tab.key === status.value) || tabs[0])

const searchFieldLabel = computed(
  () => searchFields.find((f) => f.value === searchField.value)?.label || '内容'
)

const emptyText = computed(() => {
  if (trash.value) return '回收站是空的'
  if (searchKey.value) return '没有匹配的评论'
  if (status.value !== 'all') return `暂无「${currentTab.value.label}」评论`
  return '暂无评论'
})

// ---------- 展示辅助 ----------
// 配图解析结果按当前页缓存：模板里对同一条评论会读多次，避免重复 split
const thumbsMap = computed(() => {
  const map = Object.create(null)
  for (const item of list.value) map[item.id] = parseCommentImages(item.images)
  return map
})

function imagesOf(item) {
  return thumbsMap.value[item.id] || []
}

function moreCount(item) {
  return Math.max(0, imagesOf(item).length - MAX_THUMBS)
}

// 灯箱传 { image, video }：实况图会多一个「播放实况」按钮（缩略图仍只显示封面）
function preview(item, index) {
  openLightbox(parseCommentMedia(item.images), index)
}

function authorOf(item) {
  return pickCommentAuthor(item)
}

const bindLabel = commentBindLabel

function bindTitle(item) {
  return commentBindTitle(item)
}

// 动态内容较长，卡片里只展示前 40 字
function bindTitleText(item) {
  const text = bindTitle(item)
  if (!text) return '来源已删除'
  return text.length > 40 ? `${text.slice(0, 40)}…` : text
}

function bindLink(item) {
  return commentBindLink(item)
}

// ---------- 审核 ----------
// 是否开启审核由「系统设置 → 评论配置 → 内容审核」控制（COMMENT 的 json.audit）：
// 关闭时前台提交的评论 audit 直接为 1（已通过），开启时为 0（待审核）；
// 非管理员在接口层只能看到 audit=1 的评论，管理端可以筛「待审核」并做通过 / 驳回。
const auditDialog = reactive({ visible: false, loading: false, audit: 2, reason: '', ownerUid: 0, target: null })

// 审核状态标签：已通过不显示（减少噪音），未通过带上驳回原因
function auditFlag(item) {
  const audit = Number(item?.audit || 0)
  if (audit === 1) return null
  if (audit === 2) {
    return {
      text: '未通过',
      cls: 'is-reject',
      icon: 'bi bi-x-circle',
      tip: item?.reason ? `驳回原因：${item.reason}` : '审核未通过，前台不可见'
    }
  }
  return { text: '待审核', cls: 'is-pending', icon: 'bi bi-hourglass-split', tip: '等待审核，前台暂不可见' }
}

// 只改 audit / reason，不动评论内容
async function applyAudit(item, audit, reason = '') {
  busy.value = true
  try {
    await updateComment({ id: item.id, audit, reason })
    toast.success(audit === 1 ? '已通过' : '已驳回')
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    busy.value = false
  }
}

function passItem(item) {
  return applyAudit(item, 1, '')
}

function askReject(item) {
  openAuditDialog([item], item?.reason || '', Number(item?.uid) || 0)
}

function askBatchReject() {
  openAuditDialog(selectedItems(), '', 0)
}

function openAuditDialog(target, reason = '', ownerUid = 0) {
  auditDialog.audit = 2
  auditDialog.target = target
  auditDialog.reason = reason
  auditDialog.ownerUid = ownerUid
  auditDialog.visible = true
}

function closeAuditDialog(value) {
  auditDialog.visible = value !== false
  if (!auditDialog.visible) auditDialog.target = null
}

// 批量通过：逐条调用（后端没有批量审核接口），全部成功后再刷新
async function batchPass() {
  const items = selectedItems()
  if (!items.length) return

  busy.value = true
  try {
    for (const item of items) {
      await updateComment({ id: item.id, audit: 1, reason: '' })
    }
    toast.success(`已通过 ${items.length} 条`)
    clearSelection()
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    busy.value = false
  }
}

async function runAudit(reason) {
  const items = Array.isArray(auditDialog.target) ? auditDialog.target : [auditDialog.target]
  const targets = items.filter(Boolean)
  if (!targets.length) return

  auditDialog.loading = true
  try {
    for (const item of targets) {
      await updateComment({ id: item.id, audit: 2, reason })
    }
    toast.success(`已驳回 ${targets.length} 条`)
    auditDialog.visible = false
    auditDialog.target = null
    clearSelection()
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    auditDialog.loading = false
  }
}

// 当前页选中的评论对象（批量操作要用 id + uid）
function selectedItems() {
  const ids = new Set(selectedIds.value)
  return list.value.filter((item) => ids.has(item.id))
}

const bindIcons = {
  [BIND_TYPE_ARTICLE]: 'bi bi-file-earmark-text',
  [BIND_TYPE_PAGE]: 'bi bi-file-earmark',
  [BIND_TYPE_MOMENTS]: 'bi bi-lightning-charge'
}

function bindIcon(item) {
  return bindIcons[item?.bind_type] || 'bi bi-question-circle'
}

function isReply(item) {
  return isCommentReply(item)
}

function isEdited(item) {
  return Number(item?.update_time || 0) > Number(item?.create_time || 0)
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

// ---------- 数据加载 ----------
// 请求序号：快速切换筛选 / 搜索时丢弃过期响应，避免旧数据覆盖新结果
let reqSeq = 0

async function load() {
  const run = ++reqSeq
  loading.value = true
  try {
    const res = await listComments(
      commentListParams({
        key: trash.value ? COMMENT_TRASH_KEY : status.value,
        page: page.value,
        limit: pageSize.value,
        keyword: searchKey.value,
        searchField: searchField.value,
        level: level.value,
        order: trash.value ? COMMENT_TRASH_ORDER : sortKey.value
      })
    )
    if (run !== reqSeq) return
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
  } catch {
    if (run !== reqSeq) return
    list.value = []
    total.value = 0
  } finally {
    if (run === reqSeq) loading.value = false
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
  admin?.setActive(trash.value ? COMMENT_TRASH_KEY : status.value)
  reload()
}

// 父级统计卡片点击后的回调（key 与 tabs / trash 对应）
function applyFilter(key) {
  if (key === COMMENT_TRASH_KEY) {
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
function askRemove(item) {
  const extra = isReply(item) ? '' : '（删除顶层评论不会连带删除它的回复）'
  openConfirm({
    title: '删除评论',
    message: `确定删除评论 #${item.id} 吗？${extra}删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeComments([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除评论 #${item.id} 吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteComments([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreComments([item.id])
    toast.success('已恢复')
    await afterMutation(1)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

// ---------- 批量操作 ----------
function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 条评论吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeComments(ids)
      toast.success(`已删除 ${ids.length} 条`)
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
    await restoreComments(ids)
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
    message: `确定彻底删除选中的 ${ids.length} 条评论吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteComments(ids)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有评论将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearCommentRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

// ---------- 编辑评论 ----------
const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  content: '',
  images: ''
})

// 缩略图预览只显示封面；保存 / 合并用 parseCommentRawList（保留「图片::视频」）
const editImageList = computed(() =>
  parseCommentRawList(edit.images).map((item) => parseLiveItem(item).image)
)

// ===== 评论图片：上传 / 附件库 =====
const imageFileRef = ref(null)
const uploading = ref(false)
const showImageLibrary = ref(false)

// 追加到 edit.images（逗号分隔），重复地址跳过
// 合并的是「原始项」：已经存在的实况图（图片::视频）不会在这里被截成纯封面
function appendCommentImages(urls = []) {
  const merged = [...parseCommentRawList(edit.images)]
  const exists = new Set(merged.map((item) => parseLiveItem(item).image))
  for (const url of urls) {
    if (url && !exists.has(url)) {
      merged.push(url)
      exists.add(url)
    }
  }
  edit.images = merged.join(',')
}

function pickCommentImages() {
  imageFileRef.value?.click()
}

async function onCommentImagesChange(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  if (!files.length) return

  const valid = files.filter((file) => {
    if (!file.type.startsWith('image/')) {
      toast.warning(`文件「${file.name}」不是图片`)
      return false
    }
    if (file.size > 5 * 1024 * 1024) {
      toast.warning(`图片「${file.name}」超过 5MB`)
      return false
    }
    return true
  })
  if (!valid.length) return

  uploading.value = true
  try {
    const fd = new FormData()
    valid.forEach((file) => fd.append('files', file))
    const res = await uploadAttachments(fd)
    const urls = (res.data?.results || [])
      .map((item) => item.full_url || item.url)
      .filter(Boolean)

    if (!urls.length) throw new Error('empty')

    appendCommentImages(urls)
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

// 从附件库选图：多选，追加到现有列表
// 附件里的实况图（带 live_video）拼成「图片::视频」，否则只会存下封面
function onImageLibrarySelect(urls = [], items = []) {
  showImageLibrary.value = false
  const liveByUrl = new Map(items.map((item) => [item.full_url, item.live_video || '']))
  appendCommentImages(urls.map((url) => (liveByUrl.get(url) ? `${url}::${liveByUrl.get(url)}` : url)))
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.content = item.content || ''
  // 回填原始项：实况图保留「图片::视频」，否则一保存就降级成静态图
  edit.images = parseCommentRawList(item.images).join(',')
  edit.visible = true
}

async function saveEdit() {
  const content = edit.content.trim()
  if (!content) {
    toast.warning('评论内容不能为空')
    return
  }

  edit.loading = true
  try {
    // images 传空串即清空该评论的配图；用原始项拼接，保留实况图的「图片::视频」
    await updateComment({
      id: edit.id,
      content,
      images: parseCommentRawList(edit.images).join(',')
    })
    toast.success('修改已保存')
    edit.visible = false
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出（如内容含恶意代码）
  } finally {
    edit.loading = false
  }
}

onMounted(load)

onUnmounted(() => {
  reqSeq += 1
})
</script>

<style scoped>
.comment-list {
  display: flex;
  flex-direction: column;
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
  width: 190px;
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

.comment-items {
  display: flex;
  flex-direction: column;
}
.comment-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 8px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.comment-row:last-child {
  border-bottom: none;
}
.comment-row.is-selected {
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

.comment-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  align-self: flex-start;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  border: 1px solid var(--border);
}

.comment-main {
  flex: 1;
  min-width: 0;
}
.comment-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 6px;
}
.comment-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.comment-time {
  font-size: 12px;
  color: var(--text-muted);
}

.flag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.flag.is-title {
  color: var(--warning);
  background: var(--gold-wash);
}
.flag.is-reply {
  color: var(--text-muted);
  background: var(--bg-muted);
}
/* 审核状态标签：待审核（橙）/ 未通过（红） */
.flag.is-pending {
  color: var(--warning);
  background: var(--gold-wash);
}
.flag.is-reject {
  color: var(--danger);
  background: rgba(217, 84, 77, 0.12);
}
/* 「审核通过」按钮用成功色，和「驳回」的红色区分开 */
.comment-actions .ok {
  color: var(--success);
}

.bind-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  background: var(--bg-muted);
  color: var(--text-muted);
}
.bind-chip.is-article {
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.bind-chip.is-page {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.bind-chip.is-moments {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.comment-content {
  margin: 0;
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
}

.comment-thumbs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
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

.comment-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  max-width: 320px;
  padding: 2px 8px;
  background: var(--bg-muted);
  border-radius: 3px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.meta-chip.is-link {
  color: var(--primary-deep);
  background: var(--accent-soft);
  text-decoration: none;
}
.meta-chip.is-link:hover {
  background: var(--accent-ring);
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.comment-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  align-self: center;
}
.comment-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.comment-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 弹窗 ---------- */
.image-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  /* .form-item 不是 flex 布局，和上面的输入框之间要自己留间距 */
  margin-top: 8px;
}
.thumb-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.thumb-row img {
  width: 56px;
  height: 56px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
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
  .comment-row {
    flex-wrap: wrap;
  }
  .comment-main {
    flex: 1 1 60%;
  }
  .comment-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
