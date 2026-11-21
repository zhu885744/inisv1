<template>
  <label v-if="hasText" class="form-radio" :class="{ 'is-disabled': disabled }">
    <input
      v-bind="$attrs"
      type="radio"
      class="form-radio-input"
      :value="value"
      :checked="checked"
      :disabled="disabled"
      @change="onChange"
    />
    <span class="form-radio-label"><slot>{{ label }}</slot></span>
  </label>

  <input
    v-else
    v-bind="$attrs"
    type="radio"
    class="form-radio-input is-plain"
    :value="value"
    :checked="checked"
    :disabled="disabled"
    @change="onChange"
  />
</template>

<script setup>
/**
 * 单个单选按钮（多选项一般直接用 FormRadioGroup）
 *
 * 用法：
 *   <FormRadio v-model="form.gender" value="male" label="男" name="gender" />
 */
import { computed, useSlots } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [String, Number, Boolean], default: '' },
  value: { type: [String, Number, Boolean], default: '' },
  label: { type: [String, Number], default: '' },
  disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const slots = useSlots()
const hasText = computed(() => !!(props.label || slots.default))

// 两种绑定方式都支持：
//   - 单选组：v-model 绑当前值，和 value 比较是否选中
//   - 页面自己算选中态：:model-value 传布尔（如 Number(item.audit) === opt.value）
const checked = computed(() => {
  if (typeof props.modelValue === 'boolean') return props.modelValue
  return String(props.modelValue) === String(props.value)
})

function onChange() {
  emit('update:modelValue', props.value)
}
</script>

<style scoped>
.form-radio {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
  user-select: none;
}

.form-radio.is-disabled {
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
</style>
