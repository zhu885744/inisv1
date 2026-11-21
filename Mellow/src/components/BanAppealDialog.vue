<template>
  <Teleport to="body">
    <Transition name="ban-fade">
      <div v-if="visible" class="ban-overlay" @click.self="hide">
        <div class="ban-dialog">
          <div class="ban-header">
            <span class="ban-title">
              <i class="bi bi-exclamation-octagon" /> {{ title }}
            </span>
          </div>

          <div class="ban-body">
            <ul class="ban-info">
              <li class="ban-row">
                <span class="ban-label">封禁原因</span>
                <span class="ban-value">{{ record.reason || '未说明' }}</span>
              </li>
              <li class="ban-row">
                <span class="ban-label">限制范围</span>
                <span class="ban-value">{{ typeNames }}</span>
              </li>
              <li class="ban-row">
                <span class="ban-label">封禁时长</span>
                <span class="ban-value danger">{{ durationText }}</span>
              </li>
              <li class="ban-row">
                <span class="ban-label">封禁时间</span>
                <span class="ban-value">{{ formatDate(record.ban_time) }}</span>
              </li>
              <li v-if="Number(record.expires_at) > 0" class="ban-row">
                <span class="ban-label">预计解封</span>
                <span class="ban-value">{{ formatDate(record.expires_at) }}</span>
              </li>
              <li class="ban-row">
                <span class="ban-label">违规次数</span>
                <span class="ban-value">{{ violationNum }} 次</span>
              </li>
            </ul>

            <!-- 查看他人主页：只展示封禁信息，不出现申诉表单（申诉只能本人发起） -->
            <p v-if="!appealable" class="ban-tip is-muted">
              <i class="bi bi-info-circle" /> 该账号的封禁申诉只能由账号本人发起
            </p>

            <!-- 禁止申诉 -->
            <p v-else-if="Number(record.ban_appeal) === 1" class="ban-tip is-danger">
              <i class="bi bi-slash-circle" /> 管理员已禁止该封禁申诉
            </p>
            <p v-else-if="violationNum >= 5" class="ban-tip is-danger">
              <i class="bi bi-slash-circle" /> 该封禁为永久封禁且违规次数已达上限，禁止申诉
            </p>

            <!-- 申诉已提交：展示进度 -->
            <p v-else-if="appealed" class="ban-tip is-ok">
              <i class="bi bi-clock-history" /> {{ appealStatusText }}
            </p>
            <p v-else-if="record.appeal_reply" class="ban-tip is-danger">
              <i class="bi bi-x-circle" /> 申诉被驳回：{{ record.appeal_reply }}
            </p>

            <!-- 申诉表单 -->
            <template v-else>
              <p class="ban-appeal-label">申诉说明</p>
              <FormTextarea
                v-model="content"
                class="ban-textarea"
                rows="4"
                maxlength="500"
                placeholder="请说明你的申诉理由，管理员会尽快处理…" />
              <div class="ban-count">{{ content.length }}/500</div>
            </template>
          </div>

          <div class="ban-foot">
            <button type="button" class="btn btn-sm" @click="hide">我知道了</button>
            <button
              v-if="canAppeal"
              type="button"
              class="btn btn-sm btn-primary"
              :disabled="submitting"
              @click="submit"
            >{{ submitting ? '提交中...' : '提交申诉' }}</button>
            <router-link
              v-else
              to="/blackroom"
              class="btn btn-sm btn-ghost"
              @click="hide"
            >查看小黑屋</router-link>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import { ref, computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { storeToRefs } from 'pinia'
import { submitAppeal } from '@/api/users'
import { formatDate } from '@/utils/time'
import { toast } from '@/utils/toast'

const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const visible = ref(false)
const submitting = ref(false)
const content = ref('')

/**
 * 两个使用场景：
 * 1. 全局（MainLayout）：不传参，封禁信息取当前登录态 user.result.ban（后端 Users.banInfo 提供），
 *    此时就是「自己被封禁」，可以申诉；
 * 2. 用户主页（Author.vue）：传 ban（被查看用户的 result.ban）与 appealable=false，
 *    只展示对方的封禁信息，不提供申诉（申诉只能由被封禁的本人发起）。
 */
const props = defineProps({
  ban: { type: Object, default: null },
  appealable: { type: Boolean, default: true }
})

const banInfo = computed(() => props.ban || user.value?.result?.ban || null)
const record = computed(() => banInfo.value?.record || {})

// 查看他人主页时用「该账号」，自己则是「账号」
const title = computed(() => (props.appealable ? '账号已被封禁' : '该账号已被封禁'))

const violationNum = computed(() => Number(record.value.violation_num) || 0)
// status: 0 生效中 / 3 申诉中 / 4 申诉通过 / 5 申诉驳回
const appealed = computed(() => Number(record.value.status) !== 0)
const canAppeal = computed(
  () =>
    // 只有被封禁的本人能申诉（查看他人主页时 appealable=false）
    props.appealable &&
    !appealed.value &&
    Number(record.value.ban_appeal) !== 1 &&
    violationNum.value < 5 &&
    Number(record.value.id) > 0
)

const typeNames = computed(() => {
  const types = record.value.result?.ban_types || []
  return types.length ? types.map((t) => t.name).join('、') : '全面封禁'
})

const durationText = computed(() => {
  const d = Number(record.value.duration) || 0
  return d === 0 ? '永久封禁' : `${d} 天`
})

const appealStatusText = computed(() => {
  switch (Number(record.value.status)) {
    case 3:
      return '申诉审核中，请耐心等待'
    case 4:
      return '申诉已通过，账号已恢复'
    case 5:
      return '申诉被驳回'
    default:
      return '申诉审核中'
  }
})

function show() {
  visible.value = true
}

function hide() {
  visible.value = false
}

async function submit() {
  if (!content.value.trim()) {
    toast.warning('请填写申诉说明')
    return
  }
  submitting.value = true
  try {
    await submitAppeal(record.value.id, content.value.trim())
    toast.success('申诉已提交，请耐心等待管理员审核')
    // 刷新登录态，同步最新的封禁/申诉状态
    await userStore.checkLoginState()
    hide()
  } catch {} finally {
    submitting.value = false
  }
}

defineExpose({ show, hide })
</script>

<style scoped>
.ban-overlay {
  position: fixed;
  inset: 0;
  z-index: 9500;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.ban-dialog {
  width: 100%;
  max-width: 460px;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}
.ban-header {
  padding: 16px 20px;
  background: linear-gradient(135deg, rgba(217, 84, 77, 0.14), rgba(217, 84, 77, 0.05));
  border-bottom: 1px solid var(--border-soft);
}
.ban-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--danger);
}
.ban-body {
  padding: 18px 20px;
  overflow-y: auto;
}

