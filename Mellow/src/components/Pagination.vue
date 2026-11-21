<template>
  <!-- sizes 为空时（前台列表）行为与以前完全一致：只有页码，没有条数下拉 -->
  <nav v-if="pageCount > 1 || sizes.length" class="pager" aria-label="分页导航">
    <!-- 每页条数：切换后只上报新值，回到第 1 页与重新请求由父组件处理 -->
    <FormSelect variant="pill"
      v-if="sizes.length"
      class="pager-size"
      :model-value="pageSize"
      :options="sizes"
      @update:model-value="(value) => $emit('update:pageSize', Number(value))" />
    <template v-for="(item, index) in pageItems" :key="`${item}-${index}`">
      <span v-if="item === '...'" class="pager-gap" aria-hidden="true">…</span>
      <button
        v-else
        type="button"
        class="pager-btn"
        :class="{ 'is-active': item === current }"
        :aria-label="`第 ${item} 页`"
        :aria-current="item === current ? 'page' : undefined"
        :disabled="item === current"
        @click="$emit('update:current', item)"
      >{{ item }}</button>
    </template>
    <span class="pager-total">共 {{ pageCount }} 页</span>
  </nav>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import { computed } from 'vue'

const props = defineProps({
  current: { type: Number, default: 1 },
  total: { type: Number, default: 0 },
  pageSize: { type: Number, default: 10 },
  // 每页条数下拉的选项 [{ value, label }]；不传则不渲染下拉（前台列表保持原样）
  sizes: { type: Array, default: () => [] }
})
defineEmits(['update:current', 'update:pageSize'])

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

/* 以当前页为中心取 5 个连续页码，首尾各保留 1 页，中间用省略号衔接 */
const pageItems = computed(() => {
  const count = pageCount.value
  const cur = Math.min(Math.max(1, props.current), count)
  if (count <= 7) {
    return Array.from({ length: count }, (_, i) => i + 1)
  }
  let end = Math.min(count, Math.max(1, cur - 2) + 4)
  let start = Math.max(1, end - 4)
  end = Math.min(count, start + 4)
  const items = []
  if (start > 1) {
    items.push(1)
    if (start > 2) items.push('...')
  }
  for (let i = start; i <= end; i++) items.push(i)
  if (end < count) {
    if (end < count - 1) items.push('...')
    items.push(count)
  }
  return items
})
</script>

<style scoped>
.pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 24px 0;
}
/* 条数下拉与页码按钮等高（触发器默认高度来自 --control-h-sm） */
.pager-size :deep(.select-trigger) {
  height: 32px;
}
.pager-btn {
  min-width: 32px;
  height: 32px;
  padding: 0 9px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease),
    border-color var(--dur) var(--ease);
}
.pager-btn:hover:not(:disabled) {
  border-color: var(--primary);
  background: var(--accent-wash);
  color: var(--primary);
}
.pager-btn:focus-visible {
  outline: 2px solid var(--primary-soft);
  outline-offset: 2px;
}
.pager-btn.is-active,
.pager-btn.is-active:disabled {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
  opacity: 1;
  cursor: default;
}
.pager-gap {
  min-width: 20px;
  color: var(--text-light);
  font-size: 13px;
  text-align: center;
  user-select: none;
}
.pager-total {
  margin-left: 8px;
  color: var(--text-muted);
  font-size: 13px;
}
@media (max-width: 480px) {
  .pager-total {
    display: none;
  }
}
</style>
