<template>
  <div class="theme-admin">
    <!-- 概览 -->
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">主题配色</h2>
          <p class="block-desc">
            设计 token 驱动全站：颜色、渐变、圆角、阴影与自定义变量；浅色 / 深色各一套，
            保存后立即生效（前台与管理后台同步换肤，无需重启）
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-ghost btn-sm" :disabled="loading" @click="load">
            <i class="bi bi-arrow-clockwise" /> 重新加载
          </button>
          <button class="btn btn-sm" :disabled="loading || saving" @click="askReset">
            <i class="bi bi-arrow-counterclockwise" /> 恢复默认
          </button>
          <button class="btn btn-primary btn-sm" :disabled="loading || saving" @click="onSave">
            <span v-if="saving" class="spinner" />
            <i v-else class="bi bi-check2" />
            {{ saving ? '保存中…' : '保存并应用' }}
          </button>
        </div>
      </header>

      <p v-if="loadError" class="stat-error">
        <i class="bi bi-exclamation-circle" aria-hidden="true" />
        <span>{{ loadError }}</span>
      </p>

      <div class="theme-meta">
        <span class="theme-chip">当前预设：{{ activePresetName }}</span>
        <span class="theme-chip">作用范围：全站（含未登录访客）</span>
        <span class="theme-chip">下发方式：<code>/api/theme/active</code></span>
        <span class="theme-chip" :class="{ 'is-dirty': dirty }">
          {{ dirty ? '有未保存的修改' : '配置已保存' }}
        </span>
      </div>
    </section>

    <!-- 内置预设 -->
    <section class="card card-pad">
      <header class="section-head">
        <div>
          <h3 class="section-title">内置预设</h3>
          <p class="section-desc">套用整套颜色，圆角 / 阴影 / 渐变与自定义变量保持不变</p>
        </div>
      </header>

      <div v-if="loading && !presets.length" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else class="preset-grid">
        <button
          v-for="preset in presets"
          :key="preset.key"
          type="button"
          class="preset-card"
          :class="{ 'is-active': form.preset === preset.key }"
          @click="applyPreset(preset)"
        >
          <span class="preset-swatches" aria-hidden="true">
            <i v-for="(color, index) in presetSwatches(preset)" :key="index" :style="{ background: color }" />
          </span>
          <span class="preset-body">
            <strong>{{ preset.name }}</strong>
            <small>{{ preset.desc }}</small>
          </span>
          <i v-if="form.preset === preset.key" class="bi bi-check-circle-fill preset-check" />
        </button>
      </div>
    </section>

    <!-- 编辑 + 预览 -->
    <div class="theme-layout">
      <section class="card card-pad">
        <div class="editor-head">
          <div class="mode-tabs" role="tablist" aria-label="配色模式">
            <button
              v-for="mode in modes"
              :key="mode.key"
              type="button"
              class="mode-tab"
              :class="{ 'is-active': currentMode === mode.key }"
              role="tab"
              :aria-selected="currentMode === mode.key"
              @click="currentMode = mode.key"
            >
              <i :class="mode.icon" /> {{ mode.label }}
            </button>
          </div>
          <button class="btn btn-ghost btn-sm" :disabled="!currentPreset" @click="resetMode">
            <i class="bi bi-arrow-counterclockwise" /> 本模式恢复预设
          </button>
        </div>

        <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <template v-else>
          <div v-for="group in groups" :key="group.key" class="token-group">
            <div class="token-group-head">
              <h4>{{ group.label }}</h4>
              <p v-if="group.desc">{{ group.desc }}</p>
            </div>

            <!-- 圆角组：批量统一 -->
            <div v-if="group.key === 'shape'" class="quick-radius">
              <span class="quick-radius-label">统一圆角</span>
              <FormInput v-model.number="quickRadius" class="quick-radius-input" type="number" min="0" max="40" step="1" />
              <span class="quick-radius-unit">px</span>
              <button class="btn btn-ghost btn-sm" type="button" @click="applyQuickRadius">应用到全部圆角</button>
            </div>

            <div class="token-grid">
              <div v-for="token in tokensOf(group.key)" :key="token.key" class="token-item">
                <label class="token-label">
                  {{ token.label }}
                  <code>--{{ token.key }}</code>
                </label>

                <!-- 纯色 -->
                <ColorPicker
                  v-if="token.type === 'color'"
                  :model-value="tokenValue(token)"
                  @update:model-value="setToken(token.key, $event)"
                />

                <!-- 纯色或渐变 -->
                <GradientEditor
                  v-else-if="token.type === 'paint'"
                  :model-value="tokenValue(token)"
                  @update:model-value="setToken(token.key, $event)"
                />

                <!-- 阴影 -->
                <ShadowEditor
                  v-else-if="token.type === 'shadow'"
                  :model-value="tokenValue(token)"
                  @update:model-value="setToken(token.key, $event)"
                />

                <!-- 长度 -->
                <div v-else class="length-field">
                  <FormInput
                    class="token-text"
                    :class="{ 'is-invalid': !!tokenValue(token) && !lengthValid(tokenValue(token)) }"
                    :value="tokenValue(token)"
                    type="text"
                    placeholder="10px / 0.6rem"
                    spellcheck="false"
                    @input="setToken(token.key, $event.target.value)" />
                  <span class="length-unit">{{ token.unit || 'px' }}</span>
                </div>

                <p v-if="token.desc" class="form-hint">{{ token.desc }}</p>
              </div>
            </div>
          </div>

          <!-- 自定义变量 -->
          <div class="token-group">
            <div class="token-group-head">
              <h4>自定义变量</h4>
              <p>任意 CSS 变量（写入 :root），可覆盖动效、间距、控件高度等未列出的项；浅色 / 深色共用</p>
            </div>

            <div v-if="extraKeys.length" class="extra-list">
              <div v-for="key in extraKeys" :key="key" class="extra-row">
                <code class="extra-key">--{{ key }}</code>
                <FormInput
                  class="extra-value"
                  :value="form.extras[key]"
                  type="text"
                  spellcheck="false"
                  @input="setExtra(key, $event.target.value)" />
                <button type="button" class="extra-remove" title="删除" @click="removeExtra(key)">
                  <i class="bi bi-x-lg" />
                </button>
              </div>
            </div>

            <div class="extra-add">
              <FormInput
                v-model="extraDraft.key"
                class="extra-key-input"
                type="text"
                placeholder="变量名，如 dur-fast"
                spellcheck="false" />
              <FormInput
                v-model="extraDraft.value"
                class="extra-value"
                type="text"
                placeholder="值，如 180ms"
                spellcheck="false" />
              <button type="button" class="btn btn-sm" :disabled="!extraDraft.key || !extraDraft.value" @click="addExtra">
                添加
              </button>
            </div>
            <p class="form-hint">
              变量名不需要写 <code>--</code>；带 <code>url()</code> 或 <code>; { }</code> 的值会被后端拒绝。
            </p>
          </div>
        </template>
      </section>

      <!-- 实时预览 -->
      <aside class="card card-pad preview-wrap">
        <header class="section-head">
          <div>
            <h3 class="section-title">实时预览</h3>
            <p class="section-desc">编辑任意项，这里立即变化</p>
          </div>
        </header>

        <div class="preview-pair">
          <div
            v-for="mode in modes"
            :key="`preview-${mode.key}`"
            class="theme-preview"
            :style="previewVars(mode.key)"
          >
            <span class="preview-mode">{{ mode.label }}</span>
            <span class="preview-title">纸与墨</span>
            <p class="preview-text">正文示例：暖纸为底，墨色为文，朱砂为印。</p>
            <div class="preview-row">
              <span class="preview-btn primary">主按钮</span>
              <span class="preview-btn ghost">次要</span>
              <span class="preview-chip">标签</span>
            </div>
            <div class="preview-card">
              <span class="preview-label">卡片底</span>
              <span class="preview-success">成功</span>
              <span class="preview-warning">警告</span>
            </div>
            <div class="preview-input">输入框占位…</div>
            <div class="preview-gradient" />
            <span class="preview-link">链接文字</span>
          </div>
        </div>

        <p class="form-hint">预览只覆盖本区域；点「保存并应用」后整站（含前台）才会切换。</p>
      </aside>
    </div>

    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      :confirm-text="confirm.confirmText"
      :danger="true"
      :loading="confirm.loading"
      @confirm="runConfirm"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * 主题配色（/admin/theme）
 *
 * 数据来自 /api/theme/config（app/api/controller/theme.go）：
 * - tokens / groups：带类型的设计 token（color / paint / length / shadow）
 * - presets：内置预设（颜色取值；圆角 / 阴影 / 渐变由后端默认值补齐）
 * - settings / defaults：当前配色与默认配色，extras 为自由 CSS 变量
 *
 * 保存后由 refreshTheme() 重新拉取 /api/theme/active（no-store），把新 CSS 注入
 * <style id="theme-runtime">，当前页立即换肤；不再依赖 /theme-custom.css 静态文件。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import ColorPicker from '@/components/ColorPicker.vue'
