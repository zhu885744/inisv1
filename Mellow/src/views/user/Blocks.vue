<template>
  <div class="blocks-page">
    <SectionTitle title="拉黑与举报" />

    <div class="tab-bar">
      <button :class="['tab', tab === 'blocks' && 'is-active']" @click="switchTab('blocks')">
        我的拉黑
      </button>
      <button :class="['tab', tab === 'reports' && 'is-active']" @click="switchTab('reports')">
        我的举报
      </button>
    </div>

    <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

    <!-- 我的拉黑 -->
    <template v-else-if="tab === 'blocks'">
      <EmptyState v-if="!blocks.length" icon="bi bi-shield-slash" text="还没有拉黑任何人" />
      <ul v-else class="blocks-list">
        <li v-for="item in blocks" :key="item.id" class="block-item">
          <router-link :to="`/author/${item.block_uid}`" class="block-link">
            <img :src="item.user?.avatar || defaultAvatar" class="block-avatar" alt="" loading="lazy" />
            <div class="block-info">
              <span class="block-name">{{ item.user?.nickname || '匿名用户' }}</span>
              <span class="block-sub">
                {{ fromNow(item.create_time) }}拉黑
                <template v-if="item.reason"> · {{ item.reason }}</template>
              </span>
            </div>
          </router-link>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="remove(item)">解除拉黑</button>
        </li>
      </ul>
      <p class="page-tip">
        拉黑是双向的：彼此的内容都不会再出现在对方的动态流里，也不能互相私信；
        对方不会收到任何提示。
      </p>
    </template>

    <!-- 我的举报 -->
    <template v-else>
      <EmptyState v-if="!reports.length" icon="bi bi-flag" text="还没有提交过举报" />
      <ul v-else class="reports-list">
        <li v-for="item in reports" :key="item.id" class="report-item">
          <div class="report-head">
            <span class="report-type">{{ item.target_type_text }}</span>
            <span class="report-status" :class="statusClass(item.status)">{{ item.status_text }}</span>
            <span class="report-reason">{{ item.reason_text }}</span>
            <time class="report-time">{{ fromNow(item.create_time) }}</time>
          </div>
          <p class="report-snapshot">{{ item.snapshot || '（无内容快照）' }}</p>
          <!-- 我提交的证据图片（仅管理员可见，这里回显方便自己核对） -->
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
          <p v-if="item.handle_remark" class="report-remark">处理说明：{{ item.handle_remark }}</p>
          <div class="report-actions">
            <button
              v-if="Number(item.status) === 0"
              class="btn btn-sm btn-ghost"
              :disabled="busy"
              @click="cancel(item)"
            >撤回举报</button>
            <span v-else class="report-done">已处理完成</span>
          </div>
        </li>
      </ul>
      <p class="page-tip">
        举报提交后只有管理员能看到，被举报人不会收到通知；处理结果会以站内消息告知你。
        同一内容在「待处理」期间无法重复举报，每人每天最多 20 条。
      </p>
    </template>
  </div>
</template>

<script setup>
/**
 * 用户中心「拉黑与举报」
 *
 * 两个 Tab：
 *   - 我的拉黑：我主动拉黑的人（可解除）。不展示「谁拉黑了我」——
 *     暴露这个只会制造冲突，而 Feed / 私信的屏蔽本来就是双向的
 *   - 我的举报：我提交的举报与处理进展，待处理的可撤回
 */
import { ref, onMounted } from 'vue'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import { myBlocks, unblockUser } from '@/api/user-blocks'
import { cancelReport, myReports } from '@/api/report'
import { fromNow } from '@/utils/time'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const tab = ref('blocks')
const loading = ref(false)
const busy = ref(false)
const blocks = ref([])
const reports = ref([])

function statusClass(value) {
  const status = Number(value)
  if (status === 0) return 'is-pending'
  if (status === 1) return 'is-done'
  return 'is-rejected'
}

async function loadBlocks() {
  const res = await myBlocks({ page: 1, limit: 50 })
  blocks.value = res.data?.data || []
}

async function loadReports() {
  const res = await myReports({ page: 1, limit: 50 })
  reports.value = res.data?.data || []
}

async function load() {
  loading.value = true
  try {
    if (tab.value === 'blocks') await loadBlocks()
    else await loadReports()
  } catch {
    if (tab.value === 'blocks') blocks.value = []
    else reports.value = []
  } finally {
    loading.value = false
  }
}

function switchTab(value) {
  if (tab.value === value) return
  tab.value = value
  load()
}

async function remove(item) {
  if (busy.value) return
  busy.value = true
  try {
    await unblockUser(item.block_uid)
    blocks.value = blocks.value.filter((row) => row.block_uid !== item.block_uid)
    toast.success('已解除拉黑')
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

async function cancel(item) {
  if (busy.value) return
  busy.value = true
  try {
    await cancelReport(item.id)
    reports.value = reports.value.filter((row) => row.id !== item.id)
    toast.success('已撤回举报')
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
/* 基础样式：.is-loading / .block-title 这类都不是全局样式，各页面自带一份（与其它用户中心页一致） */
.blocks-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.tab-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.tab {
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--text-soft);
  font-family: inherit;
  font-size: 14px;
  cursor: pointer;
}
.tab.is-active {
  color: var(--primary);
  font-weight: 600;
  border-bottom: 2px solid var(--primary);
}
.blocks-list,
.reports-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.block-item,
.report-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.block-item:last-child,
.report-item:last-child {
  border-bottom: none;
}
.block-link {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text);
}
.block-avatar {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  object-fit: cover;
  border: 1px solid var(--border);
  flex-shrink: 0;
}
.block-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.block-name {
  font-size: 14px;
  font-weight: 600;
}
.block-sub {
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.report-item {
  display: block;
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
.report-remark {
  margin: 4px 0;
  font-size: 12px;
  color: var(--text-muted);
}
/* 证据图片缩略图（与后台举报管理同一视觉） */
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
}
.report-actions {
  margin-top: 8px;
}
.report-done {
  font-size: 12px;
  color: var(--text-light);
}
.page-tip {
  margin-top: 14px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-light);
}
</style>
