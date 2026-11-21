<template>
  <span
    class="live-photo"
    :class="[`is-${fit}`, { 'is-square': square, 'is-playing': playing, 'is-broken': broken }]"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
    @pointerdown="onPressStart"
    @pointerup="onPressEnd"
    @pointercancel="onPressEnd"
    @pointerleave="onPressEnd"
  >
    <img
      :src="image"
      :alt="alt"
      class="live-photo-image"
      loading="lazy"
      decoding="async"
      draggable="false"
    />
    <video
      v-if="video && !broken"
      ref="videoRef"
      :src="video"
      class="live-photo-video"
      muted
      playsinline
      webkit-playsinline
      loop
      preload="metadata"
      @error="broken = true"
    />
    <span v-if="video && !broken" class="live-photo-badge">
      <i class="bi bi-record-circle-fill" aria-hidden="true" />
      LIVE
    </span>
  </span>
</template>

<script setup>
/**
 * 实况图组件（动态卡片 / 动态详情 / 后台预览共用）
 *
 * 交互：桌面端悬停播放、移开立即暂停并复位；触屏端按住播放、松手暂停。
 * 视频一律静音 + loop（实况图只有 1~3 秒，带声音会显得很吵），
 * 播不出来（编码不支持 / 文件 404）就把整层视频摘掉，只留静态图，不报错。
 */
import { onBeforeUnmount, ref } from 'vue'

const props = defineProps({
  image: { type: String, default: '' },
  video: { type: String, default: '' },
  alt: { type: String, default: '' },
  /** cover：填满格子（九宫格）；contain：完整展示（单图） */
  fit: { type: String, default: 'cover' },
  /** 正方形裁切（九宫格 / 多图） */
  square: { type: Boolean, default: true }
})

const videoRef = ref(null)
const playing = ref(false)
const broken = ref(false)

/** 有 hover 能力的设备（桌面）用悬停，触屏用按住 */
const canHover = (() => {
  if (typeof window === 'undefined' || !window.matchMedia) return true
  return window.matchMedia('(hover: hover)').matches
})()

let pressTimer = null

async function play() {
  if (!props.video || broken.value) return
  const el = videoRef.value
  if (!el) return
  try {
    el.currentTime = 0
    await el.play()
    playing.value = true
  } catch {
    // 自动播放被拦（极少见于 muted）或编码不支持：静默降级成静态图
    broken.value = true
  }
}

function stop() {
  const el = videoRef.value
  playing.value = false
  if (!el) return
  el.pause()
  try {
    el.currentTime = 0
  } catch {
    /* 某些浏览器在元数据未就绪时不允许设置 currentTime */
  }
}

function onEnter() {
  if (canHover) play()
}

function onLeave() {
  if (canHover) stop()
}

/** 触屏：按住 120ms 才播放，避免滑动列表时误触发 */
function onPressStart() {
  if (canHover) return
  clearTimeout(pressTimer)
  pressTimer = setTimeout(play, 120)
}

function onPressEnd() {
  if (canHover) return
  clearTimeout(pressTimer)
  stop()
}

onBeforeUnmount(() => {
  clearTimeout(pressTimer)
  stop()
})
</script>

<style scoped>
.live-photo {
  position: relative;
  display: block;
  width: 100%;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  line-height: 0;
}

.live-photo.is-square {
  aspect-ratio: 1;
}

.live-photo-image,
.live-photo-video {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
  transition: opacity 0.18s ease;
}

.live-photo.is-contain .live-photo-image {
  height: auto;
  max-height: 260px;
  object-fit: contain;
  margin: 0 auto;
}

.live-photo.is-contain {
  max-height: 260px;
}

.live-photo-video {
  position: absolute;
  inset: 0;
  opacity: 0;
  pointer-events: none;
  background: #000;
}

.live-photo.is-playing .live-photo-video {
  opacity: 1;
}

.live-photo-badge {
  position: absolute;
  left: 6px;
  bottom: 6px;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 6px 1px 4px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 1.6;
  pointer-events: none;
  backdrop-filter: blur(2px);
}

.live-photo-badge .bi {
  font-size: 9px;
  color: #ffd6a5;
}
</style>
