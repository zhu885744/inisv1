<template>
  <div
    ref="rootRef"
    class="select-menu"
    :class="[`is-${variant}`, { 'is-open': open, 'is-up': openUp, 'is-disabled': disabled }]"
  >
    <!-- 触发器 -->
    <button
      type="button"
      class="select-trigger"
      :disabled="disabled"
      aria-haspopup="listbox"
      :aria-expanded="open ? 'true' : 'false'"
      @click="toggle"
      @keydown="onKeydown"
    >
      <span class="select-trigger-main">
        <i v-if="icon" :class="icon" class="select-icon" aria-hidden="true" />
        <span class="select-value" :class="{ 'is-placeholder': !selected }">
          {{ selected ? selected.label : placeholder }}
        </span>
      </span>
      <i class="bi bi-chevron-down select-caret" aria-hidden="true" />
    </button>

    <!-- 选项浮层（主题内置样式，替代无法定制样式的原生 select 下拉） -->
    <Transition name="select-pop">
      <div
        v-if="open"
        ref="panelRef"
        class="select-panel"
        :class="{ 'is-floating': floating }"
        :style="panelStyle"
        role="listbox"
      >
        <button
          v-for="(opt, i) in options"
          :key="`${opt.value}-${i}`"
          type="button"
          class="select-option"
          role="option"
          :class="{
            'is-active': i === activeIndex,
            'is-selected': isSelected(opt),
            'is-disabled': opt.disabled
          }"
          :aria-selected="isSelected(opt) ? 'true' : 'false'"
          :disabled="!!opt.disabled"
          @mouseenter="activeIndex = i"
          @click="choose(opt)"
        >
          <span class="select-option-label">{{ opt.label }}</span>
          <i v-if="isSelected(opt)" class="bi bi-check2" aria-hidden="true" />
        </button>
        <div v-if="!options.length" class="select-empty">暂无可选项</div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
/**
 * 主题统一的下拉选择器 —— 全站唯一的「下拉列表」实现
 *
 * 约定（全站强制）：页面上任何下拉都必须用本组件，禁止使用浏览器原生 <select> / <datalist>：
 *   1. 原生下拉的浮层由系统绘制，CSS 完全改不动，暗色主题 / 圆角 / 动效都对不上；
 *   2. 原生 option 不能放图标、副标题、禁用态样式，扩展性差；
 *   3. 本项目已有 30+ 处用本组件，混用会让同类控件在不同页面长得不一样。
 *
 * 用法：
 *   - 筛选栏 / 工具条：默认 pill 形态  <FormSelect variant="pill" v-model="x" :options="list" icon="bi bi-funnel" />
 *   - 表单内部：variant="field"        <FormSelect v-model="x" :options="list" />
 *   options 为 [{ value, label, disabled? }]；值比较统一走 String()，数字 / 字符串都能对上。
 *
 * 浮层定位：默认贴着触发器绝对定位；当下拉处在滚动容器（弹窗 .modal-body、表格滚动区等）
 * 里、浮层会被 overflow 裁掉时，自动切成 position: fixed 按视口坐标绘制，
 * 避免「点了没反应」（其实浮层被裁在容器外了）。需要强制关闭该行为时传 floating="never"。
 */
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  // 选项：{ value, label, disabled? }
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '请选择' },
  icon: { type: String, default: '' },
  // 外观形态：pill 胶囊按钮（默认）/ field 表单输入框（对齐主题 .input）
  variant: { type: String, default: 'pill' },
  disabled: { type: Boolean, default: false },
  // 浮层方向：auto（默认，空间不足自动向上）/ bottom / top
  placement: { type: String, default: 'auto' },
  // 被滚动容器裁剪时是否改为 fixed 定位：auto（默认）/ never
  floating: { type: String, default: 'auto' }
})

const emit = defineEmits(['update:modelValue', 'change'])

const rootRef = ref(null)
const panelRef = ref(null)
const open = ref(false)
const openUp = ref(false)
const activeIndex = ref(-1)
// 浮层脱离裁剪容器时改用 fixed 定位（见文件头注释）
const floating = ref(false)
const panelStyle = ref({})

const selectedIndex = computed(() =>
  props.options.findIndex((opt) => String(opt.value) === String(props.modelValue ?? ''))
)
const selected = computed(() => (selectedIndex.value > -1 ? props.options[selectedIndex.value] : null))

const isSelected = (opt) => String(opt.value) === String(props.modelValue ?? '')

function toggle() {
  if (props.disabled) return
  open.value ? close() : openPanel()
}

