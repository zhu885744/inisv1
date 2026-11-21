<template>
  <Teleport to="body">
    <Transition name="captcha-fade">
      <div v-if="visible" class="captcha-mask" @click.self="onCancel">
        <div class="captcha-box" role="dialog" aria-modal="true" aria-label="安全验证">
          <header class="captcha-head">
            <span class="captcha-badge"><i class="bi bi-shield-check" /></span>
            <div class="captcha-head-text">
              <div class="captcha-title">安全验证</div>
              <div class="captcha-sub">拖动拼图块，与图中缺口完全重合</div>
            </div>
            <button class="captcha-close" type="button" aria-label="关闭" @click="onCancel">
              <i class="bi bi-x-lg" />
            </button>
          </header>

          <!-- 图片区：轨道与拼图块是同一张 SVG 的两个实例，
               拼图块通过负偏移把「缺口那一块图」切出来，拖到位时纹理连成一片 -->
          <div ref="trackRef" class="captcha-track" :style="{ height: `${TRACK_HEIGHT}px` }">
            <CaptchaScene :width="renderedWidth" :height="TRACK_HEIGHT" />

            <span
              class="captcha-gap"
              :class="{ 'is-aligned': aligned }"
              :style="gapStyle"
            />

            <span
              class="captcha-piece"
              :class="{ 'is-dragging': dragging, 'is-aligned': aligned }"
              :style="pieceStyle"
              @pointerdown="onPieceDown"
            >
              <span class="captcha-piece-view" :style="pieceViewStyle">
                <CaptchaScene :width="renderedWidth" :height="TRACK_HEIGHT" />
              </span>
            </span>
          </div>

          <div class="captcha-bar" :class="{ 'is-dragging': dragging }">
            <span class="captcha-bar-fill" :style="{ width: `${fillWidth}px` }" />
            <span class="captcha-bar-label">{{ barText }}</span>
            <button
              type="button"
              class="captcha-handle"
              :class="{ 'is-dragging': dragging }"
              :style="{ transform: `translateX(${handleLeft}px)` }"
              :disabled="submitting || !!limitedMessage"
              @pointerdown="onPointerDown"
            >
              <i :class="dragging ? 'bi bi-chevron-double-right' : 'bi bi-arrow-right'" />
            </button>
          </div>

          <footer class="captcha-foot">
            <span class="captcha-foot-tip" :class="{ 'is-warn': !!limitedMessage }">
              <i class="bi bi-info-circle" aria-hidden="true" />
              {{ limitedMessage || '验证通过后会自动继续刚才的操作' }}
            </span>
            <button class="captcha-refresh" type="button" :disabled="refreshing" @click="refresh">
              <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': refreshing }" /> 换一题
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * SliderCaptcha - 全局滑块验证弹窗
 *
 * 与后端的协作方式：
 *   1. 发送验证码 / 敏感操作需要人机校验时，后端返回 428 + { need_captcha, captcha }；
 *   2. request.js 的响应拦截器把 captcha 交给 captchaGate → 本组件弹出；
 *   3. 用户拖动拼图块，松手即提交 { captcha_token, captcha_value }（落点百分比）；
 *   4. 拦截器用这对票据自动重放原请求，调用方无感知。
 *
 * 视觉实现：轨道与拼图块各渲染一份 CaptchaScene（同一张 SVG），
 * 拼图块用负偏移把「缺口位置」的那块图切进来 —— 因此拖到位时纹理是连续的。
 * 前端只提交百分比，是否达标由服务端容差判定（见 app/model/captcha.go）。
 */
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { registerSliderHandler, unregisterSliderHandler } from '@/utils/captchaGate'
import { getSliderCaptcha } from '@/api/comm'
import CaptchaScene from '@/components/CaptchaScene.vue'