.ban-info {
  list-style: none;
  margin: 0 0 14px;
  padding: 12px 14px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.ban-row {
  display: flex;
  gap: 12px;
  font-size: 13px;
  line-height: 1.9;
}
.ban-label {
  flex-shrink: 0;
  width: 68px;
  color: var(--text-muted);
}
.ban-value {
  color: var(--text);
  word-break: break-word;
}
.ban-value.danger {
  color: var(--danger);
  font-weight: 600;
}

.ban-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0 0 12px;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.7;
  border-radius: var(--radius-sm);
}
.ban-tip.is-danger {
  background: rgba(217, 84, 77, 0.1);
  color: var(--danger);
}
.ban-tip.is-ok {
  background: rgba(108, 154, 77, 0.12);
  color: var(--success);
}
.ban-tip.is-muted {
  background: var(--bg-muted);
  color: var(--text-muted);
}

.ban-appeal-label {
  margin: 0 0 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
}
.ban-textarea {
  width: 100%;
  padding: 9px 12px;
  font-size: 13px;
  font-family: inherit;
  line-height: 1.7;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg);
  color: var(--text);
  resize: vertical;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.ban-textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.ban-count {
  margin-top: 4px;
  text-align: right;
  font-size: 11px;
  color: var(--text-muted);
}

.ban-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-soft);
}

.ban-fade-enter-active,
.ban-fade-leave-active {
  transition: opacity 0.25s;
}
.ban-fade-enter-active .ban-dialog,
.ban-fade-leave-active .ban-dialog {
  transition: transform 0.25s, opacity 0.25s;
}
.ban-fade-enter-from,
.ban-fade-leave-to {
  opacity: 0;
}
.ban-fade-enter-from .ban-dialog,
.ban-fade-leave-to .ban-dialog {
  transform: scale(0.95) translateY(-10px);
  opacity: 0;
}
</style>
