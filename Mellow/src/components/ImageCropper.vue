<template>
  <Teleport to="body">
    <Transition name="cropper-fade">
      <div v-if="visible" class="cropper-mask" @click.self="onCancel">
        <div class="cropper-panel" role="dialog" aria-modal="true" :aria-label="title">
          <header class="cropper-head">
            <h3 class="cropper-title">
              <i class="bi bi-crop" aria-hidden="true" />
              {{ title }}
            </h3>
            <button type="button" class="cropper-close" title="关闭" :disabled="busy" @click="onCancel">
              <i class="bi bi-x-lg" />
            </button>
          </header>

          <div class="cropper-body">
            <!--
              取景框：宽高比由 --crop-aspect 控制（同时限制最大边长，窄屏自动收缩）。
              图片用 object-fit:cover 打底，再叠一层 translate + scale —— 拖动与缩放只改
              transform，不触发重排，大图也不掉帧。
            -->
            <div
              ref="frameRef"
              class="cropper-frame"
              :class="[`is-${shape}`, { 'is-dragging': dragging }]"
              :style="{ '--crop-aspect': aspect }"
              @pointerdown="onPointerDown"
              @pointermove="onPointerMove"
              @pointerup="onPointerUp"
              @pointercancel="onPointerUp"
              @wheel.prevent="onWheel"
            >
              <img
                v-if="imgSrc"
                ref="imgRef"
                class="cropper-img"
                :src="imgSrc"
                :crossorigin="isRemote ? 'anonymous' : undefined"
                :style="imgStyle"
                alt=""
                draggable="false"
                @load="onImgLoad"
                @error="onImgError"
                @dragstart.prevent
              />
              <div class="cropper-grid" aria-hidden="true" />
              <!-- 圆形取景：四周压暗 + 虚线圆，明确头像的最终形状 -->
              <template v-if="shape === 'circle'">
                <svg class="cropper-hole" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
                  <path d="M0 0H100V100H0Z M50 1A49 49 0 1 0 50 99A49 49 0 1 0 50 1Z" fill-rule="evenodd" />
                </svg>
                <span class="cropper-ring" aria-hidden="true" />
              </template>

              <div v-if="status === 'loading'" class="cropper-overlay">
                <span class="spinner" /> 读取图片…
              </div>
              <div v-else-if="status === 'error'" class="cropper-overlay">
                <i class="bi bi-image-alt" aria-hidden="true" /> 图片读取失败
              </div>
            </div>

            <div class="cropper-toolbar">
              <i class="bi bi-zoom-out cropper-zoom-icon" aria-hidden="true" />
              <input
                v-model.number="scale"
                class="cropper-range"
                type="range"
                :min="1"
                :max="maxZoom"
                step="0.01"
                :disabled="status !== 'ready'"
                aria-label="缩放"
              />
              <i class="bi bi-zoom-in cropper-zoom-icon" aria-hidden="true" />
              <span class="cropper-size">{{ sizeText }}</span>
            </div>
          </div>

          <footer class="cropper-actions">
            <span class="cropper-tip">拖动调整位置，滚轮 / 双指缩放</span>
            <button type="button" class="btn btn-sm" :disabled="busy" @click="onCancel">取消</button>
            <button
              type="button"
              class="btn btn-sm btn-primary"
              :disabled="status !== 'ready' || busy"
              @click="onConfirm"
            >
              {{ busy ? '处理中…' : '应用' }}
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * ImageCropper - 轻量图片裁切弹窗（不依赖第三方库）
 *
 * 能力：
 * - 固定宽高比取景（头像 1:1 圆形预览 / 封面 3:1），拖动平移 + 滚轮 / 双指 / 滑条缩放
 * - canvas 导出指定尺寸的 Blob（优先 WebP，失败回落 JPEG），交给调用方上传
 * - 只接受「能安全导出」的图源：本地 File（blob: 同源）或带 CORS 头的远程图。
 *   图床没有 Access-Control-Allow-Origin 时 canvas 会被污染，这里在 onerror 阶段就
 *   抛出 unsupported，由调用方直接使用原图，避免用户拖半天却导不出来。
 *
 * 性能取舍：
 * - 拖动 / 缩放只改 CSS transform，走合成层，不重排；
 * - pointermove 用 rAF 合帧，一帧最多写一次状态；
 * - 导出尺寸由调用方给定（头像 512、封面 1620），不按原图尺寸来，避免拉爆内存。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 二选一：File（本地选择）或 src（远程 / 附件库地址）
  file: { type: Object, default: null },
  src: { type: String, default: '' },
  title: { type: String, default: '裁切图片' },
  // 宽高比：1 = 正方形（头像），3 = 3:1（封面横幅）
  aspect: { type: Number, default: 1 },
  // square 方形取景 / circle 圆形取景（仅视觉提示，导出仍是方形）
  shape: { type: String, default: 'square' },
  // 导出宽度（px），高度 = outputWidth / aspect
  outputWidth: { type: Number, default: 512 },
  maxZoom: { type: Number, default: 4 },
  // 导出中：禁用按钮与关闭
  busy: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'confirm', 'unsupported'])