// 轨道尺寸：宽度按实际渲染宽度自适应（窄屏缩放后百分比换算仍然正确）
const TRACK_WIDTH = 300
const TRACK_HEIGHT = 52
const PIECE_SIZE = 34
const HANDLE_SIZE = 48
// 松手时小于这个位移视为「只是点了一下」，不提交、不消耗题目
const MIN_DRAG = 4
// 与后端容差一致（3%）：用于「已对齐」的视觉反馈，让用户更容易命中
const TOLERANCE_PERCENT = 3

const visible = ref(false)
const dragging = ref(false)
const submitting = ref(false)
const refreshing = ref(false)
const challenge = ref(null)
const offset = ref(0)
const limitedMessage = ref('')

const trackRef = ref(null)
const renderedWidth = ref(TRACK_WIDTH)

let resolver = null
let startX = 0
let startOffset = 0

const maxOffset = computed(() => Math.max(1, renderedWidth.value - PIECE_SIZE))

/**
 * 缺口位置：把后端的 target 百分比映射到「可拖动范围」而不是整条轨道宽度
 *
 * 这样缺口永远落在拼图块拖得到的范围内（窄屏缩小时不会因为拖不到而被判错），
 * 且提交值 offset / maxOffset 与显示位置是同一套映射，往返完全对称。
 */
const gapLeft = computed(() => {
  const target = Number(challenge.value?.target ?? 50)
  const percent = Math.min(100, Math.max(0, target))
  return (maxOffset.value * percent) / 100
})

// 拖动中且已经落进容差范围：给缺口与拼图块加高亮，提示「可以松手了」
const aligned = computed(() => {
  if (!dragging.value) return false
  const tolerance = Math.max(3, (maxOffset.value * TOLERANCE_PERCENT) / 100)
  return Math.abs(offset.value - gapLeft.value) <= tolerance
})

const handleLeft = computed(() =>
  Math.min(offset.value, Math.max(0, renderedWidth.value - HANDLE_SIZE))
)

const fillWidth = computed(() => handleLeft.value + HANDLE_SIZE)

const gapStyle = computed(() => ({
  left: `${gapLeft.value}px`,
  width: `${PIECE_SIZE}px`,
  height: `${PIECE_SIZE}px`
}))

const pieceStyle = computed(() => ({
  left: `${offset.value}px`,
  width: `${PIECE_SIZE}px`,
  height: `${PIECE_SIZE}px`
}))

// 拼图块内部那张图：整体偏移到缺口位置，等效于「裁下缺口那块像素」
const pieceViewStyle = computed(() => ({
  left: `-${gapLeft.value}px`,
  top: `-${(TRACK_HEIGHT - PIECE_SIZE) / 2}px`,
  width: `${renderedWidth.value}px`,
  height: `${TRACK_HEIGHT}px`
}))

const barText = computed(() => {
  if (refreshing.value) return '正在获取题目...'
  if (submitting.value) return '验证中...'
  if (dragging.value) return aligned.value ? '已对齐，松手即可' : '松手即完成验证'
  return '按住滑块，拖到缺口处'
})

function resetOffset() {
  offset.value = 0
}

// 测量实际渲染宽度：窄屏 / 缩放环境下百分比换算仍要正确
function measureTrack() {
  const width = trackRef.value?.clientWidth
  if (width && width > 0) {
    renderedWidth.value = width
    offset.value = Math.min(offset.value, Math.max(0, width - PIECE_SIZE))
  }
}

function open(item) {
  challenge.value = item || null
  limitedMessage.value = item?.limited ? item.message || '操作过于频繁，请稍后再试！' : ''
  resetOffset()
  dragging.value = false
  submitting.value = false
  visible.value = true
  nextTick(measureTrack)
}

function close(ticket) {
  visible.value = false
  challenge.value = null
  resetOffset()
  const done = resolver
  resolver = null
  if (done) done(ticket || null)
}

function onCancel() {
  close(null)
}

// 拖动开始：底部滑块与拼图块都可以抓
function startDrag(event) {
  if (submitting.value || refreshing.value || limitedMessage.value) return
  event.preventDefault()
  dragging.value = true
  startX = event.clientX
  startOffset = offset.value
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('pointercancel', onPointerUp)
}

