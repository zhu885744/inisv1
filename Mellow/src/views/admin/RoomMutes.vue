<template>
  <div class="room-mutes-admin">
    <!-- 添加禁言 -->
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">添加禁言</h2>
          <p class="block-desc">
            一次禁言同时作用于私信与公共群聊（与现有禁言体系同一张表）；管理员之间不能互相禁言，到期自动解除
          </p>
        </div>
        <button class="btn btn-ghost btn-sm" :disabled="loading" title="刷新名单" aria-label="刷新名单" @click="load">
          <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loading }" />
        </button>
      </header>

      <div class="mute-form">
        <label class="field">
          <span class="field-label">用户 UID</span>
          <FormInput v-model="form.uid" class="field-input" type="number" min="1" placeholder="要禁言的用户 ID" />
        </label>

        <label class="field">
          <span class="field-label">时长</span>
          <FormSelect variant="pill" v-model="form.minutes" :options="durationOptions" />
        </label>

        <label class="field is-grow">
          <span class="field-label">原因（可选）</span>
          <FormInput
            v-model="form.reason"
            class="field-input"
            type="text"
            maxlength="100"
            placeholder="例如：刷屏 / 人身攻击" />
        </label>

        <button class="btn btn-primary mute-submit" :disabled="muting" @click="submitMute">
          <i class="bi bi-mic-mute" /> {{ muting ? '处理中…' : '禁言' }}
        </button>
      </div>
    </section>

    <!-- 禁言名单 -->
    <section class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">禁言名单</h2>
          <p class="block-desc">
            生效中的禁言（已到期的自动隐藏）
            <template v-if="groups.length > 0"> · 共 {{ groups.length }} 人</template>
          </p>
        </div>
        <FormSelect variant="pill"
          v-model="scope"
          :options="scopeOptions"
          icon="bi bi-filter"
          placeholder="全部范围"
          @change="load" />
      </header>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!groups.length" class="empty-row">
        <EmptyState icon="bi bi-mic" :text="scope ? '该范围下没有生效中的禁言' : '当前没有被禁言的用户'" />
      </div>

      <table v-else class="mute-table">
        <thead>
          <tr>
            <th>用户</th>
            <th class="cell-scope">范围</th>
            <th class="cell-until">到期时间</th>
            <th>原因</th>
            <th class="cell-operator">操作人</th>
            <th class="cell-time">禁言时间</th>
            <th class="cell-act">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in groups" :key="item.uid">
            <td>
              <div class="user-cell">
                <img class="user-avatar" :src="item.user?.avatar || defaultAvatar" alt="" loading="lazy" />
                <span class="user-body">
                  <span class="user-name">{{ item.user?.nickname || `用户 #${item.uid}` }}</span>
                  <span class="meta-text">#{{ item.uid }}</span>
                </span>
              </div>
            </td>
            <td class="cell-scope">
              <span v-for="s in item.scopes" :key="s" class="scope-chip" :class="`is-${s}`">
                <i :class="s === 'room' ? 'bi bi-people-fill' : 'bi bi-envelope-fill'" aria-hidden="true" />
                {{ scopeName(s) }}
              </span>
            </td>
            <td class="cell-until">
              <span class="until-chip" :class="{ 'is-forever': item.until === 0 }">
                {{ untilText(item.until) }}
              </span>
            </td>
            <td>
              <span class="reason-text" :title="item.reason">{{ item.reason || '—' }}</span>
            </td>
            <td class="cell-operator">
              <span class="meta-text">{{ item.operator?.nickname || (item.operator_uid ? `#${item.operator_uid}` : '—') }}</span>
            </td>
            <td class="cell-time">
              <span class="meta-text">{{ formatTime(item.create_time) }}</span>
            </td>
            <td class="cell-act">
              <button
                class="btn btn-ghost btn-sm"
                title="解除该用户的私信 + 群聊禁言"
                aria-label="解除禁言"
                :disabled="busy"
                @click="askUnmute(item)"
              >
                <i class="bi bi-mic" /> 解除
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      confirm-text="解除禁言"
      :loading="confirm.loading"
      @confirm="runUnmute"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