const frameRef = ref(null)
const imgRef = ref(null)

const imgSrc = ref('')
const objectUrl = ref('')
const status = ref('loading') // loading | ready | error
const natural = ref({ w: 0, h: 0 })
const frameSize = ref({ w: 0, h: 0 })

// 相对「铺满取景框」的缩放倍数：1 = 刚好铺满，小于 1 会露白，因此下限固定为 1
const scale = ref(1)
const dx = ref(0)
const dy = ref(0)
const dragging = ref(false)

const isRemote = computed(() => /^https?:\/\//i.test(imgSrc.value))

const imgStyle = computed(() => ({
  transform: `translate3d(${dx.value}px, ${dy.value}px, 0) scale(${scale.value})`
}))

const sizeText = computed(() => (
  natural.value.w && natural.value.h ? `${natural.value.w} × ${natural.value.h}` : ''
))

// 原图铺满取景框所需的像素倍数（object-fit:cover 的等效比例）
const coverScale = computed(() => {
  const { w, h } = frameSize.value
  const nat = natural.value
  if (!w || !h || !nat.w || !nat.h) return 1
  return Math.max(w / nat.w, h / nat.h)
})

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value))
}

function measureFrame() {
  const rect = frameRef.value?.getBoundingClientRect()
  if (!rect || !rect.width || !rect.height) return
  frameSize.value = { w: rect.width, h: rect.height }
}

function clampOffsets() {
  const { w, h } = frameSize.value
  const nat = natural.value
  if (!w || !h || !nat.w || !nat.h) return
  const k = coverScale.value * scale.value
  const maxX = Math.max(0, (nat.w * k - w) / 2)
  const maxY = Math.max(0, (nat.h * k - h) / 2)
  dx.value = clamp(dx.value, -maxX, maxX)
  dy.value = clamp(dy.value, -maxY, maxY)
}

// ---------- rAF 合帧：pointermove 可能一帧触发多次，只保留最后一次 ----------
let rafId = 0
let pending = null

function schedule(patch) {
  pending = { ...(pending || {}), ...patch }
  if (rafId) return
  rafId = requestAnimationFrame(() => {
    rafId = 0
    const batch = pending || {}
    pending = null
    if (batch.scale !== undefined) scale.value = clamp(batch.scale, 1, props.maxZoom)
    if (batch.dx !== undefined) dx.value = batch.dx
    if (batch.dy !== undefined) dy.value = batch.dy
    clampOffsets()
  })
}

// ---------- 图源装载 ----------
function releaseUrl() {
  if (objectUrl.value) URL.revokeObjectURL(objectUrl.value)
  objectUrl.value = ''
}

function prepare() {
  if (!props.visible) return
  releaseUrl()
  imgSrc.value = ''
  status.value = 'loading'
  scale.value = 1
  dx.value = 0
  dy.value = 0
  dragging.value = false
  natural.value = { w: 0, h: 0 }
  frameSize.value = { w: 0, h: 0 }

  if (props.file) {
    objectUrl.value = URL.createObjectURL(props.file)
    imgSrc.value = objectUrl.value
  } else if (props.src) {
    imgSrc.value = props.src
  } else {
    status.value = 'error'
  }
}

watch(
  () => [props.visible, props.file, props.src],
  () => {
    if (props.visible) prepare()
    else releaseUrl()
  },
  { immediate: true }
)

function onImgLoad() {
  const el = imgRef.value
  natural.value = { w: el?.naturalWidth || 0, h: el?.naturalHeight || 0 }
  if (!natural.value.w || !natural.value.h) {
    status.value = 'error'
    return
  }
  measureFrame()
  clampOffsets()
  status.value = 'ready'
}

function onImgError() {
  status.value = 'error'
  // 远程图大概率是图床没开 CORS：交给调用方直接使用原图
  if (isRemote.value) emit('unsupported')
}

// ---------- 手势：拖动 / 双指 / 滚轮 ----------
const pointers = new Map()
let dragStart = null
let pinchStart = null

function pointerDistance() {
  const list = [...pointers.values()]
  if (list.length < 2) return 0
  return Math.hypot(list[0].x - list[1].x, list[0].y - list[1].y)
}

