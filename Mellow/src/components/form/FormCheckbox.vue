<template>
  <!-- 带文字：label 包住方块 / 开关，整行可点 -->
  <label v-if="needWrapper" class="form-check" :class="[`is-${variant}`, { 'is-disabled': disabled }]">
    <input
      ref="inputEl"
      v-bind="$attrs"
      type="checkbox"
      class="form-check-input"
      :checked="checked"
      :disabled="disabled"
      @change="onChange"
    />
    <span v-if="hasText" class="form-check-label"><slot>{{ label }}</slot></span>
  </label>

  <!-- 不带文字：只渲染方块本身（表格多选、卡片勾选等场景） -->
  <input
    v-else
    ref="inputEl"
    v-bind="$attrs"
    type="checkbox"
    class="form-check-input is-plain"
    :checked="checked"
    :disabled="disabled"
    @change="onChange"
  />
</template>

<script setup>
/**
 * 统一复选框（含开关形态）
 *
 * 三种用法：
 *   <FormCheckbox v-model="form.agree" label="我已阅读并同意" />            // 方块 + 文字
 *   <FormCheckbox v-model="form.enabled" variant="switch" label="开启" />    // 开关（toggle）
 *   <FormCheckbox v-model="selected" />                                      // 只要方块（表格多选）
 *
 * 布尔值用 v-model 直接绑；数字开关用 :true-value="1" :false-value="0"（与原生 input 一致）。
 * 事件（@change 等）与其它属性原样透传到内部 input。
 */
import { computed, ref, useSlots, watch } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { type: [Boolean, Number, String, Array], default: false },
  label: { type: [String, Number], default: '' },
  // box：方形勾选框（默认）；switch：开关
  variant: { type: String, default: 'box' },
  disabled: { type: Boolean, default: false },
  // 数组 v-model 时用：勾选就把 value 加进数组（表格多选场景）
  value: { type: [Boolean, Number, String], default: undefined },
  trueValue: { type: [Boolean, Number, String], default: undefined },
  falseValue: { type: [Boolean, Number, String], default: undefined },
  // 半选态（表头「全选」用）：indeterminate 不是 HTML 属性，只能写 DOM 属性
  indeterminate: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const slots = useSlots()
const hasText = computed(() => !!(props.label || slots.default))
// 开关形态没有文字也要包一层 label：滑块的轨道样式画在 .form-check 上
const needWrapper = computed(() => hasText.value || props.variant === 'switch')

const checked = computed(() => {
  // 数组 v-model（v-model="ids" + :value="id"）：命中即选中
  if (Array.isArray(props.modelValue)) {
    return props.modelValue.some((item) => String(item) === String(props.value))
  }
  if (props.trueValue !== undefined) return String(props.modelValue) === String(props.trueValue)
  return !!props.modelValue
})

const inputEl = ref(null)

function syncIndeterminate() {
  if (inputEl.value) inputEl.value.indeterminate = !!props.indeterminate
}

watch(() => props.indeterminate, syncIndeterminate, { immediate: true, flush: 'post' })

/**
 * 手动控制半选态（表头全选框：选中一部分时显示横线）
 *
 * 用法：checkboxRef.value.setIndeterminate(true)
 */
function setIndeterminate(value) {
  if (inputEl.value) inputEl.value.indeterminate = !!value
}

function focus() {
  inputEl.value?.focus()
}

defineExpose({ el: inputEl, setIndeterminate, focus })

function onChange(event) {
  const isChecked = event.target.checked

  if (Array.isArray(props.modelValue)) {
    const next = isChecked
      ? [...props.modelValue, props.value]
      : props.modelValue.filter((item) => String(item) !== String(props.value))
    emit('update:modelValue', next)
    return
  }

  if (props.trueValue !== undefined) {
    emit('update:modelValue', isChecked ? props.trueValue : props.falseValue)
    return
  }

  emit('update:modelValue', isChecked)
}
</script>

<style scoped>
.form-check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  /* 控件是固定尺寸，不允许被同一行的长文案挤压（挤压时文字会变竖排） */
  flex-shrink: 0;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
  user-select: none;
}

.form-check.is-disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.form-check-input {
  flex-shrink: 0;
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: var(--primary);
  cursor: pointer;
}

.form-check-input:disabled {
  cursor: not-allowed;
}

/* ---------- 开关形态 ---------- */
.form-check.is-switch .form-check-input {
  position: relative;
  width: 40px;
  height: 22px;
  border-radius: 999px;
  background: var(--border);
  border: none;
  appearance: none;
  -webkit-appearance: none;
  transition: background var(--dur-fast) var(--ease);
}

.form-check.is-switch .form-check-input::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--bg-card);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.18);
  transition: transform var(--dur-fast) var(--ease);
}

.form-check.is-switch .form-check-input:checked {
  background: var(--primary);
}

.form-check.is-switch .form-check-input:checked::after {
  transform: translateX(18px);
}

.form-check.is-switch .form-check-input:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px var(--accent-ring);
}
</style>