import GradientEditor from '@/components/GradientEditor.vue'
import ShadowEditor from '@/components/ShadowEditor.vue'
import { getThemeConfig, resetThemeConfig, saveThemeConfig } from '@/api/theme'
import { refreshTheme } from '@/utils/theme-runtime'
import { toast } from '@/utils/toast'

const loading = ref(false)
const saving = ref(false)
const loadError = ref('')

const tokens = ref([])
const groups = ref([])
const presets = ref([])
const defaults = ref({ preset: 'cinnabar', light: {}, dark: {}, extras: {} })

const form = reactive({ preset: 'cinnabar', light: {}, dark: {}, extras: {} })
const currentMode = ref('light')
const savedSnapshot = ref('')
const ready = ref(false)

const quickRadius = ref(10)
const extraDraft = reactive({ key: '', value: '' })

const modes = [
  { key: 'light', label: '浅色模式', icon: 'bi bi-sun' },
  { key: 'dark', label: '深色模式', icon: 'bi bi-moon-stars' }
]

const confirm = reactive({
  visible: false,
  loading: false,
  title: '',
  message: '',
  confirmText: '确定'
})

const activePresetName = computed(() => {
  const item = presets.value.find((preset) => preset.key === form.preset)
  return item?.name || form.preset || '—'
})