function onPointerDown(event) {
  if (status.value !== 'ready' || props.busy) return
  frameRef.value?.setPointerCapture?.(event.pointerId)
  pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })
  if (pointers.size === 1) {
    dragStart = { x: event.clientX, y: event.clientY, dx: dx.value, dy: dy.value }
    dragging.value = true
  } else if (pointers.size === 2) {
    dragStart = null
    pinchStart = { dist: pointerDistance(), scale: scale.value }
  }
}

function onPointerMove(event) {
  if (!pointers.has(event.pointerId)) return
  pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })

  if (pointers.size >= 2 && pinchStart) {
    const dist = pointerDistance()
    if (pinchStart.dist > 0) schedule({ scale: pinchStart.scale * (dist / pinchStart.dist) })
    return
  }
  if (!dragStart) return
  schedule({
    dx: dragStart.dx + (event.clientX - dragStart.x),
    dy: dragStart.dy + (event.clientY - dragStart.y)
  })
}

function onPointerUp(event) {
  pointers.delete(event.pointerId)
  if (!pointers.size) {
    dragStart = null
    pinchStart = null
    dragging.value = false
  } else if (pointers.size === 1) {
    // 从双指退回单指：以当前指位重新开始拖动，避免位置跳变
    const [rest] = [...pointers.values()]
    dragStart = { x: rest.x, y: rest.y, dx: dx.value, dy: dy.value }
    pinchStart = null
  }
}

function onWheel(event) {
  if (status.value !== 'ready' || props.busy) return
  // 滚轮向上放大：deltaY 为负时 factor > 1
  const factor = Math.exp(-event.deltaY * 0.0015)
  schedule({ scale: scale.value * factor })
}

watch(scale, () => clampOffsets())

// ---------- 导出 ----------
function canvasToBlob(canvas) {
  return new Promise((resolve) => {
    try {
      canvas.toBlob((blob) => {
        if (blob) {
          resolve(blob)
          return
        }
        // 个别浏览器不支持 WebP 编码时回落 JPEG
        try {
          canvas.toBlob((fallback) => resolve(fallback), 'image/jpeg', 0.92)
        } catch {
          resolve(null)
        }
      }, 'image/webp', 0.92)
    } catch {
      resolve(null)
    }
  })
}

async function onConfirm() {
  if (status.value !== 'ready' || props.busy) return
  const el = imgRef.value
  const nat = natural.value
  const { w, h } = frameSize.value
  if (!el || !nat.w || !nat.h || !w || !h) return

  const outW = Math.max(1, Math.round(props.outputWidth))
  const outH = Math.max(1, Math.round(outW / (props.aspect || 1)))

  const k = coverScale.value * scale.value
  const sw = w / k
  const sh = h / k
  // 取景框中心对应的原图坐标（图片右移 dx，源图裁剪框就左移 dx / k）
  const sx = clamp(nat.w / 2 - dx.value / k - sw / 2, 0, Math.max(0, nat.w - sw))
  const sy = clamp(nat.h / 2 - dy.value / k - sh / 2, 0, Math.max(0, nat.h - sh))

  const canvas = document.createElement('canvas')
  canvas.width = outW
  canvas.height = outH
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    emit('unsupported')
    return
  }
  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'

  try {
    // 一次缩太狠（4000px → 512px）浏览器默认采样会发糊：
    // 先缩到 2 倍目标尺寸的中间画布，再缩到最终尺寸，两步降采样肉眼可见更干净。
    let source = el
    let sourceRect = { x: sx, y: sy, w: sw, h: sh }
    const stageW = outW * 2
    if (sw > stageW * 1.2) {
      const stage = document.createElement('canvas')
      stage.width = stageW
      stage.height = Math.max(1, Math.round((stageW * outH) / outW))
      const stageCtx = stage.getContext('2d')
      if (stageCtx) {
        stageCtx.imageSmoothingEnabled = true
        stageCtx.imageSmoothingQuality = 'high'
        stageCtx.drawImage(el, sx, sy, sw, sh, 0, 0, stage.width, stage.height)
        source = stage
        sourceRect = { x: 0, y: 0, w: stage.width, h: stage.height }
      }
    }
    ctx.drawImage(source, sourceRect.x, sourceRect.y, sourceRect.w, sourceRect.h, 0, 0, outW, outH)
  } catch {
    // 跨域图片被污染：交给调用方直接使用原图
    emit('unsupported')
    return
  }

  const blob = await canvasToBlob(canvas)
  if (!blob) {
    emit('unsupported')
    return
  }
  emit('confirm', { blob, width: outW, height: outH })
}

// ---------- 弹层：锁滚动 / Esc / 尺寸变化 ----------
let bodyOverflow = ''
let resizeFrame = 0

function onKeydown(event) {
  if (event.key === 'Escape') onCancel()
}

function onResize() {
  if (resizeFrame) return
  resizeFrame = requestAnimationFrame(() => {
    resizeFrame = 0
    if (status.value !== 'ready') return
    measureFrame()
    clampOffsets()
  })
}

