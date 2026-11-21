<template>
  <div class="room-settings-admin">
    <!-- 房间资料 -->
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">群聊资料</h2>
          <p class="block-desc">名称 / 简介 / 头像会实时广播给正在聊天的用户（无需刷新页面）</p>
        </div>
        <button class="btn btn-ghost btn-sm" :disabled="loadingInfo" title="重新读取" aria-label="重新读取" @click="loadInfo">
          <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loadingInfo }" />
        </button>
      </header>

      <div v-if="denied" class="denied-bar">
        <i class="bi bi-slash-circle" aria-hidden="true" />
        仅超级管理员可以修改群聊（当前账号只能查看）
      </div>

      <div v-if="loadingInfo" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else class="form-grid">
        <div class="field">
          <span class="field-label">群聊名称<span class="field-hint">最多 30 字</span></span>
          <FormInput v-model="info.name" class="field-input" type="text" maxlength="30" :disabled="denied" />
        </div>

        <div class="field">
          <span class="field-label">头像地址<span class="field-hint">留空使用默认图标，仅支持 http(s) 或站内附件地址</span></span>
          <div class="avatar-field">
            <img class="avatar-preview" :src="avatarPreview" alt="群聊头像预览" />
            <FormInput
              v-model="info.avatar"
              class="field-input"
              type="text"
              placeholder="https://… 或 /attachment/…"
              :disabled="denied" />
          </div>
        </div>

        <div class="field">
          <span class="field-label">简介<span class="field-hint">最多 200 字</span></span>
          <FormTextarea v-model="info.description" class="field-textarea" rows="3" maxlength="200" :disabled="denied" />
        </div>

        <div class="form-actions">
          <button class="btn btn-primary" :disabled="savingInfo || denied" @click="saveInfo">
            <i class="bi bi-check2" /> {{ savingInfo ? '保存中…' : '保存资料' }}
          </button>
          <span class="meta-text">
            创建于 {{ info.create_time ? formatTime(info.create_time) : '—' }} · 更新于
            {{ info.update_time ? formatTime(info.update_time) : '—' }}
          </span>
        </div>
      </div>
    </section>

    <!-- 自动清空 -->
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">自动清空</h2>
          <p class="block-desc">
            由服务端定时任务执行（app/timer/room.go）：从「上次清空」起算，到期后清理全站聊天记录，不影响私信
          </p>
        </div>
      </header>

      <div v-if="loadingClear" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else class="clear-body">
        <div class="field is-inline">
          <span class="field-label">清空间隔</span>
          <FormSelect variant="pill"
            v-model="clear.days"
            :options="dayOptions"
            icon="bi bi-calendar3"
            placeholder="选择间隔"
            :disabled="denied"
            @change="saveClear" />
        </div>

        <div class="clear-stats">
          <div class="clear-item">
            <span class="clear-label">上次清空</span>
            <span class="clear-value">{{ clear.last_time ? formatTime(clear.last_time) : '尚未清空' }}</span>
          </div>
          <div class="clear-item">
            <span class="clear-label">清理条数</span>
            <span class="clear-value">{{ clear.last_time ? `${clear.last_count} 条` : '—' }}</span>
          </div>
          <div class="clear-item">
            <span class="clear-label">下次清空</span>
            <span class="clear-value">{{ clear.next_time ? formatTime(clear.next_time) : '待服务端开始计时' }}</span>
          </div>
        </div>

        <p class="clear-tip">
          <i class="bi bi-info-circle" aria-hidden="true" />
          {{ clearTip }}
        </p>
      </div>
    </section>
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
/**
 * 群聊设置（/admin/chat/settings）
 *
 * 后端（app/api/controller/room.go、app/model/chat_room.go）：
 * - GET  room/info：房间资料（名称 / 简介 / 头像 / 状态）；
 * - PUT  room/update：只接受 name / description / avatar，名称 ≤30 字、简介 ≤200 字，
 *   头像必须是 http(s) 或站内媒体地址（isRoomMediaUrl）；
 * - GET/PUT room/auto-clear：清空间隔只接受后端给出的选项（1/3/7/30/182/365 天），
 *   真正的清空由定时任务执行，这里只改间隔，不会立刻删数据。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { getRoomInfo, updateRoom, getRoomAutoClear, saveRoomAutoClear } from '@/api/room'
import { formatTime } from '@/utils/time'
import { toast } from '@/utils/toast'

const info = reactive({ name: '', description: '', avatar: '', status: 1, create_time: 0, update_time: 0 })
const clear = reactive({ days: 3, last_time: 0, last_count: 0, next_time: 0 })

const loadingInfo = ref(false)
const loadingClear = ref(false)
const savingInfo = ref(false)
const denied = ref(false)

// 后端 RoomAutoClearOptions：182 天按「半年」、365 天按「一年」展示
const options = ref([])

const dayOptions = computed(() => {
  const source = Array.isArray(options.value) && options.value.length ? options.value : [1, 3, 7, 30, 182, 365]
  return source.map((days) => ({ value: Number(days), label: dayLabel(Number(days)) }))
})

const defaultRoomAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><path d="M22 32h36v22H22z" fill="%238a8a82"/><path d="M14 30l26-12 26 12" stroke="%238a8a82" stroke-width="4" fill="none"/></svg>'

const avatarPreview = computed(() => info.avatar || defaultRoomAvatar)

const clearTip = computed(() => {
  if (!clear.last_time) return '服务端首次运行会把「上次清空时间」设为当前时间，避免一上线就误删历史消息'
  return `距离下次清空约 ${daysText(clear.days)}，如需立即清理请到服务器执行定时任务或在数据库侧处理`
})

function dayLabel(days) {
  if (days === 1) return '每天'
  if (days === 7) return '每周'
  if (days === 30) return '每月'
  if (days === 182) return '每半年'
  if (days === 365) return '每年'
  return `每 ${days} 天`
}

function daysText(days) {
  return dayLabel(Number(days) || 0).replace('每', '')
}

async function loadInfo() {
  loadingInfo.value = true
  try {
    const res = await getRoomInfo()
    if (res?.code === 200) {
      const room = res.data?.room || {}
      info.name = room.name || ''
      info.description = room.description || ''
      info.avatar = room.avatar || ''
      info.status = Number(room.status) || 0
      info.create_time = Number(room.create_time) || 0
      info.update_time = Number(room.update_time) || 0
      // room/info 是登录接口，非超管也能读；is_admin 由服务端算好。
      // 只置 true 不置 false：auto-clear 的 403 可能先返回，别被另一个请求覆盖回去
      if (res.data?.is_admin === false) denied.value = true
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    loadingInfo.value = false
  }
}

async function saveInfo() {
  if (!info.name.trim()) {
    toast.warning('群聊名称不能为空')
    return
  }

  savingInfo.value = true
  try {
    // 这个页面只改公共大厅（id=1），群聊资料由群主在自己房间的管理面板改
    const res = await updateRoom(1, {
      name: info.name.trim(),
      description: info.description.trim(),
      avatar: info.avatar.trim()
    })
    if (res?.code === 200) {
      const room = res.data || {}
      info.name = room.name || info.name
      info.description = room.description || ''
      info.avatar = room.avatar || ''
      info.update_time = Number(room.update_time) || info.update_time
      toast.success('群聊资料已更新')
    }
  } catch {
    /* 拦截器已提示（名称超长 / 头像地址不合法等） */
  } finally {
    savingInfo.value = false
  }
}

