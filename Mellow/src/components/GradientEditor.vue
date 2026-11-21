<template>
  <div class="ge">
    <div class="gradient-head">
      <div class="gradient-modes">
        <button
          v-for="item in modeList"
          :key="item.key"
          type="button"
          class="gradient-mode"
          :class="{ 'is-active': mode === item.key }"
          @click="switchMode(item.key)"
        >
          {{ item.label }}
        </button>
      </div>
      <button type="button" class="gradient-reset" title="清空（默认 none）" @click="clear">清除</button>
    </div>

    <!-- 纯色：复用自封装颜色选择器 -->
    <ColorPicker
      v-if="mode === 'solid'"
      :model-value="solidValue"
      :show-alpha="true"
      @update:model-value="onSolid"
    />

    <template v-else>
      <div class="gradient-preview" :style="{ backgroundImage: css }">
        <span v-if="!stops.length" class="gradient-preview-empty">未设置色标</span>
      </div>

      <div class="gradient-row">
        <div class="gradient-types">
          <button
            v-for="item in typeList"
            :key="item.key"
            type="button"
            class="gradient-type"
            :class="{ 'is-active': type === item.key }"
            :title="item.label"
            @click="changeType(item.key)"
          >
            {{ item.short }}
          </button>
        </div>
        <label v-if="type === 'linear'" class="gradient-angle">
          <input v-model.number="angle" type="number" min="0" max="360" step="5" @change="emitValue" />
          <span>°</span>
        </label>
      </div>

      <div class="gradient-stops">
        <div v-for="(stop, index) in stops" :key="index" class="gradient-stop">
          <ColorPicker
            class="gradient-stop-color"
            :model-value="stop.color"
            :show-alpha="true"
            @update:model-value="updateStop(index, $event)"
          />
          <label class="gradient-pos">
            <input
              type="number"
              min="0"
              max="100"
              :value="stop.pos"
              @change="updatePos(index, $event)"
            />
            <span>%</span>
          </label>
          <button
            type="button"
            class="gradient-remove"
            :disabled="stops.length <= 2"
            title="删除色标"
            @click="removeStop(index)"
          >
            <i class="bi bi-x-lg" />
          </button>
        </div>

        <button type="button" class="gradient-add" @click="addStop">
          <i class="bi bi-plus-lg" /> 添加色标
        </button>
      </div>
    </template>
  </div>
</template>

<script setup>
/**
 * GradientEditor - 纯色 / 渐变二选一的编辑器
 *
 * 用于 paint 类型的 token（paper-0/1/2、gradient-brand、gradient-page 等）：
 * - 纯色模式：直接复用 ColorPicker
 * - 渐变模式：linear / radial / conic + 角度 + 多色标，输出标准 gradient() 字符串
 * - 默认值 none 表示不启用；「清除」会写回 none
 *
 * 解析失败（例如用户填了未知函数）时回退到纯色模式，原值仍保留在输入框里，
 * 避免一打开编辑器就把用户的值改掉。
 */
import { computed, ref, watch } from 'vue'
import ColorPicker from '@/components/ColorPicker.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const modeList = [
  { key: 'solid', label: '纯色' },
  { key: 'gradient', label: '渐变' }
]

const typeList = [
  { key: 'linear', label: '线性渐变', short: '线性' },
  { key: 'radial', label: '径向渐变', short: '径向' },
  { key: 'conic', label: '锥形渐变', short: '锥形' }
]

const mode = ref('solid')
const solidValue = ref('')
const type = ref('linear')
const angle = ref(135)
const stops = ref([
  { color: '#c7482a', pos: 0 },
  { color: '#a5371c', pos: 100 }
])

let syncing = false

const css = computed(() => {
  if (!stops.value.length) return 'none'
  const list = stops.value.map((stop) => `${stop.color} ${normalizePos(stop.pos)}%`).join(', ')
  if (type.value === 'linear') return `linear-gradient(${angle.value}deg, ${list})`
  if (type.value === 'radial') return `radial-gradient(circle at 50% 50%, ${list})`
  return `conic-gradient(from 0deg at 50% 50%, ${list})`
})

// ---------- 解析 ----------
/** 按顶层逗号拆分：rgba(0, 0, 0, 0.1) 内部的逗号不能被切开 */
function splitTopLevel(text) {
  const result = []
  let current = ''
  let depth = 0

  for (const char of String(text || '')) {
    if (char === '(') depth++
    if (char === ')') depth--
    if (char === ',' && depth === 0) {
      result.push(current.trim())
      current = ''
      continue
    }
    current += char
  }

  if (current.trim()) result.push(current.trim())
  return result
}

function parseStop(text) {
  const item = String(text || '').trim()
  if (!item) return null

  const match = item.match(/^(.*?)\s+(-?[\d.]+)%$/)
  if (match) {
    return { color: match[1].trim(), pos: Number(match[2]) }
  }
  return { color: item, pos: null }
}

