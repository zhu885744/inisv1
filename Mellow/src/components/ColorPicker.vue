<template>
  <div ref="rootRef" class="cp">
    <button
      type="button"
      class="color-picker-trigger"
      :class="{ 'is-open': open, 'is-disabled': disabled }"
      :disabled="disabled"
      @click="toggle"
    >
      <span class="color-picker-swatch" :style="swatchStyle">
        <span v-if="isTransparent" class="color-picker-swatch-empty">无</span>
        <!-- 带透明度时右下角补一个实色小点：棋盘格说明透明度，实色点说明色相，
             否则 rgba(255,255,255,0.04) 这种值看起来就是「一片空白」 -->
        <span v-else-if="alpha < 1 && !isGradientValue" class="color-picker-swatch-solid" :style="{ background: solidColor }" />
      </span>
      <span class="color-picker-value">{{ modelValue || '未设置' }}</span>
      <i class="bi bi-chevron-down color-picker-caret" />
    </button>

    <div v-if="open" class="color-picker-pop" @pointerdown.stop>
      <!-- 饱和度 / 明度面板 -->
      <div
        ref="panelRef"
        class="color-picker-panel"
        :style="{ '--cp-hue': `hsl(${hue}, 100%, 50%)` }"
        @pointerdown="startDrag('sv', $event)"
      >
        <span
          class="color-picker-handle"
          :style="{ left: `${sat}%`, top: `${100 - val}%`, background: panelColor }"
        />
      </div>

      <!-- 色相 -->
      <div class="color-picker-slider color-picker-hue" @pointerdown="startDrag('hue', $event)">
        <span class="color-picker-thumb" :style="{ left: `${(hue / 360) * 100}%` }" />
      </div>

      <!-- 透明度 -->
      <div v-if="showAlpha" class="color-picker-slider color-picker-alpha" @pointerdown="startDrag('alpha', $event)">
        <span
          class="color-picker-slider-fill"
          :style="{ background: `linear-gradient(to right, transparent, ${solidColor})` }"
        />
        <span class="color-picker-thumb" :style="{ left: `${alpha * 100}%` }" />
      </div>

      <div class="color-picker-row">
        <label class="color-picker-hex">
          <span>#</span>
          <input
            :value="hexInput"
            maxlength="8"
            spellcheck="false"
            autocomplete="off"
            @change="onHexChange"
          />
        </label>
        <label v-if="showAlpha" class="color-picker-alpha-input">
          <input type="number" min="0" max="100" :value="Math.round(alpha * 100)" @change="onAlphaChange" />
          <span>%</span>
        </label>
      </div>

      <div class="color-picker-presets">
        <button
          v-for="color in presetColors"
          :key="color"
          type="button"
          class="color-picker-preset"
          :style="{ background: color }"
          :title="color"
          @click="pickPreset(color)"
        />
      </div>

      <div class="color-picker-foot">
        <button type="button" class="color-picker-link" @click="clear">清除</button>
        <button type="button" class="color-picker-link is-primary" @click="open = false">完成</button>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * ColorPicker - 自封装的颜色选择器（不依赖任何第三方库）
 *
 * 能力：
 * - 饱和度 / 明度面板 + 色相条 + 透明度条，拖动实时生效
 * - hex 输入、透明度百分比输入、常用色板
 * - 输出规则：不透明 → #rrggbb；带透明度 → rgba(r, g, b, a)
 *   （与后端 app/model/theme.go 的 themeColorValid 白名单一致）
 *
 * 为什么不直接用 <input type="color">：原生控件无法调透明度、样式不可控，
 * 也不支持 rgba 文本回显。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  showAlpha: { type: Boolean, default: true },
  // 预设色板（由使用方按当前主题传入，默认给一组通用色）
  presets: {
    type: Array,
    default: () => [
      '#000000', '#ffffff', '#322e26', '#7b7361', '#a79e8b', '#e2d9c8', '#f6f1e8',
      '#c7482a', '#e2684a', '#e0a08a', '#a5371c', '#2e7a5c', '#5aa884', '#c08a2e',
      '#3a5ba0', '#a63d5c', 'rgba(0, 0, 0, 0.1)', 'transparent'
    ]
  }
})

const emit = defineEmits(['update:modelValue'])