/**
 * 禁言名单（/admin/chat/mutes）
 *
 * 复用现有禁言体系（app/model/chat_room.go 的 ChatRoomMutes）：
 *   - 群聊禁言：room_id = PublicRoomId(1)；
 *   - 私信禁言：room_id = ChatMuteDM(0)；
 * 一次禁言写两条记录，两处一起失效，解除也在这里一次解开。
 *
 * 后台走 chat-admin 的权限点（chat-admin/mutes、mute、unmute），
 * 不要求超级管理员；群聊页面的「管理」面板仍用 room/mute（仅超管）。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { listChatMutes, muteChatUser, unmuteChatUser } from '@/api/chat-admin'
import { formatTime } from '@/utils/time'
import { toast } from '@/utils/toast'

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const durationOptions = [
  { value: 10, label: '10 分钟' },
  { value: 60, label: '1 小时' },
  { value: 1440, label: '1 天' },
  { value: 10080, label: '7 天' },
  { value: 43200, label: '30 天' },
  { value: 0, label: '永久' }
]

const scopeOptions = [
  { value: '', label: '全部范围' },
  { value: 'dm', label: '私信' },
  { value: 'room', label: '群聊' }
]

const list = ref([])
const scope = ref('')
const loading = ref(false)
const busy = ref(false)
const muting = ref(false)

const form = reactive({ uid: '', minutes: 60, reason: '' })

const confirm = reactive({ visible: false, title: '解除禁言', message: '', loading: false, target: null })

function scopeName(value) {
  return value === 'room' ? '群聊' : '私信'
}

function untilText(until) {
  const value = Number(until) || 0
  return value === 0 ? '永久' : `${formatTime(value)} 解除`
}

// 同一个用户会有 dm / room 两条记录：按 uid 合并成一行，避免同一人出现两次
const groups = computed(() => {
  const map = new Map()

  for (const row of list.value) {
    const uid = Number(row.uid)
    let item = map.get(uid)
    if (!item) {
      item = {
        uid,
        user: row.user,
        operator: row.operator,
        operator_uid: Number(row.operator_uid) || 0,
        scopes: [],
        until: null,
        reason: '',
        create_time: Number(row.create_time) || 0
      }
      map.set(uid, item)
    }

    if (!item.scopes.includes(row.scope)) item.scopes.push(row.scope)

    // until：0 表示永久，优先级最高
    const value = Number(row.until) || 0
    item.until = item.until === null ? value : item.until === 0 || value === 0 ? 0 : Math.max(item.until, value)
    if (!item.reason) item.reason = row.reason || ''
  }

  return [...map.values()]
})

async function load() {
  loading.value = true
  try {
    const res = await listChatMutes({ scope: scope.value })
    list.value = res.data?.data || []
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

async function submitMute() {
  const uid = Number(form.uid)
  if (!uid || uid <= 0) {
    toast.warning('请填写要禁言的用户 UID')
    return
  }

  muting.value = true
  try {
    await muteChatUser(uid, Number(form.minutes) || 0, form.reason.trim())
    toast.success(Number(form.minutes) === 0 ? '已永久禁言（私信 + 群聊）' : '已禁言（私信 + 群聊）')
    form.uid = ''
    form.reason = ''
    await load()
  } catch {
    /* 拦截器已提示（自己 / 管理员 / 用户不存在等） */
  } finally {
    muting.value = false
  }
}

function askUnmute(item) {
  confirm.target = item
  confirm.message = `确定解除「${item.user?.nickname || `用户 #${item.uid}`}」的禁言吗？私信与群聊会同时恢复。`
  confirm.visible = true
}

async function runUnmute() {
  const item = confirm.target
  if (!item) {
    confirm.visible = false
    return
  }

  confirm.loading = true
  busy.value = true
  try {
    // 不传 scope：两个作用域一起解除
    await unmuteChatUser(item.uid)
    toast.success('已解除禁言')
    confirm.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    confirm.loading = false
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.room-mutes-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

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
  line-height: 1.7;
  color: var(--text-muted);
}

/* ---------- 表单 ---------- */
.mute-form {
  display: flex;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 170px;
}
.field.is-grow {
  flex: 1;
  min-width: 200px;
}
.field-label {
  font-size: 12px;
  color: var(--text-muted);
}
.field-input {
  width: 100%;
  height: var(--control-h-sm);
  padding: 0 12px;
  font-family: inherit;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.field-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.mute-submit {
  height: var(--control-h-sm);
}

/* ---------- 名单 ---------- */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}

.mute-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.mute-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.mute-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.mute-table tr:last-child td {
  border-bottom: none;
}
.mute-table tbody tr:hover td {
  background: var(--bg-muted);
}
.cell-scope {
  width: 170px;
}
.cell-until {
  width: 160px;
}
.cell-operator {
  width: 120px;
}
.cell-time {
  width: 140px;
}
.cell-act {
  width: 100px;
  text-align: right;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.user-avatar {
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.user-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.user-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}
.meta-text {
  font-size: 12px;
  color: var(--text-muted);
}

.scope-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-right: 4px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 999px;
}
.scope-chip.is-dm {
  color: var(--primary);
  background: var(--accent-soft);
}
.scope-chip.is-room {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}

.until-chip {
  display: inline-block;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--warning);
  background: var(--gold-wash);
  border-radius: 999px;
}
.until-chip.is-forever {
  color: var(--danger);
  background: var(--accent-soft);
}
.reason-text {
  display: inline-block;
  max-width: 260px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.is-spin {
  animation: rm-spin 0.9s linear infinite;
}
@keyframes rm-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .list-head {
    flex-direction: column;
  }
  .field,
  .field.is-grow {
    width: 100%;
  }
  .mute-table {
    display: block;
    overflow-x: auto;
  }
}
</style>