function onPointerDown(event) {
  startDrag(event)
}

function onPieceDown(event) {
  startDrag(event)
}

function onPointerMove(event) {
  if (!dragging.value) return
  const next = startOffset + (event.clientX - startX)
  offset.value = Math.min(maxOffset.value, Math.max(0, next))
}

function onPointerUp() {
  if (!dragging.value) return
  dragging.value = false
  removePointerListeners()

  // 只是点了一下没拖动：不消耗题目，也不提交（避免瞬间失败）
  if (offset.value < MIN_DRAG) {
    resetOffset()
    return
  }

  const token = challenge.value?.token
  if (!token) return

  // 落点按「拼图块左边缘」在可拖动范围内的百分比计算，与缺口位置口径一致
  const percent = Math.round((offset.value / maxOffset.value) * 100)

  submitting.value = true
  close({ token, value: percent })
}

function removePointerListeners() {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('pointercancel', onPointerUp)
}

// 换一题：题目过期或校验失败次数超限后使用
async function refresh() {
  if (refreshing.value) return
  refreshing.value = true
  limitedMessage.value = ''
  try {
    const res = await getSliderCaptcha()
    const data = res?.data
    if (data?.token) {
      challenge.value = data
      resetOffset()
      nextTick(measureTrack)
    } else if (data?.limited) {
      limitedMessage.value = data.message || '操作过于频繁，请稍后再试！'
    }
  } catch {
    // 取题失败：保留原题，用户仍可重试
  } finally {
    refreshing.value = false
  }
}

function onResize() {
  if (visible.value) measureTrack()
}

onMounted(() => {
  registerSliderHandler((item) => {
    // 上一次没结束就再来一次：把上一次当作「取消」，避免旧 Promise 永远挂起
    if (resolver) {
      const stale = resolver
      resolver = null
      stale(null)
    }
    return new Promise((resolve) => {
      resolver = resolve
      open(item)
    })
  })
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  unregisterSliderHandler()
  removePointerListeners()
  window.removeEventListener('resize', onResize)
  if (resolver) {
    const done = resolver
    resolver = null
    done(null)
  }
})
</script>

<style scoped>
/* ---------- 弹窗容器 ---------- */
.captcha-mask {
  position: fixed;
  inset: 0;
  z-index: 4000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(15, 23, 42, 0.5);
  backdrop-filter: blur(3px);
}
.captcha-box {
  width: 100%;
  max-width: 376px;
  padding: 16px 18px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 4px;
  box-shadow: 0 20px 52px rgba(15, 23, 42, 0.26);
}

/* ---------- 头部 ---------- */
.captcha-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.captcha-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  border-radius: 4px;
  background: linear-gradient(135deg, var(--primary), var(--primary-deep));
  color: #fff;
  font-size: 16px;
  box-shadow: 0 4px 12px rgba(140, 40, 40, 0.28);
}
.captcha-head-text {
  flex: 1;
  min-width: 0;
}
.captcha-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  line-height: 1.3;
}
.captcha-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.4;
}
.captcha-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text-light);
  cursor: pointer;
  transition: all 0.15s;
}
.captcha-close:hover {
  background: var(--bg-muted);
  color: var(--text);
}

/* ---------- 图片区 ---------- */
.captcha-track {
  position: relative;
  width: 100%;
  overflow: hidden;
  border-radius: 4px;
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.35),
    0 2px 10px rgba(15, 23, 42, 0.12);
  touch-action: none;
}

/* 缺口：像被挖掉一块（内阴影 + 半透明白虚线描边） */
.captcha-gap {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  border-radius: 4px;
  border: 1.5px dashed rgba(255, 255, 255, 0.85);
  background: rgba(15, 23, 42, 0.32);
  box-shadow: inset 0 4px 10px rgba(9, 15, 30, 0.5);
  transition: border-color 0.15s, background 0.15s;
}
.captcha-gap.is-aligned {
  border-color: #fff;
  background: rgba(15, 23, 42, 0.18);
}