const rootRef = ref(null)
const panelRef = ref(null)
const open = ref(false)

const hue = ref(0)
const sat = ref(100)
const val = ref(100)
const alpha = ref(1)

const presetColors = computed(() => props.presets || [])

// ---------- 颜色换算 ----------
function hsvToRgb(h, s, v) {
  const saturation = s / 100
  const brightness = v / 100
  const c = brightness * saturation
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = brightness - c

  let r = 0
  let g = 0
  let b = 0

  if (h < 60) [r, g, b] = [c, x, 0]
  else if (h < 120) [r, g, b] = [x, c, 0]
  else if (h < 180) [r, g, b] = [0, c, x]
  else if (h < 240) [r, g, b] = [0, x, c]
  else if (h < 300) [r, g, b] = [x, 0, c]
  else [r, g, b] = [c, 0, x]

  return {
    r: Math.round((r + m) * 255),
    g: Math.round((g + m) * 255),
    b: Math.round((b + m) * 255)
  }
}

function rgbToHsv(r, g, b) {
  const rr = r / 255
  const gg = g / 255
  const bb = b / 255
  const max = Math.max(rr, gg, bb)
  const min = Math.min(rr, gg, bb)
  const delta = max - min

  let h = 0
  if (delta !== 0) {
    if (max === rr) h = 60 * (((gg - bb) / delta) % 6)
    else if (max === gg) h = 60 * ((bb - rr) / delta + 2)
    else h = 60 * ((rr - gg) / delta + 4)
  }
  if (h < 0) h += 360

  return {
    h,
    s: max === 0 ? 0 : (delta / max) * 100,
    v: max * 100
  }
}

function toHex(r, g, b) {
  const part = (number) => number.toString(16).padStart(2, '0')
  return `#${part(r)}${part(g)}${part(b)}`
}

/** 解析颜色：支持 #rgb/#rrggbb/#rrggbbaa、rgb()/rgba()，其余返回 null */
function parseColor(text) {
  const value = String(text || '').trim().toLowerCase()
  if (!value) return null

  if (value === 'transparent') return { r: 0, g: 0, b: 0, a: 0 }
  if (value.startsWith('#')) {
    let hex = value.slice(1)
    if (hex.length === 3) hex = `${hex[0]}${hex[0]}${hex[1]}${hex[1]}${hex[2]}${hex[2]}`
    if (hex.length === 8) {
      return {
        r: parseInt(hex.slice(0, 2), 16),
        g: parseInt(hex.slice(2, 4), 16),
        b: parseInt(hex.slice(4, 6), 16),
        a: parseInt(hex.slice(6, 8), 16) / 255
      }
    }
    if (hex.length !== 6 || !/^[0-9a-f]{6}$/.test(hex)) return null
    return {
      r: parseInt(hex.slice(0, 2), 16),
      g: parseInt(hex.slice(2, 4), 16),
      b: parseInt(hex.slice(4, 6), 16),
      a: 1
    }
  }

  const match = value.match(/^rgba?\(([^)]+)\)$/)
  if (!match) return null

  const parts = match[1].split(/[,/]/).map((item) => item.trim()).filter(Boolean)
  if (parts.length < 3) return null

  return {
    r: Math.min(255, Math.max(0, Number.parseFloat(parts[0]) || 0)),
    g: Math.min(255, Math.max(0, Number.parseFloat(parts[1]) || 0)),
    b: Math.min(255, Math.max(0, Number.parseFloat(parts[2]) || 0)),
    a: parts[3] === undefined ? 1 : Math.min(1, Math.max(0, parseAlpha(parts[3])))
  }
}

function parseAlpha(text) {
  if (text.endsWith('%')) return Number.parseFloat(text) / 100
  return Number.parseFloat(text) || 0
}