const currentPreset = computed(() => presets.value.find((preset) => preset.key === form.preset))

const dirty = computed(() => ready.value && snapshot() !== savedSnapshot.value)

const extraKeys = computed(() => Object.keys(form.extras || {}))

function tokensOf(group) {
  return tokens.value.filter((token) => token.group === group)
}

// ---------- 快照 ----------
function sorted(source) {
  const result = {}
  Object.keys(source || {})
    .sort()
    .forEach((key) => {
      result[key] = source[key]
    })
  return result
}

function snapshot() {
  return JSON.stringify({
    preset: form.preset,
    light: sorted(form.light),
    dark: sorted(form.dark),
    extras: sorted(form.extras)
  })
}

// ---------- 读取 / 回填 ----------
function applySettings(settings, markSaved = true) {
  form.preset = settings?.preset || defaults.value.preset || 'cinnabar'
  form.light = { ...(settings?.light || {}) }
  form.dark = { ...(settings?.dark || {}) }
  form.extras = { ...(settings?.extras || {}) }

  // 防御旧数据：缺项的 token 用默认值补齐
  tokens.value.forEach((token) => {
    if (!form.light[token.key]) form.light[token.key] = defaults.value.light?.[token.key] || ''
    if (!form.dark[token.key]) form.dark[token.key] = defaults.value.dark?.[token.key] || ''
  })

  // 圆角只保留一套：以浅色为准，旧数据里深色残留的不同值在这里统一收敛
  tokens.value.forEach((token) => {
    if (token.shared) form.dark[token.key] = form.light[token.key]
  })

  const base = lengthNumber(form.light.radius) || lengthNumber(form.dark.radius) || 10
  quickRadius.value = base

  ready.value = true
  if (markSaved) savedSnapshot.value = snapshot()
}

