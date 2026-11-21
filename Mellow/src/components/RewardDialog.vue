<template>
  <ConfirmDialog
    :visible="visible"
    title="打赏支持"
    :message="dialogMessage"
    :confirm-text="confirmText"
    :loading="sending"
    @update:visible="(v) => emit('update:visible', v)"
    @confirm="onConfirm"
    @cancel="onCancel"
  >
    <div class="reward-integral">
      <div class="reward-balance">
        <span>我的积分</span>
        <strong>{{ isLogged ? balance : '未登录' }}</strong>
        <span v-if="summary.count" class="reward-summary">
          已获 {{ summary.count }} 次打赏 · 共 {{ summary.integral }} 积分
        </span>
      </div>

      <div class="reward-presets">
        <button
          v-for="item in presets"
          :key="item"
          type="button"
          class="chip"
          :class="{ 'is-active': value === item }"
          :disabled="item > summary.max"
          @click="value = item"
        >{{ item }}</button>
      </div>

      <label class="reward-field">
        <span class="reward-field-label">打赏积分</span>
        <FormInput
          v-model.number="value"
          type="number"
          class="reward-field-input"
          :min="summary.min"
          :max="summary.max"
          step="1"
          placeholder="输入积分数量" />
      </label>

      <label class="reward-field">
        <span class="reward-field-label">
          留言
          <em class="reward-field-tag">选填</em>
        </span>
        <FormTextarea
          v-model="message"
          class="reward-field-textarea"
          rows="2"
          :maxlength="summary.message_max"
          :placeholder="`给${toName || '作者'}说点什么…`" />
        <span class="reward-field-hint">{{ message.length }}/{{ summary.message_max }}</span>
      </label>

      <p class="reward-tip">
        打赏会从你的积分余额中扣除，并全额转入{{ toName || '对方' }}的账户，不可撤回。
      </p>
    </div>
  </ConfirmDialog>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 积分打赏弹窗（文章 / 动态 / 评论 / 用户主页共用）
 *
 * 用法：
 *   <RewardDialog
 *     :visible="reward.visible"
 *     bind-type="article"
 *     :bind-id="article.id"
 *     :to-uid="article.uid"
 *     :to-name="author.nickname"
 *     @update:visible="(v) => (reward.visible = v)"
 *     @success="onRewardSuccess"
 *   />
 *
 * 说明：
 * - 积分打赏走 tip/send，接收者由后端按「内容作者」解析，前端传 to-uid 仅用于校验；
 * - 打赏成功后 emit success({ value, balance })，调用方可据此刷新余额 / 打赏统计。
 */
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { getTipSummary, sendTip } from '@/api/tip'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 打赏对象：article / moments / comment / page / user
  bindType: { type: String, default: 'user' },
  bindId: { type: [Number, String], default: 0 },
  // 接收者 uid（内容作者 / 主页用户），后端会再校验一次
  toUid: { type: [Number, String], default: 0 },
  toName: { type: String, default: '' }
})

const emit = defineEmits(['update:visible', 'success', 'cancel'])

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const isLogged = computed(() => userStore.isLogged)

const value = ref(10)
const message = ref('')
const sending = ref(false)

// 概览：count / integral / balance / min / max / message_max
const summary = ref({ count: 0, integral: 0, balance: 0, min: 1, max: 100000, message_max: 100 })

const balance = computed(() => summary.value.balance || 0)

const presets = [1, 5, 10, 20, 50, 100]

const dialogMessage = computed(() =>
  props.toName ? `感谢你对 ${props.toName} 的支持` : '感谢你的支持'
)

const confirmText = computed(() => {
  const amount = Number(value.value) || 0
  return amount > 0 ? `打赏 ${amount} 积分` : '打赏'
})

// 打开时加载概览（顺带拿到当前余额）；关闭时还原页面滚动
watch(
  () => props.visible,
  async (val) => {
    if (!val) {
      document.body.style.overflow = ''
      return
    }

    value.value = 10
    message.value = ''
    document.body.style.overflow = 'hidden'

    await loadSummary()
  }
)

async function loadSummary() {
  const bindId = Number(props.bindId) || 0
  if (!bindId) return

  try {
    const res = await getTipSummary(props.bindType, bindId)
    const data = res?.data || {}
    summary.value = {
      count: Number(data.count) || 0,
      integral: Number(data.integral) || 0,
      balance: Number(data.balance) || 0,
      min: Number(data.min) || 1,
      max: Number(data.max) || 100000,
      message_max: Number(data.message_max) || 100
    }
  } catch {
    // 概览加载失败不阻塞打赏：用默认上下限继续
  }
}

function onCancel() {
  document.body.style.overflow = ''
  emit('cancel')
}

function close() {
  document.body.style.overflow = ''
  emit('update:visible', false)
}

/** 未登录：提示并带 redirect 跳登录页（与 ArticleDetail / MomentItem 的拦截方式一致） */
function requireLogin() {
  if (isLogged.value) return true
  toast.warning('请先登录后再打赏')
  router.push({ name: 'login', query: { redirect: route.fullPath } })
  return false
}

async function onConfirm() {
  if (!requireLogin()) return

  const amount = Number(value.value)
  if (!Number.isInteger(amount) || amount <= 0) {
    toast.warning('请输入正确的打赏积分')
    return
  }
  if (amount < summary.value.min) {
    toast.warning(`打赏积分不能少于 ${summary.value.min} 分`)
    return
  }
  if (amount > summary.value.max) {
    toast.warning(`单次打赏不能超过 ${summary.value.max} 分`)
    return
  }
  if (amount > balance.value) {
    toast.warning('积分不足，请先获取积分')
    return
  }

  sending.value = true
  try {
    const res = await sendTip({
      bind_type: props.bindType,
      bind_id: Number(props.bindId) || 0,
      to_uid: Number(props.toUid) || 0,
      value: amount,
      message: message.value.trim()
    })

    const data = res?.data || {}
    summary.value.balance = Number(data.balance) || 0
    summary.value.count += 1
    summary.value.integral += amount

    toast.success(`已打赏 ${amount} 积分，感谢支持！`)
    emit('success', { value: amount, balance: summary.value.balance })
    close()
  } catch {
    /* 请求拦截器已提示错误（积分不足 / 不能打赏自己等） */
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.reward-balance {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-bottom: 10px;
  font-size: 12px;
  color: var(--text-muted);
}
.reward-balance strong {
  font-size: 16px;
  color: var(--primary);
}
.reward-summary {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-light);
}

.reward-presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}
.chip {
  padding: 4px 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-family: inherit;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.18s;
}
.chip:hover:not(:disabled) {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.chip.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
}
.chip:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.reward-field {
  display: block;
  margin-bottom: 10px;
}
.reward-field-label {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.reward-field-tag {
  padding: 0 6px;
  font-size: 11px;
  font-style: normal;
  font-weight: 500;
  border-radius: 999px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.reward-field-input,
.reward-field-textarea {
  width: 100%;
  font-size: 13px;
  resize: vertical;
}
.reward-field-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-light);
  text-align: right;
}

.reward-tip {
  margin: 10px 0 0;
  padding-top: 10px;
  border-top: 1px dashed var(--border-soft);
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
}
</style>
