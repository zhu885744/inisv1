<template>
  <div class="form-radio-group" :class="[`is-${variant}`, { 'is-disabled': disabled }]" role="radiogroup">
    <label v-for="(option, index) in normalizedOptions" :key="`${option.value}-${index}`" class="form-radio-item">
      <input
        type="radio"
        class="form-radio-input"
        :name="name"
        :value="option.value"
        :checked="isChecked(option)"
        :disabled="disabled || option.disabled"
        @change="select(option)"
      />
      <span class="form-radio-label">{{ option.label }}</span>
    </label>
  </div>
</template>

<script setup>
/**
 * 一组单选按钮（横排，选项由数据驱动）
 *
 * 用法：
 *   <FormRadioGroup v-model="form.status" :options="STATUS_OPTIONS" />
 *   <FormRadioGroup v-model="form.mode" variant="pill" :options="[{value:'a',label:'A'}]" />
 *
 * options 与 FormSelect 同构：{ value, label, disabled? }，也接受字符串/数字数组，
 * 业务数据用 value-key / label-key 指定字段。
 */
import { computed, useId } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number, Boolean], default: '' },
  options: { type: Array, default: () => [] },
  // row：圆形 + 文字（默认）；pill：分段胶囊
  variant: { type: String, default: 'row' },
  disabled: { type: Boolean, default: false },
  // 同一组内 name 必须一致，浏览器才会互斥；不传则自动生成
  name: { type: String, default: '' },
  valueKey: { type: String, default: 'value' },
  labelKey: { type: String, default: 'label' }
})

const emit = defineEmits(['update:modelValue'])

const autoName = useId()

const normalizedOptions = computed(() =>
  (props.options || []).map((item) => {
    if (item === null || item === undefined) return { value: '', label: '' }
    if (typeof item !== 'object') return { value: item, label: String(item) }
    return {
      value: item[props.valueKey],
      label: item[props.labelKey] ?? String(item[props.valueKey] ?? ''),
      disabled: !!item.disabled
    }
  })
)

const isChecked = (option) => String(props.modelValue) === String(option.value)

function select(option) {
  if (props.disabled || option.disabled) return
  emit('update:modelValue', option.value)
}
</script>

<style scoped>
.form-radio-group {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
}

.form-radio-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
  user-select: none;
}

.form-radio-group.is-disabled .form-radio-item {
  opacity: 0.55;
  cursor: not-allowed;
}

.form-radio-input {
  flex-shrink: 0;
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: var(--primary);
  cursor: pointer;
}

/* ---------- 分段胶囊形态 ---------- */
.form-radio-group.is-pill {
  display: inline-flex;
  gap: 0;
  padding: 2px;
  background: var(--bg-muted);
  border-radius: 999px;
}

.form-radio-group.is-pill .form-radio-item {
  position: relative;
  padding: 4px 12px;
  border-radius: 999px;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}

.form-radio-group.is-pill .form-radio-input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
  pointer-events: none;
}

.form-radio-group.is-pill .form-radio-item:has(.form-radio-input:checked) {
  background: var(--bg-card);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: var(--shadow-xs, 0 1px 3px rgba(0, 0, 0, 0.08));
}
</style>