async function load() {
  if (loading.value) return
  loading.value = true
  loadError.value = ''

  try {
    const res = await getThemeConfig()
    const data = res?.data || {}

    tokens.value = Array.isArray(data.tokens) ? data.tokens : []
    groups.value = Array.isArray(data.groups) ? data.groups : []
    presets.value = Array.isArray(data.presets) ? data.presets : []
    defaults.value = data.defaults || { preset: 'cinnabar', light: {}, dark: {}, extras: {} }

    applySettings(data.settings || defaults.value)
  } catch (error) {
    loadError.value = error?.msg || '主题配色读取失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

// ---------- 编辑 ----------
// 圆角（shape 组）与明暗模式无关：读的时候统一取浅色那份，写的时候两套一起写，
// 这样在深色 tab 里看到的、改的都是同一套圆角，不会出现「深色圆角跟浅色不一致」。
function isSharedToken(token) {
  return !!token?.shared
}

function tokenValue(token) {
  if (isSharedToken(token)) {
    return form.light[token.key] || form.dark[token.key] || ''
  }
  return form[currentMode.value][token.key]
}

function setToken(key, value) {
  const token = tokens.value.find((item) => item.key === key)
  if (isSharedToken(token)) {
    form.light[key] = value
    form.dark[key] = value
    return
  }
  form[currentMode.value][key] = value
}

// 预设只带颜色；圆角 / 阴影 / 渐变 / 自定义变量是「与配色无关」的编辑，套预设时保留
function preserveNonColor() {
  const result = { light: {}, dark: {} }
  tokens.value.forEach((token) => {
    if (token.type === 'color') return
    result.light[token.key] = form.light[token.key]
    result.dark[token.key] = form.dark[token.key]
  })
  return result
}

function applyPreset(preset) {
  const keep = preserveNonColor()
  form.preset = preset.key
  form.light = { ...preset.light, ...keep.light }
  form.dark = { ...preset.dark, ...keep.dark }
}

function resetMode() {
  if (!currentPreset.value) return
  const mode = currentMode.value
  const keep = {}
  tokens.value.forEach((token) => {
    if (token.type !== 'color') keep[token.key] = form[mode][token.key]
  })
  form[mode] = { ...currentPreset.value[mode], ...keep }
}

// ---------- 长度 ----------
function lengthNumber(value) {
  const match = String(value || '').match(/^(-?[\d.]+)/)
  return match ? Number(match[1]) : 0
}

function lengthValid(value) {
  return /^(0|-?[\d.]+(px|rem|em|%|vh|vw|vmin|vmax|pt|ch))$/i.test(String(value || '').trim())
}

function applyQuickRadius() {
  const base = Math.max(0, Math.round(Number(quickRadius.value) || 0))
  // 圆角共用一套：两套模式同时写，切换 tab 不会看到不同结果
  const radius = {
    'radius-xs': `${Math.max(0, base - 4)}px`,
    'radius-sm': `${Math.max(0, base - 2)}px`,
    radius: `${base}px`,
    'radius-lg': `${base + 4}px`,
    'radius-xl': `${base + 8}px`
  }
  Object.keys(radius).forEach((key) => {
    form.light[key] = radius[key]
    form.dark[key] = radius[key]
  })
}

// ---------- 自定义变量 ----------
function addExtra() {
  const key = String(extraDraft.key || '')
    .trim()
    .toLowerCase()
    .replace(/^--/, '')

  if (!/^[a-z][a-z0-9-]{0,47}$/.test(key)) {
    toast.error('变量名只能是字母、数字与 -，且以字母开头')
    return
  }
  if (tokens.value.some((token) => token.key === key)) {
    toast.error('该变量名与内置 token 重名，请直接编辑对应项')
    return
  }
  if (!String(extraDraft.value || '').trim()) {
    toast.error('请填写变量的值')
    return
  }

  form.extras[key] = String(extraDraft.value).trim()
  extraDraft.key = ''
  extraDraft.value = ''
}

function setExtra(key, value) {
  form.extras[key] = value
}

function removeExtra(key) {
  delete form.extras[key]
}

// ---------- 预览 ----------
function previewVars(mode) {
  const values = form[mode] || {}
  const vars = {}

  Object.keys(values).forEach((key) => {
    if (values[key]) vars[`--${key}`] = values[key]
  })

  Object.keys(form.extras || {}).forEach((key) => {
    if (form.extras[key]) vars[`--${key}`] = form.extras[key]
  })

  const accentAlpha = mode === 'dark' ? ['0.1', '0.18', '0.3', '0.45'] : ['0.06', '0.12', '0.2', '0.38']
  ;['accent-wash', 'accent-soft', 'accent-ring', 'accent-glow'].forEach((name, index) => {
    vars[`--${name}`] = hexToRgba(values.cinnabar, accentAlpha[index])
  })

  const goldAlpha = mode === 'dark' ? ['0.12', '0.18', '0.34'] : ['0.1', '0.16', '0.32']
  ;['gold-wash', 'gold-soft', 'gold-line'].forEach((name, index) => {
    vars[`--${name}`] = hexToRgba(values.warning, goldAlpha[index])
  })

  // 表面类 token 可能是渐变，不能直接塞给 background-color（会失效变成透明）。
  // 这里拆成「纯色兜底 + 渐变图层」两部分，预览才既能看到底色又能看到渐变。
  vars['--bg'] = paintColor(values['paper-1'])
  vars['--bg-soft'] = paintColor(values['paper-1'])
  vars['--bg-alt'] = paintColor(values['paper-1'])
  vars['--bg-card'] = paintColor(values['paper-0'])
  vars['--bg-muted'] = paintColor(values['paper-2'])
  vars['--bg-overlay'] = hexToRgba(mode === 'dark' ? paintColor(values['paper-1']) : paintColor(values['paper-0']), '0.86')

  const pageImage = paintImage(values['gradient-page'])
  vars['--preview-page-image'] = pageImage !== 'none' ? pageImage : paintImage(values['paper-1'])
  vars['--preview-card-image'] = paintImage(values['paper-0'])

  vars['--text'] = values['ink-900']
  vars['--text-soft'] = values['ink-700']
  vars['--text-muted'] = values['ink-500']
  vars['--text-light'] = values['ink-300']

  vars['--border'] = values.line
  vars['--border-soft'] = values['line-soft']

  vars['--primary'] = values.cinnabar
  vars['--primary-deep'] = values['cinnabar-deep']
  vars['--primary-soft'] = values['cinnabar-soft']
  vars['--accent'] = values.cinnabar
  vars['--link'] = values['cinnabar-deep']
  vars['--link-hover'] = values.cinnabar

  vars['--success'] = values.bamboo
  vars['--danger'] = values.cinnabar
  vars['--warning'] = values.warning

  return vars
}

function presetSwatches(preset) {
  const values = preset.light || {}
  return [values['paper-1'], values['paper-0'], values.cinnabar, values.bamboo].filter(Boolean)
}

// ---------- 颜色工具 ----------
function isGradient(value) {
  return /gradient\(/i.test(String(value || ''))
}

/** 从渐变里取第一个颜色，作为 background-color 兜底（渐变本身进 background-image） */
function firstStopColor(value) {
  const match = String(value || '').match(/(#[0-9a-f]{3,8}|rgba?\([^)]*\)|hsla?\([^)]*\))/i)
  return match ? match[1] : ''
}

/** 纯色返回原值；渐变返回首个色标颜色（避免 background-color 失效变透明） */
function paintColor(value) {
  const text = String(value || '').trim()
  if (!text) return ''
  return isGradient(text) ? firstStopColor(text) : text
}

/** 渐变返回原值，纯色返回 none（用于 background-image） */
function paintImage(value) {
  const text = String(value || '').trim()
  return isGradient(text) ? text : 'none'
}

function hexToRgba(value, alpha) {
  const text = String(value || '').trim().toLowerCase()
  let hex = text.startsWith('#') ? text.slice(1) : ''
  if (hex.length === 8) hex = hex.slice(0, 6)
  if (hex.length === 3) hex = `${hex[0]}${hex[0]}${hex[1]}${hex[1]}${hex[2]}${hex[2]}`
  if (!/^[0-9a-f]{6}$/.test(hex)) return `rgba(128, 128, 128, ${alpha})`

  const number = parseInt(hex, 16)
  return `rgba(${(number >> 16) & 0xff}, ${(number >> 8) & 0xff}, ${number & 0xff}, ${alpha})`
}

// ---------- 保存 / 重置 ----------
/**
 * 保存后立刻把新样式注入 <style id="theme-runtime">
 * （/api/theme/active 是 no-store，不会像 /theme-custom.css 那样被 CDN 缓存住）
 */
function refreshThemeStyle() {
  refreshTheme().catch(() => {
    /* 失败时保留当前样式，下次整页加载会再校准 */
  })
}

async function onSave() {
  if (saving.value) return
  saving.value = true

  try {
    const res = await saveThemeConfig({
      preset: form.preset,
      light: form.light,
      dark: form.dark,
      extras: form.extras
    })

    applySettings(res?.data?.settings || form)
    refreshThemeStyle()
    toast.success('主题配色已保存并立即生效')
  } catch {
    /* 响应拦截器已提示错误 */
  } finally {
    saving.value = false
  }
}

function askReset() {
  confirm.title = '恢复默认配色'
  confirm.message = '将丢弃所有自定义颜色、圆角、阴影、渐变与自定义变量，恢复为「朱砂暖纸」默认预设。'
  confirm.confirmText = '恢复默认'
  confirm.visible = true
}

async function runConfirm() {
  if (confirm.loading) return
  confirm.loading = true

  try {
    const res = await resetThemeConfig()
    applySettings(res?.data?.settings || defaults.value)
    refreshThemeStyle()
    confirm.visible = false
    toast.success('已恢复默认配色')
  } catch {
    /* 响应拦截器已提示错误 */
  } finally {
    confirm.loading = false
  }
}

onMounted(load)
</script>

<style scoped>
.theme-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 概览 ---------- */
.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.block-title {
  margin: 0;
  font-size: 18px;
}
.block-desc {
  margin: 6px 0 0;
  max-width: 74ch;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.7;
}
.head-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.theme-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 14px;
}
.theme-chip {
  padding: 4px 10px;
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 12px;
}
.theme-chip code {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
}
.theme-chip.is-dirty {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.stat-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 12px 0 0;
  color: var(--danger);
  font-size: 12px;
}

/* ---------- 小节 ---------- */
.section-head {
  margin-bottom: 14px;
}
.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.section-desc {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: 12px;
}
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}

/* ---------- 预设 ---------- */
.preset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 12px;
}
.preset-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}
.preset-card:hover {
  border-color: var(--primary-soft);
}
.preset-card.is-active {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.preset-swatches {
  display: grid;
  grid-template-columns: repeat(2, 14px);
  grid-template-rows: repeat(2, 14px);
  flex-shrink: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 6px;
}
.preset-swatches i {
  display: block;
  width: 14px;
  height: 14px;
}
.preset-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.preset-body strong {
  font-size: 13px;
}
.preset-body small {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.5;
}
.preset-check {
  position: absolute;
  top: 10px;
  right: 10px;
  color: var(--primary);
  font-size: 13px;
}

/* ---------- 编辑区 ---------- */
.theme-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 16px;
  align-items: start;
}
/* grid 子项默认 min-width:auto，内部宽控件（阴影 / 渐变 / 颜色选择器）
   会把整列撑破，右侧预览被挤到视口外 */
.theme-layout > * {
  min-width: 0;
}
@media (max-width: 1180px) {
  .theme-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
.editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.mode-tabs {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: var(--radius);
  background: var(--bg-muted);
}
.mode-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: var(--control-h-sm);
  padding: 0 12px;
  border-radius: var(--radius-sm);
  color: var(--text-soft);
  font-size: 12px;
  cursor: pointer;
  transition: all var(--dur-fast) var(--ease);
}
.mode-tab:hover {
  color: var(--text);
}
.mode-tab.is-active {
  background: var(--bg-card);
  color: var(--primary);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}
.token-group + .token-group {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px dashed var(--border-soft);
}
.token-group-head h4 {
  margin: 0;
  font-size: 13px;
}
.token-group-head p {
  margin: 2px 0 12px;
  color: var(--text-muted);
  font-size: 12px;
}
.token-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.token-item {
  min-width: 0;
}
.token-label {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
  color: var(--text-soft);
  font-size: 12px;
}
.token-label code {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
}
.token-item .form-hint {
  margin: 6px 0 0;
  font-size: 11px;
}
.token-text {
  height: 38px;
  font-family: var(--font-mono);
  font-size: 12px;
}
.token-text.is-invalid {
  border-color: var(--danger);
}
.length-field {
  position: relative;
  display: flex;
  align-items: center;
  min-width: 0;
}
.length-field .token-text {
  flex: 1;
  width: auto;
  min-width: 0;
  padding-right: 38px;
}
.length-unit {
  position: absolute;
  top: 50%;
  right: 10px;
  color: var(--text-light);
  font-size: 11px;
  pointer-events: none;
  transform: translateY(-50%);
}

/* 统一圆角 */
.quick-radius {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 10px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  background: var(--bg-soft);
}
.quick-radius-label {
  color: var(--text-soft);
  font-size: 12px;
}
.quick-radius-input {
  width: 68px;
  height: 30px;
  font-family: var(--font-mono);
  font-size: 12px;
}
.quick-radius-unit {
  color: var(--text-light);
  font-size: 11px;
}

/* 自定义变量 */
.extra-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.extra-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.extra-key {
  flex: 0 0 150px;
  overflow: hidden;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.extra-value {
  flex: 1;
  height: 32px;
  font-family: var(--font-mono);
  font-size: 12px;
}
.extra-remove {
  flex: 0 0 26px;
  height: 26px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--text-light);
  font-size: 10px;
  cursor: pointer;
}
.extra-remove:hover {
  border-color: var(--border);
  color: var(--danger);
}
.extra-add {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}
.extra-key-input {
  flex: 0 1 150px;
  min-width: 120px;
  height: 32px;
  font-family: var(--font-mono);
  font-size: 12px;
}

/* ---------- 预览 ---------- */
.preview-wrap {
  position: sticky;
  top: 16px;
}
.preview-pair {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}
.theme-preview {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background-color: var(--bg);
  background-image: var(--preview-page-image, none);
  box-shadow: var(--shadow);
  color: var(--text);
}
.preview-mode {
  color: var(--text-muted);
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
.preview-title {
  color: var(--text);
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 900;
}
.preview-text {
  margin: 0;
  color: var(--text-soft);
  font-size: 12px;
  line-height: 1.6;
}
.preview-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.preview-btn {
  padding: 5px 10px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  font-size: 11px;
}
.preview-btn.primary {
  background-color: var(--primary);
  background-image: var(--gradient-brand);
  color: #fff;
}
.preview-btn.ghost {
  border-color: var(--border);
  color: var(--text-soft);
}
.preview-chip {
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 10px;
}
.preview-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background-color: var(--bg-card);
  background-image: var(--preview-card-image, none);
  box-shadow: var(--shadow-sm);
}
.preview-label {
  color: var(--text-muted);
  font-size: 11px;
}
.preview-success {
  color: var(--success);
  font-size: 11px;
}
.preview-warning {
  color: var(--warning);
  font-size: 11px;
}
.preview-input {
  display: flex;
  align-items: center;
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 11px;
}
.preview-gradient {
  height: 22px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  background-color: var(--primary-soft);
  background-image: var(--gradient-brand);
}
.preview-link {
  color: var(--link);
  font-size: 11px;
}
</style>
