<template>
  <div class="room-groups-admin">
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">群聊管理</h2>
          <p class="block-desc">
            公共大厅与用户自建的群聊：查看群主与成员数、发布或清除公告、解散违规群聊
          </p>
        </div>

        <div class="head-actions">
          <span class="meta-text">共 {{ total }} 个</span>
          <button class="btn btn-sm" :disabled="loading" @click="reload">
            <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loading }" /> 刷新
          </button>
        </div>
      </header>

      <div class="list-filter">
        <div class="search-box">
          <i class="bi bi-search" aria-hidden="true" />
          <FormInput
            v-model="keyword"
            class="search-input"
            type="search"
            placeholder="搜索群聊名称…"
            aria-label="搜索群聊名称"
            @keyup.enter="reload" />
        </div>
        <span class="filter-tip">按创建顺序排列，解散后不可恢复</span>
      </div>

      <p v-if="error" class="denied-bar">
        <i class="bi bi-slash-circle" aria-hidden="true" />{{ error }}
      </p>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState
          :icon="keyword ? 'bi bi-search' : 'bi bi-people'"
          :text="keyword ? '没有匹配的群聊' : '还没有群聊'"
        />
      </div>

      <ul v-else class="group-list">
        <li v-for="item in list" :key="item.id" class="group-row">
          <span class="group-avatar">
            <img v-if="item.avatar" :src="fullUrl(item.avatar)" alt="" loading="lazy" />
            <i v-else :class="item.is_official ? 'bi bi-people-fill' : 'bi bi-chat-square-text'" aria-hidden="true" />
          </span>

          <div class="group-main">
            <div class="group-title">
              <strong>{{ item.name }}</strong>
              <span v-if="item.is_official" class="group-tag is-official">大厅</span>
              <span class="group-tag">{{ joinTypeName(item.join_type) }}</span>
              <span v-if="Number(item.status) !== 1" class="group-tag is-danger">已关闭</span>
              <span class="group-id">#{{ item.id }}</span>
            </div>

            <div class="group-meta">
              <span>
                <i class="bi bi-person" aria-hidden="true" />
                {{ item.is_official ? '官方' : item.owner?.nickname || `用户 #${item.owner_uid}` }}
              </span>
              <span>
                <i class="bi bi-people" aria-hidden="true" />
                {{ item.member_count || 0 }}{{ item.max_members ? ` / ${item.max_members}` : '' }} 人
              </span>
              <span v-if="item.create_time">
                <i class="bi bi-clock" aria-hidden="true" />
                {{ formatTime(item.create_time) }}
              </span>
            </div>

            <p class="group-last">
              <template v-if="item.last_message">
                <template v-if="Number(item.last_message.is_revoked) === 1">[已撤回]</template>
                <template v-else-if="item.last_message.msg_type === 'image'">[图片]</template>
                <template v-else>{{ item.last_message.content }}</template>
              </template>
              <template v-else>还没有消息</template>
            </p>
          </div>

          <div class="group-actions">
            <RouterLink :to="`/chat/room/${item.id}`" class="btn btn-ghost btn-sm" target="_blank">
              <i class="bi bi-box-arrow-up-right" /> 查看
            </RouterLink>
            <button class="btn btn-ghost btn-sm" @click="openNotice(item)">
              <i class="bi bi-megaphone" /> 公告
            </button>
            <button
              v-if="!item.is_official"
              class="btn btn-ghost btn-sm group-remove"
              :disabled="dissolving === item.id"
              @click="askDissolve(item)"
            >
              <i class="bi bi-trash3" /> {{ dissolving === item.id ? '解散中…' : '解散' }}
            </button>
          </div>
        </li>
      </ul>

      <div v-if="total > size" class="pager">
        <span class="meta-text">{{ page }} / {{ pageCount }} 页</span>
        <button class="btn btn-ghost btn-sm" :disabled="page <= 1 || loading" @click="turn(-1)">上一页</button>
        <button class="btn btn-ghost btn-sm" :disabled="page * size >= total || loading" @click="turn(1)">下一页</button>
      </div>
    </section>

    <!-- ==================== 群公告 ==================== -->
    <div v-if="noticeTarget" class="notice-modal">
      <div class="notice-mask" @click="noticeTarget = null" />
      <div class="notice-card">
        <header class="notice-head">
          <span>群公告 · {{ noticeTarget.name }}</span>
          <button class="btn btn-ghost btn-sm" title="关闭" @click="noticeTarget = null">
            <i class="bi bi-x-lg" />
          </button>
        </header>

        <FormTextarea
          v-model="noticeDraft"
          class="notice-textarea"
          rows="6"
          maxlength="500"
          placeholder="最多 500 字，留空即清除公告" />

        <div class="notice-foot">
          <span class="meta-text">{{ noticeDraft.length }} / 500</span>
          <div class="notice-actions">
            <button class="btn btn-ghost" @click="noticeTarget = null">取消</button>
            <button class="btn btn-primary" :disabled="noticeSaving" @click="saveNotice">
              {{ noticeSaving ? '保存中…' : '发布公告' }}
            </button>
          </div>
        </div>
      </div>
    </div>

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
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 群聊管理（/admin/chat/groups）
 *
 * 只对超级管理员开放：room/list?scope=all 在后端用 IsRootAdmin 校验，
 * 解散复用 room/dissolve（超级管理员可解散任意群），公告复用 room/announce。
 * 定时自动清空只作用于公共大厅，用户自建的群不参与（见 app/timer/room.go）。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { announceRoom, dissolveRoom, listRooms } from '@/api/room'
