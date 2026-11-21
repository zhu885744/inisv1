<template>
  <ConfirmDialog
    :visible="visible"
    :title="title"
    :message="message"
    :confirm-text="confirmText"
    :loading="busy"
    @update:visible="(v) => emit('update:visible', v)"
    @confirm="submit"
  >
    <div class="renew-body">
      <p v-if="!options.length" class="renew-empty">
        <i class="bi bi-inbox" /> 这件商品还没有配置续期档位
      </p>

      <div v-else class="renew-list">
        <button
          v-for="option in options"
          :key="option.days"
          type="button"
          class="renew-plan"
          :class="{ 'is-active': selectedDays === option.days }"
          @click="selectedDays = option.days"
        >
          <span class="plan-main">
            <strong class="plan-name">续期 {{ option.days_text || `${option.days} 天` }}</strong>
            <span class="plan-days">到期时间顺延 {{ option.days }} 天</span>
          </span>

          <span class="plan-price">
            <em v-if="option.free">免费</em>
            <template v-else>
              <em>{{ option.price }}</em>
              <span class="plan-unit">积分</span>
            </template>
            <s v-if="Number(option.origin_price) > Number(option.price)">{{ option.origin_price }}</s>
          </span>

          <span v-if="option.tag" class="plan-tag">{{ option.tag }}</span>
          <span v-else-if="option.discount" class="plan-tag is-soft">{{ option.discount }}</span>
        </button>
      </div>

      <p v-if="selected" class="renew-after">
        续期后到期：<strong>{{ afterText }}</strong>
      </p>

      <p v-if="options.length" class="renew-tip">
        续期天数会叠加到当前到期时间上；已经过期的从今天重新计算。
      </p>
    </div>
  </ConfirmDialog>
</template>

<script setup>
/**
 * 续期选档弹窗（我的勋章 / 我的装扮共用）
 *
 * 用法：
 *   <RenewPlanDialog
 *     :visible="renew.visible"
 *     scope="badge"
 *     :target-id="renew.id"
 *     :target-name="renew.name"
 *     :current-expire="renew.expire"
 *     :options="renew.options"
 *     @update:visible="(v) => (renew.visible = v)"
 *     @done="load"
 *   />
 *
 * 档位是「这件商品自己的」：由后台在勋章 / 装扮编辑表单里配置，
 * 随 badge/mine、decoration/mine 的 renew_options 一起下发，这里不再单独请求价格表。
 * 下单只提交天数，价格由服务端按档位取。
 */
import { computed, ref, watch } from 'vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { renewBadge, renewDecoration } from '@/api/renew'
import { formatDate } from '@/utils/time'
import { toast } from '@/utils/toast'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // badge / decoration
  scope: { type: String, default: 'badge' },
  // 勋章 id 或 装扮 id
  targetId: { type: [Number, String], default: 0 },
  targetName: { type: String, default: '' },
  // 当前到期时间（unix 秒）
  currentExpire: { type: [Number, String], default: 0 },
  // 这件商品自己的续期档位：{ days, days_text, price, origin_price, tag, discount, free }
  options: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:visible', 'done'])

const busy = ref(false)
// 选中的天数（档位以天数为唯一键）
const selectedDays = ref(0)

const isBadge = computed(() => props.scope === 'badge')

const title = computed(() => {
  const name = props.targetName ? `「${props.targetName}」` : ''
  return `续期${name}`
})

const message = computed(() => {
  const expire = Number(props.currentExpire) || 0
  if (!expire) return ''
  const prefix = expire * 1000 < Date.now() ? '已于' : '当前到期'
  return `${prefix} ${formatDate(expire)} 到期`
})

const selected = computed(
  () => props.options.find((item) => Number(item.days) === Number(selectedDays.value)) || null
)

const confirmText = computed(() => {
  const option = selected.value
  if (!option) return '确认续期'
  if (option.free) return '免费续期'
  return `支付 ${option.price} 积分`
})

// 与后端 model.RenewExtend 同一套算法：未过期在原到期时间上叠加，已过期从今天算
const afterText = computed(() => {
  const option = selected.value
  if (!option) return ''

  const now = Math.floor(Date.now() / 1000)
  const base = Math.max(Number(props.currentExpire) || 0, now)
  const limit = now + 10 * 365 * 86400
  const next = Math.min(base + Number(option.days) * 86400, limit)

  return formatDate(next)
})

watch(
  () => props.visible,
  (val) => {
    if (!val) {
      document.body.style.overflow = ''
      return
    }

    document.body.style.overflow = 'hidden'
    // 默认选中天数最短的档位，减少一次点击
    selectedDays.value = Number(props.options[0]?.days) || 0
  }
)

async function submit() {
  const option = selected.value
  if (!option || busy.value) return

  const targetId = Number(props.targetId) || 0
  if (!targetId) {
    toast.warning('参数错误，请刷新后重试')
    return
  }

  busy.value = true
  try {
    const res = isBadge.value
      ? await renewBadge(targetId, option.days)
      : await renewDecoration(targetId, option.days)

    const data = res?.data?.data || res?.data || {}
    const expire = Number(data.expire_after) || 0

    toast.success(expire ? `续期成功，${formatDate(expire)} 到期` : '续期成功')
    emit('done', data)
    emit('update:visible', false)
  } catch {
    // 错误提示由请求拦截器统一处理（积分不足 / 未持有 / 永久有效等）
  } finally {
    busy.value = false
    document.body.style.overflow = ''
  }
}
</script>

<style scoped>
.renew-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.renew-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 18px 0;
  font-size: 13px;
  color: var(--text-muted);
}

.renew-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.renew-plan {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text);
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.18s, background-color 0.18s;
}

.renew-plan:hover {
  border-color: var(--primary-soft);
}

.renew-plan.is-active {
  border-color: var(--primary);
  background: var(--accent-soft);
}

.plan-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.plan-name {
  font-size: 14px;
  font-weight: 600;
}

.plan-days {
  font-size: 12px;
  color: var(--text-light);
}

.plan-price {
  display: flex;
  align-items: baseline;
  gap: 4px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
}

.plan-price em {
  font-style: normal;
  font-size: 16px;
  font-weight: 700;
  color: var(--primary);
  font-variant-numeric: tabular-nums;
}

.plan-unit {
  font-size: 11px;
}

.plan-price s {
  font-size: 11px;
  color: var(--text-light);
}

.plan-tag {
  flex-shrink: 0;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--primary);
  color: #fff;
  font-size: 11px;
  line-height: 1.6;
  white-space: nowrap;
}

.plan-tag.is-soft {
  background: var(--accent-soft);
  color: var(--primary-deep);
}

.renew-after {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--text-soft);
}

.renew-after strong {
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}

.renew-tip {
  margin: 0;
  padding-top: 8px;
  border-top: 1px dashed var(--border-soft);
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
}
</style>
