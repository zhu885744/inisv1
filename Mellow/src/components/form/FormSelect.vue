<template>
  <SelectMenu
    v-bind="$attrs"
    :model-value="modelValue"
    :options="normalizedOptions"
    :placeholder="placeholder"
    :icon="icon"
    :disabled="disabled"
    :variant="variant"
    :placement="placement"
    :floating="floating"
    @update:model-value="(val) => emit('update:modelValue', val)"
    @change="(val) => emit('change', val)"
  />
</template>

<script setup>
/**
 * 统一下拉选择（全站唯一的「下拉」入口）
 *
 * 内部复用 SelectMenu（浮层无法用原生 select 定制，见 SelectMenu.vue 的说明），
 * 对外只暴露一个组件：表单里用默认的 field 形态，筛选栏传 variant="pill"。
 *
 * 用法：
 *   <FormSelect v-model="form.status" :options="STATUS_OPTIONS" />
 *   <FormSelect v-model="filters.type" variant="pill" :options="typeOptions" icon="bi bi-funnel" />
 *   <FormSelect v-model="form.condition" :options="conditions" value-key="key" label-key="name" />
 *
 * options 里既可以是 { value, label }，也可以直接给字符串/数字数组（自动转成 value=label）；
 * 业务数据（如 [{ key, name }]）用 value-key / label-key 指定字段，不必先 map 一遍。
 */
import { computed } from 'vue'
import SelectMenu from '@/components/SelectMenu.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '请选择' },
  icon: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  // field：表单输入框形态（默认）；pill：筛选栏胶囊形态
  variant: { type: String, default: 'field' },
  placement: { type: String, default: 'auto' },
  floating: { type: String, default: 'auto' },
  valueKey: { type: String, default: 'value' },
  labelKey: { type: String, default: 'label' }
})

const emit = defineEmits(['update:modelValue', 'change'])

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
</script>
