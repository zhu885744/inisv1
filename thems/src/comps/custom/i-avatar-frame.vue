<template>
  <span class="i-avatar-frame" :style="{ width: size, height: size }">
    <!-- 底层圆形用户头像 -->
    <img
      :src="src"
      :alt="alt"
      class="i-avatar-img"
    />
    <!-- 顶层头像框：放大渲染，容纳向外延伸的装饰 -->
    <img
      v-if="frame"
      :src="frame"
      class="i-avatar-frame-overlay"
      :style="{
        transform: `translate(-50%, -50%) scale(${frameScale})`
      }"
    />
  </span>
</template>

<script setup>
defineProps({
  src: { type: String, required: true },
  frame: { type: String, default: '' },
  alt: { type: String, default: '头像' },
  size: { type: String, default: '80px' },
  /**
   * 头像框缩放系数
   * 1 = 和头像一样大
   * >1 放大头像框（推荐1.15 ~ 1.3，根据你的素材调整）
   * 例：1.2 代表头像框整体放大20%
   */
  frameScale: { type: Number, default: 1.2 }
})
</script>

<style scoped>
.i-avatar-frame {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  /* ❗不要加overflow:hidden，否则放大后的头像框装饰会被截断！ */
}

/* 用户头像：占满容器，圆形裁剪 */
.i-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
  display: block;
  z-index: 1;
}

/* 头像框居中放大 */
.i-avatar-frame-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 2;
  object-fit: contain;
  /* 初始居中，scale统一放大 */
  transform-origin: center center;
}
</style>