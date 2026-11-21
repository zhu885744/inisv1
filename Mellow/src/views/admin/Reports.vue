<template>
  <div class="reports-admin">
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">举报管理</h2>
          <p class="block-desc">
            处理用户提交的举报。待处理 {{ pending }} 条 · 合计 {{ total }} 条
            <template v-if="pending > 0">（建议优先处理「待处理」）</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm btn-soft" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </header>

      <!-- 筛选：状态 + 对象类型 -->
      <div class="report-filter">
        <div class="chip-group" aria-label="处理状态">
          <button
            v-for="opt in STATUS_OPTIONS"
            :key="opt.value"
            type="button"
            class="chip"
            :class="{ 'is-active': status === opt.value }"
            @click="setStatus(opt.value)"
          >{{ opt.label }}</button>
        </div>
        <div class="chip-group" aria-label="举报对象">
          <button
            v-for="opt in TYPE_OPTIONS"
            :key="opt.value"
            type="button"
            class="chip"
            :class="{ 'is-active': targetType === opt.value }"
            @click="setType(opt.value)"
          >{{ opt.label }}</button>
        </div>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <EmptyState v-else-if="!list.length" icon="bi bi-flag" text="没有符合条件的举报" />

      <ul v-else class="report-list">
        <li v-for="item in list" :key="item.id" class="report-item">
          <div class="report-head">
            <span class="report-type">{{ item.target_type_text }}</span>
            <span class="report-status" :class="statusClass(item.status)">
              {{ item.status_text }}
            </span>
            <span class="report-reason">{{ item.reason_text }}</span>
            <time class="report-time">{{ formatDate(item.create_time) }}</time>
          </div>

          <!-- 内容快照：原内容可能已被作者删除，这里保存的是提交时的原文 -->
          <p class="report-snapshot">{{ item.snapshot || '（无内容快照）' }}</p>

          <!-- 举报人上传的证据图片：点击放大 -->
          <div v-if="(item.images || []).length" class="report-images">
            <img
              v-for="(img, i) in item.images"
              :key="img"
              :src="img"
              class="report-image"
              alt="举报证据"
              loading="lazy"
              @click="openLightbox(item.images, i)"
            />
          </div>

          <p v-if="item.detail" class="report-detail">补充说明：{{ item.detail }}</p>

          <div class="report-meta">
            <span class="meta-text">
              举报人：{{ item.reporter?.nickname || '未知' }}（uid {{ item.uid }}）
            </span>
            <span class="meta-text">
              被举报：{{ item.target_user?.nickname || '未知' }}（uid {{ item.target_uid }}）
            </span>
            <span class="meta-text">对象 ID：{{ item.target_id }}</span>
          </div>

          <p v-if="item.handle_remark" class="report-remark">
            处理说明：{{ item.handle_remark }}
            <template v-if="Number(item.remove_content) === 1">（已移除内容）</template>
          </p>

          <div class="report-actions">
            <template v-if="Number(item.status) === 0">
              <button class="btn btn-sm btn-primary" :disabled="busy" @click="openHandle(item, 1)">
                <i class="bi bi-check2-circle" /> 属实
              </button>
              <button class="btn btn-sm btn-ghost" :disabled="busy" @click="openHandle(item, 2)">
                <i class="bi bi-slash-circle" /> 驳回
              </button>
            </template>
            <span v-else class="meta-text">已处理，无需重复操作</span>
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
        @update:current="(p) => { page = p; load() }"
      />
    </section>

    <!-- 处理弹窗：属实可勾选「同时移除被举报内容」（软删除，可在回收站恢复） -->
    <ConfirmDialog
      :visible="dialog.visible"
      :title="dialog.status === 1 ? '举报属实' : '驳回举报'"
      :danger="dialog.status === 2"
      :confirm-text="dialog.status === 1 ? '确认属实' : '确认驳回'"
      :loading="dialog.loading"
      @update:visible="(v) => (dialog.visible = v)"
      @confirm="runHandle"
    >
      <p class="dialog-line">
        举报对象：<strong>{{ dialog.item?.target_type_text }}</strong>
        · 原因：{{ dialog.item?.reason_text }}
      </p>
      <p class="dialog-snapshot">{{ dialog.item?.snapshot || '（无内容快照）' }}</p>

      <!-- 证据图片：处理前先看清楚 -->
      <div v-if="(dialog.item?.images || []).length" class="report-images">
        <img
          v-for="(img, i) in dialog.item.images"
          :key="img"
          :src="img"
          class="report-image"
          alt="举报证据"
          @click="openLightbox(dialog.item.images, i)"
          loading="lazy"
        />
      </div>

      <label v-if="dialog.status === 1 && canRemoveContent" class="dialog-check">
        <FormCheckbox v-model="dialog.removeContent" />
        <span>同时移除被举报内容（软删除，作者可在「回收站」看到）</span>
      </label>
      <p v-else-if="dialog.status === 1" class="dialog-hint">
        「用户」类举报不在这里处理账号：需要封禁请到「用户管理」，那里有申诉与留痕流程。
      </p>

      <label class="dialog-remark">
        <span class="dialog-remark-label">处理说明<em>选填，会随通知发给举报人</em></span>
        <FormTextarea
          v-model="dialog.remark"
          rows="3"
          maxlength="200"
          placeholder="例如：已警告作者 / 内容不存在 / 未发现违规" />
      </label>
    </ConfirmDialog>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 后台「举报管理」
 *
 * 与动态 / 文章审核的区别：这里处理的是**用户举报**，面向的是「是否违规」的判断，
 * 因此不直接改内容的 audit 字段，而是：
 *   - 属实（status=1）：可勾选同时移除内容（软删除，可恢复）；封禁账号仍走「用户管理」的封禁流程
 *   - 驳回（status=2）：仅记录结论
 * 处理结果会以站内信通知举报人（不含处理人身份）。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import { handleReport, listReports, REPORT_TARGETS } from '@/api/report'