import { toast } from '@/utils/toast'
import { formatTime } from '@/utils/time'
import { getFullUrl } from '@/utils/emoji'

const loading = ref(false)
const error = ref('')
const list = ref([])
const keyword = ref('')
const page = ref(1)
const size = ref(20)
const total = ref(0)
const dissolving = ref(0)

const noticeTarget = ref(null)
const noticeDraft = ref('')
const noticeSaving = ref(false)

const confirm = reactive({
  visible: false,
  title: '',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  target: null
})

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / size.value)))

function fullUrl(url) {
  return getFullUrl(url || '')
}

function joinTypeName(value) {
  return ['自由加入', '需要审批', '仅邀请'][Number(value)] || '自由加入'
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await listRooms({
      scope: 'all',
      keyword: keyword.value.trim(),
      page: page.value,
      size: size.value
    })
    if (res?.code !== 200) {
      error.value = res?.msg || '查询失败'
      return
    }
    list.value = res.data?.data || []
    total.value = Number(res.data?.total) || 0
  } catch {
    error.value = '查询失败，请确认当前账号是超级管理员'
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

function turn(delta) {
  page.value = Math.min(pageCount.value, Math.max(1, page.value + delta))
  load()
}

function openNotice(item) {
  noticeTarget.value = item
  noticeDraft.value = item.notice || ''
}

async function saveNotice() {
  if (!noticeTarget.value || noticeSaving.value) return

  noticeSaving.value = true
  try {
    const res = await announceRoom(noticeTarget.value.id, noticeDraft.value.trim())
    if (res?.code !== 200) return

    noticeTarget.value.notice = res.data?.notice || ''
    toast.success(noticeDraft.value.trim() ? '公告已发布' : '公告已清除')
    noticeTarget.value = null
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    noticeSaving.value = false
  }
}

function askDissolve(item) {
  confirm.visible = true
  confirm.title = '解散群聊'
  confirm.message = `解散「${item.name}」后，成员与入群申请都会被清空，消息也不再对成员可见。此操作不可恢复。`
  confirm.confirmText = '解散'
  confirm.danger = true
  confirm.target = item
}

async function runConfirm() {
  const item = confirm.target
  if (!item) return

  confirm.loading = true
  dissolving.value = item.id

  try {
    const res = await dissolveRoom(item.id)
    if (res?.code !== 200) return

    toast.success('群聊已解散')
    list.value = list.value.filter((row) => row.id !== item.id)
    total.value = Math.max(0, total.value - 1)
    confirm.visible = false
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    confirm.loading = false
    dissolving.value = 0
  }
}

onMounted(load)
</script>

<style scoped>
/* 这些基础类在项目里是「各页 scoped 自己定义」而不是全局样式（styles.css 只全局了 .card / .btn / .spinner），
   漏掉就会出现标题与按钮上下堆叠、搜索框缩成一小条 —— 这里按其它后台页的写法补齐。 */
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
  color: var(--text);
}
.block-desc {
  margin: 4px 0 0;
  max-width: 76ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-light);
}
.head-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 10px;
}
.list-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin: 2px 0 14px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.search-box {
  position: relative;
  width: 220px;
}
.search-box .bi {
  position: absolute;
  top: 50%;
  left: 10px;
  font-size: 13px;
  color: var(--text-light);
  transform: translateY(-50%);
}
.search-input {
  width: 100%;
  height: var(--control-h-sm, 32px);
  padding: 0 12px 0 30px;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.filter-tip {
  font-size: 12px;
  color: var(--text-light);
}
.denied-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 12px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--danger);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.empty-row {
  padding: 12px 0 4px;
}
.is-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 28px 0;
  font-size: 12px;
  color: var(--text-light);
}
.meta-text {
  font-size: 12px;
  color: var(--text-light);
}
.is-spin {
  animation: group-spin 0.9s linear infinite;
}
@keyframes group-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ==================== 列表 ==================== */
.group-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 4px 0 0;
  padding: 0;
  list-style: none;
}
.group-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px;
  background: var(--bg);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  transition: border-color var(--dur-fast) var(--ease), background var(--dur-fast) var(--ease);
}
.group-row:hover {
  border-color: var(--border);
  background: var(--bg-muted);
}
.group-avatar {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  overflow: hidden;
  font-size: 17px;
  color: var(--text-light);
  background: var(--bg-muted);
  border-radius: 10px;
}
.group-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.group-main {
  flex: 1;
  min-width: 0;
}
.group-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  line-height: 1.5;
}
.group-title strong {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.group-tag {
  padding: 0 6px;
  font-size: 10.5px;
  line-height: 17px;
  color: var(--text-light);
  background: var(--bg-muted);
  border-radius: 4px;
}
.group-row:hover .group-tag {
  background: var(--bg);
}
.group-tag.is-official {
  color: var(--primary);
  background: var(--accent-wash);
}
.group-tag.is-danger {
  color: var(--danger);
}
.group-id {
  font-size: 11px;
  color: var(--text-light);
}
.group-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
}
.group-meta span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.group-last {
  margin: 8px 0 0;
  padding-left: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  border-left: 2px solid var(--border);
}
.group-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  padding-top: 2px;
}
.group-remove {
  color: var(--danger);
}
.group-remove:hover:not(:disabled) {
  color: var(--danger);
  background: var(--bg);
}
.pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 14px;
}

/* ==================== 公告弹窗 ==================== */
.notice-modal {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}
.notice-mask {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
}
.notice-card {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(440px, 100%);
  padding: 16px 18px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md, 0 18px 48px rgba(0, 0, 0, 0.2));
}
.notice-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.notice-textarea {
  width: 100%;
  padding: 10px 12px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  outline: none;
  resize: vertical;
}
.notice-textarea:focus {
  border-color: var(--primary);
}
.notice-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 12px;
}
.notice-actions {
  display: flex;
  gap: 8px;
}

/* ==================== 窄屏 ==================== */
@media (max-width: 720px) {
  .group-row {
    flex-wrap: wrap;
  }
  .group-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