// ---------- 当前值 ----------
const rgb = computed(() => hsvToRgb(hue.value, sat.value, val.value))
const solidColor = computed(() => toHex(rgb.value.r, rgb.value.g, rgb.value.b))
const panelColor = computed(() =>
  alpha.value >= 1 ? solidColor.value : `rgba(${rgb.value.r}, ${rgb.value.g}, ${rgb.value.b}, ${alpha.value})`
)
const hexInput = computed(() => solidColor.value.replace('#', ''))
const isTransparent = computed(() => alpha.value === 0)
const isGradientValue = computed(() => /gradient\(/i.test(String(props.modelValue || '')))
const swatchStyle = computed(() => {
  const value = String(props.modelValue || '').trim()
  if (!value) return {}
  // 用 backgroundColor / backgroundImage 而不是 background 简写：
  // 简写会把色块上的棋盘格背景图一起清掉，透明色就看不出来了
  if (/gradient\(/i.test(value)) return { backgroundImage: value }
  return { backgroundColor: value }
})

// ---------- 与外部值同步 ----------
function syncFromModel() {
  const parsed = parseColor(props.modelValue)
  if (!parsed) return
  const hsv = rgbToHsv(parsed.r, parsed.g, parsed.b)
  hue.value = hsv.h
  sat.value = hsv.s
  val.value = hsv.v
  alpha.value = parsed.a
}

watch(() => props.modelValue, syncFromModel, { immediate: true })

// ---------- 输出 ----------
function emitValue() {
  const { r, g, b } = rgb.value
  const value = alpha.value >= 1
    ? toHex(r, g, b)
    : `rgba(${r}, ${g}, ${b}, ${Number(alpha.value.toFixed(2))})`
  emit('update:modelValue', value)
}

// ---------- 拖拽 ----------
let dragType = ''
// 记录按下时的元素矩形：pointermove 挂在 window 上，event.currentTarget 不是滑条本身
let dragRect = null

function startDrag(type, event) {
  if (props.disabled) return
  dragType = type
  dragRect = (type === 'sv' ? panelRef.value : event.currentTarget)?.getBoundingClientRect() || null
  updateFromPoint(event)
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', stopDrag)
  event.preventDefault()
}

function onPointerMove(event) {
  updateFromPoint(event)
}

function stopDrag() {
  dragType = ''
  dragRect = null
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', stopDrag)
}

function updateFromPoint(event) {
  if (!dragType || !dragRect) return

  if (dragType === 'sv') {
    const x = Math.min(1, Math.max(0, (event.clientX - dragRect.left) / dragRect.width))
    const y = Math.min(1, Math.max(0, (event.clientY - dragRect.top) / dragRect.height))
    sat.value = x * 100
    val.value = (1 - y) * 100
  } else {
    const x = Math.min(1, Math.max(0, (event.clientX - dragRect.left) / dragRect.width))
    if (dragType === 'hue') hue.value = x * 360
    else alpha.value = x
  }

  emitValue()
}

// ---------- 输入 ----------
function onHexChange(event) {
  const text = String(event.target.value || '').trim().replace(/^#/, '')
  const parsed = parseColor(`#${text}`)
  if (!parsed) {
    event.target.value = hexInput.value
    return
  }
  const hsv = rgbToHsv(parsed.r, parsed.g, parsed.b)
  hue.value = hsv.h
  sat.value = hsv.s
  val.value = hsv.v
  emitValue()
}

function onAlphaChange(event) {
  const value = Number.parseFloat(event.target.value)
  alpha.value = Number.isFinite(value) ? Math.min(1, Math.max(0, value / 100)) : 1
  emitValue()
}

function pickPreset(color) {
  const parsed = parseColor(color)
  if (parsed) {
    const hsv = rgbToHsv(parsed.r, parsed.g, parsed.b)
    hue.value = hsv.h
    sat.value = hsv.s
    val.value = hsv.v
    alpha.value = parsed.a
  }
  emitUnit(color)
}

/** 色板直接用原值，避免 hex 往返丢失 rgba / transparent 语义 */
function emitUnit(value) {
  emit('update:modelValue', value)
}

function clear() {
  emit('update:modelValue', '')
  open.value = false
}

// ---------- 弹层开关 ----------
function toggle() {
  if (props.disabled) return
  open.value = !open.value
}

function onDocumentPointerDown(event) {
  if (!rootRef.value?.contains(event.target)) open.value = false
}

function onKeydown(event) {
  if (event.key === 'Escape') open.value = false
}

watch(open, (value) => {
  if (value) {
    document.addEventListener('pointerdown', onDocumentPointerDown, true)
    document.addEventListener('keydown', onKeydown)
  } else {
    document.removeEventListener('pointerdown', onDocumentPointerDown, true)
    document.removeEventListener('keydown', onKeydown)
  }
})

onBeforeUnmount(() => {
  stopDrag()
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.cp {
  position: relative;
  width: 100%;
}
.color-picker-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 38px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  transition: border-color var(--dur-fast) var(--ease);
}
.color-picker-trigger:hover,
.color-picker-trigger.is-open {
  border-color: var(--primary);
}
.color-picker-trigger.is-disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.color-picker-swatch {
  position: relative;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border: 1px solid var(--border);
  border-radius: 5px;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.35);
  background-image: linear-gradient(45deg, #ccc 25%, transparent 25%),
    linear-gradient(-45deg, #ccc 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #ccc 75%),
    linear-gradient(-45deg, transparent 75%, #ccc 75%);
  background-size: 8px 8px;
  background-position: 0 0, 0 4px, 4px -4px, -4px 0;
}
.color-picker-swatch-empty {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  color: var(--text-light);
  font-size: 9px;
}
.color-picker-swatch-solid {
  position: absolute;
  right: 1px;
  bottom: 1px;
  width: 9px;
  height: 9px;
  border: 1px solid rgba(255, 255, 255, 0.85);
  border-radius: 50%;
}
.color-picker-value {
  flex: 1;
  overflow: hidden;
  font-family: var(--font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.color-picker-caret {
  color: var(--text-light);
  font-size: 10px;
}

/* ---------- 弹层 ---------- */
.color-picker-pop {
  position: absolute;
  z-index: 140;
  top: calc(100% + 6px);
  left: 0;
  width: 236px;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  box-shadow: var(--shadow-md, 0 14px 34px rgba(0, 0, 0, 0.16));
}
.color-picker-panel {
  position: relative;
  height: 132px;
  border-radius: var(--radius-sm);
  cursor: crosshair;
  background-image: linear-gradient(to top, #000, transparent),
    linear-gradient(to right, #fff, var(--cp-hue));
  touch-action: none;
}
.color-picker-handle {
  position: absolute;
  width: 12px;
  height: 12px;
  border: 2px solid #fff;
  border-radius: 50%;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35);
  transform: translate(-50%, -50%);
  pointer-events: none;
}
.color-picker-slider {
  position: relative;
  height: 12px;
  margin-top: 10px;
  border-radius: 999px;
  cursor: pointer;
  touch-action: none;
}
.color-picker-hue {
  background: linear-gradient(to right, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00);
}
.color-picker-alpha {
  background-image: linear-gradient(45deg, #ccc 25%, transparent 25%),
    linear-gradient(-45deg, #ccc 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #ccc 75%),
    linear-gradient(-45deg, transparent 75%, #ccc 75%);
  background-size: 8px 8px;
  background-position: 0 0, 0 4px, 4px -4px, -4px 0;
}
.color-picker-slider-fill {
  position: absolute;
  inset: 0;
  border-radius: 999px;
}
.color-picker-thumb {
  position: absolute;
  top: 50%;
  width: 14px;
  height: 14px;
  border: 2px solid #fff;
  border-radius: 50%;
  background: transparent;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.35);
  transform: translate(-50%, -50%);
  pointer-events: none;
}
.color-picker-row {
  display: flex;
  gap: 8px;
  margin-top: 10px;
}
.color-picker-hex,
.color-picker-alpha-input {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 4px;
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
}
.color-picker-alpha-input {
  flex: 0 0 74px;
}
.color-picker-hex input,
.color-picker-alpha-input input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
}
.color-picker-alpha-input input::-webkit-outer-spin-button,
.color-picker-alpha-input input::-webkit-inner-spin-button {
  margin: 0;
  -webkit-appearance: none;
}
.color-picker-presets {
  display: grid;
  grid-template-columns: repeat(9, 1fr);
  gap: 5px;
  margin-top: 10px;
}
.color-picker-preset {
  width: 100%;
  aspect-ratio: 1;
  border: 1px solid var(--border);
  border-radius: 5px;
  cursor: pointer;
}
.color-picker-preset:hover {
  border-color: var(--primary);
}
.color-picker-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
}
.color-picker-link {
  padding: 2px 4px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
}
.color-picker-link:hover {
  color: var(--text);
}
.color-picker-link.is-primary {
  color: var(--primary);
}
</style>