function lockBody(lock) {
  if (lock) {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', onKeydown)
    window.addEventListener('resize', onResize)
  } else {
    document.body.style.overflow = bodyOverflow || ''
    document.removeEventListener('keydown', onKeydown)
    window.removeEventListener('resize', onResize)
  }
}

function onCancel() {
  if (props.busy) return
  emit('update:visible', false)
}

watch(
  () => props.visible,
  (val) => lockBody(val),
  { immediate: true }
)

onBeforeUnmount(() => {
  releaseUrl()
  lockBody(false)
  if (rafId) cancelAnimationFrame(rafId)
  if (resizeFrame) cancelAnimationFrame(resizeFrame)
})
</script>

<style scoped>
.cropper-mask {
  position: fixed;
  inset: 0;
  /* 页面弹窗 3000 < 裁切 3400 < 附件库 3500 < 确认弹窗 3600（见 styles.css 末尾层级说明） */
  z-index: 3400;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.5);
}
.cropper-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 460px;
  max-height: calc(100vh - 32px);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

/* ---------- 头部 ---------- */
.cropper-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-soft);
}
.cropper-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.cropper-title .bi {
  color: var(--primary);
}
.cropper-close {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background-color var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.cropper-close:hover:not(:disabled) {
  background: var(--bg-muted);
  color: var(--primary);
}
.cropper-close:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 取景框 ---------- */
.cropper-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  overflow-y: auto;
}
.cropper-frame {
  /* 最大边长 300px：宽高比由 --crop-aspect 决定，窄屏按百分比自动收缩 */
  position: relative;
  width: 100%;
  max-width: calc(300px * var(--crop-aspect, 1));
  aspect-ratio: var(--crop-aspect, 1);
  margin: 0 auto;
  background: #201d19;
  border-radius: var(--radius);
  overflow: hidden;
  cursor: grab;
  touch-action: none;
  user-select: none;
}
.cropper-frame.is-dragging {
  cursor: grabbing;
}
.cropper-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transform-origin: center;
  will-change: transform;
  pointer-events: none;
}
/* 三分网格：只做构图参考，不参与交互 */
.cropper-grid {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image:
    linear-gradient(to right, transparent calc(33.33% - 0.5px), rgba(255, 255, 255, 0.24) 33.33%, transparent calc(33.33% + 0.5px)),
    linear-gradient(to right, transparent calc(66.66% - 0.5px), rgba(255, 255, 255, 0.24) 66.66%, transparent calc(66.66% + 0.5px)),
    linear-gradient(to bottom, transparent calc(33.33% - 0.5px), rgba(255, 255, 255, 0.24) 33.33%, transparent calc(33.33% + 0.5px)),
    linear-gradient(to bottom, transparent calc(66.66% - 0.5px), rgba(255, 255, 255, 0.24) 66.66%, transparent calc(66.66% + 0.5px));
}
.cropper-hole {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  fill: rgba(18, 16, 14, 0.55);
}
.cropper-ring {
  position: absolute;
  inset: 0;
  border: 1px dashed rgba(255, 255, 255, 0.7);
  border-radius: 50%;
  pointer-events: none;
}
.cropper-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(18, 16, 14, 0.55);
  color: #f1ece2;
  font-size: 12px;
}

/* ---------- 缩放条 ---------- */
.cropper-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.cropper-zoom-icon {
  flex-shrink: 0;
  font-size: 13px;
  color: var(--text-light);
}
.cropper-range {
  flex: 1;
  min-width: 0;
  height: 22px;
  margin: 0;
  accent-color: var(--primary);
  cursor: pointer;
}
.cropper-range:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
.cropper-size {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

/* ---------- 底部 ---------- */
.cropper-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--border-soft);
}
.cropper-tip {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 过渡：遮罩淡入，卡片轻微上浮 */
.cropper-fade-enter-active,
.cropper-fade-leave-active {
  transition: opacity var(--dur) var(--ease);
}
.cropper-fade-enter-active .cropper-panel,
.cropper-fade-leave-active .cropper-panel {
  transition: opacity var(--dur) var(--ease), transform var(--dur) var(--ease);
}
.cropper-fade-enter-from,
.cropper-fade-leave-to {
  opacity: 0;
}
.cropper-fade-enter-from .cropper-panel,
.cropper-fade-leave-to .cropper-panel {
  opacity: 0;
  transform: translateY(-8px) scale(0.98);
}

@media (max-width: 640px) {
  .cropper-mask {
    align-items: flex-end;
    padding: 0;
  }
  .cropper-panel {
    max-width: none;
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
    padding-bottom: env(safe-area-inset-bottom);
  }
  .cropper-tip {
    display: none;
  }
}
</style>