import { formatDate } from '@/utils/time'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

const STATUS_OPTIONS = [
  { value: '', label: '全部' },
  { value: '0', label: '待处理' },
  { value: '1', label: '已处理' },
  { value: '2', label: '已驳回' }
]

const TYPE_OPTIONS = [{ value: '', label: '全部类型' }, ...REPORT_TARGETS.map((t) => ({ value: t.value, label: t.label }))]

const list = ref([])
const total = ref(0)
const pending = ref(0)
const loading = ref(false)
const busy = ref(false)
const page = ref(1)
const pageSize = ref(PAGE_SIZE_DEFAULT)
const status = ref('0')
const targetType = ref('')

const dialog = reactive({
  visible: false,
  loading: false,
  status: 1,
  removeContent: false,
  remark: '',
  item: null
})

// 只有内容类举报才谈得上「移除内容」；用户举报走「用户管理」的封禁流程
const canRemoveContent = computed(() =>
  ['moment', 'article', 'comment'].includes(String(dialog.item?.target_type || ''))
)

function statusClass(value) {
  const status = Number(value)
  if (status === 0) return 'is-pending'
  if (status === 1) return 'is-done'
  return 'is-rejected'
}

async function load() {
  loading.value = true
  try {
    const res = await listReports({
      page: page.value,
      limit: pageSize.value,
      status: status.value,
      target_type: targetType.value
    })
    list.value = res.data?.data || []
    total.value = Number(res.data?.count) || 0
    pending.value = Number(res.data?.pending) || 0
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function setStatus(value) {
  if (status.value === value) return
  status.value = value
  page.value = 1
  load()
}

function setType(value) {
  if (targetType.value === value) return
  targetType.value = value
  page.value = 1
  load()
}

function openHandle(item, statusValue) {
  dialog.item = item
  dialog.status = statusValue
  dialog.removeContent = false
  dialog.remark = ''
  dialog.visible = true
}

async function runHandle() {
  const item = dialog.item
  if (!item) {
    dialog.visible = false
    return
  }

  dialog.loading = true
  busy.value = true
  try {
    await handleReport({
      id: item.id,
      status: dialog.status,
      remark: dialog.remark.trim(),
      remove_content: dialog.status === 1 && dialog.removeContent ? 1 : 0
    })
    toast.success(dialog.status === 1 ? '已标记为属实' : '已驳回')
    dialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    dialog.loading = false
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
/*
 * 基础样式：项目里 .panel-head / .block-title / .chip 这些**都不是全局样式**，
 * 每个页面在自己 scoped 里声明一份（见 views/admin/UserList.vue、views/admin/TagList.vue），
 * 这里照抄同一套口径，保证与其它后台页面视觉统一。
 */
.reports-admin {
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
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.chip-group {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: var(--control-h-sm);
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-family: inherit;
  font-size: 12px;
  line-height: 1.5;
  cursor: pointer;
  transition: all 0.18s;
}
.chip:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.chip.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
}

/* 筛选区：与其它后台页面的「筛选条」一致 */
.report-filter {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.report-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.report-item {
  padding: 14px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.report-item:last-child {
  border-bottom: none;
}
.report-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.report-type {
  padding: 1px 8px;
  font-size: 12px;
  border-radius: 999px;
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.report-status {
  padding: 1px 8px;
  font-size: 12px;
  border-radius: 999px;
}
.report-status.is-pending {
  color: var(--warning);
  background: var(--gold-wash);
}
.report-status.is-done {
  color: var(--success);
  background: rgba(58, 160, 106, 0.12);
}
.report-status.is-rejected {
  color: var(--text-muted);
  background: var(--bg-muted);
}
.report-reason {
  font-size: 13px;
  font-weight: 600;
}
.report-time {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-light);
}
.report-snapshot {
  margin: 8px 0 4px;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius-sm);
  word-break: break-word;
}
/* 证据图片缩略图 */
.report-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 6px 0;
}
.report-image {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: zoom-in;
  transition: transform 0.15s;
}
.report-image:hover {
  transform: scale(1.03);
}
.report-detail,
.report-remark {
  margin: 4px 0;
  font-size: 12px;
  color: var(--text-muted);
}
.report-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin: 6px 0;
}
.meta-text {
  font-size: 12px;
  color: var(--text-light);
}
.report-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.dialog-line {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--text-soft);
}
.dialog-snapshot {
  margin: 0 0 12px;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius-sm);
  word-break: break-word;
}
.dialog-check {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--text);
}
.dialog-hint {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--text-muted);
}
.dialog-remark-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
}
.dialog-remark-label em {
  margin-left: 6px;
  font-size: 12px;
  font-style: normal;
  font-weight: 400;
  color: var(--text-light);
}
.dialog-remark textarea {
  width: 100%;
  font-size: 13px;
  resize: vertical;
}
</style>