function openPanel() {
  if (props.disabled) return
  floating.value = false
  panelStyle.value = {}
  open.value = true
  activeIndex.value = selectedIndex.value > -1 ? selectedIndex.value : firstEnabledIndex()
  nextTick(() => {
    updatePlacement()
    scrollActiveIntoView()
    // 浮层渲染完再按真实高度算一次（首次只能按选项数估算）
    nextTick(updatePlacement)
  })
}

function close() {
  open.value = false
}

function firstEnabledIndex() {
  const i = props.options.findIndex((opt) => !opt.disabled)
  return i < 0 ? 0 : i
}

function choose(opt) {
  if (!opt || opt.disabled) return
  emit('update:modelValue', opt.value)
  emit('change', opt.value)
  close()
  nextTick(() => rootRef.value?.querySelector('.select-trigger')?.focus())
}

function move(step) {
  const total = props.options.length
  if (!total) return
  let i = activeIndex.value
  for (let n = 0; n < total; n++) {
    i = (i + step + total) % total
    if (!props.options[i].disabled) break
  }
  activeIndex.value = i
  scrollActiveIntoView()
}

function onKeydown(e) {
  const key = e.key
  if (key === 'Escape') {
    if (open.value) {
      e.preventDefault()
      close()
    }
    return
  }
  if (!open.value) {
    if (key === 'Enter' || key === ' ' || key === 'ArrowDown' || key === 'ArrowUp') {
      e.preventDefault()
      openPanel()
      if (key === 'ArrowUp') move(-1)
    }
    return
  }
  if (key === 'ArrowDown') {
    e.preventDefault()
    move(1)
  } else if (key === 'ArrowUp') {
    e.preventDefault()
    move(-1)
  } else if (key === 'Enter') {
    e.preventDefault()
    choose(props.options[activeIndex.value])
  } else if (key === 'Home') {
    e.preventDefault()
    activeIndex.value = 0
    scrollActiveIntoView()
  } else if (key === 'End') {
    e.preventDefault()
    activeIndex.value = props.options.length - 1
    scrollActiveIntoView()
  } else if (key === 'Tab') {
    close()
  }
}

/** 空间不足时向上展开（编辑器底部操作栏场景）；被滚动容器裁剪时改用 fixed 定位 */
function updatePlacement() {
  const root = rootRef.value
  if (!root) return

  const rect = root.getBoundingClientRect()
  const panelH = panelRef.value
    ? panelRef.value.offsetHeight
    : Math.min(props.options.length * 36 + 8, 280)
  const gap = 6
  const spaceBelow = window.innerHeight - rect.bottom - gap
  const spaceAbove = rect.top - gap

  let up = props.placement === 'top'
  if (props.placement === 'auto') up = spaceBelow < panelH && spaceAbove > spaceBelow
  openUp.value = up

  if (!clippedByAncestor(rect, panelH, up)) {
    floating.value = false
    panelStyle.value = {}
    return
  }

  // 祖先有 overflow 裁剪：浮层用 fixed 按视口坐标绘制，直接脱离滚动容器
  floating.value = true
  panelStyle.value = {
    position: 'fixed',
    left: `${Math.round(rect.left)}px`,
    minWidth: `${Math.round(rect.width)}px`,
    top: up ? 'auto' : `${Math.round(rect.bottom + gap)}px`,
    bottom: up ? `${Math.round(window.innerHeight - rect.top + gap)}px` : 'auto'
  }
}

/** 浮层按当前方向展开后，是否会被某个 overflow 容器切掉 */
function clippedByAncestor(rect, panelH, up) {
  if (props.floating === 'never' || !rootRef.value) return false

  const panelTop = up ? rect.top - 6 - panelH : rect.bottom + 6
  const panelBottom = panelTop + panelH

  let el = rootRef.value.parentElement
  while (el && el !== document.body && el !== document.documentElement) {
    const cs = getComputedStyle(el)
    const clipsY = cs.overflowY !== 'visible'
    const clipsX = cs.overflowX !== 'visible'

    if (clipsY || clipsX) {
      const box = el.getBoundingClientRect()
      if (clipsY && (panelBottom > box.bottom - 1 || panelTop < box.top + 1)) return true
      if (clipsX && (rect.right > box.right - 1 || rect.left < box.left + 1)) return true
    }

    el = el.parentElement
  }

  return false
}

