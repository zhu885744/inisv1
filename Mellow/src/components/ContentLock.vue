<template>
  <div class="content-lock" :class="{ 'is-compact': compact }">
    <i
      class="lock-icon bi"
      :class="state === 'pay' ? 'bi-lock-fill' : 'bi-box-arrow-in-right'"
      aria-hidden="true"
    />
    <div class="lock-main">
      <div class="lock-title">{{ title }}</div>
      <div class="lock-desc">{{ desc }}</div>
      <div class="lock-actions">
        <!-- 未登录，或后端判定需要登录（本地登录态可能已过期）：统一引导登录 -->
        <template v-if="state !== 'pay' || !logged">
          <button type="button" class="btn btn-primary btn-sm" @click="goLogin">登录查看</button>
          <button type="button" class="btn btn-sm" @click="goRegister">注册</button>
        </template>
        <template v-else>
          <button type="button" class="btn btn-primary btn-sm" :disabled="paying" @click="pay">
            {{ paying ? '兑换中...' : `使用 ${integral} 积分兑换查看内容` }}     
          </button>
          <span v-if="balance !== null" class="lock-balance">我的积分：{{ balance }}</span>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * 内容可见范围遮罩（文章 / 动态 / 独立页面 / 评论共用）
 *
 * 后端在无权查看时会把正文清空，并在数据里回传：
 *   scope_state = 'login' 需登录 / 'pay' 需消耗积分兑换
 *   scope_integral         兑换所需积分
 * 本组件负责展示提示并提供「登录」或「积分兑换」入口，兑换成功后 emit('unlocked')，
 * 由父级重新拉取内容（后端已改为可查看，正文会正常返回）。
 *
 * 注意：评论的 scope_state = 'author'（仅作者可见）**不**走本组件，
 * 由 CommentItem 渲染「该评论仅作者可见」占位（登录/兑换都解决不了）。
 */
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { payUnlock } from '@/api/unlock'
import { getIntegral } from '@/api/goods'
import { toast } from '@/utils/toast'

const props = defineProps({
  // 'login' 需登录 / 'pay' 需积分兑换
  state: { type: String, default: 'login' },
  // 兑换所需积分
  integral: { type: Number, default: 0 },
  // 内容类型：article / moments / page / comment
  // （评论只有「登录可见」，不会出现 state=pay，见 model.ScopePaySupported）
  bindType: { type: String, required: true },
  bindId: { type: [String, Number], required: true },
  // 紧凑模式（评论等行内场景）
  compact: { type: Boolean, default: false }
})

const emit = defineEmits(['unlocked'])

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const paying = ref(false)
const balance = ref(null)

const logged = computed(() => userStore.isLogged)
const title = computed(() => (props.state === 'pay' ? '付费内容' : '登录后查看'))
const desc = computed(() => {
  if (props.state !== 'pay') {
    return '该内容仅对已登录用户开放，请先登录后查看完整内容'
  }
  return logged.value
    ? `该内容需消耗 ${props.integral} 积分兑换后查看，兑换一次即可永久查看`
    : `该内容需登录后消耗 ${props.integral} 积分兑换查看`
})

// 已登录时展示积分余额，方便判断是否够兑换（取不到就隐藏，不影响兑换流程）
async function loadBalance() {
  if (!logged.value || props.state !== 'pay') {
    balance.value = null
    return
  }
  try {
    const res = await getIntegral()
    balance.value = Number(res?.data?.integral) || 0
  } catch {
    balance.value = null
  }
}

watch(() => [logged.value, props.state, props.bindId], loadBalance, { immediate: true })

function goLogin() {
  router.push({ name: 'login', query: { redirect: route.fullPath } })
}

function goRegister() {
  router.push({ name: 'register' })
}

async function pay() {
  if (paying.value) return
  if (props.integral <= 0) {
    toast.warning('该内容未设置兑换积分')
    return
  }
  paying.value = true
  try {
    const res = await payUnlock(props.bindType, props.bindId)
    const rest = res?.data?.balance
    if (rest !== undefined && rest !== null) balance.value = Number(rest) || 0
    toast.success(
      rest === undefined || rest === null ? '兑换成功' : `兑换成功，剩余 ${Number(rest) || 0} 积分`
    )
    emit('unlocked')
  } catch {
    /* 拦截器已提示（积分不足 / 无需兑换等） */
  } finally {
    paying.value = false
  }
}
</script>

<style scoped>
.content-lock {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin: 16px 0;
  padding: 18px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-lg);
  background: var(--bg-alt);
}
.content-lock.is-compact {
  margin: 8px 0;
  padding: 12px;
}
.lock-icon {
  flex: none;
  font-size: 20px;
  line-height: 1.2;
  color: var(--primary);
}
.is-compact .lock-icon {
  font-size: 16px;
}
.lock-main {
  flex: 1;
  min-width: 0;
}
.lock-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.is-compact .lock-title {
  font-size: 13px;
}
.lock-desc {
  margin-top: 4px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-muted);
}
.lock-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.lock-balance {
  font-size: 12px;
  color: var(--text-muted);
}
</style>
