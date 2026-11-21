<template>
  <textarea
    class="textarea"
    :class="{ 'is-invalid': invalid }"
    :value="current"
    :placeholder="placeholder"
    :disabled="disabled"
    :readonly="readonly"
    :rows="rows"
    :aria-invalid="invalid ? 'true' : undefined"
    v-bind="$attrs"
    @input="onInput"
  />
</template>

<script setup>
/**
 * 统一多行输入（对应主题的 .textarea）
 *
 * 用法：
 *   <FormTextarea v-model="form.description" rows="3" placeholder="说点什么" />
 *
 * 与 FormInput 同一套约定：视觉走 .textarea，原生属性 / 事件透传，
 * 支持 v-model 的 .number / .trim 修饰符。
 */
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  placeholder: { type: String, default: '' },
  rows: { type: [String, Number], default: 3 },
  disabled: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
  invalid: { type: Boolean, default: false },
  modelModifiers: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['update:modelValue'])

const current = computed(() => (props.modelValue === null || props.modelValue === undefined ? '' : props.modelValue))

function onInput(event) {
  let value = event.target.value
  if (props.modelModifiers.trim) value = value.trim()
  emit('update:modelValue', value)
}
</script>

<style scoped>
.textarea.is-invalid {
  border-color: var(--danger);
}

.textarea.is-invalid:focus {
  border-color: var(--danger);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--danger) 18%, transparent);
}
</style>
