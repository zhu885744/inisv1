<template>
  <Teleport to="body">
    <Transition name="lightbox-fade">
      <div
        v-if="state.visible"
        class="lightbox-overlay"
        @click.self="close"
      >
        <button
          class="lightbox-close"
          title="关闭"
          @click="close"
        ><i class="bi bi-x-lg" /></button>

        <button
          v-if="state.images.length > 1"
          class="lightbox-nav lightbox-prev"
          title="上一张"
          @click="prev"
        ><i class="bi bi-chevron-left" /></button>

        <div class="lightbox-stage" @click.stop>
          <img
            v-show="!playing"
            :key="state.index"
            :src="current.image"
            class="lightbox-img"
            alt="图片预览"
            loading="lazy"
          />
          <!-- 实况视频：不在打开时自动播放，点了下面的按钮才挂载 -->
          <video
            v-if="playing"
            ref="videoRef"
            :src="current.video"
            class="lightbox-img"
            autoplay
            muted
            loop
            playsinline
            @error="onVideoError"
          />

          <button
            v-if="current.video"
            type="button"
            class="lightbox-live"
            :title="playing ? '暂停实况' : '播放实况'"
            @click.stop="togglePlay"
          >
            <i class="bi" :class="playing ? 'bi-pause-circle' : 'bi-record-circle'" />
            {{ playing ? '暂停实况' : '播放实况' }}
          </button>
        </div>

        <button
          v-if="state.images.length > 1"
          class="lightbox-nav lightbox-next"
          title="下一张"
          @click="next"
        ><i class="bi bi-chevron-right" /></button>

        <div v-if="state.images.length > 1" class="lightbox-counter">
          {{ state.index + 1 }} / {{ state.images.length }}
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * 全局图片灯箱
 *
 * 实况图（动态配图的 图片::视频）在这里**不自动播放**：打开时只显示封面，
 * 右下角给一个「播放实况」按钮，用户点了才挂载 <video>。
 * 这样一页九张实况图也不会一次性拉九段视频，流量和首帧都省。
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { lightboxState, closeLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

const state = lightboxState
const videoRef = ref(null)
/** 实况视频是否在播放（默认关闭，点了按钮才开） */
const playing = ref(false)

const current = computed(() => state.images[state.index] || { image: '', video: '' })

function close() {
  closeLightbox()
}
function prev() {
  state.index = (state.index - 1 + state.images.length) % state.images.length
}
function next() {
  state.index = (state.index + 1) % state.images.length
}

/** 停止实况：卸掉 video 元素，停止下载与播放 */
function stopVideo() {
  playing.value = false
}

function togglePlay() {
  if (!current.value.video) return
  playing.value = !playing.value
}

function onVideoError() {
  playing.value = false
  toast.warning('实况视频加载失败，已切回封面图')
}

// 切换图片 / 关闭灯箱时停掉上一段的视频，避免串台
watch(() => state.index, stopVideo)
watch(() => state.visible, (visible) => {
  if (!visible) stopVideo()
})

function onKeydown(e) {
  if (!state.visible) return
  if (e.key === 'Escape') close()
  else if (e.key === 'ArrowLeft') prev()
  else if (e.key === 'ArrowRight') next()
  else if (e.code === 'Space' && current.value.video) {
    e.preventDefault()
    togglePlay()
  }
}

// 打开时锁定页面滚动
watch(
  () => state.visible,
  (v) => {
    document.body.style.overflow = v ? 'hidden' : ''
  }
)

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<style scoped>
.lightbox-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
}
.lightbox-img {
  max-width: 90vw;
  max-height: 85vh;
  object-fit: contain;
  border-radius: 4px;
  box-shadow: 0 8px 40px rgba(0, 0, 0, 0.5);
  cursor: zoom-out;
}
/* 图片 / 视频共用的舞台，播放按钮贴在它右下角 */
.lightbox-stage {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  max-width: 90vw;
  max-height: 85vh;
}
.lightbox-stage video {
  cursor: default;
}
.lightbox-live {
  position: absolute;
  right: 10px;
  bottom: 10px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border: none;
  border-radius: 14px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.2s;
}
.lightbox-live:hover {
  background: rgba(0, 0, 0, 0.75);
}
.lightbox-live .bi {
  font-size: 14px;
  color: #ffd6a5;
}
.lightbox-close {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  font-size: 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}
.lightbox-close:hover {
  background: rgba(255, 255, 255, 0.25);
}
.lightbox-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
  font-size: 22px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}
.lightbox-nav:hover {
  background: rgba(255, 255, 255, 0.25);
}
.lightbox-prev {
  left: 20px;
}
.lightbox-next {
  right: 20px;
}
.lightbox-counter {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  color: #fff;
  font-size: 13px;
  background: rgba(0, 0, 0, 0.4);
  padding: 4px 12px;
  border-radius: 12px;
}
.lightbox-fade-enter-active,
.lightbox-fade-leave-active {
  transition: opacity 0.25s;
}
.lightbox-fade-enter-active .lightbox-img,
.lightbox-fade-leave-active .lightbox-img {
  transition: transform 0.25s;
}
.lightbox-fade-enter-from,
.lightbox-fade-leave-to {
  opacity: 0;
}
.lightbox-fade-enter-from .lightbox-img,
.lightbox-fade-leave-to .lightbox-img {
  transform: scale(0.95);
}
</style>