async function loadClear() {
  loadingClear.value = true
  try {
    const res = await getRoomAutoClear()
    if (res?.code === 200) {
      const data = res.data || {}
      clear.days = Number(data.days) || 3
      clear.last_time = Number(data.last_time) || 0
      clear.last_count = Number(data.last_count) || 0
      clear.next_time = Number(data.next_time) || 0
      options.value = Array.isArray(data.options) ? data.options.map(Number) : []
    }
  } catch (err) {
    if (Number(err?.code) === 403) denied.value = true
  } finally {
    loadingClear.value = false
  }
}

async function saveClear() {
  const days = Number(clear.days)
  try {
    const res = await saveRoomAutoClear(days)
    if (res?.code === 200) {
      const data = res.data || {}
      clear.days = Number(data.days) || days
      clear.last_time = Number(data.last_time) || 0
      clear.last_count = Number(data.last_count) || 0
      clear.next_time = Number(data.next_time) || 0
      toast.success(`已改为${dayLabel(clear.days)}清空一次`)
    }
  } catch {
    /* 拦截器已提示 */
  }
}

onMounted(() => {
  loadInfo()
  loadClear()
})
</script>

<style scoped>
.room-settings-admin {
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
  line-height: 1.7;
  color: var(--text-muted);
}

.denied-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  padding: 10px 12px;
  font-size: 12px;
  color: var(--danger);
  background: var(--accent-soft);
  border-radius: var(--radius);
}

.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}

/* ---------- 表单 ---------- */
.form-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field.is-inline {
  flex-direction: row;
  align-items: center;
  gap: 12px;
  width: fit-content;
}
.field-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.field-hint {
  font-size: 11px;
  color: var(--text-light);
}
.field-input,
.field-textarea {
  width: 100%;
  padding: 0 12px;
  height: var(--control-h-sm);
  font-family: inherit;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.field-textarea {
  height: auto;
  padding: 8px 12px;
  line-height: 1.7;
  resize: vertical;
}
.field-input:focus,
.field-textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.field-input:disabled,
.field-textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.avatar-field {
  display: flex;
  align-items: center;
  gap: 12px;
}
.avatar-preview {
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
}

.form-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.meta-text {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 自动清空 ---------- */
.clear-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.clear-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}
.clear-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.clear-label {
  font-size: 12px;
  color: var(--text-muted);
}
.clear-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.clear-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-light);
}

.is-spin {
  animation: rs-spin 0.9s linear infinite;
}
@keyframes rs-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .field.is-inline {
    flex-direction: column;
    align-items: flex-start;
    width: 100%;
  }
  .avatar-field {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
