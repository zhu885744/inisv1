<template>
  <span class="scope-badge" :class="`is-${kind}`" :title="tip">
    <i class="bi" :class="icon" aria-hidden="true" />
    <span>{{ text }}</span>
  </span>
</template>

<script setup>
/**
 * 可见范围徽标（后台列表用）
 *
 * 数据来源：article / moments / pages / comment 的 `scope` + `scope_integral` 字段。
 * 后台列表直接展示，方便运营一眼看出哪些内容是「登录可见 / 付费可见 / 仅作者可见」。
 *
 * allowPay=false 时把 3 也按「登录可见」显示（评论不支持付费可见，
 * 历史数据里即便残留 3，也不该在后台展示成积分内容）；
 * 4（仅作者可见）只有评论支持，与 allowPay 无关，原样展示。
 */
import { computed } from 'vue'

const props = defineProps({
  scope: { type: [String, Number], default: 1 },
  integral: { type: [String, Number], default: 0 },
  allowPay: { type: Boolean, default: true }
})

const value = computed(() => {
  const scope = Number(props.scope)
  if (scope === 4) return 4
  if (scope === 3) return props.allowPay ? 3 : 2
  return scope === 2 ? 2 : 1
})

const kind = computed(() => ({ 1: 'public', 2: 'login', 3: 'pay', 4: 'author' }[value.value]))

const icon = computed(
  () => ({ 1: 'bi-globe2', 2: 'bi-person-lock', 3: 'bi-coin', 4: 'bi-eye-slash' }[value.value])
)

const text = computed(() => {
  if (value.value === 1) return '公开'
  if (value.value === 2) return '登录可见'
  if (value.value === 4) return '仅作者可见'
  return `付费 · ${Number(props.integral) || 0} 积分`
})

const tip = computed(() => {
  if (value.value === 1) return '公开：所有用户（含访客）可见'
  if (value.value === 2) return '登录可见：访客只看到「登录后查看」的提示'
  if (value.value === 4) return '仅作者可见：只有内容的作者、评论者本人与管理员可以看到'
  return `付费可见：登录用户消耗 ${Number(props.integral) || 0} 积分兑换后可看`
})
</script>

<style scoped>
.scope-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: none;
  padding: 1px 8px;
  font-size: 11px;
  line-height: 1.7;
  font-weight: 400;
  border: 1px solid transparent;
  border-radius: 999px;
  white-space: nowrap;
  vertical-align: middle;
}
.scope-badge i {
  font-size: 11px;
}
/* 公开：中性灰，低调不抢状态徽标 */
.scope-badge.is-public {
  color: var(--text-muted);
  background: var(--bg-muted);
  border-color: var(--border-soft);
}
/* 登录可见：主色淡底 */
.scope-badge.is-login {
  color: var(--primary-deep);
  background: var(--accent-soft);
}
/* 付费可见：警告色实底，文字用页面底色保证深浅模式都可读 */
.scope-badge.is-pay {
  color: var(--bg-card);
  background: var(--warning);
}
/* 仅作者可见：虚边灰底，与「公开」区分但不抢眼 */
.scope-badge.is-author {
  color: var(--text-soft);
  background: var(--bg-alt);
  border-color: var(--border-soft);
  border-style: dashed;
}
</style>