function syncFromModel(value) {
  syncing = true

  const text = String(value || '').trim()
  const match = text.match(/^(linear|radial|conic)-gradient\((.*)\)$/i)

  if (!match) {
    mode.value = 'solid'
    solidValue.value = text === 'none' ? '' : text
    syncing = false
    return
  }

  mode.value = 'gradient'
  type.value = match[1].toLowerCase()

  const parts = splitTopLevel(match[2])

  if (type.value === 'linear') {
    const angleMatch = String(parts[0] || '').match(/^(-?[\d.]+)deg$/)
    if (angleMatch) {
      angle.value = Number(angleMatch[1])
      parts.shift()
    }
  } else {
    // 第一个参数是 shape / at 描述，本编辑器固定生成，不回填
    if (parts.length > 1) parts.shift()
  }

  const parsed = parts.map(parseStop).filter(Boolean)

  // 缺省位置按顺序均分，保证预览不会因为 0/100 全叠在一起
  parsed.forEach((stop, index) => {
    if (stop.pos === null) {
      stop.pos = parsed.length > 1 ? Math.round((index / (parsed.length - 1)) * 100) : 0
    }
  })

  stops.value = parsed.length ? parsed : [{ color: '#c7482a', pos: 0 }, { color: '#a5371c', pos: 100 }]
  syncing = false
}

watch(() => props.modelValue, syncFromModel, { immediate: true })

// ---------- 输出 ----------
function normalizePos(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return 0
  return Math.min(100, Math.max(0, Math.round(number)))
}

function emitValue() {
  if (syncing) return
  emit('update:modelValue', mode.value === 'solid' ? solidValue.value : css.value)
}

// ---------- 编辑 ----------
function switchMode(next) {
  if (mode.value === next) return

  if (next === 'gradient') {
    // 从当前纯色起一个双色标渐变，减少「一片空白」的困惑
    const base = solidValue.value || '#c7482a'
    stops.value = [
      { color: base, pos: 0 },
      { color: '#a5371c', pos: 100 }
    ]
    mode.value = 'gradient'
  } else {
    solidValue.value = stops.value[0]?.color || ''
    mode.value = 'solid'
  }

  emitValue()
}

function changeType(next) {
  type.value = next
  emitValue()
}

function onSolid(value) {
  solidValue.value = value
  emitValue()
}

function updateStop(index, color) {
  if (!stops.value[index]) return
  stops.value[index] = { ...stops.value[index], color }
  emitValue()
}

function updatePos(index, event) {
  if (!stops.value[index]) return
  stops.value[index] = { ...stops.value[index], pos: normalizePos(event.target.value) }
  emitValue()
}

function addStop() {
  const last = stops.value[stops.value.length - 1]
  const position = last ? Math.min(100, normalizePos(last.pos) + 15) : 100
  stops.value.push({ color: last?.color || '#c7482a', pos: position })
  emitValue()
}

function removeStop(index) {
  if (stops.value.length <= 2) return
  stops.value.splice(index, 1)
  emitValue()
}

function clear() {
  mode.value = 'solid'
  solidValue.value = ''
  emit('update:modelValue', 'none')
}
</script>

<style scoped>
.ge {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.gradient-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.gradient-modes {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
}
.gradient-mode {
  height: 24px;
  padding: 0 10px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
}
.gradient-mode.is-active {
  background: var(--bg-card);
  color: var(--primary);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}
.gradient-reset {
  border: 0;
  background: transparent;
  color: var(--text-light);
  font-size: 11px;
  cursor: pointer;
}
.gradient-reset:hover {
  color: var(--text);
}
.gradient-preview {
  display: grid;
  place-items: center;
  height: 40px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  /* 渐变可能带透明色标，垫一层棋盘格 */
  background-color: var(--bg-muted);
  background-image: linear-gradient(45deg, rgba(0, 0, 0, 0.06) 25%, transparent 25%),
    linear-gradient(-45deg, rgba(0, 0, 0, 0.06) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, rgba(0, 0, 0, 0.06) 75%),
    linear-gradient(-45deg, transparent 75%, rgba(0, 0, 0, 0.06) 75%);
  background-size: 10px 10px;
  background-position: 0 0, 0 5px, 5px -5px, -5px 0;
}
.gradient-preview-empty {
  color: var(--text-light);
  font-size: 11px;
}
.gradient-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.gradient-types {
  display: inline-flex;
  gap: 3px;
}
.gradient-type {
  height: 24px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
}
.gradient-type.is-active {
  border-color: var(--primary);
  color: var(--primary);
}
.gradient-angle {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 24px;
  padding: 0 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
}
.gradient-angle input {
  width: 40px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  text-align: right;
}
.gradient-stops {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gradient-stop {
  display: flex;
  align-items: center;
  gap: 6px;
}
.gradient-stop-color {
  flex: 1;
  min-width: 0;
}
.gradient-pos {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex: 0 0 62px;
  height: 34px;
  padding: 0 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
}
.gradient-pos input {
  width: 100%;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  text-align: right;
}
.gradient-remove {
  flex: 0 0 26px;
  height: 26px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--text-light);
  font-size: 10px;
  cursor: pointer;
}
.gradient-remove:hover:not(:disabled) {
  border-color: var(--border);
  color: var(--danger);
}
.gradient-remove:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.gradient-add {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 30px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
}
.gradient-add:hover {
  border-color: var(--primary);
  color: var(--primary);
}
</style>
