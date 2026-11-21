<template>
  <div class="se">
    <div class="shadow-head">
      <div class="shadow-preview" :style="{ boxShadow: enabled ? css : 'none' }" />
      <button type="button" class="shadow-toggle" :class="{ 'is-active': enabled }" @click="toggle">
        {{ enabled ? '启用' : '无阴影' }}
      </button>
    </div>

    <template v-if="enabled">
      <div class="shadow-grid">
        <label v-for="field in fields" :key="field.key" class="shadow-field">
          <span>{{ field.label }}</span>
          <FormInput
            v-model.number="value[field.key]"
            type="number"
            step="1"
            @change="emitValue" />
        </label>
      </div>

      <div class="shadow-row">
        <ColorPicker
          class="shadow-color"
          :model-value="value.color"
          :show-alpha="true"
          @update:model-value="onColor"
        />
        <button
          type="button"
          class="shadow-inset"
          :class="{ 'is-active': value.inset }"
          @click="toggleInset"
        >
          内阴影
        </button>
      </div>
    </template>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * ShadowEditor - 阴影编辑器
 *
 * 解析 / 生成单段 box-shadow：`[inset] x y blur spread color`。
 * 多段阴影（逗号分隔）在解析时只取第一段，保存后会变成单段——
 * 主题里的阴影变量本来就是一个语义值，保持单段更好维护。
 */
import { computed, reactive, ref, watch } from 'vue'
import ColorPicker from '@/components/ColorPicker.vue'

const props = defineProps({
  modelValue: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue'])

const enabled = ref(true)
const value = reactive({
  x: 0,
  y: 6,
  blur: 20,
  spread: 0,
  color: 'rgba(50, 46, 38, 0.07)',
  inset: false
})

const fields = [
  { key: 'x', label: 'X' },
  { key: 'y', label: 'Y' },
  { key: 'blur', label: '模糊' },
  { key: 'spread', label: '扩散' }
]

let syncing = false

const css = computed(() => {
  const inset = value.inset ? 'inset ' : ''
  return `${inset}${value.x}px ${value.y}px ${value.blur}px ${value.spread}px ${value.color || 'rgba(0, 0, 0, 0.12)'}`
})

/** 按空格拆分，但不切开 rgba(...) 内部的空格 */
function splitFields(text) {
  const result = []
  let current = ''
  let depth = 0

  for (const char of String(text || '').trim()) {
    if (char === '(') depth++
    if (char === ')') depth--
    if (/\s/.test(char) && depth === 0) {
      if (current) {
        result.push(current)
        current = ''
      }
      continue
    }
    current += char
  }

  if (current) result.push(current)
  return result
}

function parse(text) {
  const raw = String(text || '').trim()
  if (!raw || raw === 'none') {
    enabled.value = false
    return
  }

  enabled.value = true

  // 只取第一段（顶层逗号）
  let depth = 0
  let first = ''
  for (const char of raw) {
    if (char === '(') depth++
    if (char === ')') depth--
    if (char === ',' && depth === 0) break
    first += char
  }

  const fields = splitFields(first)
  value.inset = fields[0] === 'inset'
  if (value.inset) fields.shift()

  const numbers = []
  while (fields.length && /^-?[\d.]+(px|rem|em)?$/i.test(fields[0])) {
    numbers.push(Number.parseFloat(fields.shift()) || 0)
  }

  value.x = numbers[0] ?? 0
  value.y = numbers[1] ?? 0
  value.blur = numbers[2] ?? 0
  value.spread = numbers[3] ?? 0
  value.color = fields.join(' ') || 'rgba(0, 0, 0, 0.12)'
}

watch(() => props.modelValue, parse, { immediate: true })

function emitValue() {
  if (syncing) return
  emit('update:modelValue', enabled.value ? css.value : 'none')
}

function toggle() {
  enabled.value = !enabled.value
  emitValue()
}

function toggleInset() {
  value.inset = !value.inset
  emitValue()
}

function onColor(color) {
  value.color = color
  emitValue()
}
</script>

<style scoped>
.se {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.shadow-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.shadow-preview {
  flex: 1;
  min-width: 0;
  height: 34px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  /* 用比卡片略深的底色，否则投影预览贴在卡片上几乎看不出边界 */
  background: var(--bg-muted);
  transition: box-shadow var(--dur-fast) var(--ease);
}
.shadow-toggle {
  flex: 0 0 auto;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
}
.shadow-toggle.is-active {
  border-color: var(--primary);
  color: var(--primary);
}
/* 2 列而不是 4 列：token 卡片最窄只有 260px，四列会把数字输入挤到看不清 */
.shadow-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}
.shadow-field {
  display: flex;
  flex-direction: column;
  gap: 3px;
  color: var(--text-light);
  font-size: 10px;
}
.shadow-field input {
  height: 28px;
  padding: 0 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--bg-muted);
  color: var(--text);
  font-family: var(--font-mono);
  font-size: 11px;
}
.shadow-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.shadow-color {
  flex: 1;
  min-width: 0;
}
.shadow-inset {
  flex: 0 0 auto;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
}
.shadow-inset.is-active {
  border-color: var(--primary);
  color: var(--primary);
}
</style>
