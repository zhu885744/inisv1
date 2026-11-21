<template>
  <input
    ref="inputEl"
    class="input"
    :class="{ 'is-invalid': invalid }"
    :type="type"
    :value="current"
    :placeholder="placeholder"
    :disabled="disabled"
    :readonly="readonly"
    :aria-invalid="invalid ? 'true' : undefined"
    v-bind="$attrs"
    @input="onInput"
  />
</template>

<script setup>
/**
 * 统一输入框（全站唯一的单行输入实现）
 *
 * 用法：
 *   <FormInput v-model="form.name" placeholder="标题" />
 *   <FormInput v-model.number="form.sort" type="number" min="0" />
 *   <FormInput v-model.trim="form.slug" :invalid="!!error" />
 *
 * 约定：
 *   - 视觉沿用主题的 .input（高度 / 圆角 / 聚焦环都由 var(--control-h) / --radius 令牌控制），
 *     页面里原有的 `.input` 微调样式继续生效（根元素会带上父组件的 scoped 标识）；
 *   - 原生属性（placeholder / min / max / step / maxlength / autocomplete…）与事件
 *     （@input / @change / @blur / @keyup.enter…）直接透传，不需要组件再一个个声明；
 *   - 支持 v-model 的 .number / .trim 修饰符（组件内自己处理，不依赖原生 input 的修饰符）。
 */
import { computed, ref } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
  // 校验失败态：只改边框色并由父级给 .form-error 文案
  invalid: { type: Boolean, default: false },
  // v-model 修饰符（.number / .trim）
  modelModifiers: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['update:modelValue'])

// null / undefined 统一成空串，避免受控输入显示成 "null"
const current = computed(() => (props.modelValue === null || props.modelValue === undefined ? '' : props.modelValue))

const inputEl = ref(null)

// 暴露常用 DOM 操作：页面里 ref 拿到组件后仍可 focus() / select()
defineExpose({
  el: inputEl,
  focus: () => inputEl.value?.focus(),
  blur: () => inputEl.value?.blur(),
  select: () => inputEl.value?.select()
})

function onInput(event) {
  let value = event.target.value

  if (props.modelModifiers.trim) value = value.trim()

  if (props.modelModifiers.number) {
    const num = parseFloat(value)
    // 清空输入框时保留空串，否则浏览器会把 '' 当 0
    emit('update:modelValue', Number.isNaN(num) ? value : num)
    return
  }

  emit('update:modelValue', value)
}
</script>

<style scoped>
.input.is-invalid {
  border-color: var(--danger);
}

.input.is-invalid:focus {
  border-color: var(--danger);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--danger) 18%, transparent);
}
</style>