/* 拼图块：内部是缺口那块图，靠 inset 阴影描边（不位移纹理） */
.captcha-piece {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  overflow: hidden;
  border-radius: 4px;
  cursor: grab;
  box-shadow:
    inset 0 0 0 2px rgba(255, 255, 255, 0.92),
    0 4px 12px rgba(9, 15, 30, 0.38);
  transition: box-shadow 0.15s;
}
.captcha-piece.is-dragging {
  cursor: grabbing;
  box-shadow:
    inset 0 0 0 2px rgba(255, 255, 255, 1),
    0 8px 20px rgba(9, 15, 30, 0.44);
}
.captcha-piece.is-aligned {
  box-shadow:
    inset 0 0 0 2px #ffffff,
    0 0 0 3px rgba(255, 255, 255, 0.55),
    0 8px 20px rgba(9, 15, 30, 0.4);
}
.captcha-piece-view {
  position: absolute;
  pointer-events: none;
}

/* ---------- 拖动条 ---------- */
.captcha-bar {
  position: relative;
  width: 100%;
  height: 46px;
  margin-top: 14px;
  border-radius: 4px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  overflow: hidden;
  user-select: none;
}
.captcha-bar-fill {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  background: var(--primary);
  opacity: 0.15;
  transition: width 0.06s linear;
}
.captcha-bar-label {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding-left: 10px;
  font-size: 12.5px;
  letter-spacing: 0.02em;
  color: var(--text-muted);
  pointer-events: none;
}
.captcha-bar.is-dragging .captcha-bar-label {
  color: var(--primary-deep);
}
.captcha-handle {
  position: absolute;
  top: 0;
  left: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 100%;
  border: none;
  border-radius: 4px;
  background: linear-gradient(135deg, var(--primary), var(--primary-deep));
  color: #fff;
  font-size: 17px;
  cursor: grab;
  touch-action: none;
  box-shadow: 0 2px 10px rgba(140, 40, 40, 0.32);
  transition: transform 0.06s linear, box-shadow 0.15s;
}
.captcha-handle:hover:not(:disabled) {
  box-shadow: 0 4px 16px rgba(140, 40, 40, 0.42);
}
.captcha-handle.is-dragging {
  cursor: grabbing;
}
.captcha-handle:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* ---------- 底部 ---------- */
.captcha-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 10px;
}
.captcha-foot-tip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: 12px;
  color: var(--text-light);
  line-height: 1.4;
}
.captcha-foot-tip.is-warn {
  color: var(--danger);
}
.captcha-refresh {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: transparent;
  color: var(--text-muted);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}
.captcha-refresh:hover:not(:disabled) {
  border-color: var(--primary-soft, rgba(140, 40, 40, 0.35));
  color: var(--primary);
}
.captcha-refresh:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.captcha-refresh .is-spin {
  animation: captcha-spin 0.9s linear infinite;
}
@keyframes captcha-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ---------- 过渡 ---------- */
.captcha-fade-enter-active,
.captcha-fade-leave-active {
  transition: opacity 0.18s ease;
}
.captcha-fade-enter-active .captcha-box,
.captcha-fade-leave-active .captcha-box {
  transition: transform 0.18s ease, opacity 0.18s ease;
}
.captcha-fade-enter-from,
.captcha-fade-leave-to {
  opacity: 0;
}
.captcha-fade-enter-from .captcha-box,
.captcha-fade-leave-to .captcha-box {
  transform: translateY(-8px) scale(0.97);
  opacity: 0;
}

@media (max-width: 420px) {
  .captcha-box {
    padding: 14px 14px 12px;
    border-radius: 4px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .captcha-bar-fill,
  .captcha-handle,
  .captcha-piece,
  .captcha-gap {
    transition: none;
  }
}
</style>