function scrollActiveIntoView() {
  nextTick(() => {
    const panel = panelRef.value
    const el = panel?.querySelectorAll('.select-option')[activeIndex.value]
    if (!panel || !el) return

    // 只滚浮层自身：scrollIntoView 会连带滚动弹窗 / 页面，触发「滚动即关闭」，
    // 表现就是「点了没反应」
    const top = el.offsetTop
    const bottom = top + el.offsetHeight
    if (top < panel.scrollTop) panel.scrollTop = top
    else if (bottom > panel.scrollTop + panel.clientHeight) panel.scrollTop = bottom - panel.clientHeight
  })
}

function onDocPointerDown(e) {
  if (!open.value) return
  if (!rootRef.value?.contains(e.target)) close()
}

function onWindowScroll(e) {
  if (!open.value || !rootRef.value) return
  // 浮层自身滚动不关闭
  if (e.target instanceof Node && rootRef.value.contains(e.target)) return
  close()
}

onMounted(() => {
  document.addEventListener('mousedown', onDocPointerDown)
  document.addEventListener('touchstart', onDocPointerDown, { passive: true })
  window.addEventListener('resize', updatePlacement)
  window.addEventListener('scroll', onWindowScroll, true)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', onDocPointerDown)
  document.removeEventListener('touchstart', onDocPointerDown)
  window.removeEventListener('resize', updatePlacement)
  window.removeEventListener('scroll', onWindowScroll, true)
})
</script>

<style scoped>
.select-menu {
  position: relative;
  display: inline-flex;
}

/* ---------- 触发器（与主题的胶囊按钮一致） ---------- */
.select-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 220px;
  /* 与 .btn-sm / 筛选栏其它控件等高（见 styles.css 的 --control-h-* 令牌） */
  height: var(--control-h-sm);
  padding: 0 12px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  cursor: pointer;
  transition: border-color var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}
.select-trigger:hover:not(:disabled) {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.select-menu.is-open .select-trigger {
  border-color: var(--primary);
  color: var(--primary-deep);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.select-trigger:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.select-trigger-main {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.select-icon {
  font-size: 13px;
}
.select-value {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.select-value.is-placeholder {
  color: var(--text-light);
}
.select-caret {
  font-size: 11px;
  opacity: 0.7;
  transition: transform var(--dur-fast) var(--ease);
}
.select-menu.is-open .select-caret {
  transform: rotate(180deg);
}

/* ---------- 变体：表单输入框（对齐主题 .input） ---------- */
.select-menu.is-field {
  display: flex;
  width: 100%;
}
.select-menu.is-field .select-trigger {
  width: 100%;
  max-width: none;
  /* 与 .input 等高：表单里输入框与下拉不能一高一矮 */
  height: var(--control-h);
  justify-content: space-between;
  gap: 8px;
  padding: 0 12px;
  font-size: 14px;
  line-height: 1.5;
  border-radius: var(--radius);
  border-color: var(--border);
  background: var(--bg-muted);
  color: var(--text);
}
.select-menu.is-field .select-trigger:hover:not(:disabled) {
  border-color: var(--primary-soft);
  color: var(--text);
}
.select-menu.is-field.is-open .select-trigger {
  border-color: var(--primary);
  color: var(--text);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.select-menu.is-field .select-icon {
  font-size: 14px;
  color: var(--text-muted);
}
.select-menu.is-field .select-caret {
  font-size: 12px;
}
.select-menu.is-field .select-panel {
  max-width: none;
}
.select-menu.is-field .select-option {
  font-size: 14px;
}

/* ---------- 浮层 ---------- */
.select-panel {
  position: absolute;
  left: 0;
  z-index: 60;
  min-width: 100%;
  max-width: 320px;
  max-height: 280px;
  overflow-y: auto;
  padding: 4px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
}
.select-menu:not(.is-up) .select-panel {
  top: calc(100% + 6px);
}
.select-menu.is-up .select-panel {
  bottom: calc(100% + 6px);
}

.select-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.7;
  text-align: left;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.select-option-label {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.select-option.is-active {
  background: var(--bg-muted);
}
.select-option.is-selected {
  color: var(--primary-deep);
  font-weight: 600;
}
.select-option.is-selected .bi {
  color: var(--primary);
}
.select-option:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.select-empty {
  padding: 10px;
  font-size: 12px;
  color: var(--text-light);
  text-align: center;
}

/* ---------- 动效 ---------- */
.select-pop-enter-active,
.select-pop-leave-active {
  transition: opacity var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease);
}
.select-pop-enter-from,
.select-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
.select-menu.is-up .select-pop-enter-from,
.select-menu.is-up .select-pop-leave-to {
  transform: translateY(4px);
}
</style>
